package ai

import (
	"fmt"
	"log"
	"os"

	"github.com/pocketbase/pocketbase/core"
)

// Development seed constants
const (
	// Development API key - only used when DEVELOPMENT=true
	DEV_API_KEY = "ra-dev-12345678901234567890123456789012"
	DEV_USER_EMAIL = "bob@test.com"
	DEV_USER_NAME = "Bob"
	
	// Admin user for testing
	ADMIN_USER_EMAIL = "alice@test.com"
	ADMIN_USER_NAME = "Alice Admin"
	ADMIN_PASSWORD = "password"
)

// SeedDevelopmentData creates a development user and API key for local testing
// This function should ONLY be called in development environments
func SeedDevelopmentData(app core.App) error {
	// Safety check - only run in development
	if os.Getenv("DEVELOPMENT") != "true" {
		log.Println("Skipping development seeding - not in development mode")
		return nil
	}

	log.Println("🌱 Seeding development data...")

	// Check if development user already exists
	existingUser, err := app.FindFirstRecordByFilter("users", "email = {:email}", map[string]interface{}{
		"email": DEV_USER_EMAIL,
	})
	
	var devUser *core.Record
	
	if err != nil {
		// Create development user
		log.Printf("Creating development user: %s", DEV_USER_EMAIL)
		
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return fmt.Errorf("failed to find users collection: %w", err)
		}

		devUser = core.NewRecord(usersCollection)
		devUser.Set("email", DEV_USER_EMAIL)
		devUser.Set("name", DEV_USER_NAME)
		devUser.Set("password", "password")
		devUser.Set("passwordConfirm", "password")
		devUser.Set("verified", true)
		
		if err := app.Save(devUser); err != nil {
			return fmt.Errorf("failed to create development user: %w", err)
		}
		
		log.Printf("✅ Created development user: %s (ID: %s)", DEV_USER_EMAIL, devUser.Id)
	} else {
		devUser = existingUser
		log.Printf("✅ Development user already exists: %s (ID: %s)", DEV_USER_EMAIL, devUser.Id)
	}

	// Check if development API key already exists
	keyHash := hashAPIKey(DEV_API_KEY)
	existingAPIKey, err := app.FindFirstRecordByFilter("api_keys", "key_hash = {:hash}", map[string]interface{}{
		"hash": keyHash,
	})

	if err != nil {
		// Create development API key
		log.Printf("Creating development API key: %s", DEV_API_KEY)
		
		apiKeysCollection, err := app.FindCollectionByNameOrId("api_keys")
		if err != nil {
			// API keys collection might not exist yet, create it
			log.Println("Creating api_keys collection...")
			if err := createAPIKeysCollection(app); err != nil {
				return fmt.Errorf("failed to create api_keys collection: %w", err)
			}
			
			// Retry finding the collection
			apiKeysCollection, err = app.FindCollectionByNameOrId("api_keys")
			if err != nil {
				return fmt.Errorf("failed to find api_keys collection after creation: %w", err)
			}
		}

		apiKeyRecord := core.NewRecord(apiKeysCollection)
		apiKeyRecord.Set("key_hash", keyHash)
		apiKeyRecord.Set("user_id", devUser.Id)
		apiKeyRecord.Set("active", true)
		apiKeyRecord.Set("name", "Development API Key")
		
		if err := app.Save(apiKeyRecord); err != nil {
			return fmt.Errorf("failed to create development API key: %w", err)
		}
		
		log.Printf("✅ Created development API key: %s (hash: %s)", DEV_API_KEY, keyHash[:16]+"...")
	} else {
		log.Printf("✅ Development API key already exists (hash: %s)", keyHash[:16]+"...")
		
		// Ensure it's associated with the dev user and active
		existingAPIKey.Set("user_id", devUser.Id)
		existingAPIKey.Set("active", true)
		if err := app.Save(existingAPIKey); err != nil {
			log.Printf("Warning: failed to update existing API key: %v", err)
		}
	}

	// Create admin user for testing
	if err := seedAdminUser(app); err != nil {
		log.Printf("Warning: failed to seed admin user: %v", err)
		// Don't fail the entire seeding process for admin user issues
	}

	log.Printf("🌱 Development seeding complete!")
	log.Printf("   API Key: %s", DEV_API_KEY)
	log.Printf("   User: %s", DEV_USER_EMAIL)
	log.Printf("   Admin: %s (password: %s)", ADMIN_USER_EMAIL, ADMIN_PASSWORD)
	log.Printf("   Use this API key in your Wails app for development")

	return nil
}

// createAPIKeysCollection creates the api_keys collection if it doesn't exist
func createAPIKeysCollection(app core.App) error {
	// This is a simplified version - in a real implementation, you'd use PocketBase migrations
	// For development seeding, we'll assume the collection exists or will be created manually
	log.Println("API Keys collection creation would be handled by migrations in production")
	log.Println("Please ensure the 'api_keys' collection exists with fields: key_hash, user_id, active, name")
	return fmt.Errorf("api_keys collection must be created manually or via migrations")
}

// seedAdminUser creates an admin user for testing
func seedAdminUser(app core.App) error {
	// Check if admin user already exists
	existingAdmin, err := app.FindFirstRecordByFilter("users", "email = {:email}", map[string]interface{}{
		"email": ADMIN_USER_EMAIL,
	})
	
	if err != nil {
		// Create admin user
		log.Printf("Creating admin user: %s", ADMIN_USER_EMAIL)
		
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return fmt.Errorf("failed to find users collection: %w", err)
		}

		adminUser := core.NewRecord(usersCollection)
		adminUser.Set("email", ADMIN_USER_EMAIL)
		adminUser.Set("name", ADMIN_USER_NAME)
		adminUser.Set("password", ADMIN_PASSWORD)
		adminUser.Set("passwordConfirm", ADMIN_PASSWORD)
		adminUser.Set("verified", true)
		
		// Set admin role if the field exists
		if usersCollection.Fields.GetByName("role") != nil {
			adminUser.Set("role", "admin")
		}
		
		if err := app.Save(adminUser); err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}
		
		log.Printf("✅ Created admin user: %s (ID: %s)", ADMIN_USER_EMAIL, adminUser.Id)
	} else {
		log.Printf("✅ Admin user already exists: %s (ID: %s)", ADMIN_USER_EMAIL, existingAdmin.Id)
		
		// Ensure admin role is set if the field exists
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err == nil && usersCollection.Fields.GetByName("role") != nil {
			existingAdmin.Set("role", "admin")
			if err := app.Save(existingAdmin); err != nil {
				log.Printf("Warning: failed to update admin user role: %v", err)
			}
		}
	}

	return nil
}

