package seeder

import (
	"fmt"
	"log"
	"os"

	"pocketbase/internal/payment"

	"github.com/pocketbase/pocketbase/core"
)

// PlanConfig represents a subscription plan configuration for seeding
type PlanConfig struct {
	Name              string
	PriceCents        int
	BillingInterval   string
	HoursPerMonth     float64
	ProviderPriceID   string
	ProviderProductID string
	PaymentProvider   string
	Features          []string
	DisplayOrder      int
	IsActive          bool
}

// SeedSubscriptionPlans creates default subscription plans if they don't exist
func SeedSubscriptionPlans(app core.App) error {
	log.Println("🌱 Seeding subscription plans...")

	// Check if plans already exist
	existingPlans, err := app.FindRecordsByFilter("subscription_plans", "", "", 1, 0)
	if err == nil && len(existingPlans) > 0 {
		log.Printf("📋 Subscription plans already exist (%d plans found), skipping seeding", len(existingPlans))
		return nil
	}

	// Create Stripe products and prices if we have a Stripe key
	var stripeResults map[string]*payment.ProductAndPriceResult
	stripeKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeKey != "" {
		log.Println("🔄 Creating Stripe products and prices...")
		setup := payment.NewStripeSetup(stripeKey)
		stripeResults, err = setup.SetupDefaultProductsAndPrices()
		if err != nil {
			log.Printf("⚠️  Warning: Failed to create Stripe products: %v", err)
			log.Println("📋 Continuing with placeholder price IDs...")
		}
	} else {
		log.Println("⚠️  No STRIPE_SECRET_KEY found - using placeholder price IDs")
	}

	// Define default subscription plans with dynamic Stripe IDs
	var basicPriceID, basicProductID = "price_basic_monthly", "prod_basic"
	var proPriceID, proProductID = "price_pro_monthly", "prod_pro"

	// Use real Stripe IDs if we created them
	if stripeResults != nil {
		if basic, ok := stripeResults["basic"]; ok {
			basicPriceID = basic.PriceID
			basicProductID = basic.ProductID
		}
		if pro, ok := stripeResults["pro"]; ok {
			proPriceID = pro.PriceID
			proProductID = pro.ProductID
		}
	}

	plans := []PlanConfig{
		{
			Name:              "Free",
			PriceCents:        0,
			BillingInterval:   "free",
			HoursPerMonth:     0.5, // 30 minutes
			ProviderPriceID:   "", // No Stripe price for free plan
			ProviderProductID: "",
			PaymentProvider:   "stripe",
			Features:          []string{"30 minutes per month", "Basic support"},
			DisplayOrder:      1,
			IsActive:          true,
		},
		{
			Name:              "Basic",
			PriceCents:        700, // $7
			BillingInterval:   "month",
			HoursPerMonth:     10.0,
			ProviderPriceID:   basicPriceID,
			ProviderProductID: basicProductID,
			PaymentProvider:   "stripe",
			Features:          []string{"10 hours per month", "Email support", "Priority processing"},
			DisplayOrder:      2,
			IsActive:          true,
		},
		{
			Name:              "Pro",
			PriceCents:        1500, // $15
			BillingInterval:   "month",
			HoursPerMonth:     25.0,
			ProviderPriceID:   proPriceID,
			ProviderProductID: proProductID,
			PaymentProvider:   "stripe",
			Features:          []string{"25 hours per month", "Priority support", "Fastest processing", "All features"},
			DisplayOrder:      3,
			IsActive:          true,
		},
	}

	// Get the subscription_plans collection
	collection, err := app.FindCollectionByNameOrId("subscription_plans")
	if err != nil {
		return fmt.Errorf("failed to find subscription_plans collection: %w", err)
	}

	log.Printf("✓ Found subscription_plans collection, creating %d plans", len(plans))

	// Create each plan
	for _, planConfig := range plans {
		record := core.NewRecord(collection)
		
		// Set plan fields
		record.Set("name", planConfig.Name)
		record.Set("price_cents", planConfig.PriceCents)
		record.Set("currency", "usd") // Default currency for all plans
		record.Set("billing_interval", planConfig.BillingInterval)
		record.Set("hours_per_month", planConfig.HoursPerMonth)
		record.Set("provider_price_id", planConfig.ProviderPriceID)
		record.Set("provider_product_id", planConfig.ProviderProductID)
		record.Set("payment_provider", planConfig.PaymentProvider)
		record.Set("features", planConfig.Features)
		record.Set("display_order", planConfig.DisplayOrder)
		record.Set("is_active", planConfig.IsActive)

		// Save the plan
		if err := app.Save(record); err != nil {
			log.Printf("❌ Failed to create plan %s: %v", planConfig.Name, err)
			return fmt.Errorf("failed to create plan %s: %w", planConfig.Name, err)
		}

		log.Printf("✅ Created subscription plan: %s ($%.2f, %.0f hours)", 
			planConfig.Name, float64(planConfig.PriceCents)/100, planConfig.HoursPerMonth)
	}

	log.Printf("🎉 Successfully seeded %d subscription plans", len(plans))
	return nil
}