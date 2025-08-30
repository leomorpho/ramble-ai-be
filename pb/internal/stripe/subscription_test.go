package stripe

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stripe/stripe-go/v79"
)

const testDataDir = "./test_pb_data"

// setupTestApp creates a test app with our subscription management functions
func setupTestApp(t testing.TB) *tests.TestApp {
	testApp, err := tests.NewTestApp(testDataDir)
	if err != nil {
		t.Fatal(err)
	}
	return testApp
}

// TestHandleSubscriptionEvent_NoDuplicates tests that users only have one active subscription
func TestHandleSubscriptionEvent_NoDuplicates(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	// Create test user and customer
	user := createTestUser(testApp, "test@example.com")
	customer := createTestCustomer(testApp, user.Id, "cus_123")

	// Create first subscription
	sub1 := &stripe.Subscription{
		ID:                 "sub_123",
		Customer:           &stripe.Customer{ID: "cus_123"},
		Status:             stripe.SubscriptionStatusActive,
		CurrentPeriodStart: time.Now().Unix(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0).Unix(),
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{Price: &stripe.Price{ID: "price_123"}},
			},
		},
	}

	// Handle first subscription
	err := handleSubscriptionEvent(testApp, sub1, "customer.subscription.created")
	if err != nil {
		t.Fatalf("Failed to handle first subscription: %v", err)
	}

	// Verify we have 1 active subscription
	count1 := countActiveSubscriptions(testApp, user.Id)
	if count1 != 1 {
		t.Errorf("Expected 1 active subscription after first creation, got %d", count1)
	}

	// Create second subscription (user switching plans)
	sub2 := &stripe.Subscription{
		ID:                 "sub_456", // Different subscription ID
		Customer:           &stripe.Customer{ID: "cus_123"},
		Status:             stripe.SubscriptionStatusActive,
		CurrentPeriodStart: time.Now().Unix(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0).Unix(),
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{Price: &stripe.Price{ID: "price_456"}},
			},
		},
	}

	// Handle plan change
	err = handleSubscriptionEvent(testApp, sub2, "customer.subscription.created")
	if err != nil {
		t.Fatalf("Failed to handle second subscription: %v", err)
	}

	// Should still have only 1 active subscription
	count2 := countActiveSubscriptions(testApp, user.Id)
	if count2 != 1 {
		t.Errorf("Expected 1 active subscription after plan change, got %d", count2)
	}

	// Verify the old subscription was cancelled
	oldSub, err := testApp.FindFirstRecordByFilter("user_subscriptions", 
		"stripe_subscription_id = 'sub_123'")
	
	if err == nil && oldSub.GetString("status") != "cancelled" {
		t.Errorf("Old subscription should be cancelled, got status: %s", oldSub.GetString("status"))
	}
}

// TestHandleSubscriptionEvent_ValidDates tests that timestamps are not 1970
func TestHandleSubscriptionEvent_ValidDates(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	// Create test user and customer
	user := createTestUser(testApp, "datetest@example.com")
	customer := createTestCustomer(testApp, user.Id, "cus_date")

	// Create subscription with zero timestamps (would cause 1970 dates)
	sub := &stripe.Subscription{
		ID:                 "sub_test",
		Customer:           &stripe.Customer{ID: "cus_date"},
		Status:             stripe.SubscriptionStatusActive,
		CurrentPeriodStart: 0, // This would create 1970 date
		CurrentPeriodEnd:   0, // This would create 1970 date
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{Price: &stripe.Price{ID: "price_test"}},
			},
		},
	}

	err := handleSubscriptionEvent(testApp, sub, "customer.subscription.created")
	if err != nil {
		t.Fatalf("Failed to handle subscription: %v", err)
	}

	// Check the created subscription has valid dates
	record, err := testApp.FindFirstRecordByFilter("user_subscriptions",
		"stripe_subscription_id = 'sub_test'")

	if err != nil {
		t.Fatal("Subscription record not created")
	}

	// Check dates are not 1970
	startDate := record.GetDateTime("current_period_start").Time()
	endDate := record.GetDateTime("current_period_end").Time()

	if startDate.Year() < 2000 {
		t.Errorf("Start date is invalid (1970): %v", startDate)
	}

	if endDate.Year() < 2000 {
		t.Errorf("End date is invalid (1970): %v", endDate)
	}

	// Ensure end date is after start date
	if !endDate.After(startDate) {
		t.Errorf("End date should be after start date: start=%v, end=%v", startDate, endDate)
	}
}

// TestCleanupDuplicateSubscriptions tests the cleanup function
func TestCleanupDuplicateSubscriptions(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	// Create test user
	user := createTestUser(testApp, "cleanup@example.com")

	// Create multiple active subscriptions (simulating the bug)
	createTestSubscription(testApp, user.Id, "sub_old", time.Now().AddDate(0, 0, -10))
	createTestSubscription(testApp, user.Id, "sub_middle", time.Now().AddDate(0, 0, -5))
	createTestSubscription(testApp, user.Id, "sub_newest", time.Now())

	// Create subscription with invalid dates
	createTestSubscriptionInvalid(testApp, user.Id, "sub_invalid")

	// Verify we have 4 active subscriptions (the bug state)
	initialCount := countActiveSubscriptions(testApp, user.Id)
	if initialCount != 4 {
		t.Fatalf("Expected 4 initial active subscriptions, got %d", initialCount)
	}

	// Run cleanup
	err := cleanupDuplicateSubscriptions(testApp)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Should have only 1 active subscription after cleanup
	finalCount := countActiveSubscriptions(testApp, user.Id)
	if finalCount != 1 {
		t.Errorf("Expected 1 active subscription after cleanup, got %d", finalCount)
	}

	// The remaining subscription should be the newest one
	activeRecord, _ := testApp.FindFirstRecordByFilter("user_subscriptions",
		"user_id = '" + user.Id + "' AND status = 'active'")

	if activeRecord == nil {
		t.Fatal("No active subscription found after cleanup")
	}

	if activeRecord.GetString("stripe_subscription_id") != "sub_newest" {
		t.Errorf("Expected newest subscription to remain active, got %s",
			activeRecord.GetString("stripe_subscription_id"))
	}

	// Verify dates were fixed on the remaining subscription
	startDate := activeRecord.GetDateTime("current_period_start").Time()
	endDate := activeRecord.GetDateTime("current_period_end").Time()

	if startDate.Year() < 2000 || endDate.Year() < 2000 {
		t.Errorf("Dates should be valid after cleanup: start=%v, end=%v", startDate, endDate)
	}
}

// Helper functions
func createTestUser(app *tests.TestApp, email string) *core.Record {
	collection, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(collection)
	user.Set("email", email)
	app.Save(user)
	return user
}

func createTestCustomer(app *tests.TestApp, userID, customerID string) *core.Record {
	collection, _ := app.FindCollectionByNameOrId("stripe_customers")
	customer := core.NewRecord(collection)
	customer.Set("user_id", userID)
	customer.Set("stripe_customer_id", customerID)
	app.Save(customer)
	return customer
}

func createTestSubscription(app *tests.TestApp, userID, subID string, periodEnd time.Time) *core.Record {
	collection, _ := app.FindCollectionByNameOrId("user_subscriptions")
	sub := core.NewRecord(collection)
	sub.Set("user_id", userID)
	sub.Set("stripe_subscription_id", subID)
	sub.Set("status", "active")
	sub.Set("current_period_start", time.Now().AddDate(0, -1, 0))
	sub.Set("current_period_end", periodEnd)
	sub.Set("cancel_at_period_end", false)
	app.Save(sub)
	return sub
}

func createTestSubscriptionInvalid(app *tests.TestApp, userID, subID string) *core.Record {
	collection, _ := app.FindCollectionByNameOrId("user_subscriptions")
	sub := core.NewRecord(collection)
	sub.Set("user_id", userID)
	sub.Set("stripe_subscription_id", subID)
	sub.Set("status", "active")
	sub.Set("current_period_start", time.Unix(0, 0))      // 1970 date
	sub.Set("current_period_end", time.Unix(86400, 0))    // Also 1970
	sub.Set("cancel_at_period_end", false)
	app.Save(sub)
	return sub
}

func countActiveSubscriptions(app *tests.TestApp, userID string) int {
	records, err := app.FindRecordsByFilter("user_subscriptions",
		"user_id = '" + userID + "' AND status = 'active'",
		"", 0, 0)
	
	if err != nil {
		return 0
	}
	return len(records)
}