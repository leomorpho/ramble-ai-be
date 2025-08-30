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
	log.Println("🔍 [SEEDER DEBUG] Starting SeedSubscriptionPlans...")
	
	// Safety check - only run in development or when explicitly requested
	devMode := os.Getenv("DEVELOPMENT")
	log.Printf("🔍 [SEEDER DEBUG] DEVELOPMENT env var: '%s'", devMode)
	if devMode != "true" {
		log.Println("Skipping subscription seeding - not in development mode")
		return nil
	}

	log.Println("🌱 [SEEDER DEBUG] Development mode confirmed, proceeding with seeding...")
	
	// Verify database state before seeding
	if err := verifyDatabaseState(app); err != nil {
		log.Printf("❌ [SEEDER DEBUG] Database state verification failed: %v", err)
		return err
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

	// Clean up duplicate active subscriptions (one-time fix)
	if err := cleanupDuplicateSubscriptions(app); err != nil {
		log.Printf("Warning: Failed to cleanup duplicate subscriptions: %v", err)
	}

	// Seed all existing users with Pro plan subscriptions (development only)
	if err := seedExistingUsersWithProPlan(app); err != nil {
		log.Printf("Warning: Failed to seed existing users with Pro plan: %v", err)
	}

	log.Println("🌱 Subscription seeding completed!")
	return nil
}

// createOrUpdatePlan creates or updates a subscription plan
func createOrUpdatePlan(app *pocketbase.PocketBase, config PlanConfig) error {
	log.Printf("Processing plan: %s", config.Name)

	var stripeProductID, stripePriceID string

	// Only create Stripe product/price for paid plans (not free plans)
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
	} else {
		log.Printf("Skipping Stripe integration for free plan: %s", config.Name)
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
	
	// Set price_cents directly - free plans have price 0
	record.Set("price_cents", config.PriceCents)
	
	record.Set("currency", "usd")
	record.Set("billing_interval", config.BillingInterval)
	record.Set("hours_per_month", config.HoursPerMonth)
	record.Set("is_active", true)
	record.Set("display_order", config.DisplayOrder)

	if stripeProductID != "" {
		record.Set("provider_product_id", stripeProductID)
	}
	if stripePriceID != "" {
		record.Set("provider_price_id", stripePriceID)
	}
	
	// Set payment provider - default to Stripe for all plans
	record.Set("payment_provider", "stripe")

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

// seedExistingUsersWithProPlan creates Pro plan subscriptions for all existing users
func seedExistingUsersWithProPlan(app *pocketbase.PocketBase) error {
	log.Println("🔍 [USER SEEDER DEBUG] Starting seedExistingUsersWithProPlan...")

	// Get the Pro Monthly plan
	log.Println("🔍 [USER SEEDER DEBUG] Looking for Pro Monthly plan...")
	proPlan, err := app.FindFirstRecordByFilter("subscription_plans", "name = 'Pro Monthly'", map[string]any{})
	if err != nil {
		log.Printf("❌ [USER SEEDER DEBUG] Failed to find Pro Monthly plan: %v", err)
		return fmt.Errorf("failed to find Pro Monthly plan: %w", err)
	}
	log.Printf("✅ [USER SEEDER DEBUG] Found Pro Monthly plan: ID=%s", proPlan.Id)

	// Get all users
	log.Println("🔍 [USER SEEDER DEBUG] Looking for existing users...")
	users, err := app.FindRecordsByFilter("users", "", "", 0, 0)
	if err != nil {
		log.Printf("❌ [USER SEEDER DEBUG] Failed to find users: %v", err)
		return fmt.Errorf("failed to find users: %w", err)
	}
	log.Printf("✅ [USER SEEDER DEBUG] Found %d users", len(users))

	// Get user_subscriptions collection
	log.Println("🔍 [USER SEEDER DEBUG] Looking for user_subscriptions collection...")
	userSubscriptionCollection, err := app.FindCollectionByNameOrId("user_subscriptions")
	if err != nil {
		log.Printf("❌ [USER SEEDER DEBUG] Failed to find user_subscriptions collection: %v", err)
		return fmt.Errorf("failed to find user_subscriptions collection: %w", err)
	}
	log.Printf("✅ [USER SEEDER DEBUG] Found user_subscriptions collection: %s", userSubscriptionCollection.Name)

	now := time.Now()
	oneMonthFromNow := now.AddDate(0, 1, 0) // Pro plan expires in 1 month

	for _, user := range users {
		// Check if user already has a subscription
		existingSubscription, err := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id}", map[string]any{
			"user_id": user.Id,
		})

		var subscription *core.Record
		if err != nil {
			// Create new Pro subscription for user
			subscription = core.NewRecord(userSubscriptionCollection)
			subscription.Set("user_id", user.Id)
			log.Printf("Creating new Pro subscription for user: %s", user.GetString("email"))
		} else {
			// Update existing subscription to Pro
			subscription = existingSubscription
			log.Printf("Updating existing subscription to Pro for user: %s", user.GetString("email"))
		}

		// Set subscription fields
		subscription.Set("plan_id", proPlan.Id)
		subscription.Set("status", "active")
		subscription.Set("current_period_start", now)
		subscription.Set("current_period_end", oneMonthFromNow)
		subscription.Set("cancel_at_period_end", false)
		
		// Set provider fields for Pro subscriptions
		// For test data, we don't have real Stripe subscription IDs but we set the price ID
		providerPriceID := proPlan.GetString("provider_price_id")
		if providerPriceID != "" {
			subscription.Set("provider_price_id", providerPriceID)
		}
		subscription.Set("payment_provider", "stripe")

		if err := app.Save(subscription); err != nil {
			log.Printf("❌ [USER SEEDER DEBUG] Failed to create/update Pro subscription for user %s (%s): %v", user.Id, user.GetString("email"), err)
		} else {
			log.Printf("✅ [USER SEEDER DEBUG] Successfully created/updated Pro subscription for user: %s (%s)", user.Id, user.GetString("email"))
		}
	}

	log.Printf("🎉 [USER SEEDER DEBUG] Completed processing %d users", len(users))
	return nil
}

// cleanupDuplicateSubscriptions ensures only one active subscription per user
func cleanupDuplicateSubscriptions(app *pocketbase.PocketBase) error {
	log.Println("🧹 Cleaning up duplicate active subscriptions...")
	
	// Get all users
	users, err := app.FindRecordsByFilter("users", "", "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to find users: %w", err)
	}
	
	fixedCount := 0
	for _, user := range users {
		// Get all active subscriptions for this user
		subscriptions, err := app.FindRecordsByFilter("user_subscriptions", 
			"user_id = {:user_id} && status = 'active'", 
			"-current_period_end", // Sort by newest period end first
			0, 0,
			map[string]any{"user_id": user.Id})
		
		if err != nil || len(subscriptions) <= 1 {
			continue // No duplicates for this user
		}
		
		log.Printf("Found %d active subscriptions for user %s, keeping only the most recent", 
			len(subscriptions), user.GetString("email"))
		
		// Keep the first one (most recent period_end), cancel the rest
		for i := 1; i < len(subscriptions); i++ {
			sub := subscriptions[i]
			sub.Set("status", "cancelled")
			sub.Set("canceled_at", time.Now())
			
			// Fix 1970 dates if present
			startTime := sub.GetDateTime("current_period_start").Time()
			if startTime.Year() < 2000 {
				sub.Set("current_period_start", time.Now().AddDate(0, -1, 0)) // 1 month ago
			}
			endTime := sub.GetDateTime("current_period_end").Time()
			if endTime.Year() < 2000 {
				sub.Set("current_period_end", time.Now()) // Now (since it's cancelled)
			}
			
			if err := app.Save(sub); err != nil {
				log.Printf("Failed to deactivate duplicate subscription: %v", err)
			} else {
				fixedCount++
			}
		}
		
		// Also fix the dates on the active subscription if needed
		activeSub := subscriptions[0]
		needsUpdate := false
		
		activeStartTime := activeSub.GetDateTime("current_period_start").Time()
		if activeStartTime.Year() < 2000 {
			activeSub.Set("current_period_start", time.Now())
			needsUpdate = true
		}
		activeEndTime := activeSub.GetDateTime("current_period_end").Time()
		if activeEndTime.Year() < 2000 {
			activeSub.Set("current_period_end", time.Now().AddDate(0, 1, 0)) // 1 month from now
			needsUpdate = true
		}
		
		if needsUpdate {
			if err := app.Save(activeSub); err != nil {
				log.Printf("Failed to fix dates on active subscription: %v", err)
			}
		}
	}
	
	log.Printf("✅ Cleaned up %d duplicate subscriptions", fixedCount)
	return nil
}

// verifyDatabaseState checks that all required collections exist and logs database status
func verifyDatabaseState(app *pocketbase.PocketBase) error {
	log.Println("🔍 [DATABASE DEBUG] Verifying database state...")
	
	// Check required collections
	requiredCollections := []string{"subscription_plans", "user_subscriptions", "users"}
	for _, collectionName := range requiredCollections {
		collection, err := app.FindCollectionByNameOrId(collectionName)
		if err != nil {
			log.Printf("❌ [DATABASE DEBUG] Missing required collection: %s", collectionName)
			return fmt.Errorf("missing required collection: %s", collectionName)
		}
		
		// Count records in collection
		var count int64
		if collectionName == "users" {
			// Users collection uses different counting method
			users, err := app.FindRecordsByFilter(collectionName, "", "", 0, 0)
			if err != nil {
				count = 0
			} else {
				count = int64(len(users))
			}
		} else {
			records, err := app.FindRecordsByFilter(collectionName, "", "", 0, 0)
			if err != nil {
				count = 0
			} else {
				count = int64(len(records))
			}
		}
		
		log.Printf("✅ [DATABASE DEBUG] Collection '%s' exists with %d records", collection.Name, count)
	}
	
	log.Println("🎉 [DATABASE DEBUG] Database state verification completed")
	return nil
}