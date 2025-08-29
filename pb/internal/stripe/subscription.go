package stripe

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// GetUserSubscriptionRequest represents the request for getting user subscription info
type GetUserSubscriptionRequest struct {
	UserID string `json:"user_id"`
}

// SubscriptionInfo represents comprehensive subscription information
type SubscriptionInfo struct {
	Subscription *core.Record      `json:"subscription"`
	Plan         *core.Record      `json:"plan"`
	Usage        *UsageInfo        `json:"usage"`
	AvailablePlans []*core.Record  `json:"available_plans"`
}

// GetUserSubscriptionInfo retrieves complete subscription information for a user
func GetUserSubscriptionInfo(e *core.RequestEvent, app core.App) error {
	userID := e.Request.URL.Query().Get("user_id")
	if userID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "user_id is required"})
	}

	// Get user's subscription
	subscription, err := GetUserSubscription(app, userID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "No subscription found"})
	}

	// Get subscription plan
	plan, err := GetSubscriptionPlan(app, subscription.GetString("plan_id"))
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get plan details"})
	}

	// Get usage information
	usage, err := GetUserUsageInfo(app, userID)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get usage info"})
	}

	// Get all available plans
	availablePlans, err := app.FindRecordsByFilter("subscription_plans", "is_active = true", "+display_order", 0, 0)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get available plans"})
	}

	info := &SubscriptionInfo{
		Subscription:   subscription,
		Plan:          plan,
		Usage:         usage,
		AvailablePlans: availablePlans,
	}

	return e.JSON(http.StatusOK, info)
}

// SwitchToFreePlanRequest represents the request to switch to free plan
type SwitchToFreePlanRequest struct {
	UserID string `json:"user_id"`
}

// SwitchToFreePlan moves a user to the free plan (for downgrades or cancellations)
func SwitchToFreePlan(e *core.RequestEvent, app core.App) error {
	var data SwitchToFreePlanRequest

	if err := e.BindBody(&data); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Get free plan
	freePlan, err := app.FindFirstRecordByFilter("subscription_plans", "billing_interval = 'free'", map[string]any{})
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to find free plan"})
	}

	// Get user's current subscription
	subscription, err := GetUserSubscription(app, data.UserID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "No subscription found"})
	}

	// Update subscription to free plan
	now := time.Now()
	oneYearFromNow := now.AddDate(1, 0, 0) // Free plan lasts 1 year

	subscription.Set("plan_id", freePlan.Id)
	subscription.Set("stripe_subscription_id", nil) // Remove Stripe subscription
	subscription.Set("status", "active")
	subscription.Set("current_period_start", now)
	subscription.Set("current_period_end", oneYearFromNow)
	subscription.Set("cancel_at_period_end", false)
	subscription.Set("canceled_at", now) // Mark when downgraded

	if err := app.Save(subscription); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update subscription"})
	}

	return e.JSON(http.StatusOK, map[string]string{
		"message": "Successfully switched to free plan",
		"plan_id": freePlan.Id,
	})
}

// GetAvailablePlans returns all available subscription plans
func GetAvailablePlans(e *core.RequestEvent, app core.App) error {
	plans, err := app.FindRecordsByFilter("subscription_plans", "is_active = true", "+display_order", 0, 0)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get plans"})
	}

	// Group plans by tier for easier frontend consumption
	plansByTier := make(map[string][]*core.Record)
	for _, plan := range plans {
		tier := plan.GetString("name") // Using name as tier identifier for now
		if plan.GetString("billing_interval") == "free" {
			tier = "free"
		} else if plan.GetString("name") == "Basic Monthly" || plan.GetString("name") == "Basic Yearly" {
			tier = "basic"
		} else if plan.GetString("name") == "Pro Monthly" || plan.GetString("name") == "Pro Yearly" {
			tier = "pro"
		}
		
		plansByTier[tier] = append(plansByTier[tier], plan)
	}

	return e.JSON(http.StatusOK, map[string]interface{}{
		"plans":          plans,
		"plans_by_tier": plansByTier,
	})
}

// GetUsageStats returns usage statistics for a user
func GetUsageStats(e *core.RequestEvent, app core.App) error {
	userID := e.Request.URL.Query().Get("user_id")
	if userID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "user_id is required"})
	}

	usage, err := GetUserUsageInfo(app, userID)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get usage info"})
	}

	// Add warning message if approaching limits
	warningMessage := GetUsageWarningMessage(usage)

	response := map[string]interface{}{
		"usage": usage,
	}

	if warningMessage != "" {
		response["warning"] = warningMessage
	}

	return e.JSON(http.StatusOK, response)
}

// CreateFreePlanSubscription creates a free plan subscription for a new user
func CreateFreePlanSubscription(app core.App, userID string) error {
	// Get free plan
	freePlan, err := app.FindFirstRecordByFilter("subscription_plans", "billing_interval = 'free'", map[string]any{})
	if err != nil {
		return fmt.Errorf("failed to find free plan: %w", err)
	}

	// Check if user already has a subscription
	existingSubscription, err := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id}", map[string]any{
		"user_id": userID,
	})

	if err == nil {
		// User already has a subscription
		return fmt.Errorf("user already has subscription: %s", existingSubscription.Id)
	}

	// Create new free subscription
	collection, err := app.FindCollectionByNameOrId("user_subscriptions")
	if err != nil {
		return err
	}

	now := time.Now()
	oneYearFromNow := now.AddDate(1, 0, 0) // Free plan lasts 1 year

	record := core.NewRecord(collection)
	record.Set("user_id", userID)
	record.Set("plan_id", freePlan.Id)
	record.Set("status", "active")
	record.Set("current_period_start", now)
	record.Set("current_period_end", oneYearFromNow)
	record.Set("cancel_at_period_end", false)

	return app.Save(record)
}

// GetPlanUpgrades returns available upgrade options for a user's current plan
func GetPlanUpgrades(e *core.RequestEvent, app core.App) error {
	userID := e.Request.URL.Query().Get("user_id")
	if userID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "user_id is required"})
	}

	// Get user's current subscription
	subscription, err := GetUserSubscription(app, userID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "No subscription found"})
	}

	// Get current plan
	currentPlan, err := GetSubscriptionPlan(app, subscription.GetString("plan_id"))
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get current plan"})
	}

	currentHoursLimit := currentPlan.GetFloat("hours_per_month")

	// Get all plans with higher limits (upgrades only)
	allPlans, err := app.FindRecordsByFilter("subscription_plans", "is_active = true AND hours_per_month > {:current_hours}", "+display_order", 0, 0, map[string]any{
		"current_hours": currentHoursLimit,
	})
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get upgrade options"})
	}

	return e.JSON(http.StatusOK, map[string]interface{}{
		"current_plan": currentPlan,
		"upgrades":     allPlans,
	})
}