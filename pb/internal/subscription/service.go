package subscription

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v79"
)

// Service defines the subscription management interface
type Service interface {
	// Core subscription operations
	CreateSubscription(params CreateSubscriptionParams) (*core.Record, error)
	UpdateSubscription(subscriptionID string, params UpdateSubscriptionParams) (*core.Record, error)
	GetSubscription(subscriptionID string) (*core.Record, error)
	CancelSubscription(userID string) error
	SwitchToFreePlan(userID string) (*core.Record, error)

	// Query operations
	GetUserSubscriptionInfo(userID string) (*SubscriptionInfo, error)
	GetUserActiveSubscription(userID string) (*core.Record, error)
	GetAvailablePlans() ([]*core.Record, error)
	GetPlanUpgrades(userID string) ([]*core.Record, error)

	// Webhook processing
	ProcessWebhookEvent(eventData WebhookEventData) error
	HandleSubscriptionEvent(stripeSub *stripe.Subscription, eventType string) error
	HandlePaymentSucceeded(invoice *stripe.Invoice) error
	HandlePaymentFailed(invoice *stripe.Invoice) error

	// Plan management
	ChangePlan(userID string, newPlanID string, prorationBehavior string) error
	CreateFreePlanSubscription(userID string) error

	// Utility operations
	CleanupDuplicateSubscriptions(userID string) error
	ValidateAndFixSubscriptionTimestamps(subscription *core.Record) (*core.Record, error)
}

// SubscriptionService implements the Service interface
type SubscriptionService struct {
	repo      Repository
	validator *Validator
}

// NewService creates a new subscription service
func NewService(repo Repository) Service {
	validator := NewValidator(repo)
	return &SubscriptionService{
		repo:      repo,
		validator: validator,
	}
}

// CreateSubscription creates a new subscription with validation
func (s *SubscriptionService) CreateSubscription(params CreateSubscriptionParams) (*core.Record, error) {
	// Validate input parameters
	if validationErrors := s.validator.ValidateCreateSubscription(params); len(validationErrors) > 0 {
		return nil, fmt.Errorf("validation failed: %s", validationErrors[0].Message)
	}

	// Check business rules
	if businessErrors := s.validator.ValidateBusinessRules(params.UserID, "create_active"); len(businessErrors) > 0 {
		log.Printf("Business rule violation: %s", businessErrors[0].Message)
		// Clean up existing active subscriptions before creating new one
		if err := s.repo.DeactivateAllUserSubscriptions(params.UserID); err != nil {
			log.Printf("Failed to deactivate existing subscriptions: %v", err)
		}
	}

	// Fix invalid timestamps
	params.CurrentPeriodStart, params.CurrentPeriodEnd = s.validator.FixInvalidTimestamps(params.CurrentPeriodStart, params.CurrentPeriodEnd)

	// Create the subscription
	subscription, err := s.repo.CreateSubscription(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	log.Printf("Created subscription %s for user %s", subscription.Id, params.UserID)
	return subscription, nil
}

// UpdateSubscription updates an existing subscription with validation
func (s *SubscriptionService) UpdateSubscription(subscriptionID string, params UpdateSubscriptionParams) (*core.Record, error) {
	// Validate input parameters
	if validationErrors := s.validator.ValidateUpdateSubscription(subscriptionID, params); len(validationErrors) > 0 {
		return nil, fmt.Errorf("validation failed: %s", validationErrors[0].Message)
	}

	// Fix invalid timestamps if provided
	if params.CurrentPeriodStart != nil && params.CurrentPeriodEnd != nil {
		fixedStart, fixedEnd := s.validator.FixInvalidTimestamps(*params.CurrentPeriodStart, *params.CurrentPeriodEnd)
		params.CurrentPeriodStart = &fixedStart
		params.CurrentPeriodEnd = &fixedEnd
	}

	// Update the subscription
	subscription, err := s.repo.UpdateSubscription(subscriptionID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	log.Printf("Updated subscription %s", subscriptionID)
	return subscription, nil
}

// GetSubscription retrieves a subscription by ID
func (s *SubscriptionService) GetSubscription(subscriptionID string) (*core.Record, error) {
	return s.repo.GetSubscription(subscriptionID)
}

// CancelSubscription cancels a user's active subscription
func (s *SubscriptionService) CancelSubscription(userID string) error {
	activeSubscription, err := s.repo.FindActiveSubscription(userID)
	if err != nil {
		return fmt.Errorf("no active subscription found for user %s: %w", userID, err)
	}

	now := time.Now()
	params := UpdateSubscriptionParams{
		Status:     &[]SubscriptionStatus{StatusCanceled}[0],
		CanceledAt: &now,
	}

	_, err = s.repo.UpdateSubscription(activeSubscription.Id, params)
	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	log.Printf("Cancelled subscription for user %s", userID)
	return nil
}

// SwitchToFreePlan moves a user to the free plan
func (s *SubscriptionService) SwitchToFreePlan(userID string) (*core.Record, error) {
	// Get free plan
	freePlan, err := s.repo.GetFreePlan()
	if err != nil {
		return nil, fmt.Errorf("failed to find free plan: %w", err)
	}

	// Deactivate any existing active subscriptions
	if err := s.repo.DeactivateAllUserSubscriptions(userID); err != nil {
		log.Printf("Warning: Failed to deactivate existing subscriptions: %v", err)
	}

	// Create free subscription
	now := time.Now()
	oneYearFromNow := now.AddDate(1, 0, 0) // Free plan lasts 1 year

	params := CreateSubscriptionParams{
		UserID:                userID,
		PlanID:                freePlan.Id,
		Status:                StatusActive,
		CurrentPeriodStart:    now,
		CurrentPeriodEnd:      oneYearFromNow,
		CancelAtPeriodEnd:     false,
		StripeSubscriptionID:  nil, // Free plan has no Stripe subscription
		StripePriceID:         nil,
		CanceledAt:            &now, // Mark when switched to free
	}

	subscription, err := s.CreateSubscription(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create free subscription: %w", err)
	}

	log.Printf("Switched user %s to free plan", userID)
	return subscription, nil
}

// GetUserSubscriptionInfo retrieves comprehensive subscription information for a user
func (s *SubscriptionService) GetUserSubscriptionInfo(userID string) (*SubscriptionInfo, error) {
	// Get user's active subscription
	subscription, err := s.repo.FindActiveSubscription(userID)
	if err != nil {
		return nil, fmt.Errorf("no subscription found for user %s: %w", userID, err)
	}

	// Get subscription plan
	plan, err := s.repo.GetPlan(subscription.GetString("plan_id"))
	if err != nil {
		return nil, fmt.Errorf("failed to get plan details: %w", err)
	}

	// Get usage information (placeholder - implement according to your usage tracking)
	usage := &UsageInfo{
		HoursUsedThisMonth: 0, // TODO: Implement usage tracking
		HoursLimit:         plan.GetFloat("hours_per_month"),
		FilesProcessed:     0,
		IsOverLimit:        false,
		DaysUntilReset:     0,
	}

	// Get all available plans
	availablePlans, err := s.repo.GetAllPlans()
	if err != nil {
		return nil, fmt.Errorf("failed to get available plans: %w", err)
	}

	return &SubscriptionInfo{
		Subscription:   subscription,
		Plan:          plan,
		Usage:         usage,
		AvailablePlans: availablePlans,
	}, nil
}

// GetUserActiveSubscription retrieves the active subscription for a user
func (s *SubscriptionService) GetUserActiveSubscription(userID string) (*core.Record, error) {
	return s.repo.FindActiveSubscription(userID)
}

// GetAvailablePlans retrieves all available subscription plans
func (s *SubscriptionService) GetAvailablePlans() ([]*core.Record, error) {
	return s.repo.GetAllPlans()
}

// GetPlanUpgrades returns available upgrade options for a user's current plan
func (s *SubscriptionService) GetPlanUpgrades(userID string) ([]*core.Record, error) {
	activeSubscription, err := s.repo.FindActiveSubscription(userID)
	if err != nil {
		return nil, fmt.Errorf("no active subscription found: %w", err)
	}

	currentPlanID := activeSubscription.GetString("plan_id")
	return s.repo.GetAvailableUpgrades(currentPlanID)
}

// ProcessWebhookEvent processes Stripe webhook events
func (s *SubscriptionService) ProcessWebhookEvent(eventData WebhookEventData) error {
	// Validate webhook data
	if validationErrors := s.validator.ValidateStripeWebhookData(eventData); len(validationErrors) > 0 {
		return fmt.Errorf("webhook validation failed: %s", validationErrors[0].Message)
	}

	switch eventData.EventType {
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		return s.HandleSubscriptionEvent(eventData.Subscription, eventData.EventType)
	case "invoice.payment_succeeded", "invoice.payment.paid":
		return s.HandlePaymentSucceeded(eventData.Invoice)
	case "invoice.payment_failed":
		return s.HandlePaymentFailed(eventData.Invoice)
	case "checkout.session.completed":
		// Log but don't process - wait for payment confirmation
		log.Printf("Checkout session completed: %s", eventData.CheckoutSession.ID)
		return nil
	default:
		log.Printf("Unhandled event type: %s", eventData.EventType)
		return nil
	}
}

// HandleSubscriptionEvent processes Stripe subscription lifecycle events
func (s *SubscriptionService) HandleSubscriptionEvent(stripeSub *stripe.Subscription, eventType string) error {
	if stripeSub == nil {
		return fmt.Errorf("stripe subscription data is nil")
	}

	log.Printf("Processing subscription event: %s for subscription %s", eventType, stripeSub.ID)

	// Get user ID from customer (implement this based on your customer mapping)
	userID, err := s.getUserIDFromCustomer(stripeSub.Customer.ID)
	if err != nil {
		return fmt.Errorf("failed to get user ID from customer %s: %w", stripeSub.Customer.ID, err)
	}

	// Handle deletion separately
	if eventType == "customer.subscription.deleted" {
		return s.handleSubscriptionCancellation(userID, stripeSub)
	}

	// Find the subscription plan that matches this Stripe price
	stripePriceID, err := s.validator.ExtractPriceFromSubscription(stripeSub)
	if err != nil {
		return fmt.Errorf("failed to extract price from subscription: %w", err)
	}

	plan, err := s.repo.GetPlanByStripePrice(stripePriceID)
	if err != nil {
		return fmt.Errorf("failed to find subscription plan for price %s: %w", stripePriceID, err)
	}

	// Check if this is a plan change
	existingSubscription, err := s.repo.FindSubscriptionByStripeID(stripeSub.ID)
	if err != nil {
		// No existing subscription - create new one
		return s.createSubscriptionFromStripe(userID, plan.Id, stripeSub, stripePriceID)
	}

	// Update existing subscription
	return s.updateSubscriptionFromStripe(existingSubscription, plan.Id, stripeSub, stripePriceID)
}

// HandlePaymentSucceeded handles successful payment events
func (s *SubscriptionService) HandlePaymentSucceeded(invoice *stripe.Invoice) error {
	if invoice == nil || invoice.Subscription == nil {
		return nil // Not a subscription invoice
	}

	log.Printf("Payment succeeded for subscription: %s", invoice.Subscription.ID)

	// This will trigger a subscription.updated event, so we don't need to do much here
	// Just ensure the subscription exists and is properly updated via the subscription webhook
	return nil
}

// HandlePaymentFailed handles failed payment events
func (s *SubscriptionService) HandlePaymentFailed(invoice *stripe.Invoice) error {
	if invoice == nil || invoice.Subscription == nil {
		return nil // Not a subscription invoice
	}

	// Get user ID from customer
	_, err := s.getUserIDFromCustomer(invoice.Customer.ID)
	if err != nil {
		return err
	}

	// Update subscription status to past_due
	subscription, err := s.repo.FindSubscriptionByStripeID(invoice.Subscription.ID)
	if err != nil {
		return err
	}

	status := StatusPastDue
	params := UpdateSubscriptionParams{
		Status: &status,
	}

	_, err = s.repo.UpdateSubscription(subscription.Id, params)
	return err
}

// ChangePlan changes a user's subscription plan (placeholder - implement Stripe API calls)
func (s *SubscriptionService) ChangePlan(userID string, newPlanID string, prorationBehavior string) error {
	// Validate the plan change request
	if validationErrors := s.validator.ValidatePlanChange(userID, newPlanID); len(validationErrors) > 0 {
		return fmt.Errorf("plan change validation failed: %s", validationErrors[0].Message)
	}

	// TODO: Implement Stripe API call to update subscription
	// This should update the subscription in Stripe, which will trigger a webhook
	// The webhook will then update our database

	log.Printf("Plan change requested for user %s to plan %s", userID, newPlanID)
	return fmt.Errorf("plan change not yet implemented - needs Stripe API integration")
}

// CreateFreePlanSubscription creates a free plan subscription for a new user
func (s *SubscriptionService) CreateFreePlanSubscription(userID string) error {
	// Check if user already has a subscription
	if _, err := s.repo.FindActiveSubscription(userID); err == nil {
		return fmt.Errorf("user already has an active subscription")
	}

	_, err := s.SwitchToFreePlan(userID)
	return err
}

// CleanupDuplicateSubscriptions ensures only one active subscription per user
func (s *SubscriptionService) CleanupDuplicateSubscriptions(userID string) error {
	return s.repo.CleanupDuplicateSubscriptions(userID)
}

// ValidateAndFixSubscriptionTimestamps fixes invalid timestamps in subscription records
func (s *SubscriptionService) ValidateAndFixSubscriptionTimestamps(subscription *core.Record) (*core.Record, error) {
	start := subscription.GetDateTime("current_period_start")
	end := subscription.GetDateTime("current_period_end")

	fixedStart, fixedEnd := s.validator.FixInvalidTimestamps(start.Time(), end.Time())

	params := UpdateSubscriptionParams{
		CurrentPeriodStart: &fixedStart,
		CurrentPeriodEnd:   &fixedEnd,
	}

	return s.repo.UpdateSubscription(subscription.Id, params)
}

// Private helper methods

// getUserIDFromCustomer retrieves the user ID associated with a Stripe customer ID
func (s *SubscriptionService) getUserIDFromCustomer(customerID string) (string, error) {
	// Cast the app to access PocketBase methods
	if pbApp, ok := s.repo.(*PocketBaseRepository); ok {
		record, err := pbApp.app.FindFirstRecordByFilter("stripe_customers", "stripe_customer_id = {:customer_id}", map[string]any{
			"customer_id": customerID,
		})
		if err != nil {
			return "", fmt.Errorf("customer mapping not found for %s: %w", customerID, err)
		}
		return record.GetString("user_id"), nil
	}
	return "", fmt.Errorf("unsupported repository type for customer mapping")
}

// handleSubscriptionCancellation handles subscription deletion
func (s *SubscriptionService) handleSubscriptionCancellation(userID string, stripeSub *stripe.Subscription) error {
	log.Printf("Handling subscription cancellation for user %s", userID)

	// Move user to free plan
	_, err := s.SwitchToFreePlan(userID)
	if err != nil {
		return fmt.Errorf("failed to switch user %s to free plan: %w", userID, err)
	}
	return nil
}

// getPlanByStripePrice retrieves a plan by Stripe price ID using the repository
func (s *SubscriptionService) getPlanByStripePrice(stripePriceID string) (*core.Record, error) {
	if pbRepo, ok := s.repo.(*PocketBaseRepository); ok {
		return pbRepo.GetPlanByStripePrice(stripePriceID)
	}
	return nil, fmt.Errorf("unsupported repository type for plan lookup")
}

// extractPriceFromSubscription safely extracts the price ID from a Stripe subscription
func (s *SubscriptionService) extractPriceFromSubscription(sub *stripe.Subscription) (string, error) {
	return s.validator.ExtractPriceFromSubscription(sub)
}

// createSubscriptionFromStripe creates a new subscription from Stripe data
func (s *SubscriptionService) createSubscriptionFromStripe(userID, planID string, stripeSub *stripe.Subscription, stripePriceID string) error {
	// Deactivate any existing active subscriptions first
	if err := s.repo.DeactivateAllUserSubscriptions(userID); err != nil {
		log.Printf("Failed to deactivate existing subscriptions: %v", err)
	}

	status := s.validator.MapStripeStatus(stripeSub.Status)
	start := time.Unix(stripeSub.CurrentPeriodStart, 0)
	end := time.Unix(stripeSub.CurrentPeriodEnd, 0)

	// Fix invalid timestamps
	start, end = s.validator.FixInvalidTimestamps(start, end)

	params := CreateSubscriptionParams{
		UserID:               userID,
		PlanID:               planID,
		StripeSubscriptionID: &stripeSub.ID,
		StripePriceID:        &stripePriceID,
		Status:               status,
		CurrentPeriodStart:   start,
		CurrentPeriodEnd:     end,
		CancelAtPeriodEnd:    stripeSub.CancelAtPeriodEnd,
	}

	if stripeSub.CanceledAt > 0 {
		canceledAt := time.Unix(stripeSub.CanceledAt, 0)
		params.CanceledAt = &canceledAt
	}
	if stripeSub.TrialEnd > 0 {
		trialEnd := time.Unix(stripeSub.TrialEnd, 0)
		params.TrialEnd = &trialEnd
	}

	_, err := s.CreateSubscription(params)
	return err
}

// updateSubscriptionFromStripe updates an existing subscription with Stripe data
func (s *SubscriptionService) updateSubscriptionFromStripe(subscription *core.Record, planID string, stripeSub *stripe.Subscription, stripePriceID string) error {
	status := s.validator.MapStripeStatus(stripeSub.Status)
	start := time.Unix(stripeSub.CurrentPeriodStart, 0)
	end := time.Unix(stripeSub.CurrentPeriodEnd, 0)

	// Fix invalid timestamps
	start, end = s.validator.FixInvalidTimestamps(start, end)

	params := UpdateSubscriptionParams{
		PlanID:             &planID,
		StripePriceID:      &stripePriceID,
		Status:             &status,
		CurrentPeriodStart: &start,
		CurrentPeriodEnd:   &end,
		CancelAtPeriodEnd:  &stripeSub.CancelAtPeriodEnd,
	}

	if stripeSub.CanceledAt > 0 {
		canceledAt := time.Unix(stripeSub.CanceledAt, 0)
		params.CanceledAt = &canceledAt
	}
	if stripeSub.TrialEnd > 0 {
		trialEnd := time.Unix(stripeSub.TrialEnd, 0)
		params.TrialEnd = &trialEnd
	}

	_, err := s.repo.UpdateSubscription(subscription.Id, params)
	return err
}