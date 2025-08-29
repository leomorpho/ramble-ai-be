package stripe

import (
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
	collection, err := app.FindCollectionByNameOrId("stripe_customers")
	if err != nil {
		return "", err
	}

	record := core.NewRecord(collection)
	record.Set("stripe_customer_id", stripeCustomer.ID)
	record.Set("user_id", userID)

	if err := app.Save(record); err != nil {
		return "", err
	}

	return stripeCustomer.ID, nil
}

// getStripeCustomerID retrieves the Stripe customer ID for a given user ID
func getStripeCustomerID(app *pocketbase.PocketBase, userID string) (string, error) {
	record, err := app.FindFirstRecordByFilter("stripe_customers", "user_id = {:user_id}", map[string]any{
		"user_id": userID,
	})
	if err != nil {
		return "", err
	}

	return record.GetString("stripe_customer_id"), nil
}

// getUserIDFromCustomer retrieves the user ID associated with a Stripe customer ID
func getUserIDFromCustomer(app *pocketbase.PocketBase, customerID string) (string, error) {
	record, err := app.FindFirstRecordByFilter("stripe_customers", "stripe_customer_id = {:customer_id}", map[string]any{
		"customer_id": customerID,
	})
	if err != nil {
		return "", err
	}

	return record.GetString("user_id"), nil
}