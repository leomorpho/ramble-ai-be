package stripe

import (
	"net/http"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v79"
	billingportal "github.com/stripe/stripe-go/v79/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"
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

	stripePriceID := plan.GetString("stripe_price_id")
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