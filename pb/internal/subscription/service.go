package subscription

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/subscription"
)

// CancelSubscriptionResult represents the result of a subscription cancellation
type CancelSubscriptionResult struct {
	Success               bool      `json:"success"`
	Message               string    `json:"message"`
	CancellationScheduled bool      `json:"cancellation_scheduled"`
	PeriodEndDate         time.Time `json:"period_end_date"`
	BenefitsPreserved     bool      `json:"benefits_preserved"`
}

// Service defines the subscription management interface
type Service interface {
	// Core subscription operations
	CreateSubscription(params CreateSubscriptionParams) (*core.Record, error)
	UpdateSubscription(subscriptionID string, params UpdateSubscriptionParams) (*core.Record, error)
	GetSubscription(subscriptionID string) (*core.Record, error)
	CancelSubscription(userID string) (*CancelSubscriptionResult, error)
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
	stripe    StripeService
}

// NewService creates a new subscription service with real Stripe integration
func NewService(repo Repository) Service {
	validator := NewValidator(repo)
	return &SubscriptionService{
		repo:      repo,
		validator: validator,
		stripe:    NewRealStripeService(),
	}
}

// NewServiceWithStripe creates a new subscription service with custom Stripe service (for testing)
func NewServiceWithStripe(repo Repository, stripeService StripeService) Service {
	validator := NewValidator(repo)
	return &SubscriptionService{
		repo:      repo,
		validator: validator,
		stripe:    stripeService,
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
// CancelSubscription immediately cancels a user's active subscription
// User is moved to free plan with prorated refunds handled by Stripe
func (s *SubscriptionService) CancelSubscription(userID string) (*CancelSubscriptionResult, error) {
	// Find user's active subscription
	activeSubscription, err := s.repo.FindActiveSubscription(userID)
	if err != nil {
		return nil, fmt.Errorf("no active subscription found for user %s: %w", userID, err)
	}

	// Get Stripe subscription ID
	stripeSubID := activeSubscription.GetString("provider_subscription_id")
	if stripeSubID == "" {
		return nil, fmt.Errorf("subscription %s has no Stripe subscription ID", activeSubscription.Id)
	}

	log.Printf("Cancelling Stripe subscription %s for user %s immediately", stripeSubID, userID)

	// Cancel subscription immediately in Stripe - Stripe handles prorated refunds
	_, err = subscription.Cancel(stripeSubID, &stripe.SubscriptionCancelParams{
		Prorate: stripe.Bool(true), // Ensure user gets prorated refund
	})
	if err != nil {
		return nil, fmt.Errorf("failed to cancel Stripe subscription: %w", err)
	}

	// Immediately switch user to free plan
	_, err = s.SwitchToFreePlan(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to switch user to free plan: %w", err)
	}

	log.Printf("Successfully cancelled subscription for user %s - switched to free plan with prorated refund", userID)

	return &CancelSubscriptionResult{
		Success:               true,
		Message:               "Subscription cancelled successfully with prorated refund",
		CancellationScheduled: false,
		PeriodEndDate:         time.Now(), // Immediate cancellation
		BenefitsPreserved:     false,      // No period-end preservation
	}, nil
}

// SwitchToFreePlan moves a user to the free plan
func (s *SubscriptionService) SwitchToFreePlan(userID string) (*core.Record, error) {
	// Move any existing active subscriptions to history first
	existingSubscriptions, err := s.repo.FindAllUserSubscriptions(userID)
	if err != nil {
		log.Printf("Warning: Failed to find existing subscriptions: %v", err)
	} else {
		for _, existingSub := range existingSubscriptions {
			if existingSub.GetString("status") == "active" {
				_, err := s.repo.MoveSubscriptionToHistory(existingSub, "switched_to_free_plan")
				if err != nil {
					log.Printf("Warning: Failed to move subscription %s to history: %v", existingSub.Id, err)
				}
				// Delete the current subscription after moving to history
				if err := s.repo.DeleteSubscription(existingSub.Id); err != nil {
					log.Printf("Warning: Failed to delete subscription during free plan switch: %v", err)
				}
			}
		}
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
		// No active subscription found - user should be on free plan
		log.Printf("No subscription found for user %s, assigning to free plan", userID)
		
		// Automatically assign user to free plan
		freeSubscription, freeErr := s.SwitchToFreePlan(userID)
		if freeErr != nil {
			return nil, fmt.Errorf("no subscription found for user %s and failed to assign free plan: %w", userID, freeErr)
		}
		subscription = freeSubscription
	}

	// Determine which plan to use for benefits/limits
	// CRITICAL FIX: For downgrades, user keeps current plan until period ends
	planID := subscription.GetString("plan_id")
	
	// With immediate plan changes, planID is always the current active plan
	// No complex pending logic needed

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

	// Simplified: Just sync whatever Stripe tells us - all plan changes are immediate
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

	// Find the subscription to cancel
	subscription, err := s.repo.FindSubscriptionByProviderID(stripeSub.ID)
	if err != nil {
		log.Printf("Warning: Could not find subscription to cancel: %v", err)
		// Still continue to ensure user is on free plan
	} else {
		// Move subscription to history and delete it
		_, err := s.repo.MoveSubscriptionToHistory(subscription, "subscription_cancelled")
		if err != nil {
			log.Printf("Warning: Failed to move cancelled subscription to history: %v", err)
		}
		
		// Delete the current subscription
		if err := s.repo.DeleteSubscription(subscription.Id); err != nil {
			log.Printf("Warning: Failed to delete cancelled subscription: %v", err)
		}
		
		log.Printf("User %s moved to free plan after subscription cancellation", userID)
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
	return s.createSubscriptionFromStripeInternal(userID, planID, stripeSub, stripePriceID, true)
}

// createSubscriptionFromStripeInternal creates a new subscription with option to move existing to history
func (s *SubscriptionService) createSubscriptionFromStripeInternal(userID, planID string, stripeSub *stripe.Subscription, stripePriceID string, moveExistingToHistory bool) error {
	if moveExistingToHistory {
		// Move any existing active subscriptions to history instead of just deactivating
		existingSubscriptions, err := s.repo.FindAllUserSubscriptions(userID)
		if err != nil {
			log.Printf("Warning: Failed to find existing subscriptions: %v", err)
		} else {
			for _, existingSub := range existingSubscriptions {
				if existingSub.GetString("status") == "active" {
					_, err := s.repo.MoveSubscriptionToHistory(existingSub, "replaced_by_new_subscription")
					if err != nil {
						log.Printf("Warning: Failed to move subscription %s to history: %v", existingSub.Id, err)
					}
					// Delete the current subscription after moving to history
					if err := s.repo.DeleteSubscription(existingSub.Id); err != nil {
						log.Printf("Warning: Failed to delete replaced subscription: %v", err)
					}
				}
			}
		}
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
	}

	if stripeSub.CanceledAt > 0 {
		canceledAt := time.Unix(stripeSub.CanceledAt, 0)
		params.CanceledAt = &canceledAt
	}

	_, err := s.CreateSubscription(params)
	return err
}

// updateSubscriptionFromStripe updates an existing subscription with Stripe data
func (s *SubscriptionService) updateSubscriptionFromStripe(subscription *core.Record, planID string, stripeSub *stripe.Subscription, stripePriceID string) error {
	// Check if this is a significant change that requires moving to history
	currentPlanID := subscription.GetString("plan_id")
	if currentPlanID != planID {
		log.Printf("Plan change detected: moving subscription %s to history (plan %s -> %s)", subscription.Id, currentPlanID, planID)
		// Move current subscription to history before creating/updating with new plan
		_, err := s.repo.MoveSubscriptionToHistory(subscription, "plan_change")
		if err != nil {
			log.Printf("Warning: Failed to move subscription to history: %v", err)
			// Continue with update even if history move fails
		}
		
		// Delete the current subscription record
		if err := s.repo.DeleteSubscription(subscription.Id); err != nil {
			log.Printf("Warning: Failed to delete current subscription: %v", err)
		}
		
		// Create new subscription record with the new plan
		return s.createSubscriptionFromStripeInternal(subscription.GetString("user_id"), planID, stripeSub, stripePriceID, false)
	}

	// If no plan change, just update the existing record
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
	}

	if stripeSub.CanceledAt > 0 {
		canceledAt := time.Unix(stripeSub.CanceledAt, 0)
		params.CanceledAt = &canceledAt
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
	}

	if stripeSub.CanceledAt > 0 {
		canceledAt := time.Unix(stripeSub.CanceledAt, 0)
		params.CanceledAt = &canceledAt
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

	// Simplified: All plan changes are immediate with Stripe prorations
	stripePriceID := targetPlan.GetString("provider_price_id")
	if stripePriceID == "" {
		return nil, fmt.Errorf("target plan has no Stripe price ID")
	}

	log.Printf("Processing immediate plan change: %s -> %s", currentPlan.GetString("name"), targetPlan.GetString("name"))

	// Update Stripe subscription immediately - Stripe handles prorations
	err = s.updateStripeSubscription(stripeSubID, stripePriceID)
	if err != nil {
		return nil, fmt.Errorf("failed to update Stripe subscription: %w", err)
	}

	// Update local database immediately to match the Stripe change
	_, err = s.repo.UpdateSubscription(currentSub.Id, UpdateSubscriptionParams{
		PlanID:          &newPlanID,
		ProviderPriceID: &stripePriceID,
	})
	if err != nil {
		log.Printf("Warning: Stripe updated successfully but local database update failed: %v", err)
		// Don't fail the request since Stripe succeeded - webhook will eventually sync
	}
	changeType := "upgrade"
	if !isUpgrade {
		changeType = "downgrade"
	}

	return &ChangePlanResult{
		Success:       true,
		Message:       fmt.Sprintf("Plan changed to %s - changes take effect immediately", targetPlan.GetString("name")),
		ChangeType:    changeType,
		NewPlan:       targetPlan.Id,
		EffectiveDate: "immediately",
		PendingChange: false,
	}, nil
}



// updateStripeSubscription immediately updates a Stripe subscription price with prorations
func (s *SubscriptionService) updateStripeSubscription(subID string, priceID string) error {
	log.Printf("Updating Stripe subscription %s to priceID=%s (immediate with prorations)", subID, priceID)
	return s.stripe.UpdateSubscription(subID, priceID)
}

func (s *SubscriptionService) getStripeSubscription(subID string) (*stripe.Subscription, error) {
	return s.stripe.GetSubscription(subID)
}


// Helper functions for pointer types
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}