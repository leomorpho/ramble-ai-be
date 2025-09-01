package ai

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// MockApp implements the core.App interface for testing
type MockApp struct {
	monthlyUsageRecords map[string]map[string]interface{} // key: userID-yearMonth
	subscriptions       map[string]map[string]interface{} // key: userID
	plans               map[string]map[string]interface{} // key: planID
	saveError           error
	findError           error
}

func NewMockApp() *MockApp {
	return &MockApp{
		monthlyUsageRecords: make(map[string]map[string]interface{}),
		subscriptions:       make(map[string]map[string]interface{}),
		plans:               make(map[string]map[string]interface{}),
	}
}

func (m *MockApp) FindFirstRecordByFilter(collectionName string, filter string, params map[string]interface{}) (MockRecord, error) {
	if m.findError != nil {
		return MockRecord{}, m.findError
	}

	switch collectionName {
	case "monthly_usage":
		userID := params["user_id"].(string)
		month := params["month"].(string)
		key := fmt.Sprintf("%s-%s", userID, month)
		if data, exists := m.monthlyUsageRecords[key]; exists {
			return MockRecord{Data: data, Id: key}, nil
		}
		return MockRecord{}, fmt.Errorf("record not found")

	case "current_user_subscriptions":
		userID := params["user_id"].(string)
		if data, exists := m.subscriptions[userID]; exists {
			return MockRecord{Data: data, Id: userID}, nil
		}
		return MockRecord{}, fmt.Errorf("subscription not found")

	default:
		return MockRecord{}, fmt.Errorf("collection %s not supported in mock", collectionName)
	}
}

func (m *MockApp) FindCollectionByNameOrId(name string) (MockCollection, error) {
	return MockCollection{Name: name}, nil
}

func (m *MockApp) Save(record MockRecord) error {
	if m.saveError != nil {
		return m.saveError
	}

	// Update our in-memory store based on record data
	if userID, ok := record.Data["user_id"].(string); ok {
		if yearMonth, ok := record.Data["year_month"].(string); ok {
			// This is a monthly usage record
			key := fmt.Sprintf("%s-%s", userID, yearMonth)
			m.monthlyUsageRecords[key] = record.Data
		}
	}

	return nil
}

// Mock helper methods
func (m *MockApp) SetMonthlyUsage(userID string, yearMonth string, hoursUsed float64, filesProcessed int) {
	key := fmt.Sprintf("%s-%s", userID, yearMonth)
	m.monthlyUsageRecords[key] = map[string]interface{}{
		"user_id":         userID,
		"year_month":      yearMonth,
		"hours_used":      hoursUsed,
		"files_processed": filesProcessed,
	}
}

func (m *MockApp) SetUserSubscription(userID string, planID string) {
	m.subscriptions[userID] = map[string]interface{}{
		"user_id": userID,
		"plan_id": planID,
		"status":  "active",
	}
}

func (m *MockApp) SetPlan(planID string, name string, hoursPerMonth float64) {
	m.plans[planID] = map[string]interface{}{
		"id":               planID,
		"name":             name,
		"hours_per_month":  hoursPerMonth,
	}
}

// Mock record type
type MockRecord struct {
	Data map[string]interface{}
	Id   string
}

func (r MockRecord) GetFloat(field string) float64 {
	if val, exists := r.Data[field]; exists {
		if f, ok := val.(float64); ok {
			return f
		}
		if i, ok := val.(int); ok {
			return float64(i)
		}
	}
	return 0
}

func (r MockRecord) GetInt(field string) int {
	if val, exists := r.Data[field]; exists {
		if i, ok := val.(int); ok {
			return i
		}
		if f, ok := val.(float64); ok {
			return int(f)
		}
	}
	return 0
}

func (r MockRecord) GetString(field string) string {
	if val, exists := r.Data[field]; exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func (r MockRecord) Set(field string, value interface{}) {
	if r.Data == nil {
		r.Data = make(map[string]interface{})
	}
	r.Data[field] = value
}

type MockCollection struct {
	Name string
}

func NewRecord(collection MockCollection) MockRecord {
	return MockRecord{
		Data: make(map[string]interface{}),
		Id:   fmt.Sprintf("new_record_%d", time.Now().UnixNano()),
	}
}

// Mock subscription service for testing
type MockSubscriptionInfo struct {
	Plan MockRecord
}

func (m *MockApp) GetUserSubscriptionInfo(userID string) (*MockSubscriptionInfo, error) {
	// Find user's subscription
	if subData, exists := m.subscriptions[userID]; exists {
		planID := subData["plan_id"].(string)
		
		// Find plan
		if planData, exists := m.plans[planID]; exists {
			return &MockSubscriptionInfo{
				Plan: MockRecord{Data: planData, Id: planID},
			}, nil
		}
		return nil, fmt.Errorf("plan not found")
	}
	return nil, fmt.Errorf("subscription not found")
}

// Adapted validation functions to work with mock types
func validateUsageLimitsMock(app *MockApp, userID string, hoursToAdd float64) error {
	currentMonth := time.Now().Format("2006-01")
	
	monthlyUsageRecord, err := app.FindFirstRecordByFilter("monthly_usage", 
		"user_id = {:user_id} && year_month = {:month}", 
		map[string]interface{}{
			"user_id": userID,
			"month":   currentMonth,
		})
	
	var currentHoursUsed float64
	if err != nil {
		currentHoursUsed = 0
	} else {
		currentHoursUsed = monthlyUsageRecord.GetFloat("hours_used")
	}
	
	subscriptionInfo, err := app.GetUserSubscriptionInfo(userID)
	if err != nil {
		return fmt.Errorf("unable to determine subscription plan: %w", err)
	}
	
	monthlyLimitHours := subscriptionInfo.Plan.GetFloat("hours_per_month")
	projectedUsage := currentHoursUsed + hoursToAdd
	
	if projectedUsage > monthlyLimitHours {
		planName := subscriptionInfo.Plan.GetString("name")
		return fmt.Errorf("monthly limit of %.1f hours exceeded for %s plan (currently used: %.2f hours, requested: %.2f hours)", 
			monthlyLimitHours, planName, currentHoursUsed, hoursToAdd)
	}
	
	return nil
}

func updateUsageAfterProcessingMock(app *MockApp, userID string, durationSeconds float64) error {
	hoursUsed := durationSeconds / 3600.0
	currentMonth := time.Now().Format("2006-01")
	
	monthlyUsageRecord, err := app.FindFirstRecordByFilter("monthly_usage",
		"user_id = {:user_id} && year_month = {:month}",
		map[string]interface{}{
			"user_id": userID,
			"month":   currentMonth,
		})
	
	if err != nil {
		collection, _ := app.FindCollectionByNameOrId("monthly_usage")
		record := NewRecord(collection)
		record.Set("user_id", userID)
		record.Set("year_month", currentMonth)
		record.Set("hours_used", hoursUsed)
		record.Set("files_processed", 1)
		record.Set("last_processing_date", time.Now())
		
		return app.Save(record)
	} else {
		currentHours := monthlyUsageRecord.GetFloat("hours_used")
		currentFiles := monthlyUsageRecord.GetInt("files_processed")
		
		monthlyUsageRecord.Set("hours_used", currentHours + hoursUsed)
		monthlyUsageRecord.Set("files_processed", currentFiles + 1)
		monthlyUsageRecord.Set("last_processing_date", time.Now())
		
		return app.Save(monthlyUsageRecord)
	}
}

// UNIT TESTS

// Test validateUsageLimits - User under limit should be allowed
func TestValidateUsageLimits_UnderLimit_ShouldAllow(t *testing.T) {
	app := NewMockApp()
	
	// Setup: User with 10-hour plan, used 5 hours
	userID := "test_user_1"
	planID := "basic_plan"
	app.SetPlan(planID, "Basic Plan", 10.0)
	app.SetUserSubscription(userID, planID)
	
	currentMonth := time.Now().Format("2006-01")
	app.SetMonthlyUsage(userID, currentMonth, 5.0, 10)
	
	// Try to add 2 more hours (should be allowed: 5 + 2 = 7 < 10)
	err := validateUsageLimitsMock(app, userID, 2.0)
	
	if err != nil {
		t.Fatalf("Expected no error for user under limit, got: %v", err)
	}
}

// Test validateUsageLimits - User at limit should be blocked
func TestValidateUsageLimits_AtLimit_ShouldBlock(t *testing.T) {
	app := NewMockApp()
	
	// Setup: User with 10-hour plan, used exactly 10 hours
	userID := "test_user_2"
	planID := "basic_plan"
	app.SetPlan(planID, "Basic Plan", 10.0)
	app.SetUserSubscription(userID, planID)
	
	currentMonth := time.Now().Format("2006-01")
	app.SetMonthlyUsage(userID, currentMonth, 10.0, 20)
	
	// Try to add 0.5 more hours (should be blocked: 10 + 0.5 > 10)
	err := validateUsageLimitsMock(app, userID, 0.5)
	
	if err == nil {
		t.Fatal("Expected error for user at limit, got nil")
	}
	
	if !strings.Contains(err.Error(), "monthly limit of 10.0 hours exceeded") {
		t.Fatalf("Expected limit exceeded error, got: %s", err.Error())
	}
}

// Test validateUsageLimits - User over limit should be blocked  
func TestValidateUsageLimits_OverLimit_ShouldBlock(t *testing.T) {
	app := NewMockApp()
	
	// Setup: User with 2-hour plan (free), used 3 hours (already over)
	userID := "test_user_3"
	planID := "free_plan"
	app.SetPlan(planID, "Free Plan", 2.0)
	app.SetUserSubscription(userID, planID)
	
	currentMonth := time.Now().Format("2006-01")
	app.SetMonthlyUsage(userID, currentMonth, 3.0, 6)
	
	// Try to add 1 more hour (should be blocked: 3 + 1 > 2)
	err := validateUsageLimitsMock(app, userID, 1.0)
	
	if err == nil {
		t.Fatal("Expected error for user over limit, got nil")
	}
	
	if !strings.Contains(err.Error(), "monthly limit of 2.0 hours exceeded") {
		t.Fatalf("Expected limit exceeded error, got: %s", err.Error())
	}
}

// Test validateUsageLimits - New user should be allowed
func TestValidateUsageLimits_NewUser_ShouldAllow(t *testing.T) {
	app := NewMockApp()
	
	// Setup: User with 10-hour plan but NO usage record
	userID := "new_user"
	planID := "basic_plan"
	app.SetPlan(planID, "Basic Plan", 10.0)
	app.SetUserSubscription(userID, planID)
	
	// No monthly usage record exists (new user)
	// Try to add 3 hours (should be allowed: 0 + 3 = 3 < 10)
	err := validateUsageLimitsMock(app, userID, 3.0)
	
	if err != nil {
		t.Fatalf("Expected no error for new user, got: %v", err)
	}
}

// Test validateUsageLimits - Different plan limits
func TestValidateUsageLimits_DifferentPlanLimits(t *testing.T) {
	app := NewMockApp()
	currentMonth := time.Now().Format("2006-01")
	
	// Test Free Plan (2 hours)
	freeUserID := "free_user"
	freePlanID := "free_plan"
	app.SetPlan(freePlanID, "Free", 2.0)
	app.SetUserSubscription(freeUserID, freePlanID)
	app.SetMonthlyUsage(freeUserID, currentMonth, 1.5, 3)
	
	// Should allow 0.4 hours (1.5 + 0.4 = 1.9 < 2.0)
	err := validateUsageLimitsMock(app, freeUserID, 0.4)
	if err != nil {
		t.Fatalf("Free plan should allow usage under 2 hours, got: %v", err)
	}
	
	// Should block 0.6 hours (1.5 + 0.6 = 2.1 > 2.0)
	err = validateUsageLimitsMock(app, freeUserID, 0.6)
	if err == nil {
		t.Fatal("Free plan should block usage over 2 hours")
	}
	
	// Test Pro Plan (50 hours)
	proUserID := "pro_user"
	proPlanID := "pro_plan"
	app.SetPlan(proPlanID, "Pro", 50.0)
	app.SetUserSubscription(proUserID, proPlanID)
	app.SetMonthlyUsage(proUserID, currentMonth, 45.0, 100)
	
	// Should allow 4 hours (45 + 4 = 49 < 50)
	err = validateUsageLimitsMock(app, proUserID, 4.0)
	if err != nil {
		t.Fatalf("Pro plan should allow usage under 50 hours, got: %v", err)
	}
	
	// Should block 6 hours (45 + 6 = 51 > 50)
	err = validateUsageLimitsMock(app, proUserID, 6.0)
	if err == nil {
		t.Fatal("Pro plan should block usage over 50 hours")
	}
}

// Test validateUsageLimits - Error cases
func TestValidateUsageLimits_ErrorCases(t *testing.T) {
	app := NewMockApp()
	
	// Test invalid user ID (no subscription)
	err := validateUsageLimitsMock(app, "invalid_user", 1.0)
	if err == nil {
		t.Fatal("Expected error for invalid user ID, got nil")
	}
	if !strings.Contains(err.Error(), "unable to determine subscription plan") {
		t.Fatalf("Expected subscription error, got: %s", err.Error())
	}
}

// Test updateUsageAfterProcessing - New user record creation
func TestUpdateUsageAfterProcessing_NewUser_CreatesRecord(t *testing.T) {
	app := NewMockApp()
	userID := "new_usage_user"
	
	// Process 1800 seconds (0.5 hours)
	err := updateUsageAfterProcessingMock(app, userID, 1800.0)
	if err != nil {
		t.Fatalf("Expected no error creating new usage record, got: %v", err)
	}
	
	// Verify record was created
	currentMonth := time.Now().Format("2006-01")
	key := fmt.Sprintf("%s-%s", userID, currentMonth)
	
	if data, exists := app.monthlyUsageRecords[key]; exists {
		if hoursUsed := data["hours_used"].(float64); hoursUsed != 0.5 {
			t.Fatalf("Expected 0.5 hours used, got: %f", hoursUsed)
		}
		if filesProcessed := data["files_processed"].(int); filesProcessed != 1 {
			t.Fatalf("Expected 1 file processed, got: %d", filesProcessed)
		}
	} else {
		t.Fatal("Expected monthly usage record to be created")
	}
}

// Test updateUsageAfterProcessing - Existing record update  
func TestUpdateUsageAfterProcessing_ExistingRecord_UpdatesValues(t *testing.T) {
	app := NewMockApp()
	userID := "existing_usage_user"
	
	// Create existing usage record: 2.5 hours, 5 files
	currentMonth := time.Now().Format("2006-01")
	app.SetMonthlyUsage(userID, currentMonth, 2.5, 5)
	
	// Process 3600 seconds (1.0 hours)
	err := updateUsageAfterProcessingMock(app, userID, 3600.0)
	if err != nil {
		t.Fatalf("Expected no error updating usage record, got: %v", err)
	}
	
	// Verify record was updated
	key := fmt.Sprintf("%s-%s", userID, currentMonth)
	if data, exists := app.monthlyUsageRecords[key]; exists {
		// Should be 2.5 + 1.0 = 3.5 hours
		if hoursUsed := data["hours_used"].(float64); hoursUsed != 3.5 {
			t.Fatalf("Expected 3.5 hours used, got: %f", hoursUsed)
		}
		// Should be 5 + 1 = 6 files
		if filesProcessed := data["files_processed"].(int); filesProcessed != 6 {
			t.Fatalf("Expected 6 files processed, got: %d", filesProcessed)
		}
	} else {
		t.Fatal("Expected monthly usage record to exist")
	}
}

// Test end-to-end usage limit enforcement workflow
func TestUsageLimitWorkflow_EndToEnd(t *testing.T) {
	app := NewMockApp()
	
	// Create user with 2-hour limit
	userID := "workflow_user"
	planID := "limited_plan"
	app.SetPlan(planID, "Limited Plan", 2.0)
	app.SetUserSubscription(userID, planID)
	
	// First file: 1.5 hours (should work)
	err := validateUsageLimitsMock(app, userID, 1.5)
	if err != nil {
		t.Fatalf("First file should be allowed, got: %v", err)
	}
	
	err = updateUsageAfterProcessingMock(app, userID, 5400.0) // 1.5 hours in seconds
	if err != nil {
		t.Fatalf("Failed to update usage after first file: %v", err)
	}
	
	// Second file: 0.3 hours (should work, total = 1.8 < 2.0)
	err = validateUsageLimitsMock(app, userID, 0.3)
	if err != nil {
		t.Fatalf("Second file should be allowed, got: %v", err)
	}
	
	err = updateUsageAfterProcessingMock(app, userID, 1080.0) // 0.3 hours in seconds
	if err != nil {
		t.Fatalf("Failed to update usage after second file: %v", err)
	}
	
	// Third file: 0.5 hours (should be blocked, total would be 2.3 > 2.0)
	err = validateUsageLimitsMock(app, userID, 0.5)
	if err == nil {
		t.Fatal("Third file should be blocked due to limit")
	}
	
	if !strings.Contains(err.Error(), "monthly limit of 2.0 hours exceeded") {
		t.Fatalf("Expected limit error message, got: %s", err.Error())
	}
	
	// Verify final usage
	currentMonth := time.Now().Format("2006-01")
	key := fmt.Sprintf("%s-%s", userID, currentMonth)
	if data, exists := app.monthlyUsageRecords[key]; exists {
		expectedHours := 1.8
		actualHours := data["hours_used"].(float64)
		if abs(actualHours-expectedHours) > 0.001 {
			t.Fatalf("Expected %.1f hours used, got: %f", expectedHours, actualHours)
		}
		
		if filesProcessed := data["files_processed"].(int); filesProcessed != 2 {
			t.Fatalf("Expected 2 files processed, got: %d", filesProcessed)
		}
	} else {
		t.Fatal("Expected monthly usage record to exist")
	}
}

// Helper function
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}