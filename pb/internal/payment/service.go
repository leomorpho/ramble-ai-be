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
			CancelAtPeriodEnd: stripe.Bool(true),
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

	// For now, return basic event - proper parsing would need to be implemented
	// TODO: Parse event data based on type like in stripe/webhooks.go
	
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
		CancelAtPeriodEnd:  stripeSub.CancelAtPeriodEnd,
		Metadata:           stripeSub.Metadata,
	}

	if stripeSub.CanceledAt > 0 {
		canceledAt := time.Unix(stripeSub.CanceledAt, 0)
		sub.CanceledAt = &canceledAt
	}

	if stripeSub.TrialEnd > 0 {
		trialEnd := time.Unix(stripeSub.TrialEnd, 0)
		sub.TrialEnd = &trialEnd
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