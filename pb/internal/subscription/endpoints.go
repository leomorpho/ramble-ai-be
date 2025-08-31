package subscription

import (
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

// ChangePlanHandler handles requests to change subscription plans
func ChangePlanHandler(e *core.RequestEvent, app core.App, subscriptionService Service) error {
	// Parse request body
	var req struct {
		PlanID string `json:"plan_id"`
		UserID string `json:"user_id"`
	}
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// This endpoint should only be used for actual plan changes, not cancellations
	// For cancellations, clients should use /api/subscription/cancel endpoint
	
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

	// For paid plan changes, use checkout flow
	return e.JSON(http.StatusBadRequest, map[string]string{"error": "Use checkout for paid plan changes"})
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