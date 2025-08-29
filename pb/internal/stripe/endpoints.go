package stripe

import (
	"net/http"
	"os"

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

	// Free plan doesn't need Stripe checkout
	if plan.GetString("billing_interval") == "free" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Cannot create checkout for free plan"})
	}

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
		SuccessURL: stripe.String(os.Getenv("STRIPE_SUCCESS_URL")),
		CancelURL:  stripe.String(os.Getenv("STRIPE_CANCEL_URL")),
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

	// Get Stripe customer ID directly for user
	customerID, err := getStripeCustomerID(app, data.UserID)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Create portal session
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(os.Getenv("HOST") + "/billing"),
	}

	ps, err := billingportal.New(params)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]string{"url": ps.URL})
}