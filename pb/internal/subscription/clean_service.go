package subscription

import (
	"fmt"
	"log"
	"time"

	"github.com/stripe/stripe-go/v79"
)

// CleanSubscriptionService handles subscription business logic using domain models
// This is the new clean version that works with domain structs instead of PocketBase records
type CleanSubscriptionService struct {
	store     SubscriptionStore
	validator *CleanValidator
}

// NewCleanSubscriptionService creates a new service with dependency injection
func NewCleanSubscriptionService(store SubscriptionStore, validator *CleanValidator) *CleanSubscriptionService {
	return &CleanSubscriptionService{
		store:     store,
		validator: validator,
	}
}

// ProcessCancellationWebhook handles the critical cancellation logic that was failing
func (s *CleanSubscriptionService) ProcessCancellationWebhook(userID string, stripeSub *stripe.Subscription) error {
	log.Printf("Processing cancellation webhook for user %s, cancel_at_period_end: %v", userID, stripeSub.CancelAtPeriodEnd)

	// Find the user's current subscription
	currentSub, err := s.store.GetActiveSubscription(userID)
	if err != nil {
		log.Printf("Warning: Could not find active subscription for user %s: %v", userID, err)
		// User might already be on free plan or have no subscription
		return nil
	}

	// CRITICAL LOGIC: Handle cancel_at_period_end vs immediate cancellation
	if stripeSub.CancelAtPeriodEnd {
		// This is the key scenario that was failing
		log.Printf("Subscription marked for period-end cancellation - preserving benefits until %v", 
			time.Unix(stripeSub.CurrentPeriodEnd, 0))
		
		return s.handlePeriodEndCancellation(currentSub, stripeSub)
	} else {
		// Immediate cancellation - subscription is actually being deleted
		log.Printf("Processing immediate subscription deletion")
		return s.handleImmediateCancellation(currentSub)
	}
}

// handlePeriodEndCancellation preserves current benefits and sets free plan as pending
func (s *CleanSubscriptionService) handlePeriodEndCancellation(currentSub *Subscription, stripeSub *stripe.Subscription) error {
	// Get free plan to set as pending
	freePlan, err := s.store.GetFreePlan()
	if err != nil {
		return fmt.Errorf("failed to get free plan: %w", err)
	}

	// Update subscription to preserve current benefits but set pending transition
	effectiveDate := time.Unix(stripeSub.CurrentPeriodEnd, 0)
	currentSub.CancelAtPeriodEnd = true
	currentSub.PendingPlanID = freePlan.ID
	currentSub.PendingChangeEffectiveDate = &effectiveDate
	currentSub.PendingChangeReason = "cancellation_to_free_plan"
	
	// CRITICAL: Do NOT change current PlanID - user keeps benefits
	err = s.store.UpdateSubscription(currentSub)
	if err != nil {
		return fmt.Errorf("failed to update subscription with pending cancellation: %w", err)
	}

	log.Printf("Successfully preserved %s benefits until %v, free plan set as pending", 
		currentSub.PlanID, effectiveDate)
	return nil
}

// handleImmediateCancellation moves user to free plan and saves to history
func (s *CleanSubscriptionService) handleImmediateCancellation(currentSub *Subscription) error {
	// Check if there was a pending free plan (period-end cancellation completing)
	if currentSub.PendingPlanID != "" {
		log.Printf("Activating pending plan %s after period-end cancellation", currentSub.PendingPlanID)
		
		freePlan, err := s.store.GetFreePlan()
		if err != nil {
			return fmt.Errorf("failed to get free plan: %w", err)
		}
		
		if currentSub.PendingPlanID == freePlan.ID {
			// Move to history with pending activation reason
			err = s.store.SaveToHistory(currentSub, "period_end_cancellation_completed")
			if err != nil {
				log.Printf("Warning: Failed to save cancelled subscription to history: %v", err)
			}
			
			// Delete current subscription
			err = s.store.DeleteSubscription(currentSub.ID)
			if err != nil {
				log.Printf("Warning: Failed to delete cancelled subscription: %v", err)
			}
			
			// User is now on free plan (no active subscription)
			log.Printf("User moved to free plan after period-end cancellation completed")
			return nil
		}
	}

	// Regular cancellation - save to history and delete
	err := s.store.SaveToHistory(currentSub, "immediate_cancellation")
	if err != nil {
		log.Printf("Warning: Failed to save subscription to history: %v", err)
	}
	
	err = s.store.DeleteSubscription(currentSub.ID)
	if err != nil {
		log.Printf("Warning: Failed to delete cancelled subscription: %v", err)
	}
	
	log.Printf("User moved to free plan after immediate cancellation")
	return nil
}

// ProcessPlanChange handles subscription updates (not cancellations)
func (s *CleanSubscriptionService) ProcessPlanChange(userID string, stripeSub *stripe.Subscription) error {
	// Find subscription plan that matches this Stripe price
	stripePriceID, err := s.validator.ExtractPriceFromSubscription(stripeSub)
	if err != nil {
		return fmt.Errorf("failed to extract price from subscription: %w", err)
	}

	plan, err := s.store.GetPlanByProviderPrice(stripePriceID)
	if err != nil {
		return fmt.Errorf("failed to find subscription plan for price %s: %w", stripePriceID, err)
	}

	// Check if subscription exists
	currentSub, err := s.store.GetActiveSubscription(userID)
	if err != nil {
		// No existing subscription - create new one
		return s.createSubscriptionFromStripe(userID, plan, stripeSub, stripePriceID)
	}

	// Update existing subscription
	return s.updateSubscriptionFromStripe(currentSub, plan, stripeSub, stripePriceID)
}

// createSubscriptionFromStripe creates a new subscription from Stripe data
func (s *CleanSubscriptionService) createSubscriptionFromStripe(userID string, plan *Plan, stripeSub *stripe.Subscription, stripePriceID string) error {
	// Deactivate any existing subscriptions first
	err := s.store.DeactivateAllUserSubscriptions(userID)
	if err != nil {
		log.Printf("Warning: Failed to deactivate existing subscriptions: %v", err)
	}

	status := s.validator.MapStripeStatus(stripeSub.Status)
	start := time.Unix(stripeSub.CurrentPeriodStart, 0)
	end := time.Unix(stripeSub.CurrentPeriodEnd, 0)

	// Fix invalid timestamps
	start, end = s.validator.FixInvalidTimestamps(start, end)

	newSub := &Subscription{
		UserID:                   userID,
		PlanID:                   plan.ID,
		ProviderSubscriptionID:   stripeSub.ID,
		ProviderPriceID:          stripePriceID,
		PaymentProvider:          "stripe",
		Status:                   status,
		CurrentPeriodStart:       start,
		CurrentPeriodEnd:         end,
		CancelAtPeriodEnd:        stripeSub.CancelAtPeriodEnd,
	}

	if stripeSub.CanceledAt > 0 {
		canceledAt := time.Unix(stripeSub.CanceledAt, 0)
		newSub.CanceledAt = &canceledAt
	}
	if stripeSub.TrialEnd > 0 {
		trialEnd := time.Unix(stripeSub.TrialEnd, 0)
		newSub.TrialEnd = &trialEnd
	}

	return s.store.CreateSubscription(newSub)
}

// updateSubscriptionFromStripe updates an existing subscription
func (s *CleanSubscriptionService) updateSubscriptionFromStripe(currentSub *Subscription, plan *Plan, stripeSub *stripe.Subscription, stripePriceID string) error {
	status := s.validator.MapStripeStatus(stripeSub.Status)
	start := time.Unix(stripeSub.CurrentPeriodStart, 0)
	end := time.Unix(stripeSub.CurrentPeriodEnd, 0)

	// Fix invalid timestamps
	start, end = s.validator.FixInvalidTimestamps(start, end)

	// Update subscription fields
	currentSub.PlanID = plan.ID
	currentSub.ProviderPriceID = stripePriceID
	currentSub.Status = status
	currentSub.CurrentPeriodStart = start
	currentSub.CurrentPeriodEnd = end
	currentSub.CancelAtPeriodEnd = stripeSub.CancelAtPeriodEnd

	if stripeSub.CanceledAt > 0 {
		canceledAt := time.Unix(stripeSub.CanceledAt, 0)
		currentSub.CanceledAt = &canceledAt
	}
	if stripeSub.TrialEnd > 0 {
		trialEnd := time.Unix(stripeSub.TrialEnd, 0)
		currentSub.TrialEnd = &trialEnd
	}

	// Clear pending fields since change is now confirmed
	currentSub.PendingPlanID = ""
	currentSub.PendingChangeEffectiveDate = nil
	currentSub.PendingChangeReason = ""

	return s.store.UpdateSubscription(currentSub)
}

// SwitchToFreePlan moves a user to the free plan
func (s *CleanSubscriptionService) SwitchToFreePlan(userID string) error {
	// Move any existing subscriptions to history
	existingSubs, err := s.store.GetAllUserSubscriptions(userID)
	if err != nil {
		log.Printf("Warning: Failed to find existing subscriptions: %v", err)
	} else {
		for _, sub := range existingSubs {
			if sub.Status == "active" {
				err = s.store.SaveToHistory(sub, "switched_to_free_plan")
				if err != nil {
					log.Printf("Warning: Failed to save subscription to history: %v", err)
				}
				err = s.store.DeleteSubscription(sub.ID)
				if err != nil {
					log.Printf("Warning: Failed to delete subscription: %v", err)
				}
			}
		}
	}

	log.Printf("User %s switched to free plan", userID)
	return nil
}

// CreateFreePlanSubscription assigns the free plan to a new user
func (s *CleanSubscriptionService) CreateFreePlanSubscription(userID string) error {
	log.Printf("Creating free plan subscription for new user: %s", userID)
	
	// Check if user already has an active subscription
	existingSub, err := s.store.GetActiveSubscription(userID)
	if err == nil && existingSub != nil {
		log.Printf("User %s already has active subscription %s, skipping free plan creation", userID, existingSub.ID)
		return nil
	}
	
	// Get the free plan from subscription_plans
	freePlan, err := s.store.GetFreePlan()
	if err != nil {
		return fmt.Errorf("failed to get free plan for user onboarding: %w", err)
	}
	
	// Create a free plan subscription record
	now := time.Now()
	freeSubscription := &Subscription{
		UserID:             userID,
		PlanID:             freePlan.ID,
		PaymentProvider:    "none", // Free plan doesn't use payment provider
		Status:             "active",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(1, 0, 0), // Free plan valid for 1 year
		CancelAtPeriodEnd:  false,
	}
	
	// Save the free subscription
	err = s.store.CreateSubscription(freeSubscription)
	if err != nil {
		return fmt.Errorf("failed to create free plan subscription for user %s: %w", userID, err)
	}
	
	log.Printf("Successfully created free plan subscription for user %s with %f hours per month", 
		userID, freePlan.HoursPerMonth)
	return nil
}

// GetUserSubscriptionInfo retrieves comprehensive subscription information
func (s *CleanSubscriptionService) GetUserSubscriptionInfo(userID string) (*Subscription, *Plan, error) {
	// Try to get active subscription
	subscription, err := s.store.GetActiveSubscription(userID)
	if err != nil {
		// No active subscription - this should be rare with new user onboarding
		// but we'll handle it gracefully by creating virtual free subscription
		log.Printf("No active subscription found for user %s, returning virtual free subscription", userID)
		
		freePlan, err := s.store.GetFreePlan()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get free plan: %w", err)
		}
		
		// Create a virtual free subscription for display (fallback behavior)
		freeSubscription := &Subscription{
			ID:                 "free_virtual",
			UserID:            userID,
			PlanID:            freePlan.ID,
			Status:            "active",
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:  time.Now().AddDate(1, 0, 0), // Free plan valid for 1 year
		}
		
		return freeSubscription, freePlan, nil
	}

	// Get the plan for the active subscription
	plan, err := s.store.GetPlan(subscription.PlanID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get plan: %w", err)
	}

	return subscription, plan, nil
}