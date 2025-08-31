package subscription

import (
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v79"
)

// CleanValidator handles validation logic for the clean service
// This works with domain models instead of PocketBase records
type CleanValidator struct{}

// NewCleanValidator creates a new validator for clean service
func NewCleanValidator() *CleanValidator {
	return &CleanValidator{}
}

// ExtractPriceFromSubscription safely extracts the price ID from a Stripe subscription
func (v *CleanValidator) ExtractPriceFromSubscription(sub *stripe.Subscription) (string, error) {
	if sub.Items == nil || len(sub.Items.Data) == 0 {
		return "", fmt.Errorf("subscription has no items")
	}

	item := sub.Items.Data[0]
	if item.Price == nil {
		return "", fmt.Errorf("subscription item has no price")
	}

	return item.Price.ID, nil
}

// MapStripeStatus converts Stripe subscription status to our internal status
func (v *CleanValidator) MapStripeStatus(status stripe.SubscriptionStatus) string {
	switch status {
	case stripe.SubscriptionStatusActive:
		return "active"
	case stripe.SubscriptionStatusCanceled:
		return "cancelled"
	case stripe.SubscriptionStatusPastDue:
		return "past_due"
	case stripe.SubscriptionStatusTrialing:
		return "trialing"
	case stripe.SubscriptionStatusIncomplete:
		return "incomplete"
	case stripe.SubscriptionStatusIncompleteExpired:
		return "incomplete_expired"
	case stripe.SubscriptionStatusUnpaid:
		return "unpaid"
	default:
		return "active" // Safe default
	}
}

// FixInvalidTimestamps fixes common timestamp issues (like 1970 dates)
func (v *CleanValidator) FixInvalidTimestamps(start, end time.Time) (time.Time, time.Time) {
	// Fix 1970 epoch timestamps that sometimes come from Stripe
	epoch := time.Date(1971, 1, 1, 0, 0, 0, 0, time.UTC)
	
	if start.Before(epoch) {
		start = time.Now()
	}
	
	if end.Before(epoch) {
		end = start.AddDate(0, 1, 0) // Default to 1 month from start
	}
	
	// Ensure end is after start
	if end.Before(start) || end.Equal(start) {
		end = start.AddDate(0, 1, 0)
	}
	
	return start, end
}

// ValidateSubscription performs basic subscription validation
func (v *CleanValidator) ValidateSubscription(sub *Subscription) []string {
	var errors []string
	
	if sub.UserID == "" {
		errors = append(errors, "user ID is required")
	}
	
	if sub.PlanID == "" {
		errors = append(errors, "plan ID is required")
	}
	
	if sub.Status == "" {
		errors = append(errors, "status is required")
	}
	
	if sub.CurrentPeriodEnd.Before(sub.CurrentPeriodStart) {
		errors = append(errors, "period end must be after period start")
	}
	
	return errors
}