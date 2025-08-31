package payment

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/stripe/stripe-go/v79"
	billingportal "github.com/stripe/stripe-go/v79/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/customer"
	"github.com/stripe/stripe-go/v79/paymentmethod"
	"github.com/stripe/stripe-go/v79/subscription"
	"github.com/stripe/stripe-go/v79/webhook"
)

// NewStripeService creates a new payment service with Stripe provider
func NewStripeService() (*Service, error) {
	secretKey := os.Getenv("STRIPE_SECRET_KEY")
	webhookSecret := os.Getenv("STRIPE_SECRET_WHSEC")
	
	if secretKey == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY environment variable is required")
	}
	
	if webhookSecret == "" {
		log.Printf("Warning: STRIPE_SECRET_WHSEC not set - webhook verification will be disabled")
	}

	// Create Stripe provider using a factory function approach
	provider := newStripeProvider(secretKey, webhookSecret)
	
	// Create payment service with Stripe provider
	config := Config{
		ProviderType:  ProviderStripe,
		SecretKey:     secretKey,
		WebhookSecret: webhookSecret,
	}
	
	return NewService(provider, config), nil
}

// newStripeProvider creates a Stripe provider implementation
func newStripeProvider(secretKey, webhookSecret string) Provider {
	stripe.Key = secretKey
	return &stripeProviderImpl{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
	}
}

// stripeProviderImpl implements the Provider interface for Stripe
type stripeProviderImpl struct {
	secretKey     string
	webhookSecret string
}

// Implement Provider interface methods
func (p *stripeProviderImpl) GetProviderName() string {
	return "Stripe"
}

func (p *stripeProviderImpl) GetProviderType() ProviderType {
	return ProviderStripe
}

func (p *stripeProviderImpl) CreateCheckoutSession(params CheckoutSessionParams) (*CheckoutSession, error) {
	stripeParams := &stripe.CheckoutSessionParams{
		Customer: stripe.String(params.CustomerID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(params.PriceID),
				Quantity: stripe.Int64(params.Quantity),
			},
		},
		Mode:       stripe.String(params.Mode),
		SuccessURL: stripe.String(params.SuccessURL),
		CancelURL:  stripe.String(params.CancelURL),
	}

	if params.AllowPromoCodes {
		stripeParams.AllowPromotionCodes = stripe.Bool(true)
	}

	// Add metadata
	stripeParams.Metadata = map[string]string{
		"user_id": params.UserID,
		"plan_id": params.PlanID,
	}

	session, err := checkoutsession.New(stripeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return &CheckoutSession{
		ID:         session.ID,
		URL:        session.URL,
		CustomerID: session.Customer.ID,
		Status:     string(session.Status),
		Metadata:   session.Metadata,
	}, nil
}

func (p *stripeProviderImpl) CreateBillingPortalLink(customerID string, returnURL string) (*PortalLink, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	session, err := billingportal.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create billing portal link: %w", err)
	}

	return &PortalLink{
		URL: session.URL,
	}, nil
}

func (p *stripeProviderImpl) ChangeSubscriptionPlan(subscriptionID string, newPriceID string, prorationBehavior string) (*Subscription, error) {
	// Get current subscription to modify items
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if len(sub.Items.Data) == 0 {
		return nil, fmt.Errorf("subscription has no items")
	}

	// Update subscription with new price
	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(sub.Items.Data[0].ID),
				Price: stripe.String(newPriceID),
			},
		},
		ProrationBehavior: stripe.String(prorationBehavior),
	}

	updatedSub, err := subscription.Update(subscriptionID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	return p.convertStripeSubscription(updatedSub), nil
}

func (p *stripeProviderImpl) CancelSubscription(subscriptionID string, cancelAtPeriodEnd bool) (*Subscription, error) {
	if cancelAtPeriodEnd {
		// Set to cancel at period end
		params := &stripe.SubscriptionParams{
			}
		updatedSub, err := subscription.Update(subscriptionID, params)
		if err != nil {
			return nil, fmt.Errorf("failed to schedule cancellation: %w", err)
		}
		return p.convertStripeSubscription(updatedSub), nil
	} else {
		// Cancel immediately
		canceledSub, err := subscription.Cancel(subscriptionID, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to cancel subscription: %w", err)
		}
		return p.convertStripeSubscription(canceledSub), nil
	}
}

func (p *stripeProviderImpl) CreateCustomer(params CustomerParams) (*Customer, error) {
	stripeParams := &stripe.CustomerParams{
		Email: stripe.String(params.Email),
		Name:  stripe.String(params.Name),
		Metadata: map[string]string{
			"user_id": params.UserID,
		},
	}

	// Add any additional metadata
	for k, v := range params.Metadata {
		stripeParams.Metadata[k] = v
	}

	cust, err := customer.New(stripeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	return &Customer{
		ID:       cust.ID,
		Email:    cust.Email,
		Name:     cust.Name,
		Created:  time.Unix(cust.Created, 0),
		Metadata: cust.Metadata,
	}, nil
}

func (p *stripeProviderImpl) GetCustomer(customerID string) (*Customer, error) {
	cust, err := customer.Get(customerID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	return &Customer{
		ID:       cust.ID,
		Email:    cust.Email,
		Name:     cust.Name,
		Created:  time.Unix(cust.Created, 0),
		Metadata: cust.Metadata,
	}, nil
}

func (p *stripeProviderImpl) HasValidPaymentMethod(customerID string) (*PaymentMethodStatus, error) {
	// List all payment methods for the customer
	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(customerID),
		Type:     stripe.String("card"), // Focus on card payments for now
	}
	params.Filters.AddFilter("limit", "", "10") // Limit to 10 most recent

	iter := paymentmethod.List(params)
	
	paymentMethods := 0
	var defaultPaymentMethod *string
	var lastUsed *time.Time
	hasValidPaymentMethod := false
	
	// Count payment methods and check their status
	for iter.Next() {
		pm := iter.PaymentMethod()
		paymentMethods++
		
		// Check if this is a valid, non-expired card
		if pm.Card != nil {
			// Card is valid if it's not expired
			currentTime := time.Now()
			if int(pm.Card.ExpYear) > currentTime.Year() || 
			   (int(pm.Card.ExpYear) == currentTime.Year() && int(pm.Card.ExpMonth) >= int(currentTime.Month())) {
				hasValidPaymentMethod = true
				
				// Check if this is the customer's default payment method
				if pm.ID == customerID { // This logic might need adjustment based on how you track default
					defaultPaymentMethod = &pm.ID
				}
			}
		}
		
		// Track the most recent created payment method as "last used"
		if lastUsed == nil || time.Unix(pm.Created, 0).After(*lastUsed) {
			created := time.Unix(pm.Created, 0)
			lastUsed = &created
		}
	}
	
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list payment methods: %w", err)
	}
	
	// Also check if customer has an active subscription (indicates working payment)
	canProcessPayments := hasValidPaymentMethod
	if hasValidPaymentMethod {
		// Check if there are any recent failed payments that would indicate issues
		// For now, we'll assume if they have valid cards, they can process payments
		canProcessPayments = true
	}
	
	return &PaymentMethodStatus{
		HasValidPaymentMethod: hasValidPaymentMethod,
		PaymentMethods:        paymentMethods,
		DefaultPaymentMethod:  defaultPaymentMethod,
		LastUsed:              lastUsed,
		RequiresUpdate:        !hasValidPaymentMethod && paymentMethods > 0, // Has cards but they're expired
		CanProcessPayments:    canProcessPayments,
	}, nil
}

func (p *stripeProviderImpl) ParseWebhookEvent(payload []byte, signature string) (*WebhookEvent, error) {
	// Verify webhook signature
	event, err := webhook.ConstructEventWithOptions(payload, signature, p.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return nil, fmt.Errorf("webhook signature verification failed: %w", err)
	}

	// Create the payment webhook event
	webhookEvent := &WebhookEvent{
		ID:           event.ID,
		Type:         string(event.Type),
		Created:      time.Unix(event.Created, 0),
		ProviderType: ProviderStripe,
		Data:         WebhookEventData{},
	}

	// Parse event data based on type
	// Note: event.Data.Object is map[string]interface{}, we need to parse it safely
	switch event.Type {
	case "customer.created", "customer.updated":
		if data := event.Data.Object; data != nil {
			webhookEvent.Data.Customer = &Customer{
				ID:       getStringFromMap(data, "id"),
				Email:    getStringFromMap(data, "email"),
				Name:     getStringFromMap(data, "name"),
				Metadata: getStringMapFromMap(data, "metadata"),
			}
		}

	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		if data := event.Data.Object; data != nil {
			webhookEvent.Data.Subscription = &Subscription{
				ID:                 getStringFromMap(data, "id"),
				CustomerID:         getStringFromMap(data, "customer"),
				Status:             SubscriptionStatus(getStringFromMap(data, "status")),
				CurrentPeriodStart: time.Unix(getInt64FromMap(data, "current_period_start"), 0),
				CurrentPeriodEnd:   time.Unix(getInt64FromMap(data, "current_period_end"), 0),
				Metadata:           getStringMapFromMap(data, "metadata"),
			}
			
			// Handle optional fields
			if canceledAt := getInt64FromMap(data, "canceled_at"); canceledAt > 0 {
				t := time.Unix(canceledAt, 0)
				webhookEvent.Data.Subscription.CanceledAt = &t
			}
			

			// Get price ID from items
			if items := getMapFromMap(data, "items"); items != nil {
				if itemsData, ok := items["data"].([]interface{}); ok && len(itemsData) > 0 {
					if firstItem, ok := itemsData[0].(map[string]interface{}); ok {
						if price := getMapFromMap(firstItem, "price"); price != nil {
							webhookEvent.Data.Subscription.PriceID = getStringFromMap(price, "id")
						}
					}
				}
			}
		}

	case "checkout.session.completed":
		if data := event.Data.Object; data != nil {
			webhookEvent.Data.CheckoutSession = &CheckoutSession{
				ID:         getStringFromMap(data, "id"),
				URL:        getStringFromMap(data, "url"),
				CustomerID: getStringFromMap(data, "customer"),
				Status:     getStringFromMap(data, "status"),
				Metadata:   getStringMapFromMap(data, "metadata"),
			}
		}

	case "invoice.payment_succeeded", "invoice.payment_failed":
		if data := event.Data.Object; data != nil {
			invoice := &Invoice{
				ID:         getStringFromMap(data, "id"),
				CustomerID: getStringFromMap(data, "customer"),
				Status:     getStringFromMap(data, "status"),
				Total:      getInt64FromMap(data, "total"),
				Currency:   getStringFromMap(data, "currency"),
				Metadata:   getStringMapFromMap(data, "metadata"),
			}
			
			if subscription := getStringFromMap(data, "subscription"); subscription != "" {
				invoice.SubscriptionID = &subscription
			}
			
			if statusTransitions := getMapFromMap(data, "status_transitions"); statusTransitions != nil {
				if paidAtTimestamp := getInt64FromMap(statusTransitions, "paid_at"); paidAtTimestamp > 0 {
					paidAt := time.Unix(paidAtTimestamp, 0)
					invoice.PaidAt = &paidAt
				}
			}
			
			webhookEvent.Data.Invoice = invoice
		}
	}
	
	return webhookEvent, nil
}

// GetStripeHelpers returns Stripe-specific helper functions
// This is a temporary bridge until we fully migrate away from direct Stripe calls
type StripeHelpers struct {
	// Add methods as needed for backward compatibility
}

// GetStripeHelpers returns stripe helper functions for backward compatibility
func (s *Service) GetStripeHelpers() *StripeHelpers {
	return &StripeHelpers{}
}

// Helper method to convert Stripe subscription to generic subscription
func (p *stripeProviderImpl) convertStripeSubscription(stripeSub *stripe.Subscription) *Subscription {
	sub := &Subscription{
		ID:                 stripeSub.ID,
		CustomerID:         stripeSub.Customer.ID,
		Status:             p.convertSubscriptionStatus(stripeSub.Status),
		CurrentPeriodStart: time.Unix(stripeSub.CurrentPeriodStart, 0),
		CurrentPeriodEnd:   time.Unix(stripeSub.CurrentPeriodEnd, 0),
		Metadata:           stripeSub.Metadata,
	}

	if stripeSub.CanceledAt > 0 {
		canceledAt := time.Unix(stripeSub.CanceledAt, 0)
		sub.CanceledAt = &canceledAt
	}


	// Extract price ID from subscription items
	if stripeSub.Items != nil && len(stripeSub.Items.Data) > 0 {
		sub.PriceID = stripeSub.Items.Data[0].Price.ID
	}

	return sub
}

// Helper method to convert Stripe subscription status to generic status
func (p *stripeProviderImpl) convertSubscriptionStatus(stripeStatus stripe.SubscriptionStatus) SubscriptionStatus {
	switch stripeStatus {
	case stripe.SubscriptionStatusActive:
		return SubscriptionStatusActive
	case stripe.SubscriptionStatusCanceled:
		return SubscriptionStatusCanceled
	case stripe.SubscriptionStatusIncomplete:
		return SubscriptionStatusIncomplete
	case stripe.SubscriptionStatusIncompleteExpired:
		return SubscriptionStatusIncompleteExpired
	case stripe.SubscriptionStatusPastDue:
		return SubscriptionStatusPastDue
	case stripe.SubscriptionStatusTrialing:
		return SubscriptionStatusTrialing
	case stripe.SubscriptionStatusUnpaid:
		return SubscriptionStatusUnpaid
	default:
		log.Printf("Unknown Stripe subscription status: %s, defaulting to active", stripeStatus)
		return SubscriptionStatusActive
	}
}

// Helper functions for safely extracting data from map[string]interface{}
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt64FromMap(m map[string]interface{}, key string) int64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return 0
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func getMapFromMap(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key]; ok {
		if mapVal, ok := val.(map[string]interface{}); ok {
			return mapVal
		}
	}
	return nil
}

func getStringMapFromMap(m map[string]interface{}, key string) map[string]string {
	if val, ok := m[key]; ok {
		if mapVal, ok := val.(map[string]interface{}); ok {
			result := make(map[string]string)
			for k, v := range mapVal {
				if str, ok := v.(string); ok {
					result[k] = str
				}
			}
			return result
		}
	}
	return nil
}