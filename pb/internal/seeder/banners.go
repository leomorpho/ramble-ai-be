package seeder

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// BannerConfig represents a banner configuration for seeding
type BannerConfig struct {
	Title        string
	Message      string
	Type         string    // info, warning, success, error
	Active       bool
	RequiresAuth bool
	ActionURL    string
	ActionText   string
	ExpiresAt    *time.Time // nil means no expiration
	CreatedOffset time.Duration // Offset from now for created timestamp
}

// SeedBanners creates sample banners for development testing
// This function should ONLY be called in development environments
func SeedBanners(app core.App) error {
	log.Println("🌱 Seeding banners...")

	// Check if banners already exist
	existingBanners, err := app.FindRecordsByFilter("banners", "", "", 1, 0)
	if err == nil && len(existingBanners) > 0 {
		log.Println("Banners already exist, skipping seeding")
		return nil
	}

	// Get future and past times for testing
	now := time.Now()
	futureExpiry := now.Add(30 * 24 * time.Hour) // 30 days from now
	pastExpiry := now.Add(-7 * 24 * time.Hour)   // 7 days ago (expired)

	// Define diverse test banners
	banners := []BannerConfig{
		// App Update Banner - Most important, should be visible
		{
			Title:         "🚀 New Version Available!",
			Message:       "Ramble AI v1.2.0 is now available with enhanced AI processing and better performance. Download now to get the latest features and improvements.",
			Type:          "info",
			Active:        true,
			RequiresAuth:  false,
			ActionURL:     "https://ramble.goosebyteshq.com/download",
			ActionText:    "Download Update",
			ExpiresAt:     &futureExpiry,
			CreatedOffset: -1 * time.Hour, // 1 hour ago
		},

		// General Info Banner
		{
			Title:         "💡 Pro Tip: Batch Processing",
			Message:       "Did you know you can process multiple audio files at once? Select multiple files in the upload dialog to save time and improve your workflow.",
			Type:          "info", 
			Active:        true,
			RequiresAuth:  false,
			ActionURL:     "", // No action button
			ActionText:    "",
			ExpiresAt:     nil, // No expiration
			CreatedOffset: -2 * time.Hour, // 2 hours ago
		},

		// Maintenance Warning
		{
			Title:         "⚠️ Scheduled Maintenance",
			Message:       "We'll be performing system maintenance on Sunday, March 15th from 2:00 AM to 4:00 AM EST. Some features may be temporarily unavailable.",
			Type:          "warning",
			Active:        true,
			RequiresAuth:  false,
			ActionURL:     "https://status.ramble.goosebyteshq.com",
			ActionText:    "View Status Page",
			ExpiresAt:     &futureExpiry,
			CreatedOffset: -3 * time.Hour, // 3 hours ago
		},

		// Success/Feature Banner
		{
			Title:         "✨ New Feature: Smart Chapters",
			Message:       "Introducing automatic chapter detection! Our AI now intelligently identifies natural breakpoints in your content for better organization.",
			Type:          "success",
			Active:        true,
			RequiresAuth:  false,
			ActionURL:     "https://docs.ramble.goosebyteshq.com/features/smart-chapters",
			ActionText:    "Learn More",
			ExpiresAt:     nil,
			CreatedOffset: -6 * time.Hour, // 6 hours ago
		},

		// Authenticated-only Banner (Premium Feature)
		{
			Title:         "🎯 Premium: Advanced AI Models",
			Message:       "As a premium user, you now have access to GPT-4 and Claude for even more accurate transcriptions and better content optimization.",
			Type:          "success",
			Active:        true,
			RequiresAuth:  true, // Only show to authenticated users
			ActionURL:     "https://ramble.goosebyteshq.com/premium/ai-models",
			ActionText:    "Explore Models",
			ExpiresAt:     nil,
			CreatedOffset: -12 * time.Hour, // 12 hours ago
		},

		// Security/Error Banner
		{
			Title:         "🔒 Security Update Required",
			Message:       "An important security update is available. Please update to the latest version to ensure your data remains protected.",
			Type:          "error",
			Active:        true,
			RequiresAuth:  false,
			ActionURL:     "https://ramble.goosebyteshq.com/security/update-guide",
			ActionText:    "Update Now",
			ExpiresAt:     &futureExpiry,
			CreatedOffset: -24 * time.Hour, // 1 day ago
		},

		// Expired Banner (for testing expiration functionality)
		{
			Title:         "📅 Past Event: AI Workshop",
			Message:       "Thanks to everyone who joined our AI workshop last week! The recording is available in your account dashboard.",
			Type:          "info",
			Active:        true,
			RequiresAuth:  true,
			ActionURL:     "https://ramble.goosebyteshq.com/workshop-recording",
			ActionText:    "Watch Recording",
			ExpiresAt:     &pastExpiry, // Already expired - should not show
			CreatedOffset: -10 * 24 * time.Hour, // 10 days ago
		},

		// Another app update banner with different priority
		{
			Title:         "📱 Mobile App Coming Soon",
			Message:       "We're working on a mobile companion app for Ramble AI. Sign up to be notified when it's available!",
			Type:          "info",
			Active:        true,
			RequiresAuth:  false,
			ActionURL:     "https://ramble.goosebyteshq.com/mobile-signup",
			ActionText:    "Join Waitlist",
			ExpiresAt:     nil,
			CreatedOffset: -4 * 24 * time.Hour, // 4 days ago
		},

		// Feature announcement for authenticated users
		{
			Title:         "🔧 New: API Access",
			Message:       "Developers can now integrate Ramble AI into their applications using our REST API. Check out the documentation to get started.",
			Type:          "info",
			Active:        true,
			RequiresAuth:  true, // Only for registered users
			ActionURL:     "https://docs.ramble.goosebyteshq.com/api",
			ActionText:    "View API Docs",
			ExpiresAt:     nil,
			CreatedOffset: -7 * 24 * time.Hour, // 1 week ago
		},
	}

	// Get the banners collection
	collection, err := app.FindCollectionByNameOrId("banners")
	if err != nil {
		return fmt.Errorf("failed to find banners collection: %w", err)
	}

	// Create each banner record
	successCount := 0
	for _, banner := range banners {
		record := core.NewRecord(collection)
		
		// Set the data
		record.Set("title", banner.Title)
		record.Set("message", banner.Message)
		record.Set("type", banner.Type)
		record.Set("active", banner.Active)
		record.Set("requires_auth", banner.RequiresAuth)
		record.Set("action_url", banner.ActionURL)
		record.Set("action_text", banner.ActionText)
		
		// Handle optional expiration
		if banner.ExpiresAt != nil {
			record.Set("expires_at", banner.ExpiresAt.Format(time.RFC3339))
		}
		
		// Set created timestamp with offset
		createdTime := now.Add(banner.CreatedOffset)
		record.Set("created", createdTime)
		record.Set("updated", createdTime)

		// Save the record
		if err := app.Save(record); err != nil {
			log.Printf("Failed to create banner '%s': %v", banner.Title, err)
			continue
		}

		// Log creation with key details
		authStatus := "public"
		if banner.RequiresAuth {
			authStatus = "auth-required"
		}
		
		expiryStatus := "no expiry"
		if banner.ExpiresAt != nil {
			if banner.ExpiresAt.Before(now) {
				expiryStatus = "expired"
			} else {
				expiryStatus = "expires " + banner.ExpiresAt.Format("Jan 2")
			}
		}

		log.Printf("✓ Created banner: %s [%s] (%s, %s)", 
			banner.Title, banner.Type, authStatus, expiryStatus)
		successCount++
	}

	log.Printf("🎉 Successfully seeded %d/%d banners", successCount, len(banners))
	log.Println("📢 Banner types created:")
	log.Println("   - App update banners (download links)")
	log.Println("   - Info & tip banners")
	log.Println("   - Maintenance warnings")
	log.Println("   - Feature announcements")
	log.Println("   - Security alerts")
	log.Println("   - Premium user content (auth required)")
	log.Println("   - Expired banners (for testing)")
	log.Println("")
	log.Println("🔍 Test scenarios available:")
	log.Println("   - Public banners (visible to all)")
	log.Println("   - Auth-required banners (API key needed)")
	log.Println("   - Action buttons with external links")
	log.Println("   - Expiration handling (some expired)")
	log.Println("   - Different priority levels (by creation time)")
	
	return nil
}