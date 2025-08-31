package payment

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pocketbase/pocketbase/core"
)

// CreateCheckoutSessionHandler handles requests to create a Stripe checkout session
func CreateCheckoutSessionHandler(e *core.RequestEvent, app core.App, paymentService *Service) error {
	if paymentService == nil {
		return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Payment service not available"})
	}

	// Parse request body
	var req struct {
		PlanID string `json:"plan_id"`
		UserID string `json:"user_id"`
	}
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Get the plan details
	plan, err := app.FindRecordById("subscription_plans", req.PlanID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "Plan not found"})
	}

	// Check if this is a free plan
	if plan.GetInt("price_cents") == 0 {
		// For free plans, don't create a checkout session, just return success
		// The frontend should handle this by calling the change-plan endpoint
		return e.JSON(http.StatusOK, map[string]interface{}{
			"message": "Free plan selected - use change-plan endpoint",
			"is_free": true,
		})
	}

	// Get or create customer
	user, err := app.FindRecordById("users", req.UserID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	// Check if customer exists
	customers, err := app.FindRecordsByFilter("payment_customers", fmt.Sprintf("user_id = '%s'", req.UserID), "", 1, 0)
	var customerID string
	if err != nil || len(customers) == 0 {
		// Create new customer
		customer, err := paymentService.CreateCustomer(CustomerParams{
			Email:  user.GetString("email"),
			Name:   user.GetString("name"),
			UserID: req.UserID,
		})
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to create customer: %v", err)})
		}
		
		// Save customer record
		collection, err := app.FindCollectionByNameOrId("payment_customers")
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to find payment_customers collection: %v", err)})
		}
		record := core.NewRecord(collection)
		record.Set("user_id", req.UserID)
		record.Set("provider_customer_id", customer.ID)
		if err := app.Save(record); err != nil {
			log.Printf("Failed to save customer record: %v", err)
		}
		customerID = customer.ID
	} else {
		customerID = customers[0].GetString("provider_customer_id")
	}

	// Create checkout session
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	checkoutParams := CheckoutSessionParams{
		CustomerID:      customerID,
		PriceID:         plan.GetString("provider_price_id"),
		Quantity:        1,
		Mode:            "subscription",
		SuccessURL:      fmt.Sprintf("%s/pricing?success=true", frontendURL),
		CancelURL:       fmt.Sprintf("%s/pricing?canceled=true", frontendURL),
		AllowPromoCodes: true,
		UserID:          req.UserID,
		PlanID:          req.PlanID,
	}

	session, err := paymentService.CreateCheckoutSession(checkoutParams)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to create checkout session: %v", err)})
	}

	return e.JSON(http.StatusOK, map[string]string{"url": session.URL})
}

// CreatePortalLinkHandler handles requests to create a billing portal link
func CreatePortalLinkHandler(e *core.RequestEvent, app core.App, paymentService *Service) error {
	if paymentService == nil {
		return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Payment service not available"})
	}

	// Parse request body
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Get customer
	customers, err := app.FindRecordsByFilter("payment_customers", fmt.Sprintf("user_id = '%s'", req.UserID), "", 1, 0)
	if err != nil || len(customers) == 0 {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "Customer not found"})
	}

	customerID := customers[0].GetString("provider_customer_id")
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	portalLink, err := paymentService.CreateBillingPortalLink(customerID, fmt.Sprintf("%s/pricing", frontendURL))
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to create portal link: %v", err)})
	}

	return e.JSON(http.StatusOK, map[string]string{"url": portalLink.URL})
}