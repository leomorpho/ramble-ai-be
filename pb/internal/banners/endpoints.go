package banners

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// GetPublicBannersHandler handles requests for public banners
func GetPublicBannersHandler(e *core.RequestEvent, app core.App) error {
	// Find active banners that don't require authentication and haven't expired
	records, err := app.FindRecordsByFilter(
		"banners",
		"active = true && requires_auth = false && (expires_at = '' || expires_at > {:now})",
		"created",
		-1,
		0,
		map[string]interface{}{
			"now": time.Now().Format(time.RFC3339),
		},
	)
	if err != nil {
		return e.JSON(500, map[string]string{"error": "Failed to fetch banners"})
	}

	return e.JSON(200, map[string]interface{}{
		"banners": records,
	})
}

// GetAuthenticatedBannersHandler handles requests for authenticated user banners
func GetAuthenticatedBannersHandler(e *core.RequestEvent, app core.App) error {
	// Validate API key (reuse existing AI validation logic)
	apiKey := extractBearerToken(e.Request.Header.Get("Authorization"))
	if apiKey == "" {
		return e.JSON(401, map[string]string{"error": "Missing or invalid API key"})
	}

	// Validate API key using existing validation
	_, err := validateAPIKey(app, apiKey)
	if err != nil {
		return e.JSON(401, map[string]string{"error": "Invalid API key"})
	}

	// Find active banners (both public and auth-required) that haven't expired
	records, err := app.FindRecordsByFilter(
		"banners",
		"active = true && (expires_at = '' || expires_at > {:now})",
		"created",
		-1,
		0,
		map[string]interface{}{
			"now": time.Now().Format(time.RFC3339),
		},
	)
	if err != nil {
		return e.JSON(500, map[string]string{"error": "Failed to fetch banners"})
	}

	return e.JSON(200, map[string]interface{}{
		"banners": records,
	})
}

// Helper functions (reused from AI endpoints)

func extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}

func validateAPIKey(app core.App, apiKey string) (*core.Record, error) {
	keyHash := hashAPIKey(apiKey)
	
	// Find API key record
	apiKeyRecord, err := app.FindFirstRecordByFilter("api_keys", "key_hash = {:hash} && active = true", map[string]interface{}{
		"hash": keyHash,
	})
	if err != nil {
		return nil, err
	}

	// Get user record
	userRecord, err := app.FindRecordById("users", apiKeyRecord.GetString("user_id"))
	if err != nil {
		return nil, err
	}

	return userRecord, nil
}

func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}