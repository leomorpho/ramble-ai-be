package subscription

import "time"

// Domain models for subscription business logic
// These are simple structs with no dependencies on PocketBase ORM

// Subscription represents a user's subscription state
type Subscription struct {
	ID                       string
	UserID                   string
	PlanID                   string
	ProviderSubscriptionID   string
	ProviderPriceID          string
	PaymentProvider          string
	Status                   string
	CurrentPeriodStart       time.Time
	CurrentPeriodEnd         time.Time
	CancelAtPeriodEnd        bool
	CanceledAt               *time.Time
	TrialEnd                 *time.Time
	PendingPlanID            string
	PendingChangeEffectiveDate *time.Time
	PendingChangeReason      string
}

// Plan represents a subscription plan
type Plan struct {
	ID               string
	Name             string
	ProviderPriceID  string
	BillingInterval  string
	HoursPerMonth    float64
	PriceCents       int
	DisplayOrder     int
	IsActive         bool
	IsFree           bool
}

// SubscriptionStore defines the interface for subscription storage operations
// This abstracts away PocketBase ORM for easier testing
type SubscriptionStore interface {
	// Subscription operations
	CreateSubscription(sub *Subscription) error
	UpdateSubscription(sub *Subscription) error
	GetSubscription(id string) (*Subscription, error)
	DeleteSubscription(id string) error
	
	// Query operations
	GetActiveSubscription(userID string) (*Subscription, error)
	GetSubscriptionByProviderID(providerSubID string) (*Subscription, error)
	GetAllUserSubscriptions(userID string) ([]*Subscription, error)
	GetActiveSubscriptionsCount(userID string) (int, error)
	
	// Plan operations
	GetPlan(planID string) (*Plan, error)
	GetPlanByProviderPrice(providerPriceID string) (*Plan, error)
	GetFreePlan() (*Plan, error)
	GetAllPlans() ([]*Plan, error)
	GetAvailableUpgrades(currentPlanID string) ([]*Plan, error)
	
	// History operations
	SaveToHistory(sub *Subscription, reason string) error
	GetHistory(userID string) ([]*Subscription, error)
	
	// Bulk operations
	DeactivateAllUserSubscriptions(userID string) error
	CleanupDuplicateSubscriptions(userID string) error
}