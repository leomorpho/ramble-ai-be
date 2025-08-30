package payment

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/stripe/stripe-go/v79"
	billingportal "github.com/stripe/stripe-go/v79/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/customer"
	"github.com/stripe/stripe-go/v79/subscription"
	"github.com/stripe/stripe-go/v79/webhook"
)

// StripeProvider implements the Provider interface for Stripe
type StripeProvider struct {
	secretKey     string
	webhookSecret string
}

// NewStripeProvider creates a new Stripe provider
func NewStripeProvider(secretKey, webhookSecret string) Provider {
	stripe.Key = secretKey
	return &StripeProvider{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
	}
}

// GetProviderName returns the provider name
func (s *StripeProvider) GetProviderName() string {
	return "Stripe"
}

// GetProviderType returns the provider type
func (s *StripeProvider) GetProviderType() ProviderType {
	return ProviderStripe
}

// CreateCheckoutSession creates a Stripe checkout session
func (s *StripeProvider) CreateCheckoutSession(params CheckoutSessionParams) (*CheckoutSession, error) {
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

	if params.Mode == "subscription" {
		stripeParams.SubscriptionData = &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"user_id": params.UserID,
				"plan_id": params.PlanID,
			},
		}
	}

	if params.AllowPromoCodes {
		stripeParams.AllowPromotionCodes = stripe.Bool(true)
	}

	session, err := checkoutsession.New(stripeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe checkout session: %w", err)
	}

	return &CheckoutSession{
		ID:         session.ID,
		URL:        session.URL,
		CustomerID: session.Customer.ID,
		Status:     string(session.Status),
		Metadata: map[string]string{
			"user_id": params.UserID,
			"plan_id": params.PlanID,
		},
	}, nil
}

// CreateBillingPortalLink creates a Stripe billing portal link
func (s *StripeProvider) CreateBillingPortalLink(customerID string, returnURL string) (*PortalLink, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	session, err := billingportal.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe billing portal: %w", err)
	}

	return &PortalLink{
		URL: session.URL,
	}, nil
}

// ChangeSubscriptionPlan changes a Stripe subscription plan
func (s *StripeProvider) ChangeSubscriptionPlan(subscriptionID string, newPriceID string, prorationBehavior string) (*Subscription, error) {
	// Get current subscription to find the item ID
	currentSub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get current subscription: %w", err)
	}

	if len(currentSub.Items.Data) == 0 {
		return nil, fmt.Errorf("subscription has no items to update")
	}

	// Update the subscription
	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(currentSub.Items.Data[0].ID),
				Price: stripe.String(newPriceID),
			},
		},
		ProrationBehavior: stripe.String(prorationBehavior),
	}

	updatedSub, err := subscription.Update(subscriptionID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update Stripe subscription: %w", err)
	}

	return s.convertStripeSubscription(updatedSub), nil
}

// CancelSubscription cancels a Stripe subscription
func (s *StripeProvider) CancelSubscription(subscriptionID string, cancelAtPeriodEnd bool) (*Subscription, error) {
	var params *stripe.SubscriptionParams
	var updatedSub *stripe.Subscription
	var err error

	if cancelAtPeriodEnd {
		params = &stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		}
		updatedSub, err = subscription.Update(subscriptionID, params)
	} else {
		cancelParams := &stripe.SubscriptionCancelParams{}
		updatedSub, err = subscription.Cancel(subscriptionID, cancelParams)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to cancel Stripe subscription: %w", err)
	}

	return s.convertStripeSubscription(updatedSub), nil
}

// CreateCustomer creates a Stripe customer
func (s *StripeProvider) CreateCustomer(params CustomerParams) (*Customer, error) {
	stripeParams := &stripe.CustomerParams{
		Email: stripe.String(params.Email),
		Name:  stripe.String(params.Name),
		Metadata: map[string]string{
			"user_id": params.UserID,
		},
	}

	// Add any additional metadata
	for key, value := range params.Metadata {
		stripeParams.Metadata[key] = value
	}

	stripeCustomer, err := customer.New(stripeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	return &Customer{
		ID:       stripeCustomer.ID,
		Email:    stripeCustomer.Email,
		Name:     stripeCustomer.Name,
		Created:  time.Unix(stripeCustomer.Created, 0),
		Metadata: stripeCustomer.Metadata,
	}, nil
}

// GetCustomer retrieves a Stripe customer
func (s *StripeProvider) GetCustomer(customerID string) (*Customer, error) {
	stripeCustomer, err := customer.Get(customerID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe customer: %w", err)
	}

	return &Customer{
		ID:       stripeCustomer.ID,
		Email:    stripeCustomer.Email,
		Name:     stripeCustomer.Name,
		Created:  time.Unix(stripeCustomer.Created, 0),
		Metadata: stripeCustomer.Metadata,
	}, nil
}

// ParseWebhookEvent parses a Stripe webhook event
func (s *StripeProvider) ParseWebhookEvent(payload []byte, signature string) (*WebhookEvent, error) {
	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to verify Stripe webhook signature: %w", err)
	}

	webhookEvent := &WebhookEvent{
		ID:           event.ID,
		Type:         string(event.Type),
		Created:      time.Unix(event.Created, 0),
		ProviderType: ProviderStripe,
		Data:         WebhookEventData{},
	}

	// Parse event data based on type
	switch event.Type {
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		var stripeSub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &stripeSub); err != nil {
			return nil, fmt.Errorf("failed to parse subscription: %w", err)
		}
		webhookEvent.Data.Subscription = s.convertStripeSubscription(&stripeSub)

	case "invoice.payment_succeeded", "invoice.payment_failed", "invoice.payment.paid":
		var stripeInvoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &stripeInvoice); err != nil {
			return nil, fmt.Errorf("failed to parse invoice: %w", err)
		}
		webhookEvent.Data.Invoice = s.convertStripeInvoice(&stripeInvoice)

	case "checkout.session.completed":
		var stripeSession stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &stripeSession); err != nil {
			return nil, fmt.Errorf("failed to parse checkout session: %w", err)
		}
		webhookEvent.Data.CheckoutSession = &CheckoutSession{
			ID:         stripeSession.ID,
			CustomerID: stripeSession.Customer.ID,
			Status:     string(stripeSession.Status),
			Metadata:   stripeSession.Metadata,
		}

	default:
		log.Printf("Unhandled Stripe webhook event type: %s", event.Type)
	}

	return webhookEvent, nil
}

// Helper method to convert Stripe subscription to generic subscription
func (s *StripeProvider) convertStripeSubscription(stripeSub *stripe.Subscription) *Subscription {
	sub := &Subscription{
		ID:                 stripeSub.ID,
		CustomerID:         stripeSub.Customer.ID,
		Status:             s.convertSubscriptionStatus(stripeSub.Status),
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

// Helper method to convert Stripe invoice to generic invoice
func (s *StripeProvider) convertStripeInvoice(stripeInvoice *stripe.Invoice) *Invoice {
	invoice := &Invoice{
		ID:         stripeInvoice.ID,
		CustomerID: stripeInvoice.Customer.ID,
		Status:     string(stripeInvoice.Status),
		Total:      stripeInvoice.Total,
		Currency:   string(stripeInvoice.Currency),
		Metadata:   stripeInvoice.Metadata,
	}

	if stripeInvoice.Subscription != nil {
		subscriptionID := stripeInvoice.Subscription.ID
		invoice.SubscriptionID = &subscriptionID
	}

	if stripeInvoice.StatusTransitions != nil && stripeInvoice.StatusTransitions.PaidAt > 0 {
		paidAt := time.Unix(stripeInvoice.StatusTransitions.PaidAt, 0)
		invoice.PaidAt = &paidAt
	}

	return invoice
}

// Helper method to convert Stripe subscription status to generic status
func (s *StripeProvider) convertSubscriptionStatus(stripeStatus stripe.SubscriptionStatus) SubscriptionStatus {
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