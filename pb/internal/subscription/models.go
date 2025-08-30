package subscription

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v79"
)

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	StatusActive   SubscriptionStatus = "active"
	StatusCanceled SubscriptionStatus = "cancelled"
	StatusPastDue  SubscriptionStatus = "past_due"
	StatusTrialing SubscriptionStatus = "trialing"
)

// SubscriptionInfo represents comprehensive subscription information
type SubscriptionInfo struct {
	Subscription   *core.Record     `json:"subscription"`
	Plan           *core.Record     `json:"plan"`
	Usage          *UsageInfo       `json:"usage"`
	AvailablePlans []*core.Record   `json:"available_plans"`
}

// UsageInfo represents user usage statistics
type UsageInfo struct {
	HoursUsedThisMonth float64 `json:"hours_used_this_month"`
	HoursLimit         float64 `json:"hours_limit"`
	FilesProcessed     int     `json:"files_processed"`
	IsOverLimit        bool    `json:"is_over_limit"`
	DaysUntilReset     int     `json:"days_until_reset"`
}

// CreateSubscriptionParams represents parameters for creating a subscription
type CreateSubscriptionParams struct {
	UserID                   string
	PlanID                   string
	ProviderSubscriptionID   *string
	ProviderPriceID          *string
	PaymentProvider          *string
	Status                   SubscriptionStatus
	CurrentPeriodStart       time.Time
	CurrentPeriodEnd         time.Time
	CancelAtPeriodEnd        bool
	CanceledAt               *time.Time
	TrialEnd                 *time.Time
}

// UpdateSubscriptionParams represents parameters for updating a subscription
type UpdateSubscriptionParams struct {
	PlanID                   *string
	ProviderSubscriptionID   *string
	ProviderPriceID          *string
	PaymentProvider          *string
	Status                   *SubscriptionStatus
	CurrentPeriodStart       *time.Time
	CurrentPeriodEnd         *time.Time
	CancelAtPeriodEnd        *bool
	CanceledAt               *time.Time
	TrialEnd                 *time.Time
}

// SubscriptionQuery represents query parameters for finding subscriptions
type SubscriptionQuery struct {
	UserID                 string
	Status                 *SubscriptionStatus
	ProviderSubscriptionID *string
	PlanID                 *string
	PaymentProvider        *string
}

// PlanChangeRequest represents a request to change subscription plans
type PlanChangeRequest struct {
	UserID     string `json:"user_id"`
	NewPlanID  string `json:"new_plan_id"`
	ProrationBehavior string `json:"proration_behavior,omitempty"`
}

// WebhookEventData represents data extracted from Stripe webhook events
type WebhookEventData struct {
	EventType     string
	Subscription  *stripe.Subscription
	Invoice       *stripe.Invoice
	Customer      *stripe.Customer
	CheckoutSession *stripe.CheckoutSession
}

// ValidationError represents a subscription validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// BusinessRuleError represents a business rule violation
type BusinessRuleError struct {
	Rule    string
	Message string
}

func (e BusinessRuleError) Error() string {
	return e.Message
}