package stripe

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/price"
	"github.com/stripe/stripe-go/v79/product"
)

// PlanConfig represents a subscription plan configuration
type PlanConfig struct {
	Name           string
	Tier           string
	PriceCents     int64
	BillingInterval string
	HoursPerMonth   float64
	Features       []string
	DisplayOrder   int
}

// SeedSubscriptionPlans creates Stripe products/prices and seeds the database
// This should only run in development or during initial setup
func SeedSubscriptionPlans(app *pocketbase.PocketBase) error {
	// Safety check - only run in development or when explicitly requested
	if os.Getenv("DEVELOPMENT") != "true" {
		log.Println("Skipping subscription seeding - not in development mode")
		return nil
	}

	log.Println("🌱 Seeding subscription plans...")

	// Define our 3 subscription plans (monthly only)
	plans := []PlanConfig{
		{
			Name:            "Free",
			Tier:            "free",
			PriceCents:      0,
			BillingInterval: "free",
			HoursPerMonth:   1,
			Features:        []string{"1 hour of media processing", "Unlimited video quality exports", "Basic support"},
			DisplayOrder:    1,
		},
		{
			Name:            "Basic Monthly",
			Tier:            "basic",
			PriceCents:      500, // $5.00
			BillingInterval: "month",
			HoursPerMonth:   10,
			Features:        []string{"10 hours of media processing", "Unlimited video quality exports", "Email support", "Priority processing"},
			DisplayOrder:    2,
		},
		{
			Name:            "Pro Monthly",
			Tier:            "pro",
			PriceCents:      1000, // $10.00
			BillingInterval: "month",
			HoursPerMonth:   25,
			Features:        []string{"25 hours of media processing", "Unlimited video quality exports", "Priority support", "Advanced AI models", "Bulk processing"},
			DisplayOrder:    3,
		},
	}

	// Process each plan
	for _, planConfig := range plans {
		if err := createOrUpdatePlan(app, planConfig); err != nil {
			log.Printf("Error processing plan %s: %v", planConfig.Name, err)
			// Continue with other plans instead of failing entirely
		}
	}

	// Seed all existing users with Free plan subscriptions
	if err := seedExistingUsersWithFreePlan(app); err != nil {
		log.Printf("Warning: Failed to seed existing users with free plan: %v", err)
	}

	// Create Pro membership for bob@test.com dated 6 months ago (development only)
	if err := createBobProMembership(app); err != nil {
		log.Printf("Warning: Failed to create Bob's Pro membership: %v", err)
	}

	log.Println("🌱 Subscription seeding completed!")
	return nil
}

// createOrUpdatePlan creates or updates a subscription plan
func createOrUpdatePlan(app *pocketbase.PocketBase, config PlanConfig) error {
	log.Printf("Processing plan: %s", config.Name)

	var stripeProductID, stripePriceID string

	// Create Stripe product and price for paid plans
	if config.BillingInterval != "free" {
		productID, priceID, err := createStripeProductAndPrice(config)
		if err != nil {
			log.Printf("Warning: Failed to create Stripe product/price for %s: %v", config.Name, err)
			log.Printf("Creating plan in database without Stripe integration...")
			// Continue to create the plan in database without Stripe integration
		} else {
			stripeProductID = productID
			stripePriceID = priceID
		}
	}

	// Create or update the plan in PocketBase
	if err := upsertSubscriptionPlan(app, config, stripeProductID, stripePriceID); err != nil {
		return fmt.Errorf("failed to upsert plan in database: %w", err)
	}

	log.Printf("✅ Plan created/updated: %s", config.Name)
	return nil
}

// createStripeProductAndPrice creates a Stripe product and price
func createStripeProductAndPrice(config PlanConfig) (string, string, error) {
	// Create product
	productParams := &stripe.ProductParams{
		Name:        stripe.String(config.Name),
		Description: stripe.String(fmt.Sprintf("%s plan with %.0f hours of media processing per month", config.Tier, config.HoursPerMonth)),
		Metadata: map[string]string{
			"tier":             config.Tier,
			"hours_per_month":  fmt.Sprintf("%.0f", config.HoursPerMonth),
			"billing_interval": config.BillingInterval,
		},
	}

	stripeProduct, err := product.New(productParams)
	if err != nil {
		return "", "", fmt.Errorf("failed to create Stripe product: %w", err)
	}

	// Create price
	var recurringParams *stripe.PriceRecurringParams
	if config.BillingInterval == "month" {
		recurringParams = &stripe.PriceRecurringParams{
			Interval: stripe.String("month"),
		}
	} else if config.BillingInterval == "year" {
		recurringParams = &stripe.PriceRecurringParams{
			Interval: stripe.String("year"),
		}
	}

	priceParams := &stripe.PriceParams{
		Product:    stripe.String(stripeProduct.ID),
		UnitAmount: stripe.Int64(config.PriceCents),
		Currency:   stripe.String("usd"),
		Recurring:  recurringParams,
		Metadata: map[string]string{
			"tier":             config.Tier,
			"hours_per_month":  fmt.Sprintf("%.0f", config.HoursPerMonth),
			"billing_interval": config.BillingInterval,
		},
	}

	stripePrice, err := price.New(priceParams)
	if err != nil {
		return "", "", fmt.Errorf("failed to create Stripe price: %w", err)
	}

	return stripeProduct.ID, stripePrice.ID, nil
}

// upsertSubscriptionPlan creates or updates a subscription plan in the database
func upsertSubscriptionPlan(app *pocketbase.PocketBase, config PlanConfig, stripeProductID, stripePriceID string) error {
	collection, err := app.FindCollectionByNameOrId("subscription_plans")
	if err != nil {
		return err
	}

	// Try to find existing plan by name
	existingRecord, err := app.FindFirstRecordByFilter("subscription_plans", "name = {:name}", map[string]any{
		"name": config.Name,
	})

	var record *core.Record
	if err != nil {
		// Create new record
		record = core.NewRecord(collection)
		log.Printf("Creating new plan record: %s", config.Name)
	} else {
		// Update existing record
		record = existingRecord
		log.Printf("Updating existing plan record: %s", config.Name)
	}

	// Set all fields - ensure price_cents is always set, even for free plans
	record.Set("name", config.Name)
	
	// Handle PocketBase's quirk where 0 is treated as "blank" for required fields
	// For free plans, use 0.01 (1 cent) instead of 0 to avoid validation error
	var priceCents int64
	if config.BillingInterval == "free" && config.PriceCents == 0 {
		priceCents = 1 // Use 1 cent instead of 0 for free plans to avoid PocketBase validation
		log.Printf("Using 1 cent instead of 0 for free plan: %s", config.Name)
	} else {
		priceCents = config.PriceCents
	}
	
	log.Printf("Setting price_cents for %s to: %d (type: %T)", config.Name, priceCents, priceCents)
	record.Set("price_cents", priceCents)
	
	record.Set("currency", "usd")
	record.Set("billing_interval", config.BillingInterval)
	record.Set("hours_per_month", config.HoursPerMonth)
	record.Set("is_active", true)
	record.Set("display_order", config.DisplayOrder)

	if stripeProductID != "" {
		record.Set("stripe_product_id", stripeProductID)
	}
	if stripePriceID != "" {
		record.Set("stripe_price_id", stripePriceID)
	}

	// Convert features to JSON
	if featuresJSON, err := json.Marshal(config.Features); err == nil {
		record.Set("features", string(featuresJSON))
	}

	if err := app.Save(record); err != nil {
		log.Printf("Full error details for %s: %+v", config.Name, err)
		return fmt.Errorf("failed to upsert plan in database: %w", err)
	}
	
	return nil
}

// seedExistingUsersWithFreePlan creates free plan subscriptions for all existing users
func seedExistingUsersWithFreePlan(app *pocketbase.PocketBase) error {
	log.Println("Seeding existing users with Free plan subscriptions...")

	// Get the Free plan
	freePlan, err := app.FindFirstRecordByFilter("subscription_plans", "billing_interval = 'free'", map[string]any{})
	if err != nil {
		return fmt.Errorf("failed to find free plan: %w", err)
	}

	// Get all users
	users, err := app.FindRecordsByFilter("users", "", "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to find users: %w", err)
	}

	userSubscriptionCollection, err := app.FindCollectionByNameOrId("user_subscriptions")
	if err != nil {
		return fmt.Errorf("failed to find user_subscriptions collection: %w", err)
	}

	now := time.Now()
	oneYearFromNow := now.AddDate(1, 0, 0) // Free plan expires in 1 year

	for _, user := range users {
		// Check if user already has a subscription
		existingSubscription, err := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id}", map[string]any{
			"user_id": user.Id,
		})

		if err != nil {
			// Create new free subscription for user
			subscription := core.NewRecord(userSubscriptionCollection)
			subscription.Set("user_id", user.Id)
			subscription.Set("plan_id", freePlan.Id)
			subscription.Set("status", "active")
			subscription.Set("current_period_start", now)
			subscription.Set("current_period_end", oneYearFromNow)
			subscription.Set("cancel_at_period_end", false)

			if err := app.Save(subscription); err != nil {
				log.Printf("Warning: Failed to create free subscription for user %s: %v", user.Id, err)
			} else {
				log.Printf("✅ Created free subscription for user: %s", user.GetString("email"))
			}
		} else {
			log.Printf("✅ User %s already has subscription: %s", user.GetString("email"), existingSubscription.Id)
		}
	}

	return nil
}

// createBobProMembership updates bob@test.com to have a Pro membership dated 6 months ago
func createBobProMembership(app *pocketbase.PocketBase) error {
	log.Println("Creating Pro membership for bob@test.com dated 6 months ago...")

	// Find bob@test.com user
	bobUser, err := app.FindFirstRecordByFilter("users", "email = {:email}", map[string]any{
		"email": "bob@test.com",
	})
	if err != nil {
		return fmt.Errorf("failed to find bob@test.com: %w", err)
	}

	// Find Pro Monthly plan
	proMonthlyPlan, err := app.FindFirstRecordByFilter("subscription_plans", "name = 'Pro Monthly'", map[string]any{})
	if err != nil {
		return fmt.Errorf("failed to find Pro Monthly plan: %w", err)
	}

	// Find bob's current subscription
	bobSubscription, err := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id}", map[string]any{
		"user_id": bobUser.Id,
	})
	if err != nil {
		return fmt.Errorf("failed to find bob's subscription: %w", err)
	}

	// Calculate dates 6 months ago
	now := time.Now()
	sixMonthsAgo := now.AddDate(0, -6, 0) // 6 months ago
	oneMonthLater := sixMonthsAgo.AddDate(0, 1, 0) // One month after start

	// Update subscription to Pro Monthly with historical dates
	bobSubscription.Set("plan_id", proMonthlyPlan.Id)
	bobSubscription.Set("current_period_start", sixMonthsAgo)
	bobSubscription.Set("current_period_end", oneMonthLater)
	bobSubscription.Set("status", "active")

	if err := app.Save(bobSubscription); err != nil {
		return fmt.Errorf("failed to update bob's subscription: %w", err)
	}

	log.Printf("✅ Updated bob@test.com to Pro Monthly plan (period: %s to %s)", 
		sixMonthsAgo.Format("2006-01-02"), oneMonthLater.Format("2006-01-02"))

	return nil
}