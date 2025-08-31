package subscription

import (
	"fmt"
	
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/subscription"
)

// StripeService interface for Stripe operations (enables dependency injection)
type StripeService interface {
	UpdateSubscription(subID string, priceID string) error
	GetSubscription(subID string) (*stripe.Subscription, error)
}

// RealStripeService implements StripeService using actual Stripe API
type RealStripeService struct{}

// NewRealStripeService creates a new real Stripe service
func NewRealStripeService() StripeService {
	return &RealStripeService{}
}

// UpdateSubscription immediately updates a Stripe subscription price with prorations
func (s *RealStripeService) UpdateSubscription(subID string, priceID string) error {
	if priceID == "" {
		return fmt.Errorf("price ID is required")
	}
	
	// Get current subscription to access the subscription item
	currentStripeSub, err := subscription.Get(subID, nil)
	if err != nil {
		return err
	}
	
	if len(currentStripeSub.Items.Data) == 0 {
		return fmt.Errorf("no subscription items found")
	}
	
	// Update the price immediately with prorations
	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(currentStripeSub.Items.Data[0].ID),
				Price: stripe.String(priceID),
			},
		},
		ProrationBehavior: stripe.String("always_invoice"), // Handle prorations immediately
	}
	
	_, err = subscription.Update(subID, params)
	return err
}

// GetSubscription retrieves a Stripe subscription
func (s *RealStripeService) GetSubscription(subID string) (*stripe.Subscription, error) {
	return subscription.Get(subID, nil)
}

// MockStripeService implements StripeService for testing
type MockStripeService struct {
	// Track method calls for test assertions
	UpdateCalls []MockUpdateCall
	GetCalls    []string
	// Control return values
	UpdateError error
	GetError    error
	GetResult   *stripe.Subscription
}

// MockUpdateCall represents a call to UpdateSubscription for testing
type MockUpdateCall struct {
	SubID   string
	PriceID string
}

// NewMockStripeService creates a new mock Stripe service for testing
func NewMockStripeService() *MockStripeService {
	return &MockStripeService{
		UpdateCalls: []MockUpdateCall{},
		GetCalls:    []string{},
	}
}

// UpdateSubscription mocks updating a Stripe subscription
func (m *MockStripeService) UpdateSubscription(subID string, priceID string) error {
	// Record the call
	m.UpdateCalls = append(m.UpdateCalls, MockUpdateCall{
		SubID:   subID,
		PriceID: priceID,
	})
	
	// Return configured error if any
	return m.UpdateError
}

// GetSubscription mocks retrieving a Stripe subscription
func (m *MockStripeService) GetSubscription(subID string) (*stripe.Subscription, error) {
	// Record the call
	m.GetCalls = append(m.GetCalls, subID)
	
	// Return configured error if any
	if m.GetError != nil {
		return nil, m.GetError
	}
	
	// Return configured result or a default mock
	if m.GetResult != nil {
		return m.GetResult, nil
	}
	
	// Default mock subscription
	return &stripe.Subscription{
		ID: subID,
		CurrentPeriodEnd: 1725091200, // Mock timestamp
	}, nil
}