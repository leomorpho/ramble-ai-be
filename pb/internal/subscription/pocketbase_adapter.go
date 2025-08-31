package subscription

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// PocketBaseAdapter adapts the clean service domain models to PocketBase storage
// This bridges the gap between our domain-focused clean architecture and PocketBase ORM
type PocketBaseAdapter struct {
	app *pocketbase.PocketBase
}

// NewPocketBaseAdapter creates a new adapter for PocketBase storage
func NewPocketBaseAdapter(app *pocketbase.PocketBase) *PocketBaseAdapter {
	return &PocketBaseAdapter{app: app}
}

// CreateSubscription creates a new subscription record in PocketBase
func (a *PocketBaseAdapter) CreateSubscription(sub *Subscription) error {
	collection, err := a.app.FindCollectionByNameOrId("current_user_subscriptions")
	if err != nil {
		return fmt.Errorf("failed to find current_user_subscriptions collection: %w", err)
	}

	record := core.NewRecord(collection)
	a.mapSubscriptionToRecord(sub, record)

	if err := a.app.Save(record); err != nil {
		return fmt.Errorf("failed to save subscription: %w", err)
	}

	// Set the ID back to the domain object
	sub.ID = record.Id
	return nil
}

// UpdateSubscription updates an existing subscription record
func (a *PocketBaseAdapter) UpdateSubscription(sub *Subscription) error {
	record, err := a.app.FindRecordById("current_user_subscriptions", sub.ID)
	if err != nil {
		return fmt.Errorf("failed to find subscription %s: %w", sub.ID, err)
	}

	a.mapSubscriptionToRecord(sub, record)
	return a.app.Save(record)
}

// GetSubscription retrieves a subscription by ID
func (a *PocketBaseAdapter) GetSubscription(id string) (*Subscription, error) {
	record, err := a.app.FindRecordById("current_user_subscriptions", id)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	return a.mapRecordToSubscription(record), nil
}

// DeleteSubscription deletes a subscription record
func (a *PocketBaseAdapter) DeleteSubscription(id string) error {
	record, err := a.app.FindRecordById("current_user_subscriptions", id)
	if err != nil {
		return fmt.Errorf("failed to find subscription %s: %w", id, err)
	}

	return a.app.Delete(record)
}

// GetActiveSubscription finds the active subscription for a user
func (a *PocketBaseAdapter) GetActiveSubscription(userID string) (*Subscription, error) {
	records, err := a.app.FindRecordsByFilter(
		"current_user_subscriptions",
		"user_id = {:user_id} && status = 'active'",
		"-created",
		1, 0,
		map[string]any{"user_id": userID},
	)
	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("no active subscription found")
	}

	return a.mapRecordToSubscription(records[0]), nil
}

// GetSubscriptionByProviderID finds a subscription by provider subscription ID
func (a *PocketBaseAdapter) GetSubscriptionByProviderID(providerSubID string) (*Subscription, error) {
	records, err := a.app.FindRecordsByFilter(
		"current_user_subscriptions",
		"provider_subscription_id = {:provider_sub_id}",
		"-created",
		1, 0,
		map[string]any{"provider_sub_id": providerSubID},
	)
	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("subscription not found")
	}

	return a.mapRecordToSubscription(records[0]), nil
}

// GetAllUserSubscriptions retrieves all subscriptions for a user
func (a *PocketBaseAdapter) GetAllUserSubscriptions(userID string) ([]*Subscription, error) {
	records, err := a.app.FindRecordsByFilter(
		"current_user_subscriptions",
		"user_id = {:user_id}",
		"-created",
		100, 0,
		map[string]any{"user_id": userID},
	)
	if err != nil {
		return nil, err
	}

	subs := make([]*Subscription, len(records))
	for i, record := range records {
		subs[i] = a.mapRecordToSubscription(record)
	}
	return subs, nil
}

// GetActiveSubscriptionsCount counts active subscriptions for a user
func (a *PocketBaseAdapter) GetActiveSubscriptionsCount(userID string) (int, error) {
	records, err := a.app.FindRecordsByFilter(
		"current_user_subscriptions",
		"user_id = {:user_id} && status = 'active'",
		"-created",
		100, 0,
		map[string]any{"user_id": userID},
	)
	if err != nil {
		return 0, err
	}
	return len(records), nil
}

// GetPlan retrieves a plan by ID
func (a *PocketBaseAdapter) GetPlan(planID string) (*Plan, error) {
	record, err := a.app.FindRecordById("subscription_plans", planID)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}

	return a.mapRecordToPlan(record), nil
}

// GetPlanByProviderPrice finds a plan by provider price ID
func (a *PocketBaseAdapter) GetPlanByProviderPrice(providerPriceID string) (*Plan, error) {
	records, err := a.app.FindRecordsByFilter(
		"subscription_plans",
		"provider_price_id = {:price_id}",
		"",
		1, 0,
		map[string]any{"price_id": providerPriceID},
	)
	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("plan not found for price")
	}

	return a.mapRecordToPlan(records[0]), nil
}

// GetFreePlan retrieves the free plan
func (a *PocketBaseAdapter) GetFreePlan() (*Plan, error) {
	records, err := a.app.FindRecordsByFilter(
		"subscription_plans",
		"billing_interval = 'free' && is_active = true",
		"created",
		1, 0,
		nil,
	)
	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("free plan not found")
	}

	return a.mapRecordToPlan(records[0]), nil
}

// GetAllPlans retrieves all active plans
func (a *PocketBaseAdapter) GetAllPlans() ([]*Plan, error) {
	records, err := a.app.FindRecordsByFilter(
		"subscription_plans",
		"is_active = true",
		"display_order",
		100, 0,
		nil,
	)
	if err != nil {
		return nil, err
	}

	plans := make([]*Plan, len(records))
	for i, record := range records {
		plans[i] = a.mapRecordToPlan(record)
	}
	return plans, nil
}

// GetAvailableUpgrades finds plans with more hours than the current plan
func (a *PocketBaseAdapter) GetAvailableUpgrades(currentPlanID string) ([]*Plan, error) {
	currentPlan, err := a.GetPlan(currentPlanID)
	if err != nil {
		return nil, err
	}

	records, err := a.app.FindRecordsByFilter(
		"subscription_plans",
		"is_active = true && hours_per_month > {:current_hours}",
		"display_order",
		100, 0,
		map[string]any{"current_hours": currentPlan.HoursPerMonth},
	)
	if err != nil {
		return nil, err
	}

	plans := make([]*Plan, len(records))
	for i, record := range records {
		plans[i] = a.mapRecordToPlan(record)
	}
	return plans, nil
}

// SaveToHistory moves a subscription to history
func (a *PocketBaseAdapter) SaveToHistory(sub *Subscription, reason string) error {
	collection, err := a.app.FindCollectionByNameOrId("subscription_history")
	if err != nil {
		return fmt.Errorf("failed to find subscription_history collection: %w", err)
	}

	record := core.NewRecord(collection)
	a.mapSubscriptionToRecord(sub, record)
	record.Set("change_reason", reason)
	record.Set("moved_to_history_at", time.Now())

	return a.app.Save(record)
}

// GetHistory retrieves subscription history for a user
func (a *PocketBaseAdapter) GetHistory(userID string) ([]*Subscription, error) {
	records, err := a.app.FindRecordsByFilter(
		"subscription_history",
		"user_id = {:user_id}",
		"-moved_to_history_at",
		100, 0,
		map[string]any{"user_id": userID},
	)
	if err != nil {
		return nil, err
	}

	subs := make([]*Subscription, len(records))
	for i, record := range records {
		subs[i] = a.mapRecordToSubscription(record)
	}
	return subs, nil
}

// DeactivateAllUserSubscriptions deactivates all user subscriptions
func (a *PocketBaseAdapter) DeactivateAllUserSubscriptions(userID string) error {
	records, err := a.app.FindRecordsByFilter(
		"current_user_subscriptions",
		"user_id = {:user_id} && status = 'active'",
		"",
		100, 0,
		map[string]any{"user_id": userID},
	)
	if err != nil {
		return err
	}

	for _, record := range records {
		record.Set("status", "cancelled")
		record.Set("canceled_at", time.Now())
		if err := a.app.Save(record); err != nil {
			return fmt.Errorf("failed to deactivate subscription %s: %w", record.Id, err)
		}
	}
	return nil
}

// CleanupDuplicateSubscriptions ensures only one active subscription per user
func (a *PocketBaseAdapter) CleanupDuplicateSubscriptions(userID string) error {
	records, err := a.app.FindRecordsByFilter(
		"current_user_subscriptions",
		"user_id = {:user_id} && status = 'active'",
		"-created",
		100, 0,
		map[string]any{"user_id": userID},
	)
	if err != nil {
		return err
	}

	// Keep the most recent, cancel the rest
	for i := 1; i < len(records); i++ {
		record := records[i]
		record.Set("status", "cancelled")
		record.Set("canceled_at", time.Now())
		if err := a.app.Save(record); err != nil {
			return fmt.Errorf("failed to cleanup duplicate subscription %s: %w", record.Id, err)
		}
	}
	return nil
}

// Helper methods for mapping between domain objects and PocketBase records

func (a *PocketBaseAdapter) mapSubscriptionToRecord(sub *Subscription, record *core.Record) {
	record.Set("user_id", sub.UserID)
	record.Set("plan_id", sub.PlanID)
	record.Set("provider_subscription_id", sub.ProviderSubscriptionID)
	record.Set("provider_price_id", sub.ProviderPriceID)
	record.Set("payment_provider", sub.PaymentProvider)
	record.Set("status", sub.Status)
	record.Set("current_period_start", sub.CurrentPeriodStart)
	record.Set("current_period_end", sub.CurrentPeriodEnd)
	record.Set("cancel_at_period_end", sub.CancelAtPeriodEnd)
	
	if sub.CanceledAt != nil {
		record.Set("canceled_at", *sub.CanceledAt)
	}
	if sub.TrialEnd != nil {
		record.Set("trial_end", *sub.TrialEnd)
	}
	
	record.Set("pending_plan_id", sub.PendingPlanID)
	if sub.PendingChangeEffectiveDate != nil {
		record.Set("pending_change_effective_date", *sub.PendingChangeEffectiveDate)
	}
	record.Set("pending_change_reason", sub.PendingChangeReason)
}

func (a *PocketBaseAdapter) mapRecordToSubscription(record *core.Record) *Subscription {
	sub := &Subscription{
		ID:                       record.Id,
		UserID:                   record.GetString("user_id"),
		PlanID:                   record.GetString("plan_id"),
		ProviderSubscriptionID:   record.GetString("provider_subscription_id"),
		ProviderPriceID:          record.GetString("provider_price_id"),
		PaymentProvider:          record.GetString("payment_provider"),
		Status:                   record.GetString("status"),
		CurrentPeriodStart:       record.GetDateTime("current_period_start").Time(),
		CurrentPeriodEnd:         record.GetDateTime("current_period_end").Time(),
		CancelAtPeriodEnd:        record.GetBool("cancel_at_period_end"),
		PendingPlanID:            record.GetString("pending_plan_id"),
		PendingChangeReason:      record.GetString("pending_change_reason"),
	}

	if canceledAt := record.GetDateTime("canceled_at"); !canceledAt.IsZero() {
		t := canceledAt.Time()
		sub.CanceledAt = &t
	}
	if trialEnd := record.GetDateTime("trial_end"); !trialEnd.IsZero() {
		t := trialEnd.Time()
		sub.TrialEnd = &t
	}
	if pendingDate := record.GetDateTime("pending_change_effective_date"); !pendingDate.IsZero() {
		t := pendingDate.Time()
		sub.PendingChangeEffectiveDate = &t
	}

	return sub
}

func (a *PocketBaseAdapter) mapRecordToPlan(record *core.Record) *Plan {
	return &Plan{
		ID:               record.Id,
		Name:             record.GetString("name"),
		ProviderPriceID:  record.GetString("provider_price_id"),
		BillingInterval:  record.GetString("billing_interval"),
		HoursPerMonth:    record.GetFloat("hours_per_month"),
		PriceCents:       record.GetInt("price_cents"),
		DisplayOrder:     record.GetInt("display_order"),
		IsActive:         record.GetBool("is_active"),
		IsFree:           record.GetString("billing_interval") == "free",
	}
}