package stripe

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v79"
	billingportal "github.com/stripe/stripe-go/v79/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/subscription"
)

// CreateCheckoutSessionRequest represents the request payload for creating a checkout session
type CreateCheckoutSessionRequest struct {
	PlanID string `json:"plan_id"`
	UserID string `json:"user_id"`
}

// CreatePortalLinkRequest represents the request payload for creating a portal link
type CreatePortalLinkRequest struct {
	UserID string `json:"user_id"`
}

// ChangePlanRequest represents the request payload for directly changing plans
type ChangePlanRequest struct {
	UserID string `json:"user_id"`
	PlanID string `json:"plan_id"`
}

// getBaseURL returns the base URL for the application, falling back to localhost:8090 if HOST is not set
func getBaseURL() string {
	host := os.Getenv("HOST")
	if host == "" {
		host = "http://localhost:8090"
	}
	// Ensure HOST doesn't have trailing slash
	host = strings.TrimSuffix(host, "/")
	return host
}

// getFrontendURL returns the frontend URL for redirects, falling back to localhost:5173 if FRONTEND_URL is not set
func getFrontendURL() string {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}
	// Ensure FRONTEND_URL doesn't have trailing slash
	frontendURL = strings.TrimSuffix(frontendURL, "/")
	return frontendURL
}

// CreateCheckoutSession handles the creation of Stripe checkout sessions
func CreateCheckoutSession(e *core.RequestEvent, app *pocketbase.PocketBase) error {
	var data CreateCheckoutSessionRequest

	if err := e.BindBody(&data); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Get the subscription plan
	plan, err := app.FindRecordById("subscription_plans", data.PlanID)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid plan ID"})
	}

	// All plans (including free $0.00 plans) can go through Stripe checkout

	stripePriceID := plan.GetString("provider_price_id")
	if stripePriceID == "" {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Plan has no Stripe price ID"})
	}

	// Get or create Stripe customer for user
	customerID, err := getOrCreateStripeCustomer(app, data.UserID)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Create checkout session
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(stripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String("subscription"),
		SuccessURL: stripe.String(getFrontendURL() + "/pricing?success=true"),
		CancelURL:  stripe.String(getFrontendURL() + "/pricing?canceled=true"),
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"user_id": data.UserID,
				"plan_id": data.PlanID,
			},
		},
		AllowPromotionCodes: stripe.Bool(true),
	}

	s, err := checkoutsession.New(params)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]string{"url": s.URL})
}

// CreatePortalLink handles the creation of Stripe billing portal links
func CreatePortalLink(e *core.RequestEvent, app *pocketbase.PocketBase) error {
	var data CreatePortalLinkRequest

	if err := e.BindBody(&data); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Check if user has an active subscription
	_, err := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id} && status = 'active'", map[string]any{
		"user_id": data.UserID,
	})
	if err != nil {
		// No active subscription found, redirect to pricing page
		return e.JSON(http.StatusOK, map[string]string{"url": getBaseURL() + "/pricing"})
	}

	// Get or create Stripe customer ID for all users (including free plan users)
	customerID, err := getOrCreateStripeCustomer(app, data.UserID)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create Stripe customer. Please contact support."})
	}

	// Create portal session for all users
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(getFrontendURL() + "/pricing"),
	}

	ps, err := billingportal.New(params)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]string{"url": ps.URL})
}

// ChangePlan handles direct plan changes without checkout (for downgrades/cancellations)
func ChangePlan(e *core.RequestEvent, app *pocketbase.PocketBase) error {
	var data ChangePlanRequest

	if err := e.BindBody(&data); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Get the user's current active subscription
	currentSub, err := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id} && status = 'active'", map[string]any{
		"user_id": data.UserID,
	})
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "No active subscription found"})
	}

	// Get the target plan details
	targetPlan, err := app.FindFirstRecordByFilter("subscription_plans", "id = {:plan_id}", map[string]any{
		"plan_id": data.PlanID,
	})
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid plan ID"})
	}

	// Get current plan details to determine if this is an upgrade or downgrade
	currentPlan, err := app.FindFirstRecordByFilter("subscription_plans", "id = {:plan_id}", map[string]any{
		"plan_id": currentSub.GetString("plan_id"),
	})
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Current plan not found"})
	}

	// Determine if this is an upgrade or downgrade based on pricing
	currentPrice := currentPlan.GetInt("price_cents")
	targetPrice := targetPlan.GetInt("price_cents")
	isUpgrade := targetPrice > currentPrice

	log.Printf("Plan change: %s (%d cents) -> %s (%d cents), isUpgrade: %v", 
		currentPlan.GetString("name"), currentPrice, 
		targetPlan.GetString("name"), targetPrice, isUpgrade)

	// Get Stripe subscription ID
	stripeSubID := currentSub.GetString("provider_subscription_id")
	if stripeSubID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "No Stripe subscription ID found"})
	}

	// Handle upgrades vs downgrades differently
	if isUpgrade {
		// UPGRADES: Apply immediately with proration (user gets better benefits right away)
		return handleUpgrade(e, stripeSubID, targetPlan)
	} else {
		// DOWNGRADES: Use cancel_at_period_end to preserve current benefits until period ends
		return handleDowngrade(e, app, stripeSubID, currentSub, targetPlan, data.UserID)
	}

	// This code should not be reached due to the upgrade/downgrade handling above
	return e.JSON(http.StatusInternalServerError, map[string]string{
		"error": "Unexpected code path in plan change",
	})
}

// handleUpgrade processes plan upgrades - applies changes immediately with proration
func handleUpgrade(e *core.RequestEvent, stripeSubID string, targetPlan *core.Record) error {
	log.Printf("Processing UPGRADE: Applying immediately with proration")
	
	stripePriceID := targetPlan.GetString("provider_price_id")
	
	// Get the current subscription to find the subscription item ID to update
	currentStripeSub, err := subscription.Get(stripeSubID, nil)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get current subscription from Stripe: " + err.Error(),
		})
	}
	
	if len(currentStripeSub.Items.Data) == 0 {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Current subscription has no items to update",
		})
	}
	
	// Update the subscription by replacing the existing item with the new price
	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(currentStripeSub.Items.Data[0].ID),
				Price: stripe.String(stripePriceID),
			},
		},
	}
	
	// For upgrades, apply changes immediately with proration
	params.ProrationBehavior = stripe.String("always_invoice")
	
	updatedSub, err := subscription.Update(stripeSubID, params)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update subscription in Stripe: " + err.Error(),
		})
	}

	return e.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Upgrade applied immediately - you now have access to enhanced features!",
		"provider_subscription_id": updatedSub.ID,
		"new_plan": targetPlan.Id,
		"change_type": "upgrade",
	})
}

// handleDowngrade processes plan downgrades - preserves current benefits until period end
func handleDowngrade(e *core.RequestEvent, app *pocketbase.PocketBase, stripeSubID string, currentSub *core.Record, targetPlan *core.Record, userID string) error {
	log.Printf("Processing DOWNGRADE: Preserving current benefits until period end")
	
	// CRITICAL: For downgrades, we need to preserve current benefits until the billing period ends
	// We do this by setting cancel_at_period_end=true and creating a scheduled plan change
	
	// Step 1: Set current subscription to cancel at period end
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	
	updatedSub, err := subscription.Update(stripeSubID, params)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to schedule downgrade in Stripe: " + err.Error(),
		})
	}

	// Step 2: Create a record of the pending plan change for the period end
	// This will be processed when the subscription actually cancels/renews
	if err := createPendingPlanChange(app, userID, targetPlan.Id, updatedSub.CurrentPeriodEnd); err != nil {
		log.Printf("Warning: Failed to create pending plan change record: %v", err)
		// Don't fail the request - the Stripe webhook will handle the actual change
	}

	// Format the period end date for user-friendly display
	periodEndDate := time.Unix(updatedSub.CurrentPeriodEnd, 0).Format("January 2, 2006")

	return e.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Downgrade scheduled - you'll keep your current benefits until %s, then switch to the new plan", periodEndDate),
		"provider_subscription_id": updatedSub.ID,
		"new_plan": targetPlan.Id,
		"change_type": "downgrade",
		"effective_date": periodEndDate,
		"cancel_at_period_end": true,
	})
}

// createPendingPlanChange creates a record to track plan changes that will happen at period end
func createPendingPlanChange(app *pocketbase.PocketBase, userID string, newPlanID string, effectiveDate int64) error {
	// This is a helper function to track pending plan changes
	// In a full implementation, you might want a separate collection for this
	// For now, we'll just log it - the webhook handling will manage the actual change
	
	log.Printf("Pending plan change for user %s: new plan %s effective %s", 
		userID, newPlanID, time.Unix(effectiveDate, 0).Format("2006-01-02"))
	
	// TODO: Implement pending_plan_changes collection if needed for complex scenarios
	// This would track scheduled plan changes that need to happen at period end
	
	return nil
}