package payment

import (
	"io"
	"log"
	"net/http"
	"time"

	"pocketbase/internal/subscription"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// HandleWebhook processes payment provider webhooks and routes them to the subscription service
func (s *Service) HandleWebhook(e *core.RequestEvent, app *pocketbase.PocketBase) error {
	// Read the request body
	payload, err := io.ReadAll(e.Request.Body)
	if err != nil {
		log.Printf("Error reading webhook payload: %v", err)
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to read request body"})
	}

	// Get webhook signature from headers
	signature := e.Request.Header.Get("Stripe-Signature")
	if signature == "" {
		log.Printf("Missing webhook signature")
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Missing webhook signature"})
	}

	// Parse webhook event using the payment provider
	webhookEvent, err := s.ParseWebhookEvent(payload, signature)
	if err != nil {
		log.Printf("Webhook signature verification failed: %v", err)
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	log.Printf("Processing webhook event: %s (ID: %s)", webhookEvent.Type, webhookEvent.ID)

	// Create subscription service to handle the business logic
	repo := subscription.NewRepository(app)
	subscriptionService := subscription.NewService(repo)

	// Route webhook events to appropriate handlers
	switch webhookEvent.Type {
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		if webhookEvent.Data.Subscription == nil {
			log.Printf("No subscription data in webhook")
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Missing subscription data"})
		}
		
		// Convert payment.Subscription back to webhook event data format for subscription service
		eventData := subscription.WebhookEventData{
			EventType: webhookEvent.Type,
			// TODO: Fix this conversion when implementing proper webhook parsing
			// Subscription: convertPaymentSubscriptionToStripe(webhookEvent.Data.Subscription),
		}
		
		if err := subscriptionService.ProcessWebhookEvent(eventData); err != nil {
			log.Printf("Error processing subscription webhook: %v", err)
			// Don't return error to Stripe - we've received the event
		}

	case "invoice.payment_succeeded", "invoice.payment_failed":
		if webhookEvent.Data.Invoice == nil {
			log.Printf("No invoice data in webhook")
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Missing invoice data"})
		}
		
		// Handle invoice events
		eventData := subscription.WebhookEventData{
			EventType: webhookEvent.Type,
			// TODO: Fix this conversion when implementing proper webhook parsing
			// Invoice: convertPaymentInvoiceToStripe(webhookEvent.Data.Invoice),
		}
		
		if err := subscriptionService.ProcessWebhookEvent(eventData); err != nil {
			log.Printf("Error processing invoice webhook: %v", err)
			// Don't return error to Stripe - we've received the event
		}

	case "checkout.session.completed":
		// Log but don't process - wait for payment confirmation via subscription events
		log.Printf("Checkout session completed: %s", webhookEvent.Data.CheckoutSession.ID)

	default:
		log.Printf("Unhandled webhook event type: %s", webhookEvent.Type)
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// Helper function to convert payment.Subscription to stripe.Subscription format expected by subscription service
// This is a temporary bridge until we refactor the subscription service to use payment types
func convertPaymentSubscriptionToStripe(sub *Subscription) interface{} {
	// This would need to return a stripe.Subscription-like object
	// For now, we'll need to create a simple adapter
	return map[string]interface{}{
		"id":                   sub.ID,
		"customer":             map[string]interface{}{"id": sub.CustomerID},
		"status":               string(sub.Status),
		"current_period_start": sub.CurrentPeriodStart.Unix(),
		"current_period_end":   sub.CurrentPeriodEnd.Unix(),
		"cancel_at_period_end": sub.CancelAtPeriodEnd,
		"canceled_at":          getUnixTime(sub.CanceledAt),
		"trial_end":            getUnixTime(sub.TrialEnd),
		"items": map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"price": map[string]interface{}{"id": sub.PriceID},
				},
			},
		},
		"metadata": sub.Metadata,
	}
}

// Helper function to convert payment.Invoice to stripe.Invoice format
func convertPaymentInvoiceToStripe(invoice *Invoice) interface{} {
	return map[string]interface{}{
		"id":       invoice.ID,
		"customer": map[string]interface{}{"id": invoice.CustomerID},
		"subscription": map[string]interface{}{
			"id": getStringValue(invoice.SubscriptionID),
		},
		"status":   invoice.Status,
		"total":    invoice.Total,
		"currency": invoice.Currency,
		"metadata": invoice.Metadata,
	}
}

// Helper functions
func getUnixTime(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.Unix()
}

func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}