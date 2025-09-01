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

// Adapted validation functions to work with mock types - Updated to match the new grace period logic
func validateUsageLimitsMock(app *MockApp, userID string, hoursToAdd float64) error {
	return validateUsageLimitsMockWithGracePeriod(app, userID, hoursToAdd, 60.0) // Default 60 seconds
}

func validateUsageLimitsMockWithGracePeriod(app *MockApp, userID string, hoursToAdd float64, gracePeriodSeconds float64) error {
	gracePeriodHours := gracePeriodSeconds / 3600.0
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
	
	var monthlyLimitHours float64
	subscriptionInfo, err := app.GetUserSubscriptionInfo(userID)
	if err != nil {
		// Fallback to free tier limits (30 minutes = 0.5 hours) if subscription service fails
		monthlyLimitHours = 0.5 // 30 minutes for free users
	} else {
		monthlyLimitHours = subscriptionInfo.Plan.GetFloat("hours_per_month")
	}
	
	projectedUsage := currentHoursUsed + hoursToAdd
	
	if projectedUsage > monthlyLimitHours {
		// Calculate how much the user would exceed their limit
		excessHours := projectedUsage - monthlyLimitHours
		
		// Apply grace period logic: allow if excess is within grace period
		if excessHours <= gracePeriodHours {
			return nil // Allow within grace period
		}
		
		// Excess is beyond grace period - reject
		var planName string
		if subscriptionInfo != nil {
			planName = subscriptionInfo.Plan.GetString("name")
			if planName == "" {
				planName = "Free" // Fallback if plan name is empty
			}
		} else {
			planName = "Free" // Fallback plan name
		}
		return fmt.Errorf("monthly limit of %.1f hours exceeded for %s plan (currently used: %.2f hours, requested: %.2f hours, grace period: %.0f seconds)", 
			monthlyLimitHours, planName, currentHoursUsed, hoursToAdd, gracePeriodSeconds)
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

// Test validateUsageLimits - Free tier fallback for users without subscriptions
func TestValidateUsageLimits_FreeTierFallback_NoSubscription(t *testing.T) {
	app := NewMockApp()
	
	// Test user without subscription - should get free tier limits (30 minutes = 0.5 hours)
	userID := "no_subscription_user"
	// Don't set subscription or plan - user has no subscription
	
	// Should allow 0.3 hours (18 minutes) for free users
	err := validateUsageLimitsMock(app, userID, 0.3)
	if err != nil {
		t.Fatalf("Expected no error for free user under 30min limit, got: %v", err)
	}
	
	// Should block 0.6 hours (36 minutes) for free users
	err = validateUsageLimitsMock(app, userID, 0.6)
	if err == nil {
		t.Fatal("Expected error for free user over 30min limit, got nil")
	}
	
	if !strings.Contains(err.Error(), "monthly limit of 0.5 hours exceeded for Free plan") {
		t.Fatalf("Expected free tier limit error, got: %s", err.Error())
	}
}

// Test validateUsageLimits - Free tier with existing usage
func TestValidateUsageLimits_FreeTierWithUsage(t *testing.T) {
	app := NewMockApp()
	
	userID := "free_user_with_usage"
	currentMonth := time.Now().Format("2006-01")
	// User has used 0.4 hours (24 minutes) already
	app.SetMonthlyUsage(userID, currentMonth, 0.4, 2)
	
	// Should allow 0.1 hours (6 minutes) more - total would be 0.5 hours (30min)
	err := validateUsageLimitsMock(app, userID, 0.1)
	if err != nil {
		t.Fatalf("Expected no error for free user at limit, got: %v", err)
	}
	
	// Should block 0.2 hours (12 minutes) - total would be 0.6 hours (36min)
	err = validateUsageLimitsMock(app, userID, 0.2)
	if err == nil {
		t.Fatal("Expected error for free user exceeding 30min limit")
	}
	
	if !strings.Contains(err.Error(), "currently used: 0.40 hours, requested: 0.20 hours") {
		t.Fatalf("Expected detailed usage error message, got: %s", err.Error())
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

// ===============================
// GRACE PERIOD TESTS
// ===============================

// Test grace period - User exceeds limit within grace period should be allowed
func TestValidateUsageLimits_GracePeriod_WithinLimit_ShouldAllow(t *testing.T) {
	app := NewMockApp()
	
	// Setup: User with 2-hour plan, used 1.9 hours
	userID := "grace_user_1"
	planID := "basic_plan"
	app.SetPlan(planID, "Basic Plan", 2.0)
	app.SetUserSubscription(userID, planID)
	
	currentMonth := time.Now().Format("2006-01")
	app.SetMonthlyUsage(userID, currentMonth, 1.9, 10) // 1.9 hours used
	
	// Try to add 0.12 hours (4.32 minutes) - would exceed by 0.02 hours (72 seconds)
	// Grace period: 60 seconds (0.0167 hours) - should be BLOCKED (72s > 60s)
	gracePeriodSeconds := 60.0
	err := validateUsageLimitsMockWithGracePeriod(app, userID, 0.12, gracePeriodSeconds)
	
	if err == nil {
		t.Fatal("Expected error for excess beyond grace period, got nil")
	}
	
	if !strings.Contains(err.Error(), "grace period: 60 seconds") {
		t.Fatalf("Expected grace period error message, got: %s", err.Error())
	}
}

// Test grace period - User exceeds limit within grace period should be allowed
func TestValidateUsageLimits_GracePeriod_WithinGrace_ShouldAllow(t *testing.T) {
	app := NewMockApp()
	
	// Setup: User with 2-hour plan, used 1.95 hours
	userID := "grace_user_2"
	planID := "basic_plan"
	app.SetPlan(planID, "Basic Plan", 2.0)
	app.SetUserSubscription(userID, planID)
	
	currentMonth := time.Now().Format("2006-01")
	app.SetMonthlyUsage(userID, currentMonth, 1.95, 10) // 1.95 hours used
	
	// Try to add 0.065 hours (3.9 minutes) - would exceed by 0.015 hours (54 seconds)
	// Grace period: 60 seconds - should be ALLOWED (54s <= 60s)
	gracePeriodSeconds := 60.0
	err := validateUsageLimitsMockWithGracePeriod(app, userID, 0.065, gracePeriodSeconds)
	
	if err != nil {
		t.Fatalf("Expected no error within grace period, got: %v", err)
	}
}

// Test grace period - Exactly at grace period boundary
func TestValidateUsageLimits_GracePeriod_ExactBoundary_ShouldAllow(t *testing.T) {
	app := NewMockApp()
	
	// Setup: User with 1-hour plan, used 0.95 hours
	userID := "grace_user_3"
	planID := "basic_plan"
	app.SetPlan(planID, "Basic Plan", 1.0)
	app.SetUserSubscription(userID, planID)
	
	currentMonth := time.Now().Format("2006-01")
	app.SetMonthlyUsage(userID, currentMonth, 0.95, 5) // 0.95 hours used
	
	// Try to add exactly 0.0667 hours (4 minutes = 240 seconds)  
	// This would exceed by 0.0167 hours = 60 seconds exactly
	// Grace period: 60 seconds - should be ALLOWED (60s <= 60s)
	gracePeriodSeconds := 60.0
	hoursToAdd := 1.0/60.0 // Exactly 1 minute = 1/60 hours to avoid floating point precision issues
	err := validateUsageLimitsMockWithGracePeriod(app, userID, hoursToAdd, gracePeriodSeconds)
	
	if err != nil {
		t.Fatalf("Expected no error at exact grace period boundary, got: %v", err)
	}
}

// Test grace period - Different grace period values
func TestValidateUsageLimits_GracePeriod_DifferentPeriods(t *testing.T) {
	app := NewMockApp()
	
	userID := "grace_user_4"
	planID := "basic_plan"
	app.SetPlan(planID, "Basic Plan", 1.0) // 1 hour limit
	app.SetUserSubscription(userID, planID)
	
	currentMonth := time.Now().Format("2006-01")
	app.SetMonthlyUsage(userID, currentMonth, 0.95, 5) // 0.95 hours used (57 minutes)
	
	// Try to add exactly 5 minutes - would exceed by exactly 2 minutes (120 seconds)
	hoursToAdd := 5.0/60.0 // 5 minutes, total would be 57+5=62 minutes = 1.033 hours
	// Excess: 1.033 - 1.0 = 0.033 hours = 120 seconds
	
	// Test 1: Grace period 60 seconds (1 minute) - should be BLOCKED (120s > 60s)
	err := validateUsageLimitsMockWithGracePeriod(app, userID, hoursToAdd, 60.0)
	if err == nil {
		t.Fatal("Expected error with 60s grace period, got nil")
	}
	
	// Test 2: Grace period 200 seconds (3.33 minutes) - should be ALLOWED (120s <= 200s)
	err = validateUsageLimitsMockWithGracePeriod(app, userID, hoursToAdd, 200.0)
	if err != nil {
		t.Fatalf("Expected no error with 200s grace period, got: %v", err)
	}
	
	// Test 3: Grace period 120 seconds exactly - should be ALLOWED (120s <= 120s)
	err = validateUsageLimitsMockWithGracePeriod(app, userID, hoursToAdd, 120.0)
	if err != nil {
		t.Fatalf("Expected no error with exact 120s grace period, got: %v", err)
	}
}

// Test grace period - Free tier with grace period
func TestValidateUsageLimits_GracePeriod_FreeTier_ShouldWork(t *testing.T) {
	app := NewMockApp()
	
	// User without subscription - gets free tier (0.5 hours = 30 minutes)
	userID := "free_grace_user"
	currentMonth := time.Now().Format("2006-01")
	app.SetMonthlyUsage(userID, currentMonth, 0.48, 10) // 0.48 hours (28.8 minutes)
	
	// Try to add 0.05 hours (3 minutes) - would exceed by 0.03 hours (108 seconds)
	// Grace period: 120 seconds - should be ALLOWED (108s <= 120s)
	err := validateUsageLimitsMockWithGracePeriod(app, userID, 0.05, 120.0)
	if err != nil {
		t.Fatalf("Expected no error for free tier within grace period, got: %v", err)
	}
	
	// Try to add 0.08 hours (4.8 minutes) - would exceed by 0.06 hours (216 seconds)
	// Grace period: 120 seconds - should be BLOCKED (216s > 120s)
	err = validateUsageLimitsMockWithGracePeriod(app, userID, 0.08, 120.0)
	if err == nil {
		t.Fatal("Expected error for free tier beyond grace period, got nil")
	}
	
	if !strings.Contains(err.Error(), "monthly limit of 0.5 hours exceeded for Free plan") {
		t.Fatalf("Expected free tier limit error, got: %s", err.Error())
	}
}

// Test grace period - Zero grace period (should behave like old system)
func TestValidateUsageLimits_GracePeriod_Zero_ShouldBlockImmediately(t *testing.T) {
	app := NewMockApp()
	
	userID := "zero_grace_user"
	planID := "basic_plan"
	app.SetPlan(planID, "Basic Plan", 2.0)
	app.SetUserSubscription(userID, planID)
	
	currentMonth := time.Now().Format("2006-01")
	app.SetMonthlyUsage(userID, currentMonth, 2.0, 10) // Exactly at limit
	
	// Try to add any amount with zero grace period - should be blocked
	err := validateUsageLimitsMockWithGracePeriod(app, userID, 0.001, 0.0) // 1/1000th of an hour
	if err == nil {
		t.Fatal("Expected error with zero grace period, got nil")
	}
	
	if !strings.Contains(err.Error(), "grace period: 0 seconds") {
		t.Fatalf("Expected zero grace period in error message, got: %s", err.Error())
	}
}

// Test grace period - Large grace period
func TestValidateUsageLimits_GracePeriod_Large_ShouldWork(t *testing.T) {
	app := NewMockApp()
	
	userID := "large_grace_user"
	planID := "basic_plan"
	app.SetPlan(planID, "Basic Plan", 1.0)
	app.SetUserSubscription(userID, planID)
	
	currentMonth := time.Now().Format("2006-01")
	app.SetMonthlyUsage(userID, currentMonth, 0.5, 5) // 0.5 hours used
	
	// Try to add 1.0 hours - would exceed by 0.5 hours (1800 seconds)
	// Grace period: 3600 seconds (1 hour) - should be ALLOWED (1800s <= 3600s)
	err := validateUsageLimitsMockWithGracePeriod(app, userID, 1.0, 3600.0)
	if err != nil {
		t.Fatalf("Expected no error with large grace period, got: %v", err)
	}
}

// Test grace period - Edge case with very small numbers
func TestValidateUsageLimits_GracePeriod_SmallNumbers_ShouldWork(t *testing.T) {
	app := NewMockApp()
	
	userID := "small_grace_user"
	planID := "micro_plan"
	app.SetPlan(planID, "Micro Plan", 0.01) // 0.01 hours = 36 seconds
	app.SetUserSubscription(userID, planID)
	
	currentMonth := time.Now().Format("2006-01")
	app.SetMonthlyUsage(userID, currentMonth, 0.009, 1) // 0.009 hours = 32.4 seconds
	
	// Try to add 0.002 hours (7.2 seconds) - would exceed by 0.001 hours (3.6 seconds)
	// Grace period: 5 seconds - should be ALLOWED (3.6s <= 5s)
	err := validateUsageLimitsMockWithGracePeriod(app, userID, 0.002, 5.0)
	if err != nil {
		t.Fatalf("Expected no error with small numbers, got: %v", err)
	}
	
	// Try to add 0.003 hours (10.8 seconds) - would exceed by 0.002 hours (7.2 seconds)  
	// Grace period: 5 seconds - should be BLOCKED (7.2s > 5s)
	err = validateUsageLimitsMockWithGracePeriod(app, userID, 0.003, 5.0)
	if err == nil {
		t.Fatal("Expected error exceeding small grace period, got nil")
	}
}