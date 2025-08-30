package subscription

import (
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v79"
)

// Validator handles business rules and validation for subscriptions
type Validator struct {
	repo Repository
}

// NewValidator creates a new subscription validator
func NewValidator(repo Repository) *Validator {
	return &Validator{repo: repo}
}

// ValidateCreateSubscription validates parameters for creating a subscription
func (v *Validator) ValidateCreateSubscription(params CreateSubscriptionParams) []ValidationError {
	var errors []ValidationError

	// Required fields
	if params.UserID == "" {
		errors = append(errors, ValidationError{
			Field:   "user_id",
			Message: "User ID is required",
		})
	}
	if params.PlanID == "" {
		errors = append(errors, ValidationError{
			Field:   "plan_id",
			Message: "Plan ID is required",
		})
	}

	// Validate status
	if !isValidStatus(params.Status) {
		errors = append(errors, ValidationError{
			Field:   "status",
			Message: "Invalid subscription status",
		})
	}

	// Validate dates
	if params.CurrentPeriodStart.IsZero() {
		errors = append(errors, ValidationError{
			Field:   "current_period_start",
			Message: "Current period start is required",
		})
	}
	if params.CurrentPeriodEnd.IsZero() {
		errors = append(errors, ValidationError{
			Field:   "current_period_end",
			Message: "Current period end is required",
		})
	}
	if !params.CurrentPeriodStart.IsZero() && !params.CurrentPeriodEnd.IsZero() && params.CurrentPeriodEnd.Before(params.CurrentPeriodStart) {
		errors = append(errors, ValidationError{
			Field:   "current_period_end",
			Message: "Current period end must be after start date",
		})
	}

	// Validate Unix timestamps (avoid 1970 dates)
	if !params.CurrentPeriodStart.IsZero() && params.CurrentPeriodStart.Year() < 2020 {
		errors = append(errors, ValidationError{
			Field:   "current_period_start",
			Message: "Invalid start date (appears to be Unix timestamp 0)",
		})
	}
	if !params.CurrentPeriodEnd.IsZero() && params.CurrentPeriodEnd.Year() < 2020 {
		errors = append(errors, ValidationError{
			Field:   "current_period_end",
			Message: "Invalid end date (appears to be Unix timestamp 0)",
		})
	}

	return errors
}

// ValidateUpdateSubscription validates parameters for updating a subscription
func (v *Validator) ValidateUpdateSubscription(subscriptionID string, params UpdateSubscriptionParams) []ValidationError {
	var errors []ValidationError

	if subscriptionID == "" {
		errors = append(errors, ValidationError{
			Field:   "subscription_id",
			Message: "Subscription ID is required",
		})
	}

	// Validate status if provided
	if params.Status != nil && !isValidStatus(*params.Status) {
		errors = append(errors, ValidationError{
			Field:   "status",
			Message: "Invalid subscription status",
		})
	}

	// Validate dates if provided
	if params.CurrentPeriodStart != nil && params.CurrentPeriodEnd != nil {
		if params.CurrentPeriodEnd.Before(*params.CurrentPeriodStart) {
			errors = append(errors, ValidationError{
				Field:   "current_period_end",
				Message: "Current period end must be after start date",
			})
		}
	}

	// Validate Unix timestamps
	if params.CurrentPeriodStart != nil && params.CurrentPeriodStart.Year() < 2020 {
		errors = append(errors, ValidationError{
			Field:   "current_period_start",
			Message: "Invalid start date (appears to be Unix timestamp 0)",
		})
	}
	if params.CurrentPeriodEnd != nil && params.CurrentPeriodEnd.Year() < 2020 {
		errors = append(errors, ValidationError{
			Field:   "current_period_end",
			Message: "Invalid end date (appears to be Unix timestamp 0)",
		})
	}

	return errors
}

// ValidateBusinessRules checks business rules for subscription operations
func (v *Validator) ValidateBusinessRules(userID string, operation string) []BusinessRuleError {
	var errors []BusinessRuleError

	switch operation {
	case "create_active":
		// Check if user already has active subscription
		activeCount, err := v.repo.FindActiveSubscriptionsCount(userID)
		if err == nil && activeCount > 0 {
			errors = append(errors, BusinessRuleError{
				Rule:    "single_active_subscription",
				Message: "User already has an active subscription",
			})
		}
	}

	return errors
}

// ValidatePlanChange validates a plan change request
func (v *Validator) ValidatePlanChange(userID string, newPlanID string) []ValidationError {
	var errors []ValidationError

	if userID == "" {
		errors = append(errors, ValidationError{
			Field:   "user_id",
			Message: "User ID is required",
		})
	}
	if newPlanID == "" {
		errors = append(errors, ValidationError{
			Field:   "new_plan_id",
			Message: "New plan ID is required",
		})
	}

	// Validate new plan exists
	if newPlanID != "" {
		if _, err := v.repo.GetPlan(newPlanID); err != nil {
			errors = append(errors, ValidationError{
				Field:   "new_plan_id",
				Message: "Plan does not exist",
			})
		}
	}

	return errors
}

// ValidateStripeWebhookData validates Stripe webhook event data
func (v *Validator) ValidateStripeWebhookData(data WebhookEventData) []ValidationError {
	var errors []ValidationError

	if data.EventType == "" {
		errors = append(errors, ValidationError{
			Field:   "event_type",
			Message: "Event type is required",
		})
	}

	switch data.EventType {
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		if data.Subscription == nil {
			errors = append(errors, ValidationError{
				Field:   "subscription",
				Message: "Subscription data is required for subscription events",
			})
		} else {
			// Validate subscription data
			if data.Subscription.Customer == nil {
				errors = append(errors, ValidationError{
					Field:   "subscription.customer",
					Message: "Customer data is required",
				})
			}
		}

	case "invoice.payment_succeeded", "invoice.payment_failed":
		if data.Invoice == nil {
			errors = append(errors, ValidationError{
				Field:   "invoice",
				Message: "Invoice data is required for invoice events",
			})
		}

	case "checkout.session.completed":
		if data.CheckoutSession == nil {
			errors = append(errors, ValidationError{
				Field:   "checkout_session",
				Message: "Checkout session data is required for checkout events",
			})
		}
	}

	return errors
}

// FixInvalidTimestamps fixes timestamps that are Unix timestamp 0 (1970)
func (v *Validator) FixInvalidTimestamps(start, end time.Time) (time.Time, time.Time) {
	now := time.Now()
	
	fixedStart := start
	fixedEnd := end

	// Fix start date if it's invalid (before 2020)
	if start.IsZero() || start.Year() < 2020 {
		fixedStart = now
	}

	// Fix end date if it's invalid (before 2020)
	if end.IsZero() || end.Year() < 2020 {
		// Default to 30 days from start for monthly subscriptions
		fixedEnd = fixedStart.AddDate(0, 1, 0)
	}

	return fixedStart, fixedEnd
}

// MapStripeStatus maps Stripe subscription status to internal status
func (v *Validator) MapStripeStatus(stripeStatus stripe.SubscriptionStatus) SubscriptionStatus {
	switch stripeStatus {
	case stripe.SubscriptionStatusActive:
		return StatusActive
	case stripe.SubscriptionStatusCanceled:
		return StatusCanceled
	case stripe.SubscriptionStatusPastDue:
		return StatusPastDue
	case stripe.SubscriptionStatusTrialing:
		return StatusTrialing
	default:
		return StatusActive // Default fallback
	}
}

// ExtractPriceFromSubscription extracts the price ID from a Stripe subscription
func (v *Validator) ExtractPriceFromSubscription(stripeSub *stripe.Subscription) (string, error) {
	if stripeSub == nil {
		return "", fmt.Errorf("subscription is nil")
	}
	if stripeSub.Items == nil || len(stripeSub.Items.Data) == 0 {
		return "", fmt.Errorf("subscription has no items")
	}
	if stripeSub.Items.Data[0].Price == nil {
		return "", fmt.Errorf("subscription item has no price")
	}
	return stripeSub.Items.Data[0].Price.ID, nil
}

// isValidStatus checks if a subscription status is valid
func isValidStatus(status SubscriptionStatus) bool {
	switch status {
	case StatusActive, StatusCanceled, StatusPastDue, StatusTrialing:
		return true
	default:
		return false
	}
}

// GetUsageWarningMessage returns a warning message if user is approaching limits
func (v *Validator) GetUsageWarningMessage(usage *UsageInfo) string {
	if usage == nil {
		return ""
	}

	if usage.IsOverLimit {
		return fmt.Sprintf("You have exceeded your monthly limit of %.1f hours. Additional processing may be restricted.", usage.HoursLimit)
	}

	usagePercent := (usage.HoursUsedThisMonth / usage.HoursLimit) * 100

	if usagePercent >= 90 {
		remaining := usage.HoursLimit - usage.HoursUsedThisMonth
		return fmt.Sprintf("You have %.1f hours remaining this month (%.0f%% used).", remaining, usagePercent)
	}

	if usagePercent >= 75 {
		return fmt.Sprintf("You have used %.0f%% of your monthly hours limit.", usagePercent)
	}

	return ""
}