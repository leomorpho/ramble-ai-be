package stripe

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// UsageInfo represents a user's current usage status
type UsageInfo struct {
	UserID              string  `json:"user_id"`
	CurrentPlanID       string  `json:"current_plan_id"`
	PlanName            string  `json:"plan_name"`
	HoursLimit          float64 `json:"hours_limit"`
	HoursUsed           float64 `json:"hours_used"`
	HoursRemaining      float64 `json:"hours_remaining"`
	UsagePercentage     float64 `json:"usage_percentage"`
	FilesProcessed      int     `json:"files_processed"`
	PeriodStart         string  `json:"period_start"`
	PeriodEnd           string  `json:"period_end"`
	IsOverLimit         bool    `json:"is_over_limit"`
	CanProcessMore      bool    `json:"can_process_more"`
	SubscriptionStatus  string  `json:"subscription_status"`
	BillingInterval     string  `json:"billing_interval"`
}

// GetUserUsageInfo retrieves comprehensive usage information for a user
func GetUserUsageInfo(app core.App, userID string) (*UsageInfo, error) {
	// Get user's current subscription
	subscription, err := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id}", map[string]any{
		"user_id": userID,
	})
	if err != nil {
		return nil, fmt.Errorf("no subscription found for user %s: %w", userID, err)
	}

	// Get the subscription plan
	plan, err := app.FindRecordById("subscription_plans", subscription.GetString("plan_id"))
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription plan: %w", err)
	}

	// Get current month usage
	currentMonth := time.Now().Format("2006-01")
	usage, err := getCurrentMonthUsage(app, userID, currentMonth)
	if err != nil {
		return nil, fmt.Errorf("failed to get current month usage: %w", err)
	}

	// Calculate usage info
	hoursLimit := plan.GetFloat("hours_per_month")
	hoursUsed := usage.HoursUsed
	hoursRemaining := hoursLimit - hoursUsed
	if hoursRemaining < 0 {
		hoursRemaining = 0
	}

	var usagePercentage float64
	if hoursLimit > 0 {
		usagePercentage = (hoursUsed / hoursLimit) * 100
	}

	isOverLimit := hoursUsed >= hoursLimit
	canProcessMore := !isOverLimit

	return &UsageInfo{
		UserID:              userID,
		CurrentPlanID:       plan.Id,
		PlanName:            plan.GetString("name"),
		HoursLimit:          hoursLimit,
		HoursUsed:           hoursUsed,
		HoursRemaining:      hoursRemaining,
		UsagePercentage:     usagePercentage,
		FilesProcessed:      usage.FilesProcessed,
		PeriodStart:         subscription.GetDateTime("current_period_start").Time().Format("2006-01-02"),
		PeriodEnd:           subscription.GetDateTime("current_period_end").Time().Format("2006-01-02"),
		IsOverLimit:         isOverLimit,
		CanProcessMore:      canProcessMore,
		SubscriptionStatus:  subscription.GetString("status"),
		BillingInterval:     plan.GetString("billing_interval"),
	}, nil
}

// MonthlyUsage represents usage data for a month
type MonthlyUsage struct {
	HoursUsed      float64
	FilesProcessed int
}

// getCurrentMonthUsage gets or creates the current month's usage record
func getCurrentMonthUsage(app core.App, userID, yearMonth string) (*MonthlyUsage, error) {
	// Try to find existing usage record
	record, err := app.FindFirstRecordByFilter("monthly_usage", "user_id = {:user_id} AND year_month = {:year_month}", map[string]any{
		"user_id":    userID,
		"year_month": yearMonth,
	})

	if err != nil {
		// Create new usage record
		collection, err := app.FindCollectionByNameOrId("monthly_usage")
		if err != nil {
			return nil, err
		}

		record = core.NewRecord(collection)
		record.Set("user_id", userID)
		record.Set("year_month", yearMonth)
		record.Set("hours_used", 0.0)
		record.Set("files_processed", 0)
		record.Set("last_processing_date", time.Now())

		if err := app.Save(record); err != nil {
			return nil, err
		}

		return &MonthlyUsage{
			HoursUsed:      0.0,
			FilesProcessed: 0,
		}, nil
	}

	return &MonthlyUsage{
		HoursUsed:      record.GetFloat("hours_used"),
		FilesProcessed: int(record.GetFloat("files_processed")),
	}, nil
}

// ValidateUsageLimits checks if a user can process more files based on their plan limits
func ValidateUsageLimits(app core.App, userID string, additionalHours float64) error {
	usageInfo, err := GetUserUsageInfo(app, userID)
	if err != nil {
		return fmt.Errorf("failed to get usage info: %w", err)
	}

	// Check if subscription is active
	if usageInfo.SubscriptionStatus != "active" && usageInfo.SubscriptionStatus != "trialing" {
		return fmt.Errorf("subscription is not active (status: %s)", usageInfo.SubscriptionStatus)
	}

	// Check if adding this file would exceed the limit
	totalAfterProcessing := usageInfo.HoursUsed + additionalHours
	if totalAfterProcessing > usageInfo.HoursLimit {
		return fmt.Errorf("processing this file would exceed your monthly limit of %.1f hours. You have %.2f hours remaining", 
			usageInfo.HoursLimit, usageInfo.HoursRemaining)
	}

	return nil
}

// UpdateUsageAfterProcessing updates usage statistics after successful file processing
func UpdateUsageAfterProcessing(app core.App, userID string, durationSeconds float64) error {
	// Convert seconds to hours
	hoursProcessed := durationSeconds / 3600.0

	currentMonth := time.Now().Format("2006-01")

	// Get or create usage record
	collection, err := app.FindCollectionByNameOrId("monthly_usage")
	if err != nil {
		return err
	}

	record, err := app.FindFirstRecordByFilter("monthly_usage", "user_id = {:user_id} AND year_month = {:year_month}", map[string]any{
		"user_id":    userID,
		"year_month": currentMonth,
	})

	if err != nil {
		// Create new record
		record = core.NewRecord(collection)
		record.Set("user_id", userID)
		record.Set("year_month", currentMonth)
		record.Set("hours_used", hoursProcessed)
		record.Set("files_processed", 1)
		record.Set("last_processing_date", time.Now())
	} else {
		// Update existing record
		currentHours := record.GetFloat("hours_used")
		currentFiles := int(record.GetFloat("files_processed"))

		record.Set("hours_used", currentHours+hoursProcessed)
		record.Set("files_processed", currentFiles+1)
		record.Set("last_processing_date", time.Now())
	}

	if err := app.Save(record); err != nil {
		return fmt.Errorf("failed to update usage: %w", err)
	}

	log.Printf("Updated usage for user %s: +%.2f hours (total: %.2f hours)", userID, hoursProcessed, record.GetFloat("hours_used"))
	return nil
}

// GetUsageWarningMessage returns a warning message if user is approaching their limits
func GetUsageWarningMessage(usageInfo *UsageInfo) string {
	if usageInfo.UsagePercentage >= 90 {
		return fmt.Sprintf("⚠️ You've used %.1f%% of your monthly quota (%.1f/%.1f hours). Consider upgrading your plan.", 
			usageInfo.UsagePercentage, usageInfo.HoursUsed, usageInfo.HoursLimit)
	}
	
	if usageInfo.UsagePercentage >= 75 {
		return fmt.Sprintf("You've used %.1f hours of your %.1f hour monthly quota (%.1f%%).", 
			usageInfo.HoursUsed, usageInfo.HoursLimit, usageInfo.UsagePercentage)
	}

	return ""
}

// ResetMonthlyUsage resets usage for a specific month (for testing or corrections)
func ResetMonthlyUsage(app core.App, userID, yearMonth string) error {
	record, err := app.FindFirstRecordByFilter("monthly_usage", "user_id = {:user_id} AND year_month = {:year_month}", map[string]any{
		"user_id":    userID,
		"year_month": yearMonth,
	})

	if err != nil {
		return fmt.Errorf("usage record not found: %w", err)
	}

	record.Set("hours_used", 0.0)
	record.Set("files_processed", 0)
	record.Set("last_processing_date", time.Now())

	return app.Save(record)
}

// GetUserSubscription gets the current active subscription for a user
func GetUserSubscription(app core.App, userID string) (*core.Record, error) {
	return app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id}", map[string]any{
		"user_id": userID,
	})
}

// GetSubscriptionPlan gets a subscription plan by ID
func GetSubscriptionPlan(app core.App, planID string) (*core.Record, error) {
	return app.FindRecordById("subscription_plans", planID)
}