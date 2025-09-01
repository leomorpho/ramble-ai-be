package subscription

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// Repository handles all database operations for subscriptions
type Repository interface {
	// Core CRUD operations
	CreateSubscription(params CreateSubscriptionParams) (*core.Record, error)
	UpdateSubscription(subscriptionID string, params UpdateSubscriptionParams) (*core.Record, error)
	GetSubscription(subscriptionID string) (*core.Record, error)
	DeleteSubscription(subscriptionID string) error

	// Query operations
	FindSubscription(query SubscriptionQuery) (*core.Record, error)
	FindActiveSubscription(userID string) (*core.Record, error)
	FindSubscriptionByProviderID(providerSubID string) (*core.Record, error)
	FindAllUserSubscriptions(userID string) ([]*core.Record, error)
	FindActiveSubscriptionsCount(userID string) (int, error)

	// Plan operations
	GetPlan(planID string) (*core.Record, error)
	GetPlanByProviderPrice(providerPriceID string) (*core.Record, error)
	GetFreePlan() (*core.Record, error)
	GetAllPlans() ([]*core.Record, error)
	GetAvailableUpgrades(currentPlanID string) ([]*core.Record, error)

	// Bulk operations
	DeactivateAllUserSubscriptions(userID string) error
	CleanupDuplicateSubscriptions(userID string) error
	
	// Subscription history operations
	MoveSubscriptionToHistory(subscriptionRecord *core.Record, reason string) (*core.Record, error)
	GetUserSubscriptionHistory(userID string) ([]*core.Record, error)
}

// PocketBaseRepository implements Repository using PocketBase
type PocketBaseRepository struct {
	app core.App
}

// NewRepository creates a new PocketBase repository
func NewRepository(app core.App) Repository {
	return &PocketBaseRepository{app: app}
}

// CreateSubscription creates a new subscription record
func (r *PocketBaseRepository) CreateSubscription(params CreateSubscriptionParams) (*core.Record, error) {
	collection, err := r.app.FindCollectionByNameOrId("current_user_subscriptions")
	if err != nil {
		return nil, fmt.Errorf("failed to find current_user_subscriptions collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("user_id", params.UserID)
	record.Set("plan_id", params.PlanID)
	record.Set("status", string(params.Status))
	record.Set("current_period_start", params.CurrentPeriodStart)
	record.Set("current_period_end", params.CurrentPeriodEnd)

	if params.ProviderSubscriptionID != nil {
		record.Set("provider_subscription_id", *params.ProviderSubscriptionID)
	}
	if params.ProviderPriceID != nil {
		record.Set("provider_price_id", *params.ProviderPriceID)
	}
	if params.PaymentProvider != nil {
		record.Set("payment_provider", *params.PaymentProvider)
	}
	if params.CanceledAt != nil {
		record.Set("canceled_at", *params.CanceledAt)
	}

	if err := r.app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return record, nil
}

// UpdateSubscription updates an existing subscription record
func (r *PocketBaseRepository) UpdateSubscription(subscriptionID string, params UpdateSubscriptionParams) (*core.Record, error) {
	record, err := r.app.FindRecordById("current_user_subscriptions", subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription %s: %w", subscriptionID, err)
	}

	if params.PlanID != nil {
		record.Set("plan_id", *params.PlanID)
	}
	if params.ProviderSubscriptionID != nil {
		record.Set("provider_subscription_id", *params.ProviderSubscriptionID)
	}
	if params.ProviderPriceID != nil {
		record.Set("provider_price_id", *params.ProviderPriceID)
	}
	if params.PaymentProvider != nil {
		record.Set("payment_provider", *params.PaymentProvider)
	}
	if params.Status != nil {
		record.Set("status", string(*params.Status))
	}
	if params.CurrentPeriodStart != nil {
		record.Set("current_period_start", *params.CurrentPeriodStart)
	}
	if params.CurrentPeriodEnd != nil {
		record.Set("current_period_end", *params.CurrentPeriodEnd)
	}
	if params.CanceledAt != nil {
		record.Set("canceled_at", *params.CanceledAt)
	}

	if err := r.app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to update subscription %s: %w", subscriptionID, err)
	}

	return record, nil
}

// GetSubscription retrieves a subscription by ID
func (r *PocketBaseRepository) GetSubscription(subscriptionID string) (*core.Record, error) {
	record, err := r.app.FindRecordById("current_user_subscriptions", subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription %s: %w", subscriptionID, err)
	}
	return record, nil
}

// DeleteSubscription removes a subscription record
func (r *PocketBaseRepository) DeleteSubscription(subscriptionID string) error {
	record, err := r.app.FindRecordById("current_user_subscriptions", subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to find subscription %s: %w", subscriptionID, err)
	}

	if err := r.app.Delete(record); err != nil {
		return fmt.Errorf("failed to delete subscription %s: %w", subscriptionID, err)
	}

	return nil
}

// FindSubscription finds a subscription based on query parameters
func (r *PocketBaseRepository) FindSubscription(query SubscriptionQuery) (*core.Record, error) {
	filter := "user_id = {:user_id}"
	params := map[string]any{"user_id": query.UserID}

	if query.Status != nil {
		filter += " && status = {:status}"
		params["status"] = string(*query.Status)
	}
	if query.ProviderSubscriptionID != nil {
		filter += " && provider_subscription_id = {:stripe_sub_id}"
		params["stripe_sub_id"] = *query.ProviderSubscriptionID
	}
	if query.PlanID != nil {
		filter += " && plan_id = {:plan_id}"
		params["plan_id"] = *query.PlanID
	}

	record, err := r.app.FindFirstRecordByFilter("current_user_subscriptions", filter, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription: %w", err)
	}
	return record, nil
}

// FindActiveSubscription finds the active subscription for a user
func (r *PocketBaseRepository) FindActiveSubscription(userID string) (*core.Record, error) {
	status := StatusActive
	query := SubscriptionQuery{
		UserID: userID,
		Status: &status,
	}
	return r.FindSubscription(query)
}

// FindSubscriptionByProviderID finds a subscription by Stripe subscription ID
func (r *PocketBaseRepository) FindSubscriptionByProviderID(stripeSubID string) (*core.Record, error) {
	record, err := r.app.FindFirstRecordByFilter("current_user_subscriptions", "provider_subscription_id = {:stripe_sub_id}", map[string]any{
		"stripe_sub_id": stripeSubID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription by Stripe ID %s: %w", stripeSubID, err)
	}
	return record, nil
}

// FindAllUserSubscriptions retrieves all subscriptions for a user
func (r *PocketBaseRepository) FindAllUserSubscriptions(userID string) ([]*core.Record, error) {
	records, err := r.app.FindRecordsByFilter("current_user_subscriptions", "user_id = {:user_id}", "-created", 100, 0, map[string]any{
		"user_id": userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find user subscriptions: %w", err)
	}
	return records, nil
}

// FindActiveSubscriptionsCount counts active subscriptions for a user
func (r *PocketBaseRepository) FindActiveSubscriptionsCount(userID string) (int, error) {
	records, err := r.app.FindRecordsByFilter("current_user_subscriptions", "user_id = {:user_id} && status = 'active'", "-created", 100, 0, map[string]any{
		"user_id": userID,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count active subscriptions: %w", err)
	}
	return len(records), nil
}

// GetPlan retrieves a subscription plan by ID
func (r *PocketBaseRepository) GetPlan(planID string) (*core.Record, error) {
	record, err := r.app.FindRecordById("subscription_plans", planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan %s: %w", planID, err)
	}
	return record, nil
}

// GetPlanByProviderPrice retrieves a plan by Stripe price ID
func (r *PocketBaseRepository) GetPlanByProviderPrice(stripePriceID string) (*core.Record, error) {
	record, err := r.app.FindFirstRecordByFilter("subscription_plans", "provider_price_id = {:price_id}", map[string]any{
		"price_id": stripePriceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find plan for price %s: %w", stripePriceID, err)
	}
	return record, nil
}

// GetFreePlan retrieves the free plan
func (r *PocketBaseRepository) GetFreePlan() (*core.Record, error) {
	record, err := r.app.FindFirstRecordByFilter("subscription_plans", "billing_interval = 'free'", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("failed to find free plan: %w", err)
	}
	return record, nil
}

// GetAllPlans retrieves all available subscription plans ordered by price (cheapest to most expensive)
func (r *PocketBaseRepository) GetAllPlans() ([]*core.Record, error) {
	records, err := r.app.FindRecordsByFilter("subscription_plans", "is_active = true", "+price_cents", 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get all plans: %w", err)
	}
	return records, nil
}

// GetAvailableUpgrades returns plans with higher limits than the current plan
func (r *PocketBaseRepository) GetAvailableUpgrades(currentPlanID string) ([]*core.Record, error) {
	currentPlan, err := r.GetPlan(currentPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current plan: %w", err)
	}

	currentHoursLimit := currentPlan.GetFloat("hours_per_month")

	records, err := r.app.FindRecordsByFilter("subscription_plans", "is_active = true && hours_per_month > {:current_hours}", "+price_cents", 0, 0, map[string]any{
		"current_hours": currentHoursLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get upgrade options: %w", err)
	}
	return records, nil
}

// DeactivateAllUserSubscriptions marks all user subscriptions as cancelled
func (r *PocketBaseRepository) DeactivateAllUserSubscriptions(userID string) error {
	subscriptions, err := r.app.FindRecordsByFilter("current_user_subscriptions", "user_id = {:user_id} && status = 'active'", "-created", 100, 0, map[string]any{
		"user_id": userID,
	})
	if err != nil {
		return fmt.Errorf("failed to find active subscriptions: %w", err)
	}

	for _, sub := range subscriptions {
		sub.Set("status", "cancelled")
		sub.Set("canceled_at", time.Now())
		if err := r.app.Save(sub); err != nil {
			log.Printf("Failed to deactivate subscription %s: %v", sub.Id, err)
		}
	}

	log.Printf("Deactivated %d subscriptions for user %s", len(subscriptions), userID)
	return nil
}

// CleanupDuplicateSubscriptions ensures only one active subscription per user
func (r *PocketBaseRepository) CleanupDuplicateSubscriptions(userID string) error {
	activeSubscriptions, err := r.app.FindRecordsByFilter("current_user_subscriptions", "user_id = {:user_id} && status = 'active'", "-created", 100, 0, map[string]any{
		"user_id": userID,
	})
	if err != nil {
		return fmt.Errorf("failed to find active subscriptions: %w", err)
	}

	if len(activeSubscriptions) <= 1 {
		return nil // No duplicates to clean up
	}

	// Keep the most recently created subscription, deactivate the rest
	for i := 1; i < len(activeSubscriptions); i++ {
		sub := activeSubscriptions[i]
		sub.Set("status", "cancelled")
		sub.Set("canceled_at", time.Now())
		if err := r.app.Save(sub); err != nil {
			log.Printf("Failed to deactivate duplicate subscription %s: %v", sub.Id, err)
		}
	}

	log.Printf("Cleaned up %d duplicate subscriptions for user %s", len(activeSubscriptions)-1, userID)
	return nil
}

// MoveSubscriptionToHistory moves a current subscription to the history table
func (r *PocketBaseRepository) MoveSubscriptionToHistory(subscriptionRecord *core.Record, reason string) (*core.Record, error) {
	// Get subscription history collection
	historyCollection, err := r.app.FindCollectionByNameOrId("subscription_history")
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription_history collection: %w", err)
	}
	
	// Create history record with current subscription data
	historyRecord := core.NewRecord(historyCollection)
	
	// Copy all fields except pending fields and IDs
	historyRecord.Set("user_id", subscriptionRecord.GetString("user_id"))
	historyRecord.Set("plan_id", subscriptionRecord.GetString("plan_id"))
	historyRecord.Set("provider_subscription_id", subscriptionRecord.GetString("provider_subscription_id"))
	historyRecord.Set("provider_price_id", subscriptionRecord.GetString("provider_price_id"))
	historyRecord.Set("payment_provider", subscriptionRecord.GetString("payment_provider"))
	historyRecord.Set("status", subscriptionRecord.GetString("status"))
	historyRecord.Set("current_period_start", subscriptionRecord.Get("current_period_start"))
	historyRecord.Set("current_period_end", subscriptionRecord.Get("current_period_end"))
	historyRecord.Set("canceled_at", subscriptionRecord.Get("canceled_at"))
	
	// Set history-specific fields
	historyRecord.Set("replaced_at", time.Now())
	historyRecord.Set("replacement_reason", reason)
	
	// Save to history
	if err := r.app.Save(historyRecord); err != nil {
		return nil, fmt.Errorf("failed to save subscription to history: %w", err)
	}
	
	log.Printf("Moved subscription %s to history with reason: %s", subscriptionRecord.Id, reason)
	return historyRecord, nil
}

// GetUserSubscriptionHistory retrieves all historical subscriptions for a user
func (r *PocketBaseRepository) GetUserSubscriptionHistory(userID string) ([]*core.Record, error) {
	records, err := r.app.FindRecordsByFilter("subscription_history", "user_id = {:user_id}", "-replaced_at", 100, 0, map[string]any{
		"user_id": userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription history: %w", err)
	}
	return records, nil
}