package jobs

import (
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// CleanupExpiredOTPs removes all user_otps entries that have expired (older than their expires_at timestamp)
func CleanupExpiredOTPs(app core.App) {
	log.Printf("[OTP_CLEANUP] Starting cleanup of expired OTP entries...")
	
	startTime := time.Now()
	
	// Delete all expired OTP entries
	// expires_at < datetime('now') finds all entries that have passed their expiration time
	query := app.DB().NewQuery("DELETE FROM user_otps WHERE expires_at < datetime('now')")
	
	result, err := query.Execute()
	if err != nil {
		log.Printf("[OTP_CLEANUP] ERROR: Failed to delete expired OTP entries: %v", err)
		return
	}
	
	// Get the number of affected rows
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[OTP_CLEANUP] WARNING: Could not get affected rows count: %v", err)
		rowsAffected = 0
	}
	
	duration := time.Since(startTime)
	log.Printf("[OTP_CLEANUP] Cleanup completed successfully. Deleted %d expired OTP entries in %v", rowsAffected, duration)
}