package subscription

import (
	"errors"
	"fmt"
	"time"
)

// TestStore is a simple in-memory implementation for testing
type TestStore struct {
	subscriptions map[string]*Subscription
	plans        map[string]*Plan
	plansByPrice map[string]*Plan
	history      []*Subscription
	idCounter    int
}

// NewTestStore creates a new test store with some default data
func NewTestStore() *TestStore {
	store := &TestStore{
		subscriptions: make(map[string]*Subscription),
		plans:        make(map[string]*Plan),
		plansByPrice: make(map[string]*Plan),
		history:      []*Subscription{},
		idCounter:    1,
	}
	
	// Add default plans
	store.setupDefaultPlans()
	return store
}

func (t *TestStore) setupDefaultPlans() {
	// Free plan
	freePlan := &Plan{
		ID:              "free_plan",
		Name:            "Free Plan",
		ProviderPriceID: "",
		BillingInterval: "free",
		HoursPerMonth:   5.0,
		PriceCents:      0,
		DisplayOrder:    1,
		IsActive:        true,
		IsFree:          true,
	}
	t.plans[freePlan.ID] = freePlan
	
	// Basic plan
	basicPlan := &Plan{
		ID:              "basic_plan",
		Name:            "Basic Plan",
		ProviderPriceID: "price_basic",
		BillingInterval: "monthly",
		HoursPerMonth:   50.0,
		PriceCents:      999,
		DisplayOrder:    2,
		IsActive:        true,
		IsFree:          false,
	}
	t.plans[basicPlan.ID] = basicPlan
	t.plansByPrice[basicPlan.ProviderPriceID] = basicPlan
	
	// Pro plan
	proPlan := &Plan{
		ID:              "pro_plan",
		Name:            "Pro Plan", 
		ProviderPriceID: "price_pro",
		BillingInterval: "monthly",
		HoursPerMonth:   200.0,
		PriceCents:      2999,
		DisplayOrder:    3,
		IsActive:        true,
		IsFree:          false,
	}
	t.plans[proPlan.ID] = proPlan
	t.plansByPrice[proPlan.ProviderPriceID] = proPlan
}

func (t *TestStore) nextID() string {
	t.idCounter++
	return fmt.Sprintf("test_id_%d", t.idCounter)
}

// Subscription operations
func (t *TestStore) CreateSubscription(sub *Subscription) error {
	if sub.ID == "" {
		sub.ID = t.nextID()
	}
	t.subscriptions[sub.ID] = sub
	return nil
}

func (t *TestStore) UpdateSubscription(sub *Subscription) error {
	if _, exists := t.subscriptions[sub.ID]; !exists {
		return errors.New("subscription not found")
	}
	t.subscriptions[sub.ID] = sub
	return nil
}

func (t *TestStore) GetSubscription(id string) (*Subscription, error) {
	sub, exists := t.subscriptions[id]
	if !exists {
		return nil, errors.New("subscription not found")
	}
	return sub, nil
}

func (t *TestStore) DeleteSubscription(id string) error {
	delete(t.subscriptions, id)
	return nil
}

// Query operations
func (t *TestStore) GetActiveSubscription(userID string) (*Subscription, error) {
	for _, sub := range t.subscriptions {
		if sub.UserID == userID && sub.Status == "active" {
			return sub, nil
		}
	}
	return nil, errors.New("no active subscription found")
}

func (t *TestStore) GetSubscriptionByProviderID(providerSubID string) (*Subscription, error) {
	for _, sub := range t.subscriptions {
		if sub.ProviderSubscriptionID == providerSubID {
			return sub, nil
		}
	}
	return nil, errors.New("subscription not found")
}

func (t *TestStore) GetAllUserSubscriptions(userID string) ([]*Subscription, error) {
	var result []*Subscription
	for _, sub := range t.subscriptions {
		if sub.UserID == userID {
			result = append(result, sub)
		}
	}
	return result, nil
}

func (t *TestStore) GetActiveSubscriptionsCount(userID string) (int, error) {
	count := 0
	for _, sub := range t.subscriptions {
		if sub.UserID == userID && sub.Status == "active" {
			count++
		}
	}
	return count, nil
}

// Plan operations
func (t *TestStore) GetPlan(planID string) (*Plan, error) {
	plan, exists := t.plans[planID]
	if !exists {
		return nil, errors.New("plan not found")
	}
	return plan, nil
}

func (t *TestStore) GetPlanByProviderPrice(providerPriceID string) (*Plan, error) {
	plan, exists := t.plansByPrice[providerPriceID]
	if !exists {
		return nil, errors.New("plan not found for price")
	}
	return plan, nil
}

func (t *TestStore) GetFreePlan() (*Plan, error) {
	for _, plan := range t.plans {
		if plan.IsFree {
			return plan, nil
		}
	}
	return nil, errors.New("free plan not found")
}

func (t *TestStore) GetAllPlans() ([]*Plan, error) {
	var result []*Plan
	for _, plan := range t.plans {
		if plan.IsActive {
			result = append(result, plan)
		}
	}
	return result, nil
}

func (t *TestStore) GetAvailableUpgrades(currentPlanID string) ([]*Plan, error) {
	currentPlan, err := t.GetPlan(currentPlanID)
	if err != nil {
		return nil, err
	}
	
	var result []*Plan
	for _, plan := range t.plans {
		if plan.IsActive && plan.HoursPerMonth > currentPlan.HoursPerMonth {
			result = append(result, plan)
		}
	}
	return result, nil
}

// History operations
func (t *TestStore) SaveToHistory(sub *Subscription, reason string) error {
	// Create a copy for history
	historySub := *sub
	historySub.ID = "history_" + sub.ID
	historySub.PendingChangeReason = reason
	
	t.history = append(t.history, &historySub)
	return nil
}

func (t *TestStore) GetHistory(userID string) ([]*Subscription, error) {
	var result []*Subscription
	for _, sub := range t.history {
		if sub.UserID == userID {
			result = append(result, sub)
		}
	}
	return result, nil
}

// Bulk operations
func (t *TestStore) DeactivateAllUserSubscriptions(userID string) error {
	for _, sub := range t.subscriptions {
		if sub.UserID == userID && sub.Status == "active" {
			sub.Status = "cancelled"
			sub.CanceledAt = &time.Time{}
			*sub.CanceledAt = time.Now()
		}
	}
	return nil
}

func (t *TestStore) CleanupDuplicateSubscriptions(userID string) error {
	var activeCount int
	var lastActiveSub *Subscription
	
	for _, sub := range t.subscriptions {
		if sub.UserID == userID && sub.Status == "active" {
			activeCount++
			lastActiveSub = sub
		}
	}
	
	// If more than one active, keep the last one and cancel the rest
	if activeCount > 1 {
		for _, sub := range t.subscriptions {
			if sub.UserID == userID && sub.Status == "active" && sub != lastActiveSub {
				sub.Status = "cancelled"
				sub.CanceledAt = &time.Time{}
				*sub.CanceledAt = time.Now()
			}
		}
	}
	
	return nil
}

// Helper methods for testing
func (t *TestStore) AddSubscription(sub *Subscription) {
	if sub.ID == "" {
		sub.ID = t.nextID()
	}
	t.subscriptions[sub.ID] = sub
}

func (t *TestStore) GetHistoryCount() int {
	return len(t.history)
}

func (t *TestStore) GetLastHistoryEntry() *Subscription {
	if len(t.history) == 0 {
		return nil
	}
	return t.history[len(t.history)-1]
}