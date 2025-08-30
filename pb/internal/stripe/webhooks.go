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

	// Handle the event
	switch event.Type {
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if err := handleSubscriptionEvent(app, &sub, string(event.Type)); err != nil {
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
		if err := handlePaymentFailed(app, &invoice); err != nil {
			log.Printf("Error handling payment failure: %v", err)
		}

	case "invoice.payment_succeeded":
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
		plan, err := app.FindFirstRecordByFilter("subscription_plans", "stripe_price_id = {:price_id}", map[string]any{
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

	// Try to find existing subscription for this user
	record, err := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id} AND stripe_subscription_id = {:sub_id}", map[string]any{
		"user_id": userID,
		"sub_id": stripeSub.ID,
	})

	if err != nil {
		// Create new record
		record = core.NewRecord(collection)
		log.Printf("Creating new subscription record for user %s", userID)
	} else {
		log.Printf("Updating existing subscription record for user %s", userID)
	}

	// Update record fields
	record.Set("user_id", userID)
	if planID != "" {
		record.Set("plan_id", planID)
	}
	record.Set("stripe_subscription_id", stripeSub.ID)
	record.Set("status", mapStripeStatus(stripeSub.Status))
	record.Set("current_period_start", time.Unix(stripeSub.CurrentPeriodStart, 0))
	record.Set("current_period_end", time.Unix(stripeSub.CurrentPeriodEnd, 0))
	record.Set("cancel_at_period_end", stripeSub.CancelAtPeriodEnd)

	if stripeSub.CanceledAt > 0 {
		record.Set("canceled_at", time.Unix(stripeSub.CanceledAt, 0))
	}
	if stripeSub.TrialEnd > 0 {
		record.Set("trial_end", time.Unix(stripeSub.TrialEnd, 0))
	}

	return app.Save(record)
}

// handleSubscriptionCancellation handles subscription deletion
func handleSubscriptionCancellation(app *pocketbase.PocketBase, userID string, stripeSub *stripe.Subscription) error {
	log.Printf("Handling subscription cancellation for user %s", userID)

	// Move user to free plan
	freePlan, err := app.FindFirstRecordByFilter("subscription_plans", "billing_interval = 'free'", map[string]any{})
	if err != nil {
		return fmt.Errorf("failed to find free plan: %w", err)
	}

	// Update user's subscription to free plan
	collection, err := app.FindCollectionByNameOrId("user_subscriptions")
	if err != nil {
		return err
	}

	// Find user's current subscription
	record, err := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id} AND stripe_subscription_id = {:sub_id}", map[string]any{
		"user_id": userID,
		"sub_id": stripeSub.ID,
	})

	if err != nil {
		// If no existing subscription, create a free one
		record = core.NewRecord(collection)
		record.Set("user_id", userID)
	}

	// Set to free plan
	now := time.Now()
	oneYearFromNow := now.AddDate(1, 0, 0)

	record.Set("plan_id", freePlan.Id)
	record.Set("stripe_subscription_id", nil) // Remove Stripe subscription ID
	record.Set("status", "active")
	record.Set("current_period_start", now)
	record.Set("current_period_end", oneYearFromNow)
	record.Set("cancel_at_period_end", false)
	record.Set("canceled_at", time.Unix(stripeSub.CanceledAt, 0))

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
	record, err := app.FindFirstRecordByFilter("user_subscriptions", "user_id = {:user_id} AND stripe_subscription_id = {:sub_id}", map[string]any{
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

	// Handle the subscription creation/update now that payment is confirmed
	return handleSubscriptionEvent(app, sub, "customer.subscription.created")
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