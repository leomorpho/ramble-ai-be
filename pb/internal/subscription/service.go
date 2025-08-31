package subscription

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/subscription"
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
	ChangePlan(userID string, newPlanID string) (*ChangePlanResult, error)
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
	// Deactivate any existing active subscriptions first
	if err := s.repo.DeactivateAllUserSubscriptions(userID); err != nil {
		log.Printf("Warning: Failed to deactivate existing subscriptions: %v", err)
		// Continue anyway - important to not have active subscriptions
	}

	// Get the free plan
	freePlan, err := s.repo.GetFreePlan()
	if err != nil {
		return nil, fmt.Errorf("failed to get free plan: %w", err)
	}

	// Create a new subscription record for the free plan
	now := time.Now()
	paymentProvider := "stripe"
	params := CreateSubscriptionParams{
		UserID:                userID,
		PlanID:                freePlan.Id,
		Status:                StatusActive,
		CurrentPeriodStart:    now,
		CurrentPeriodEnd:      now.AddDate(1, 0, 0), // Free plan active for 1 year
		CancelAtPeriodEnd:     false,
		ProviderSubscriptionID: nil, // No Stripe subscription for free plan
		ProviderPriceID:       nil, // No Stripe price for free plan
		PaymentProvider:       &paymentProvider, // Consistent with other plans
	}

	record, err := s.repo.CreateSubscription(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create free plan subscription: %w", err)
	}

	log.Printf("User %s switched to free plan", userID)
	return record, nil
}

// GetUserSubscriptionInfo retrieves comprehensive subscription information for a user
func (s *SubscriptionService) GetUserSubscriptionInfo(userID string) (*SubscriptionInfo, error) {
	// Get user's active subscription
	subscription, err := s.repo.FindActiveSubscription(userID)
	if err != nil {
		return nil, fmt.Errorf("no subscription found for user %s: %w", userID, err)
	}

	// Determine which plan to use for benefits/limits
	// CRITICAL FIX: For downgrades, user keeps current plan until period ends
	planID := subscription.GetString("plan_id")
	
	// If user has a pending downgrade and period hasn't ended, they keep current benefits
	if subscription.GetBool("cancel_at_period_end") && 
		subscription.GetString("pending_plan_id") != "" &&
		subscription.GetDateTime("current_period_end").Time().After(time.Now()) {
		// User keeps current plan benefits until period actually ends
		planID = subscription.GetString("plan_id")
	}

	// Get plan details (this determines user's current benefits/limits)
	plan, err := s.repo.GetPlan(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan details: %w", err)
	}

	// Get usage information based on plan limits
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

	plan, err := s.repo.GetPlanByProviderPrice(stripePriceID)
	if err != nil {
		return fmt.Errorf("failed to find subscription plan for price %s: %w", stripePriceID, err)
	}

	// Check if this is a plan change
	existingSubscription, err := s.repo.FindSubscriptionByProviderID(stripeSub.ID)
	if err != nil {
		// No existing subscription - create new one
		return s.createSubscriptionFromStripe(userID, plan.Id, stripeSub, stripePriceID)
	}

	// CRITICAL: Check if subscription is set to cancel at period end
	// If so, preserve current plan but store new plan as pending
	if stripeSub.CancelAtPeriodEnd {
		log.Printf("Subscription %s is set to cancel at period end - preserving current plan until %v, storing new plan %s as pending", 
			stripeSub.ID, time.Unix(stripeSub.CurrentPeriodEnd, 0), plan.Id)
		
		// Preserve current plan but store the new (lower) plan as pending
		return s.updateSubscriptionWithPendingPlan(existingSubscription, plan.Id, stripeSub)
	}

	// Update existing subscription (plan can change immediately if not canceling at period end)
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
	subscription, err := s.repo.FindSubscriptionByProviderID(invoice.Subscription.ID)
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

// This old ChangePlan method has been replaced with the new implementation below

// CreateFreePlanSubscription ensures a user is on the free plan (no subscription record needed)
func (s *SubscriptionService) CreateFreePlanSubscription(userID string) error {
	// For free plan, we simply ensure the user has no active subscription
	// Check if user already has a subscription and deactivate it
	if _, err := s.repo.FindActiveSubscription(userID); err == nil {
		// User has active subscription, switch them to free (deactivate it)
		_, err := s.SwitchToFreePlan(userID)
		return err
	}
	
	// No active subscription - user is already on free plan
	return nil
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
		record, err := pbApp.app.FindFirstRecordByFilter("payment_customers", "provider_customer_id = {:customer_id}", map[string]any{
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
		return pbRepo.GetPlanByProviderPrice(stripePriceID)
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
		ProviderSubscriptionID: &stripeSub.ID,
		ProviderPriceID:        &stripePriceID,
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
		ProviderPriceID:      &stripePriceID,
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

// updateSubscriptionMetadataOnly updates subscription metadata without changing the plan
// This is used when a subscription is set to cancel at period end - we preserve the current plan
// until the billing period actually ends
func (s *SubscriptionService) updateSubscriptionMetadataOnly(subscription *core.Record, stripeSub *stripe.Subscription) error {
	status := s.validator.MapStripeStatus(stripeSub.Status)
	start := time.Unix(stripeSub.CurrentPeriodStart, 0)
	end := time.Unix(stripeSub.CurrentPeriodEnd, 0)

	// Fix invalid timestamps
	start, end = s.validator.FixInvalidTimestamps(start, end)

	// CRITICAL: Do NOT update PlanID or ProviderPriceID when cancel_at_period_end is true
	// The user should keep their current plan benefits until the period ends
	params := UpdateSubscriptionParams{
		// PlanID and ProviderPriceID are intentionally omitted - preserve current plan
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

	log.Printf("Updating subscription metadata only (preserving current plan) for subscription %s", subscription.Id)
	_, err := s.repo.UpdateSubscription(subscription.Id, params)
	return err
}

// ChangePlan handles plan changes through the service layer (SINGLE ENTRY POINT)
func (s *SubscriptionService) ChangePlan(userID string, newPlanID string) (*ChangePlanResult, error) {
	log.Printf("Processing plan change for user %s to plan %s", userID, newPlanID)

	// Get user's current active subscription
	currentSub, err := s.repo.FindActiveSubscription(userID)
	if err != nil {
		return nil, fmt.Errorf("no active subscription found for user %s: %w", userID, err)
	}

	// Get current and target plan details
	currentPlan, err := s.repo.GetPlan(currentSub.GetString("plan_id"))
	if err != nil {
		return nil, fmt.Errorf("failed to get current plan: %w", err)
	}

	targetPlan, err := s.repo.GetPlan(newPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target plan: %w", err)
	}

	// Validate the plan change
	if validationErrors := s.validator.ValidatePlanChange(userID, newPlanID); len(validationErrors) > 0 {
		return nil, fmt.Errorf("plan change validation failed: %s", validationErrors[0].Message)
	}

	// Determine if this is an upgrade or downgrade
	currentPrice := int64(currentPlan.GetInt("price_cents"))
	targetPrice := int64(targetPlan.GetInt("price_cents"))
	isUpgrade := targetPrice > currentPrice

	log.Printf("Plan change: %s (%d cents) -> %s (%d cents), isUpgrade: %v",
		currentPlan.GetString("name"), currentPrice,
		targetPlan.GetString("name"), targetPrice, isUpgrade)

	// Get Stripe subscription ID
	stripeSubID := currentSub.GetString("provider_subscription_id")
	if stripeSubID == "" {
		return nil, fmt.Errorf("no Stripe subscription ID found for user %s", userID)
	}

	// Handle upgrades vs downgrades differently
	if isUpgrade {
		return s.handleUpgrade(currentSub, targetPlan, stripeSubID)
	} else {
		return s.handleDowngrade(currentSub, targetPlan, stripeSubID)
	}
}

// handleUpgrade processes immediate plan upgrades with proration
func (s *SubscriptionService) handleUpgrade(currentSub *core.Record, targetPlan *core.Record, stripeSubID string) (*ChangePlanResult, error) {
	log.Printf("Processing UPGRADE: applying immediately with proration")

	stripePriceID := targetPlan.GetString("provider_price_id")
	if stripePriceID == "" {
		return nil, fmt.Errorf("target plan has no Stripe price ID")
	}

	// Update Stripe subscription immediately
	err := s.updateStripeSubscription(stripeSubID, stripePriceID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to update Stripe subscription: %w", err)
	}

	// Update the current subscription record (not creating new one!)
	params := UpdateSubscriptionParams{
		PlanID:           &targetPlan.Id,
		ProviderPriceID:  &stripePriceID,
		PendingPlanID:    stringPtr(""), // Clear any pending plan
	}

	_, err = s.repo.UpdateSubscription(currentSub.Id, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription record: %w", err)
	}

	return &ChangePlanResult{
		Success:       true,
		Message:       "Upgrade applied immediately - you now have access to enhanced features!",
		ChangeType:    "upgrade",
		NewPlan:       targetPlan.Id,
		EffectiveDate: "immediately",
	}, nil
}

// handleDowngrade processes plan downgrades with period-end preservation
func (s *SubscriptionService) handleDowngrade(currentSub *core.Record, targetPlan *core.Record, stripeSubID string) (*ChangePlanResult, error) {
	log.Printf("Processing DOWNGRADE: preserving current benefits until period end")

	// Set cancel_at_period_end=true in Stripe
	err := s.updateStripeSubscription(stripeSubID, "", true)
	if err != nil {
		return nil, fmt.Errorf("failed to schedule downgrade in Stripe: %w", err)
	}

	// Get current period end from Stripe for user display
	updatedSub, err := s.getStripeSubscription(stripeSubID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated Stripe subscription: %w", err)
	}

	effectiveDate := time.Unix(updatedSub.CurrentPeriodEnd, 0)
	
	// Update the current subscription record with pending plan info
	params := UpdateSubscriptionParams{
		PendingPlanID:             &targetPlan.Id,
		CancelAtPeriodEnd:         boolPtr(true),
	}

	_, err = s.repo.UpdateSubscription(currentSub.Id, params)
	if err != nil {
		return nil, fmt.Errorf("failed to store pending plan change: %w", err)
	}

	return &ChangePlanResult{
		Success:       true,
		Message:       fmt.Sprintf("Downgrade scheduled - you'll keep your current benefits until %s, then switch to %s", effectiveDate.Format("January 2, 2006"), targetPlan.GetString("name")),
		ChangeType:    "downgrade",
		NewPlan:       targetPlan.Id,
		EffectiveDate: effectiveDate.Format("January 2, 2006"),
		PendingChange: true,
	}, nil
}

// updateStripeSubscription updates a Stripe subscription
func (s *SubscriptionService) updateStripeSubscription(subID string, priceID string, cancelAtPeriodEnd bool) error {
	params := &stripe.SubscriptionParams{}
	
	if cancelAtPeriodEnd {
		// For downgrades: set cancel_at_period_end=true
		params.CancelAtPeriodEnd = stripe.Bool(true)
		log.Printf("Setting cancel_at_period_end=true for subscription %s", subID)
	} else if priceID != "" {
		// For upgrades: update the subscription item with new price
		currentStripeSub, err := subscription.Get(subID, nil)
		if err != nil {
			return fmt.Errorf("failed to get current subscription from Stripe: %w", err)
		}
		
		if len(currentStripeSub.Items.Data) == 0 {
			return fmt.Errorf("current subscription has no items to update")
		}
		
		params.Items = []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(currentStripeSub.Items.Data[0].ID),
				Price: stripe.String(priceID),
			},
		}
		params.ProrationBehavior = stripe.String("always_invoice")
		log.Printf("Updating subscription %s to price %s with immediate proration", subID, priceID)
	}
	
	_, err := subscription.Update(subID, params)
	return err
}

func (s *SubscriptionService) getStripeSubscription(subID string) (*stripe.Subscription, error) {
	return subscription.Get(subID, nil)
}

// updateSubscriptionWithPendingPlan updates subscription with pending plan info when cancel_at_period_end=true
// This preserves the current plan benefits while storing the new plan to apply at period end
func (s *SubscriptionService) updateSubscriptionWithPendingPlan(subscription *core.Record, pendingPlanID string, stripeSub *stripe.Subscription) error {
	status := s.validator.MapStripeStatus(stripeSub.Status)
	start := time.Unix(stripeSub.CurrentPeriodStart, 0)
	end := time.Unix(stripeSub.CurrentPeriodEnd, 0)
	
	// Fix invalid timestamps
	start, end = s.validator.FixInvalidTimestamps(start, end)
	
	log.Printf("Preserving current plan for user, storing pending plan %s for period end", pendingPlanID)
	
	// CRITICAL: Do NOT update PlanID or ProviderPriceID - preserve current benefits
	// Only update metadata and store the pending plan info
	params := UpdateSubscriptionParams{
		Status:              &status,
		CurrentPeriodStart:  &start,
		CurrentPeriodEnd:    &end,
		CancelAtPeriodEnd:   boolPtr(stripeSub.CancelAtPeriodEnd),
		PendingPlanID:       &pendingPlanID, // Store the new plan to apply later
	}

	if stripeSub.CanceledAt > 0 {
		canceledAt := time.Unix(stripeSub.CanceledAt, 0)
		params.CanceledAt = &canceledAt
	}
	if stripeSub.TrialEnd > 0 {
		trialEnd := time.Unix(stripeSub.TrialEnd, 0)
		params.TrialEnd = &trialEnd
	}

	log.Printf("Updating subscription metadata only (preserving current plan) for subscription %s", subscription.Id)
	_, err := s.repo.UpdateSubscription(subscription.Id, params)
	return err
}

// Helper functions for pointer types
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}