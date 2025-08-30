package stripe

import (
	"fmt"
	"time"
	
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/customer"
)

// getOrCreateStripeCustomer retrieves an existing Stripe customer ID or creates a new one for a user
func getOrCreateStripeCustomer(app *pocketbase.PocketBase, userID string) (string, error) {
	// First try to get existing customer
	customerID, err := getStripeCustomerID(app, userID)
	if err == nil {
		return customerID, nil
	}

	// Get user info
	user, err := app.FindRecordById("users", userID)
	if err != nil {
		return "", err
	}

	// Create new Stripe customer
	params := &stripe.CustomerParams{
		Email: stripe.String(user.GetString("email")),
		Name:  stripe.String(user.GetString("name")),
		Metadata: map[string]string{
			"user_id": userID,
		},
	}

	stripeCustomer, err := customer.New(params)
	if err != nil {
		return "", err
	}

	// Save customer record in PocketBase
	collection, err := app.FindCollectionByNameOrId("payment_customers")
	if err != nil {
		return "", err
	}

	record := core.NewRecord(collection)
	record.Set("provider_customer_id", stripeCustomer.ID)
	record.Set("user_id", userID)

	if err := app.Save(record); err != nil {
		return "", err
	}

	return stripeCustomer.ID, nil
}

// getStripeCustomerID retrieves the Stripe customer ID for a given user ID
func getStripeCustomerID(app *pocketbase.PocketBase, userID string) (string, error) {
	record, err := app.FindFirstRecordByFilter("payment_customers", "user_id = {:user_id}", map[string]any{
		"user_id": userID,
	})
	if err != nil {
		return "", err
	}

	return record.GetString("provider_customer_id"), nil
}

// getUserIDFromCustomer retrieves the user ID associated with a Stripe customer ID
func getUserIDFromCustomer(app *pocketbase.PocketBase, customerID string) (string, error) {
	record, err := app.FindFirstRecordByFilter("payment_customers", "provider_customer_id = {:customer_id}", map[string]any{
		"customer_id": customerID,
	})
	if err != nil {
		return "", err
	}

	return record.GetString("user_id"), nil
}

// validateAndFixTimestamps validates Unix timestamps and returns Go times
// Returns false if timestamps are invalid (0 or end before start)
func validateAndFixTimestamps(startUnix, endUnix int64) (time.Time, time.Time, bool) {
	// Check for 1970 timestamps (invalid)
	if startUnix <= 0 || endUnix <= 0 {
		return time.Time{}, time.Time{}, false
	}
	
	startTime := time.Unix(startUnix, 0)
	endTime := time.Unix(endUnix, 0)
	
	// Check if end is after start
	if !endTime.After(startTime) {
		return time.Time{}, time.Time{}, false
	}
	
	return startTime, endTime, true
}

// extractPriceFromSubscription safely extracts the price ID from a Stripe subscription
func extractPriceFromSubscription(sub *stripe.Subscription) (string, error) {
	if sub.Items == nil || len(sub.Items.Data) == 0 {
		return "", fmt.Errorf("subscription has no items")
	}
	
	if sub.Items.Data[0].Price == nil {
		return "", fmt.Errorf("subscription item has no price")
	}
	
	return sub.Items.Data[0].Price.ID, nil
}