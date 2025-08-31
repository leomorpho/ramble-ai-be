package jobs

import (
	"log"

	"github.com/pocketbase/pocketbase/core"
)

// RegisterJobs registers all scheduled jobs with the PocketBase cron scheduler
func RegisterJobs(app core.App) error {
	log.Printf("[JOBS] Registering scheduled jobs...")
	
	// Register OTP cleanup job to run every 10 minutes
	// Cron expression: */10 * * * * means "every 10 minutes"
	err := app.Cron().Add("otp_cleanup", "*/10 * * * *", func() {
		CleanupExpiredOTPs(app)
	})
	
	if err != nil {
		log.Printf("[JOBS] ERROR: Failed to register OTP cleanup job: %v", err)
		return err
	}
	
	log.Printf("[JOBS] Successfully registered OTP cleanup job (runs every 10 minutes)")
	log.Printf("[JOBS] All scheduled jobs registered successfully")
	
	return nil
}