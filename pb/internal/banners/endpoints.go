package banners

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// GetBannersHandler handles all banner requests with optional authentication and filtering
func GetBannersHandler(e *core.RequestEvent, app core.App) error {
	// Check for API key
	apiKey := extractBearerToken(e.Request.Header.Get("Authorization"))
	
	// Get query parameter to determine if we should include dismissed banners
	includeDismissed := e.Request.URL.Query().Get("include_dismissed") == "true"
	
	var records []*core.Record
	var err error
	
	if apiKey == "" {
		// No API key - return only public banners
		records, err = app.FindRecordsByFilter(
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
	
	// Validate API key
	_, err = validateAPIKey(app, apiKey)
	if err != nil {
		return e.JSON(401, map[string]string{"error": "Invalid API key"})
	}
	
	// Authenticated request - get all accessible banners
	records, err = app.FindRecordsByFilter(
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
	
	// Add dismissal status to each banner
	keyHash := hashAPIKey(apiKey)
	bannersWithStatus := make([]map[string]interface{}, 0, len(records))
	
	for _, banner := range records {
		bannerData := make(map[string]interface{})
		
		// Copy all banner fields manually
		bannerData["id"] = banner.Id
		bannerData["created"] = banner.GetString("created")
		bannerData["updated"] = banner.GetString("updated")
		bannerData["title"] = banner.GetString("title")
		bannerData["message"] = banner.GetString("message")
		bannerData["type"] = banner.GetString("type")
		bannerData["action_text"] = banner.GetString("action_text")
		bannerData["action_url"] = banner.GetString("action_url")
		bannerData["expires_at"] = banner.GetString("expires_at")
		bannerData["requires_auth"] = banner.GetBool("requires_auth")
		bannerData["active"] = banner.GetBool("active")
		
		// Check if this banner has been dismissed by this API key
		dismissalID := keyHash + "_" + banner.Id
		_, err := app.FindFirstRecordByFilter("banner_dismissals", 
			"id = {:dismissal_id}", 
			map[string]interface{}{
				"dismissal_id": dismissalID,
			},
		)
		
		// If dismissal found (no error), mark as dismissed
		isDismissed := (err == nil)
		bannerData["dismissed"] = isDismissed
		
		// Filter logic: if includeDismissed=false and banner is dismissed, skip it
		if !includeDismissed && isDismissed {
			continue
		}
		
		bannersWithStatus = append(bannersWithStatus, bannerData)
	}
	
	return e.JSON(200, map[string]interface{}{
		"banners": bannersWithStatus,
	})
}

// DismissBannerHandler handles dismissing a banner for a specific API key
func DismissBannerHandler(e *core.RequestEvent, app core.App) error {
	// Validate API key
	apiKey := extractBearerToken(e.Request.Header.Get("Authorization"))
	if apiKey == "" {
		return e.JSON(401, map[string]string{"error": "Missing or invalid API key"})
	}

	// Validate API key using existing validation
	userRecord, err := validateAPIKey(app, apiKey)
	if err != nil {
		return e.JSON(401, map[string]string{"error": "Invalid API key"})
	}

	// Get banner ID from URL parameter
	bannerID := e.Request.PathValue("id")
	if bannerID == "" {
		return e.JSON(400, map[string]string{"error": "Missing banner ID"})
	}

	// Verify banner exists
	bannerRecord, err := app.FindRecordById("banners", bannerID)
	if err != nil {
		return e.JSON(404, map[string]string{"error": "Banner not found"})
	}

	// Create or update dismissal record
	// We'll store dismissals using a combination of API key hash and banner ID
	keyHash := hashAPIKey(apiKey)
	dismissalID := keyHash + "_" + bannerID

	// Check if dismissal already exists
	existingDismissal, err := app.FindFirstRecordByFilter("banner_dismissals", 
		"id = {:dismissal_id}", 
		map[string]interface{}{
			"dismissal_id": dismissalID,
		},
	)

	if err != nil && err.Error() != "no rows in result set" {
		return e.JSON(500, map[string]string{"error": "Failed to check existing dismissal"})
	}

	if existingDismissal != nil {
		// Already dismissed
		return e.JSON(200, map[string]interface{}{
			"success": true,
			"message": "Banner already dismissed",
		})
	}

	// Create new dismissal record
	dismissalsCollection, err := app.FindCollectionByNameOrId("banner_dismissals")
	if err != nil {
		return e.JSON(500, map[string]string{"error": "Dismissals collection not found"})
	}

	dismissalRecord := core.NewRecord(dismissalsCollection)
	dismissalRecord.Set("id", dismissalID)
	dismissalRecord.Set("banner_id", bannerID)
	dismissalRecord.Set("user_id", userRecord.Id) // For reference, though we primarily use API key hash
	dismissalRecord.Set("api_key_hash", keyHash)
	dismissalRecord.Set("dismissed_at", time.Now().Format(time.RFC3339))

	if err := app.Save(dismissalRecord); err != nil {
		return e.JSON(500, map[string]string{"error": "Failed to save dismissal"})
	}

	return e.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Banner dismissed successfully",
		"banner_title": bannerRecord.GetString("title"),
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