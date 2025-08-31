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

	// Check if switching to free plan
	plan, err := app.FindRecordById("subscription_plans", req.PlanID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "Plan not found"})
	}

	if plan.GetInt("price_cents") == 0 {
		// Switch to free plan
		_, err := subscriptionService.SwitchToFreePlan(req.UserID)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to switch to free plan: %v", err)})
		}
		return e.JSON(http.StatusOK, map[string]interface{}{"success": true, "message": "Successfully switched to free plan"})
	}

	// For paid plans, redirect to checkout
	return e.JSON(http.StatusBadRequest, map[string]string{"error": "Use checkout for paid plan changes"})
}

// Note: GET operations (subscription info, plans, usage stats, plan upgrades) 
// should use PocketBase JavaScript SDK with RLS rules instead of custom endpoints.

// SwitchToFreePlanHandler handles requests to switch to free plan
func SwitchToFreePlanHandler(e *core.RequestEvent, app core.App, subscriptionService Service) error {
	// TODO: Implement switch to free plan
	return e.JSON(http.StatusNotImplemented, map[string]string{"error": "Not implemented yet"})
}