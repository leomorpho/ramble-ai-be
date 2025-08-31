package subscription

import (
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

// ChangePlanHandler handles requests to change subscription plans with automatic upgrade/downgrade detection
func ChangePlanHandler(e *core.RequestEvent, app core.App, subscriptionService Service) error {
	// Get user info from auth (standard PocketBase pattern)
	user := e.Auth
	if user == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	// Parse request body
	var req struct {
		PlanID string `json:"plan_id"`
		UserID string `json:"user_id"` // Optional - will use authenticated user if not provided
	}
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Use authenticated user ID (ignore request user_id for security)
	userID := user.Id

	// Validate that the target plan exists
	plan, err := app.FindRecordById("subscription_plans", req.PlanID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "Plan not found"})
	}

	// Check if this is a free plan (cancellation attempt)
	if plan.GetInt("price_cents") == 0 {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Use /api/subscription/cancel endpoint for subscription cancellations",
			"hint": "This preserves your benefits until the billing period ends",
		})
	}

	// Use the subscription service to handle the plan change with automatic upgrade/downgrade detection
	// This will compare prices and route upgrades vs downgrades appropriately
	result, err := subscriptionService.ChangePlan(userID, req.PlanID)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to change plan: %v", err),
		})
	}

	return e.JSON(http.StatusOK, result)
}

// Note: GET operations (subscription info, plans, usage stats, plan upgrades) 
// should use PocketBase JavaScript SDK with RLS rules instead of custom endpoints.

// CancelSubscriptionHandler handles requests to cancel a subscription properly via Stripe
func CancelSubscriptionHandler(e *core.RequestEvent, app core.App, subscriptionService Service) error {
	// Get user info from auth (standard PocketBase pattern)
	user := e.Auth
	if user == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	// Cancel subscription via Stripe (sets cancel_at_period_end=true)
	result, err := subscriptionService.CancelSubscription(user.Id)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to cancel subscription: %v", err),
		})
	}

	return e.JSON(http.StatusOK, result)
}

// SwitchToFreePlanHandler handles requests to switch to free plan
func SwitchToFreePlanHandler(e *core.RequestEvent, app core.App, subscriptionService Service) error {
	// TODO: Implement switch to free plan
	return e.JSON(http.StatusNotImplemented, map[string]string{"error": "Not implemented yet"})
}