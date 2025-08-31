package payment

import (
	"fmt"
	"log"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/price"
	"github.com/stripe/stripe-go/v79/product"
)

// StripeSetup contains utilities for setting up Stripe products and prices
type StripeSetup struct {
	secretKey string
}

// NewStripeSetup creates a new Stripe setup utility
func NewStripeSetup(secretKey string) *StripeSetup {
	stripe.Key = secretKey
	return &StripeSetup{secretKey: secretKey}
}

// ProductAndPriceResult contains the created product and price IDs
type ProductAndPriceResult struct {
	ProductID string
	PriceID   string
}

// CreateProductAndPrice creates a Stripe product and its associated price
func (s *StripeSetup) CreateProductAndPrice(name string, priceCents int64, interval string) (*ProductAndPriceResult, error) {
	// Create the product
	productParams := &stripe.ProductParams{
		Name: stripe.String(name),
		Metadata: map[string]string{
			"created_by": "pocketbase_seeder",
		},
	}

	stripeProduct, err := product.New(productParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create product %s: %w", name, err)
	}

	log.Printf("Created Stripe product: %s (ID: %s)", name, stripeProduct.ID)

	// Create the price
	priceParams := &stripe.PriceParams{
		Product:    stripe.String(stripeProduct.ID),
		UnitAmount: stripe.Int64(priceCents),
		Currency:   stripe.String("usd"),
		Metadata: map[string]string{
			"created_by": "pocketbase_seeder",
		},
	}

	// Add recurring billing if not a one-time payment
	if interval != "one_time" && interval != "" {
		priceParams.Recurring = &stripe.PriceRecurringParams{
			Interval: stripe.String(interval),
		}
	}

	stripePrice, err := price.New(priceParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create price for product %s: %w", name, err)
	}

	log.Printf("Created Stripe price: %s for %s (ID: %s)", 
		fmt.Sprintf("$%.2f/%s", float64(priceCents)/100, interval), name, stripePrice.ID)

	return &ProductAndPriceResult{
		ProductID: stripeProduct.ID,
		PriceID:   stripePrice.ID,
	}, nil
}

// SetupDefaultProductsAndPrices creates the default products and prices needed by the application
func (s *StripeSetup) SetupDefaultProductsAndPrices() (map[string]*ProductAndPriceResult, error) {
	results := make(map[string]*ProductAndPriceResult)

	plans := []struct {
		Key      string
		Name     string
		Price    int64
		Interval string
	}{
		{"basic", "Basic Plan", 700, "month"},   // $7/month
		{"pro", "Pro Plan", 1500, "month"},     // $15/month
	}

	for _, plan := range plans {
		result, err := s.CreateProductAndPrice(plan.Name, plan.Price, plan.Interval)
		if err != nil {
			return nil, fmt.Errorf("failed to create plan %s: %w", plan.Key, err)
		}
		results[plan.Key] = result
	}

	return results, nil
}