package subscription

import (
	"testing"
	"time"

	"github.com/stripe/stripe-go/v79"
)

// Test the critical cancellation webhook processing that was failing
func TestProcessCancellationWebhook_WithCancelAtPeriodEnd_PreservesCurrentPlan(t *testing.T) {
	// Setup
	store := NewTestStore()
	validator := NewCleanValidator()
	service := NewCleanSubscriptionService(store, validator)

	// Create a user with basic plan
	basicPlan, _ := store.GetPlan("basic_plan")
	userSub := &Subscription{
		ID:                     "test_sub_1",
		UserID:                 "user_123",
		PlanID:                 basicPlan.ID,
		ProviderSubscriptionID: "stripe_sub_123",
		Status:                 "active",
		CurrentPeriodStart:     time.Now(),
		CurrentPeriodEnd:       time.Now().AddDate(0, 1, 0),
		CancelAtPeriodEnd:      false,
	}
	store.AddSubscription(userSub)

	// Create Stripe webhook: subscription.updated with cancel_at_period_end=true
	periodEnd := time.Now().AddDate(0, 1, 0)
	stripeSub := &stripe.Subscription{
		ID:                 "stripe_sub_123",
		Status:             stripe.SubscriptionStatusActive,
		CancelAtPeriodEnd:  true, // This is the key - marked for cancellation
		CurrentPeriodStart: time.Now().Unix(),
		CurrentPeriodEnd:   periodEnd.Unix(),
	}

	// Process the cancellation webhook
	err := service.ProcessCancellationWebhook("user_123", stripeSub)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// CRITICAL ASSERTIONS: Verify current plan is preserved
	updatedSub, err := store.GetActiveSubscription("user_123")
	if err != nil {
		t.Fatal("User should still have active subscription")
	}

	// Should still be on basic plan (current benefits preserved)
	if updatedSub.PlanID != "basic_plan" {
		t.Errorf("Expected to preserve basic plan, got: %s", updatedSub.PlanID)
	}

	// Should have cancel_at_period_end set
	if !updatedSub.CancelAtPeriodEnd {
		t.Error("Expected cancel_at_period_end to be true")
	}

	// Should have free plan as pending
	if updatedSub.PendingPlanID != "free_plan" {
		t.Errorf("Expected free plan as pending, got: %s", updatedSub.PendingPlanID)
	}

	// Should have correct pending reason
	if updatedSub.PendingChangeReason != "cancellation_to_free_plan" {
		t.Errorf("Expected cancellation_to_free_plan reason, got: %s", updatedSub.PendingChangeReason)
	}

	// Should have effective date set
	if updatedSub.PendingChangeEffectiveDate == nil {
		t.Error("Expected pending change effective date to be set")
	}

	// Verify no history record created yet (subscription still active)
	if store.GetHistoryCount() != 0 {
		t.Error("Expected no history records during cancel_at_period_end phase")
	}
}

func TestProcessCancellationWebhook_ImmediateDeletion_MovesToFreePlan(t *testing.T) {
	// Setup
	store := NewTestStore()
	validator := NewCleanValidator()
	service := NewCleanSubscriptionService(store, validator)

	// Create a user with subscription that has pending free plan (period-end scenario)
	userSub := &Subscription{
		ID:                         "test_sub_1",
		UserID:                     "user_123",
		PlanID:                     "basic_plan",
		ProviderSubscriptionID:     "stripe_sub_123",
		Status:                     "active",
		CancelAtPeriodEnd:          true,
		PendingPlanID:              "free_plan",
		PendingChangeReason:        "cancellation_to_free_plan",
		PendingChangeEffectiveDate: &time.Time{},
	}
	*userSub.PendingChangeEffectiveDate = time.Now()
	store.AddSubscription(userSub)

	// Create Stripe webhook: subscription.deleted (period has ended)
	stripeSub := &stripe.Subscription{
		ID:                 "stripe_sub_123",
		Status:             stripe.SubscriptionStatusCanceled,
		CancelAtPeriodEnd:  false, // No longer relevant
		CurrentPeriodStart: time.Now().AddDate(0, -1, 0).Unix(),
		CurrentPeriodEnd:   time.Now().Unix(), // Period has ended
	}

	// Process the deletion webhook
	err := service.ProcessCancellationWebhook("user_123", stripeSub)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// CRITICAL ASSERTIONS: Verify user is now on free plan
	// Original subscription should be deleted
	_, err = store.GetSubscription("test_sub_1")
	if err == nil {
		t.Error("Original subscription should have been deleted")
	}

	// Should have no active subscription (= free plan)
	_, err = store.GetActiveSubscription("user_123")
	if err == nil {
		t.Error("User should have no active subscription (= free plan)")
	}

	// Should have history record
	if store.GetHistoryCount() != 1 {
		t.Errorf("Expected 1 history record, got: %d", store.GetHistoryCount())
	}

	historyEntry := store.GetLastHistoryEntry()
	if historyEntry.PendingChangeReason != "period_end_cancellation_completed" {
		t.Errorf("Expected history reason 'period_end_cancellation_completed', got: %s", historyEntry.PendingChangeReason)
	}
}

func TestProcessCancellationWebhook_NoExistingSubscription_GracefulHandling(t *testing.T) {
	// Setup
	store := NewTestStore()
	validator := NewCleanValidator()
	service := NewCleanSubscriptionService(store, validator)

	// Try to process cancellation for user with no subscription
	stripeSub := &stripe.Subscription{
		ID:                 "stripe_sub_nonexistent",
		Status:             stripe.SubscriptionStatusCanceled,
		CancelAtPeriodEnd:  false,
		CurrentPeriodStart: time.Now().AddDate(0, -1, 0).Unix(),
		CurrentPeriodEnd:   time.Now().Unix(),
	}

	// Should handle gracefully without error
	err := service.ProcessCancellationWebhook("user_no_sub", stripeSub)
	if err != nil {
		t.Fatalf("Expected graceful handling, got error: %v", err)
	}

	// Should have no side effects
	if store.GetHistoryCount() != 0 {
		t.Error("Expected no history records for non-existent subscription")
	}
}

func TestSwitchToFreePlan_MovesSubscriptionToHistory(t *testing.T) {
	// Setup
	store := NewTestStore()
	validator := NewCleanValidator()
	service := NewCleanSubscriptionService(store, validator)

	// Create user with active subscription
	userSub := &Subscription{
		ID:                     "test_sub_1",
		UserID:                 "user_123",
		PlanID:                 "basic_plan",
		ProviderSubscriptionID: "stripe_sub_123",
		Status:                 "active",
	}
	store.AddSubscription(userSub)

	// Switch to free plan
	err := service.SwitchToFreePlan("user_123")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have no active subscription (= free plan)
	_, err = store.GetActiveSubscription("user_123")
	if err == nil {
		t.Error("User should have no active subscription after switching to free")
	}

	// Should have history record
	if store.GetHistoryCount() != 1 {
		t.Errorf("Expected 1 history record, got: %d", store.GetHistoryCount())
	}

	historyEntry := store.GetLastHistoryEntry()
	if historyEntry.PendingChangeReason != "switched_to_free_plan" {
		t.Errorf("Expected history reason 'switched_to_free_plan', got: %s", historyEntry.PendingChangeReason)
	}
}

func TestGetUserSubscriptionInfo_WithActiveSubscription_ReturnsCorrectInfo(t *testing.T) {
	// Setup
	store := NewTestStore()
	validator := NewCleanValidator()
	service := NewCleanSubscriptionService(store, validator)

	// Create user with basic plan
	userSub := &Subscription{
		ID:     "test_sub_1",
		UserID: "user_123",
		PlanID: "basic_plan",
		Status: "active",
	}
	store.AddSubscription(userSub)

	// Get subscription info
	sub, plan, err := service.GetUserSubscriptionInfo("user_123")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if sub.PlanID != "basic_plan" {
		t.Errorf("Expected basic plan, got: %s", sub.PlanID)
	}

	if plan.Name != "Basic Plan" {
		t.Errorf("Expected Basic Plan, got: %s", plan.Name)
	}
}

func TestGetUserSubscriptionInfo_NoActiveSubscription_ReturnsFreePlan(t *testing.T) {
	// Setup
	store := NewTestStore()
	validator := NewCleanValidator()
	service := NewCleanSubscriptionService(store, validator)

	// Get subscription info for user with no subscription
	sub, plan, err := service.GetUserSubscriptionInfo("user_no_sub")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if sub.PlanID != "free_plan" {
		t.Errorf("Expected free plan, got: %s", sub.PlanID)
	}

	if plan.Name != "Free Plan" {
		t.Errorf("Expected Free Plan, got: %s", plan.Name)
	}

	if sub.ID != "free_virtual" {
		t.Errorf("Expected virtual free subscription, got: %s", sub.ID)
	}
}

func TestPendingStateManagement_DuringCancellation_PreservesBenefits(t *testing.T) {
	// Setup
	store := NewTestStore()
	validator := NewCleanValidator()
	service := NewCleanSubscriptionService(store, validator)

	// Create user with pro plan
	proPlan, _ := store.GetPlan("pro_plan")
	userSub := &Subscription{
		ID:                     "test_sub_1",
		UserID:                 "user_123",
		PlanID:                 proPlan.ID,
		ProviderSubscriptionID: "stripe_sub_123",
		Status:                 "active",
		CurrentPeriodEnd:       time.Now().AddDate(0, 1, 0),
	}
	store.AddSubscription(userSub)

	// Process period-end cancellation
	periodEnd := time.Now().AddDate(0, 1, 0)
	stripeSub := &stripe.Subscription{
		ID:                 "stripe_sub_123",
		Status:             stripe.SubscriptionStatusActive,
		CancelAtPeriodEnd:  true,
		CurrentPeriodEnd:   periodEnd.Unix(),
	}

	err := service.ProcessCancellationWebhook("user_123", stripeSub)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Get subscription info - should still show pro benefits
	sub, plan, err := service.GetUserSubscriptionInfo("user_123")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// User should still get pro plan benefits
	if plan.ID != "pro_plan" {
		t.Errorf("Expected user to still have pro benefits, got: %s", plan.ID)
	}

	if plan.HoursPerMonth != 200.0 {
		t.Errorf("Expected 200 hours, got: %f", plan.HoursPerMonth)
	}

	// But should have pending transition to free
	if sub.PendingPlanID != "free_plan" {
		t.Errorf("Expected pending free plan, got: %s", sub.PendingPlanID)
	}

	// Now simulate period end - process deletion webhook
	deletionStripe := &stripe.Subscription{
		ID:                 "stripe_sub_123",
		Status:             stripe.SubscriptionStatusCanceled,
		CancelAtPeriodEnd:  false,
		CurrentPeriodEnd:   time.Now().Unix(),
	}

	err = service.ProcessCancellationWebhook("user_123", deletionStripe)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Now should be on free plan
	sub, plan, err = service.GetUserSubscriptionInfo("user_123")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if plan.ID != "free_plan" {
		t.Errorf("Expected free plan after period end, got: %s", plan.ID)
	}

	if plan.HoursPerMonth != 5.0 {
		t.Errorf("Expected 5 hours for free plan, got: %f", plan.HoursPerMonth)
	}
}

// Edge case tests
func TestMultipleRapidCancellations_HandledGracefully(t *testing.T) {
	store := NewTestStore()
	validator := NewCleanValidator()
	service := NewCleanSubscriptionService(store, validator)

	// Create subscription
	userSub := &Subscription{
		ID:                     "test_sub_1",
		UserID:                 "user_123",
		PlanID:                 "basic_plan",
		ProviderSubscriptionID: "stripe_sub_123",
		Status:                 "active",
	}
	store.AddSubscription(userSub)

	// Process multiple cancellation webhooks
	stripeSub := &stripe.Subscription{
		ID:                 "stripe_sub_123",
		Status:             stripe.SubscriptionStatusActive,
		CancelAtPeriodEnd:  true,
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0).Unix(),
	}

	// First cancellation
	err := service.ProcessCancellationWebhook("user_123", stripeSub)
	if err != nil {
		t.Fatalf("Expected no error on first cancellation, got: %v", err)
	}

	// Second cancellation (duplicate webhook)
	err = service.ProcessCancellationWebhook("user_123", stripeSub)
	if err != nil {
		t.Fatalf("Expected no error on duplicate cancellation, got: %v", err)
	}

	// Should still preserve benefits
	_, plan, err := service.GetUserSubscriptionInfo("user_123")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if plan.ID != "basic_plan" {
		t.Errorf("Expected basic plan preserved, got: %s", plan.ID)
	}
}

// USER ONBOARDING TESTS

func TestCreateFreePlanSubscription_NewUser_CreatesFreePlanRecord(t *testing.T) {
	// Setup
	store := NewTestStore()
	validator := NewCleanValidator()
	service := NewCleanSubscriptionService(store, validator)
	
	userID := "new_user_123"
	
	// Create free plan subscription for new user
	err := service.CreateFreePlanSubscription(userID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	// Verify user now has an active subscription
	subscription, err := store.GetActiveSubscription(userID)
	if err != nil {
		t.Fatalf("Expected user to have active subscription, got error: %v", err)
	}
	
	// Should be on free plan
	if subscription.PlanID != "free_plan" {
		t.Errorf("Expected free plan, got: %s", subscription.PlanID)
	}
	
	// Should have correct subscription details
	if subscription.Status != "active" {
		t.Errorf("Expected active status, got: %s", subscription.Status)
	}
	
	if subscription.PaymentProvider != "none" {
		t.Errorf("Expected no payment provider for free plan, got: %s", subscription.PaymentProvider)
	}
	
	if subscription.CancelAtPeriodEnd {
		t.Error("Expected cancel_at_period_end to be false for new free subscription")
	}
	
	// Period should be set to 1 year
	expectedEnd := subscription.CurrentPeriodStart.AddDate(1, 0, 0)
	if !subscription.CurrentPeriodEnd.Equal(expectedEnd) {
		t.Errorf("Expected period end to be 1 year from start, got: start=%v, end=%v", 
			subscription.CurrentPeriodStart, subscription.CurrentPeriodEnd)
	}
}

func TestCreateFreePlanSubscription_UserWithExistingSubscription_SkipsCreation(t *testing.T) {
	// Setup
	store := NewTestStore()
	validator := NewCleanValidator()
	service := NewCleanSubscriptionService(store, validator)
	
	userID := "existing_user_123"
	
	// User already has a basic plan subscription
	existingSub := &Subscription{
		ID:                 "existing_sub_1",
		UserID:             userID,
		PlanID:             "basic_plan",
		Status:             "active",
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0),
	}
	store.AddSubscription(existingSub)
	
	// Try to create free plan subscription
	err := service.CreateFreePlanSubscription(userID)
	if err != nil {
		t.Fatalf("Expected no error when user already has subscription, got: %v", err)
	}
	
	// Should still be on original plan
	subscription, err := store.GetActiveSubscription(userID)
	if err != nil {
		t.Fatalf("Expected user to still have active subscription, got error: %v", err)
	}
	
	if subscription.PlanID != "basic_plan" {
		t.Errorf("Expected to preserve existing basic plan, got: %s", subscription.PlanID)
	}
	
	// Should only have one subscription
	if store.GetHistoryCount() != 0 {
		t.Error("Expected no history records when skipping free plan creation")
	}
}

func TestCreateFreePlanSubscription_Integration_GetUserSubscriptionInfo(t *testing.T) {
	// Setup
	store := NewTestStore()
	validator := NewCleanValidator()
	service := NewCleanSubscriptionService(store, validator)
	
	userID := "integration_user_123"
	
	// Create free plan subscription
	err := service.CreateFreePlanSubscription(userID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	// Get user subscription info should return the actual subscription record
	subscription, plan, err := service.GetUserSubscriptionInfo(userID)
	if err != nil {
		t.Fatalf("Expected no error getting subscription info, got: %v", err)
	}
	
	// Should not be virtual subscription
	if subscription.ID == "free_virtual" {
		t.Error("Expected real subscription record, got virtual subscription")
	}
	
	// Should have correct plan details
	if plan.Name != "Free Plan" {
		t.Errorf("Expected Free Plan, got: %s", plan.Name)
	}
	
	if plan.HoursPerMonth != 5.0 {
		t.Errorf("Expected 5.0 hours per month for free plan, got: %f", plan.HoursPerMonth)
	}
	
	if !plan.IsFree {
		t.Error("Expected free plan to be marked as free")
	}
}

func TestGetUserSubscriptionInfo_NoSubscription_FallbackToVirtual(t *testing.T) {
	// Setup - this tests the fallback behavior for users who somehow don't have subscriptions
	store := NewTestStore()
	validator := NewCleanValidator()
	service := NewCleanSubscriptionService(store, validator)
	
	userID := "orphaned_user_123"
	// Don't create any subscription for this user
	
	// Get subscription info should fall back to virtual free subscription
	subscription, plan, err := service.GetUserSubscriptionInfo(userID)
	if err != nil {
		t.Fatalf("Expected no error with fallback behavior, got: %v", err)
	}
	
	// Should be virtual subscription (fallback)
	if subscription.ID != "free_virtual" {
		t.Errorf("Expected virtual subscription as fallback, got: %s", subscription.ID)
	}
	
	// Should still get correct free plan
	if plan.Name != "Free Plan" {
		t.Errorf("Expected Free Plan, got: %s", plan.Name)
	}
}