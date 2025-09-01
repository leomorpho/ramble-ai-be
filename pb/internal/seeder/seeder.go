package seeder

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase/core"
)

// SeedAll runs all seeding functions in development mode
func SeedAll(app core.App) error {
	// Safety check - only run in development
	if os.Getenv("DEVELOPMENT") != "true" {
		log.Println("Skipping seeding - not in development mode")
		return nil
	}

	log.Println("🌱 Starting development seeding...")

	// Seed subscription plans
	if err := SeedSubscriptionPlans(app); err != nil {
		log.Printf("Warning: Failed to seed subscription plans: %v", err)
	}

	// Seed app versions
	if err := SeedAppVersions(app); err != nil {
		log.Printf("Warning: Failed to seed app versions: %v", err)
	}

	// Seed banners
	if err := SeedBanners(app); err != nil {
		log.Printf("Warning: Failed to seed banners: %v", err)
	}

	log.Println("🎉 Seeding completed")
	return nil
}