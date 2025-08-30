package subscription

import (
	"errors"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v79"
)

// MockRepository implements Repository interface for testing
type MockRepository struct {
	subscriptions        map[string]*core.Record
	plans               map[string]*core.Record
	activeSubscriptions map[string]*core.Record
	createError         error
	updateError         error
	findError           error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		subscriptions:       make(map[string]*core.Record),
		plans:              make(map[string]*core.Record),
		activeSubscriptions: make(map[string]*core.Record),
	}
}

func (m *MockRepository) CreateSubscription(params CreateSubscriptionParams) (*core.Record, error) {
	if m.createError != nil {
		return nil, m.createError
	}
	
	// Create a mock record
	record := &core.Record{}
	record.Id = "test_subscription_id"
	m.subscriptions[record.Id] = record
	m.activeSubscriptions[params.UserID] = record
	return record, nil
}

func (m *MockRepository) UpdateSubscription(subscriptionID string, params UpdateSubscriptionParams) (*core.Record, error) {
	if m.updateError != nil {
		return nil, m.updateError
	}
	
	record, exists := m.subscriptions[subscriptionID]
	if !exists {
		return nil, errors.New("subscription not found")
	}
	return record, nil
}

func (m *MockRepository) GetSubscription(subscriptionID string) (*core.Record, error) {
	record, exists := m.subscriptions[subscriptionID]
	if !exists {
		return nil, errors.New("subscription not found")
	}
	return record, nil
}

func (m *MockRepository) DeleteSubscription(subscriptionID string) error {
	delete(m.subscriptions, subscriptionID)
	return nil
}

func (m *MockRepository) FindSubscription(query SubscriptionQuery) (*core.Record, error) {
	if m.findError != nil {
		return nil, m.findError
	}
	
	// Simple mock implementation
	if record, exists := m.activeSubscriptions[query.UserID]; exists {
		return record, nil
	}
	return nil, errors.New("subscription not found")
}

func (m *MockRepository) FindActiveSubscription(userID string) (*core.Record, error) {
	query := SubscriptionQuery{UserID: userID, Status: &[]SubscriptionStatus{StatusActive}[0]}
	return m.FindSubscription(query)
}

func (m *MockRepository) FindSubscriptionByStripeID(stripeSubID string) (*core.Record, error) {
	// Simple mock - return first subscription
	for _, record := range m.subscriptions {
		return record, nil
	}
	return nil, errors.New("subscription not found")
}

func (m *MockRepository) FindAllUserSubscriptions(userID string) ([]*core.Record, error) {
	var records []*core.Record
	for _, record := range m.subscriptions {
		records = append(records, record)
	}
	return records, nil
}

func (m *MockRepository) FindActiveSubscriptionsCount(userID string) (int, error) {
	if _, exists := m.activeSubscriptions[userID]; exists {
		return 1, nil
	}
	return 0, nil
}

func (m *MockRepository) GetPlan(planID string) (*core.Record, error) {
	record, exists := m.plans[planID]
	if !exists {
		return nil, errors.New("plan not found")
	}
	return record, nil
}

func (m *MockRepository) GetPlanByStripePrice(stripePriceID string) (*core.Record, error) {
	// Return a mock plan
	record := &core.Record{}
	record.Id = "test_plan_id"
	return record, nil
}

func (m *MockRepository) GetFreePlan() (*core.Record, error) {
	record := &core.Record{}
	record.Id = "free_plan_id"
	return record, nil
}

func (m *MockRepository) GetAllPlans() ([]*core.Record, error) {
	var records []*core.Record
	for _, record := range m.plans {
		records = append(records, record)
	}
	return records, nil
}

func (m *MockRepository) GetAvailableUpgrades(currentPlanID string) ([]*core.Record, error) {
	return []*core.Record{}, nil
}

func (m *MockRepository) DeactivateAllUserSubscriptions(userID string) error {
	delete(m.activeSubscriptions, userID)
	return nil
}

func (m *MockRepository) CleanupDuplicateSubscriptions(userID string) error {
	return nil
}

// Test helper functions
func createTestService() Service {
	repo := NewMockRepository()
	return NewService(repo)
}

func createValidCreateParams() CreateSubscriptionParams {
	return CreateSubscriptionParams{
		UserID:             "test_user_id",
		PlanID:             "test_plan_id",
		Status:             StatusActive,
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0),
		CancelAtPeriodEnd:  false,
	}
}

// Unit tests
func TestCreateSubscription_Success(t *testing.T) {
	service := createTestService()
	params := createValidCreateParams()

	subscription, err := service.CreateSubscription(params)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if subscription == nil {
		t.Fatal("Expected subscription to be created, got nil")
	}
}

func TestCreateSubscription_ValidationError(t *testing.T) {
	service := createTestService()
	
	// Test with empty user ID
	params := createValidCreateParams()
	params.UserID = ""

	_, err := service.CreateSubscription(params)
	if err == nil {
		t.Fatal("Expected validation error for empty user ID")
	}
}

func TestCreateSubscription_InvalidDates(t *testing.T) {
	service := createTestService()
	
	// Test with end date before start date
	params := createValidCreateParams()
	params.CurrentPeriodEnd = params.CurrentPeriodStart.Add(-time.Hour)

	_, err := service.CreateSubscription(params)
	if err == nil {
		t.Fatal("Expected validation error for invalid date range")
	}
}

func TestCreateSubscription_FixesInvalidTimestamps(t *testing.T) {
	// This test verifies that the validator rejects invalid timestamps during validation
	// The service should reject timestamps that appear to be Unix timestamp 0 (1970)
	service := createTestService()
	
	// Test with 1970 Unix timestamp (0)
	params := createValidCreateParams()
	params.CurrentPeriodStart = time.Unix(0, 0)
	params.CurrentPeriodEnd = time.Unix(0, 0)

	_, err := service.CreateSubscription(params)
	if err == nil {
		t.Fatal("Expected validation error for 1970 timestamps")
	}
	
	// Verify the error is about invalid timestamps
	if err.Error() != "validation failed: Invalid start date (appears to be Unix timestamp 0)" {
		t.Fatalf("Expected timestamp validation error, got: %v", err)
	}
}

func TestUpdateSubscription_Success(t *testing.T) {
	service := createTestService()
	
	// First create a subscription
	createParams := createValidCreateParams()
	subscription, err := service.CreateSubscription(createParams)
	if err != nil {
		t.Fatalf("Failed to create test subscription: %v", err)
	}

	// Now update it
	status := StatusCanceled
	updateParams := UpdateSubscriptionParams{
		Status: &status,
	}

	updatedSubscription, err := service.UpdateSubscription(subscription.Id, updateParams)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if updatedSubscription == nil {
		t.Fatal("Expected updated subscription, got nil")
	}
}

func TestUpdateSubscription_ValidationError(t *testing.T) {
	service := createTestService()
	
	// Test with empty subscription ID
	updateParams := UpdateSubscriptionParams{}
	_, err := service.UpdateSubscription("", updateParams)
	if err == nil {
		t.Fatal("Expected validation error for empty subscription ID")
	}
}

func TestSwitchToFreePlan_Success(t *testing.T) {
	service := createTestService()
	userID := "test_user_id"

	subscription, err := service.SwitchToFreePlan(userID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if subscription == nil {
		t.Fatal("Expected free plan subscription to be created")
	}
}

func TestCancelSubscription_NoActiveSubscription(t *testing.T) {
	repo := NewMockRepository()
	repo.findError = errors.New("no subscription found")
	service := NewService(repo)

	err := service.CancelSubscription("nonexistent_user")
	if err == nil {
		t.Fatal("Expected error when no active subscription exists")
	}
}

func TestProcessWebhookEvent_SubscriptionCreated(t *testing.T) {
	service := createTestService()

	stripeSub := &stripe.Subscription{
		ID:     "stripe_sub_id",
		Status: stripe.SubscriptionStatusActive,
		Customer: &stripe.Customer{
			ID: "stripe_customer_id",
		},
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{
					Price: &stripe.Price{
						ID: "stripe_price_id",
					},
				},
			},
		},
		CurrentPeriodStart: time.Now().Unix(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0).Unix(),
	}

	eventData := WebhookEventData{
		EventType:    "customer.subscription.created",
		Subscription: stripeSub,
	}

	// This will fail because getUserIDFromCustomer is not implemented
	// But it tests the validation and routing logic
	err := service.ProcessWebhookEvent(eventData)
	if err == nil {
		t.Fatal("Expected error due to unimplemented getUserIDFromCustomer")
	}
	
	// Check that the error contains the expected message
	expectedSubstring := "unsupported repository type for customer mapping"
	if !containsString(err.Error(), expectedSubstring) {
		t.Fatalf("Expected error to contain '%s', got: %v", expectedSubstring, err)
	}
}

func TestProcessWebhookEvent_InvalidData(t *testing.T) {
	service := createTestService()

	// Test with empty event type
	eventData := WebhookEventData{
		EventType: "",
	}

	err := service.ProcessWebhookEvent(eventData)
	if err == nil {
		t.Fatal("Expected validation error for empty event type")
	}
}

func TestProcessWebhookEvent_UnhandledEventType(t *testing.T) {
	service := createTestService()

	eventData := WebhookEventData{
		EventType: "unhandled.event.type",
	}

	err := service.ProcessWebhookEvent(eventData)
	if err != nil {
		t.Fatalf("Expected no error for unhandled event type, got: %v", err)
	}
}

// Test validator functions directly
func TestMapStripeStatus(t *testing.T) {
	repo := NewMockRepository()
	validator := NewValidator(repo)

	tests := []struct {
		input    stripe.SubscriptionStatus
		expected SubscriptionStatus
	}{
		{stripe.SubscriptionStatusActive, StatusActive},
		{stripe.SubscriptionStatusCanceled, StatusCanceled},
		{stripe.SubscriptionStatusPastDue, StatusPastDue},
		{stripe.SubscriptionStatusTrialing, StatusTrialing},
		{stripe.SubscriptionStatus("unknown"), StatusActive}, // Default fallback
	}

	for _, test := range tests {
		result := validator.MapStripeStatus(test.input)
		if result != test.expected {
			t.Errorf("MapStripeStatus(%v) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestFixInvalidTimestamps(t *testing.T) {
	repo := NewMockRepository()
	validator := NewValidator(repo)

	// Test fixing 1970 timestamps
	invalidStart := time.Unix(0, 0)
	invalidEnd := time.Unix(0, 0)

	fixedStart, fixedEnd := validator.FixInvalidTimestamps(invalidStart, invalidEnd)

	if fixedStart.Year() < 2020 {
		t.Errorf("Expected fixed start date to be after 2020, got %v", fixedStart)
	}
	if fixedEnd.Year() < 2020 {
		t.Errorf("Expected fixed end date to be after 2020, got %v", fixedEnd)
	}
	if !fixedEnd.After(fixedStart) {
		t.Error("Expected fixed end date to be after start date")
	}
}

func TestExtractPriceFromSubscription(t *testing.T) {
	repo := NewMockRepository()
	validator := NewValidator(repo)

	// Test with valid subscription
	stripeSub := &stripe.Subscription{
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{
					Price: &stripe.Price{
						ID: "price_123",
					},
				},
			},
		},
	}

	priceID, err := validator.ExtractPriceFromSubscription(stripeSub)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if priceID != "price_123" {
		t.Errorf("Expected price_123, got %s", priceID)
	}

	// Test with nil subscription
	_, err = validator.ExtractPriceFromSubscription(nil)
	if err == nil {
		t.Fatal("Expected error for nil subscription")
	}

	// Test with no items
	emptySubscription := &stripe.Subscription{}
	_, err = validator.ExtractPriceFromSubscription(emptySubscription)
	if err == nil {
		t.Fatal("Expected error for subscription with no items")
	}
}

func TestGetUsageWarningMessage(t *testing.T) {
	repo := NewMockRepository()
	validator := NewValidator(repo)

	// Test over limit
	usage := &UsageInfo{
		HoursUsedThisMonth: 15.0,
		HoursLimit:         10.0,
		IsOverLimit:        true,
	}
	message := validator.GetUsageWarningMessage(usage)
	if message == "" {
		t.Error("Expected warning message for over limit usage")
	}

	// Test 90% usage
	usage = &UsageInfo{
		HoursUsedThisMonth: 9.0,
		HoursLimit:         10.0,
		IsOverLimit:        false,
	}
	message = validator.GetUsageWarningMessage(usage)
	if message == "" {
		t.Error("Expected warning message for 90% usage")
	}

	// Test 50% usage (no warning)
	usage = &UsageInfo{
		HoursUsedThisMonth: 5.0,
		HoursLimit:         10.0,
		IsOverLimit:        false,
	}
	message = validator.GetUsageWarningMessage(usage)
	if message != "" {
		t.Error("Expected no warning message for 50% usage")
	}

	// Test nil usage
	message = validator.GetUsageWarningMessage(nil)
	if message != "" {
		t.Error("Expected empty message for nil usage")
	}
}

// Test helper function
func containsString(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(substr) == 0 ||
		(len(str) > len(substr) && str[:len(substr)] == substr) ||
		(len(str) > len(substr) && str[len(str)-len(substr):] == substr) ||
		(len(str) > len(substr) && findSubstring(str, substr)))
}

func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}