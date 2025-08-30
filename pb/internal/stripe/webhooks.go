package stripe

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/subscription"
	"github.com/stripe/stripe-go/v79/webhook"

	subscriptionService "pocketbase/internal/subscription"
)

// HandleWebhook processes Stripe webhook events
func HandleWebhook(e *core.RequestEvent, app *pocketbase.PocketBase) error {
	const MaxBodyBytes = int64(65536)
	e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, MaxBodyBytes)

	payload, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
	}

	// Verify webhook signature
	endpointSecret := os.Getenv("STRIPE_SECRET_WHSEC")
	
	// Debug logging for environment variable
	if endpointSecret == "" {
		log.Printf("[WEBHOOK_DEBUG] STRIPE_SECRET_WHSEC environment variable is empty or not set")
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "webhook secret not configured"})
	} else {
		// Log first and last 6 characters for verification (security safe)
		secretLen := len(endpointSecret)
		if secretLen > 12 {
			log.Printf("[WEBHOOK_DEBUG] STRIPE_SECRET_WHSEC loaded: %s...%s (length: %d)", 
				endpointSecret[:6], endpointSecret[secretLen-6:], secretLen)
		} else {
			log.Printf("[WEBHOOK_DEBUG] STRIPE_SECRET_WHSEC loaded (length: %d)", secretLen)
		}
	}
	
	event, err := webhook.ConstructEventWithOptions(payload, e.Request.Header.Get("Stripe-Signature"), endpointSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		log.Printf("[WEBHOOK_DEBUG] Signature verification failed: %v", err)
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Create subscription service
	repo := subscriptionService.NewRepository(app)
	service := subscriptionService.NewService(repo)

	// Handle the event using the subscription service
	// IMPORTANT: When adding/removing webhook events, update README.md payment provider section
	switch event.Type {
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if err := service.HandleSubscriptionEvent(&sub, string(event.Type)); err != nil {
			log.Printf("Error handling subscription event: %v", err)
		}

	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		// Log checkout completion but don't update subscriptions yet
		// Wait for invoice.payment_succeeded to confirm payment before updating user plans
		log.Printf("Checkout session completed: %s (subscription: %v)", session.ID, session.Subscription != nil)

	case "invoice.payment_failed":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("Error parsing invoice webhook JSON: %v", err)
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if err := service.HandlePaymentFailed(&invoice); err != nil {
			log.Printf("Error handling payment failure: %v", err)
		}

	case "invoice.payment_succeeded", "invoice.payment.paid", "invoice_payment.paid":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("Error parsing invoice webhook JSON: %v", err)
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if err := handlePaymentSucceeded(app, &invoice); err != nil {
			log.Printf("Error handling payment success: %v", err)
		}

	default:
		log.Printf("Unhandled event type: %s", event.Type)
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleSubscriptionEvent processes Stripe subscription lifecycle events
func handleSubscriptionEvent(app *pocketbase.PocketBase, stripeSub *stripe.Subscription, eventType string) error {
	log.Printf("Processing subscription event: %s for subscription %s", eventType, stripeSub.ID)

	// Get user ID from customer
	userID, err := getUserIDFromCustomer(app, stripeSub.Customer.ID)
	if err != nil {
		return err
	}

	// Find the subscription plan that matches this Stripe price
	var planID string
	if stripeSub.Items != nil && len(stripeSub.Items.Data) > 0 {
		stripePriceID := stripeSub.Items.Data[0].Price.ID
		plan, err := app.FindFirstRecordByFilter("subscription_plans", "provider_price_id = {:price_id}", map[string]any{
			"price_id": stripePriceID,
		})
		if err != nil {
			return fmt.Errorf("failed to find subscription plan for price %s: %w", stripePriceID, err)
		}
		planID = plan.Id
	}

	// Handle deletion separately
	if eventType == "customer.subscription.deleted" {
		return handleSubscriptionCancellation(app, userID, stripeSub)
	}

	// Get or create subscription record
	collection, err := app.FindCollectionByNameOrId("user_subscriptions")
	if err != nil {
		return err
	}

	// First, deactivate any existing active subscriptions for this user
	existingActive, _ := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id} AND status = 'active'", map[string]any{
		"user_id": userID,
	})
	
	// Get current Stripe price ID for comparison
	var currentStripePriceID string
	if stripeSub.Items != nil && len(stripeSub.Items.Data) > 0 {
		currentStripePriceID = stripeSub.Items.Data[0].Price.ID
	}
	
	if existingActive != nil {
		existingStripePriceID := existingActive.GetString("provider_price_id")
		existingStripeSubID := existingActive.GetString("provider_subscription_id")
		
		// Detect plan changes: either different subscription ID OR different price ID within same subscription
		isPlanChange := (existingStripeSubID != stripeSub.ID) || 
						(existingStripeSubID == stripeSub.ID && existingStripePriceID != currentStripePriceID)
		
		if isPlanChange {
			// User is switching plans - mark old subscription as cancelled
			existingActive.Set("status", "cancelled")
			existingActive.Set("canceled_at", time.Now())
			if err := app.Save(existingActive); err != nil {
				log.Printf("Failed to deactivate old subscription: %v", err)
			}
			log.Printf("Deactivated old subscription for user %s (plan change detected: %s -> %s)", 
				userID, existingStripePriceID, currentStripePriceID)
		}
	}

	// SINGLE SUBSCRIPTION APPROACH: Find current active subscription for the user
	// instead of looking for specific Stripe subscription ID
	record, err := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id} AND status = 'active'", map[string]any{
		"user_id": userID,
	})

	if err != nil {
		// No active subscription found - create new record
		record = core.NewRecord(collection)
		log.Printf("Creating new subscription record for user %s with Stripe ID %s", userID, stripeSub.ID)
	} else {
		// Found existing active subscription
		existingStripeSubID := record.GetString("provider_subscription_id")
		
		// Check if this is an update to the same subscription or a new one
		if existingStripeSubID == stripeSub.ID {
			log.Printf("Updating existing subscription record for user %s with Stripe ID %s", userID, stripeSub.ID)
		} else if existingStripeSubID != "" && existingStripeSubID != stripeSub.ID {
			// This is a different Stripe subscription - this should not happen with single subscription approach
			log.Printf("Warning: User %s has different Stripe subscription (%s vs %s) - updating to new one", 
				userID, existingStripeSubID, stripeSub.ID)
		}
		
		// Special handling for cancel_at_period_end updates
		if stripeSub.CancelAtPeriodEnd {
			log.Printf("Subscription %s has cancel_at_period_end=true - preserving current plan until period end", stripeSub.ID)
			// Don't change plan_id or provider_price_id when cancel_at_period_end is true
			// This preserves user's current benefits until period actually ends
		}
	}

	// Update record fields
	record.Set("user_id", userID)
	
	// CRITICAL: Only update plan/price when NOT in cancel_at_period_end state
	// This preserves user's current benefits until Stripe actually cancels the subscription
	if !stripeSub.CancelAtPeriodEnd {
		if planID != "" {
			record.Set("plan_id", planID)
		}
		if currentStripePriceID != "" {
			record.Set("provider_price_id", currentStripePriceID)
		}
	} else {
		log.Printf("Preserving current plan for user %s - subscription has cancel_at_period_end=true", userID)
	}
	
	record.Set("provider_subscription_id", stripeSub.ID)
	record.Set("status", mapStripeStatus(stripeSub.Status))
	
	// Log and set period dates with validation
	if stripeSub.CurrentPeriodStart > 0 {
		record.Set("current_period_start", time.Unix(stripeSub.CurrentPeriodStart, 0))
	} else {
		log.Printf("Warning: CurrentPeriodStart is 0 for subscription %s", stripeSub.ID)
		record.Set("current_period_start", time.Now())
	}
	
	if stripeSub.CurrentPeriodEnd > 0 {
		record.Set("current_period_end", time.Unix(stripeSub.CurrentPeriodEnd, 0))
	} else {
		log.Printf("Warning: CurrentPeriodEnd is 0 for subscription %s", stripeSub.ID)
		// Default to 30 days from now for monthly subscriptions
		record.Set("current_period_end", time.Now().AddDate(0, 1, 0))
	}
	
	record.Set("cancel_at_period_end", stripeSub.CancelAtPeriodEnd)

	if stripeSub.CanceledAt > 0 {
		record.Set("canceled_at", time.Unix(stripeSub.CanceledAt, 0))
	}
	if stripeSub.TrialEnd > 0 {
		record.Set("trial_end", time.Unix(stripeSub.TrialEnd, 0))
	}

	return app.Save(record)
}

// handleSubscriptionCancellation handles subscription deletion with pending plan support
func handleSubscriptionCancellation(app *pocketbase.PocketBase, userID string, stripeSub *stripe.Subscription) error {
	log.Printf("Handling subscription cancellation for user %s", userID)

	// Find user's current subscription record
	record, err := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id} AND provider_subscription_id = {:sub_id}", map[string]any{
		"user_id": userID,
		"sub_id": stripeSub.ID,
	})

	if err != nil {
		log.Printf("No subscription record found for user %s, subscription %s", userID, stripeSub.ID)
		return fmt.Errorf("subscription record not found: %w", err)
	}

	// CRITICAL: Check if there's a pending plan change
	pendingPlanID := record.GetString("pending_plan_id")
	
	var targetPlan *core.Record
	
	if pendingPlanID != "" {
		// User has a pending plan change - use that plan
		log.Printf("Processing pending plan change for user %s: switching to plan %s", userID, pendingPlanID)
		
		targetPlan, err = app.FindFirstRecordByFilter("subscription_plans", "id = {:plan_id}", map[string]any{
			"plan_id": pendingPlanID,
		})
		if err != nil {
			log.Printf("Warning: Failed to find pending target plan %s for user %s, defaulting to free", pendingPlanID, userID)
			targetPlan = nil // Will fall back to free plan below
		} else {
			log.Printf("Successfully found pending target plan: %s (%s)", targetPlan.GetString("name"), targetPlan.Id)
		}
	}
	
	// If no pending plan or pending plan not found, use free plan
	if targetPlan == nil {
		log.Printf("No pending plan for user %s, moving to free plan", userID)
		targetPlan, err = app.FindFirstRecordByFilter("subscription_plans", "billing_interval = 'free'", map[string]any{})
		if err != nil {
			return fmt.Errorf("failed to find free plan: %w", err)
		}
	}

	// Update the EXISTING subscription record (single subscription approach)
	now := time.Now()
	oneYearFromNow := now.AddDate(1, 0, 0)

	record.Set("plan_id", targetPlan.Id)
	record.Set("provider_subscription_id", "") // Remove Stripe subscription ID
	record.Set("provider_price_id", "")        // Clear Stripe price ID
	record.Set("status", "active")
	record.Set("current_period_start", now)
	record.Set("current_period_end", oneYearFromNow)
	record.Set("cancel_at_period_end", false)
	record.Set("canceled_at", time.Unix(stripeSub.CanceledAt, 0))
	
	// Clear pending plan fields since the change has now been applied
	record.Set("pending_plan_id", "")
	record.Set("pending_change_effective_date", "")
	
	log.Printf("Updated subscription for user %s to plan %s (%s)", userID, targetPlan.GetString("name"), targetPlan.Id)
	
	return app.Save(record)
}

// handlePaymentFailed handles failed payment events
func handlePaymentFailed(app *pocketbase.PocketBase, invoice *stripe.Invoice) error {
	if invoice.Subscription == nil {
		return nil // Not a subscription invoice
	}

	// Get user ID from customer
	userID, err := getUserIDFromCustomer(app, invoice.Customer.ID)
	if err != nil {
		return err
	}

	// Update subscription status to past_due
	record, err := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id} AND provider_subscription_id = {:sub_id}", map[string]any{
		"user_id": userID,
		"sub_id": invoice.Subscription.ID,
	})

	if err != nil {
		return err
	}

	record.Set("status", "past_due")
	return app.Save(record)
}

// handlePaymentSucceeded handles successful payment events
func handlePaymentSucceeded(app *pocketbase.PocketBase, invoice *stripe.Invoice) error {
	if invoice.Subscription == nil {
		return nil // Not a subscription invoice
	}

	log.Printf("Payment succeeded for subscription: %s", invoice.Subscription.ID)

	// Get the full subscription details from Stripe
	sub, err := subscription.Get(invoice.Subscription.ID, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve subscription: %w", err)
	}

	// Check if subscription already exists in our database
	existingRecord, err := app.FindFirstRecordByFilter("user_subscriptions", "provider_subscription_id = {:sub_id}", map[string]any{
		"sub_id": sub.ID,
	})

	// If subscription exists, this is an update (plan change), otherwise it's creation
	eventType := "customer.subscription.created"
	if existingRecord != nil {
		eventType = "customer.subscription.updated"
	}

	log.Printf("Processing subscription %s as %s event", sub.ID, eventType)
	return handleSubscriptionEvent(app, sub, eventType)
}

// mapStripeStatus maps Stripe subscription statuses to our internal statuses
func mapStripeStatus(stripeStatus stripe.SubscriptionStatus) string {
	switch stripeStatus {
	case stripe.SubscriptionStatusActive:
		return "active"
	case stripe.SubscriptionStatusCanceled:
		return "cancelled"
	case stripe.SubscriptionStatusPastDue:
		return "past_due"
	case stripe.SubscriptionStatusTrialing:
		return "trialing"
	default:
		return "active" // Default fallback
	}
}