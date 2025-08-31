package payment

import (
	"io"
	"log"
	"net/http"
	"time"

	"pocketbase/internal/subscription"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v79"
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
	case "customer.created":
		// Customer creation is handled automatically by payment service
		// This webhook is mostly for logging and potential future processing
		if webhookEvent.Data.Customer != nil {
			log.Printf("Customer created: %s", webhookEvent.Data.Customer.ID)
		} else {
			log.Printf("Customer created but no customer data provided")
		}
		
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		if webhookEvent.Data.Subscription == nil {
			log.Printf("No subscription data in webhook")
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Missing subscription data"})
		}
		
		// Convert payment.Subscription back to webhook event data format for subscription service
		eventData := subscription.WebhookEventData{
			EventType:    webhookEvent.Type,
			Subscription: convertPaymentSubscriptionToStripe(webhookEvent.Data.Subscription),
		}
		
		// Add customer data if available
		if webhookEvent.Data.Customer != nil {
			eventData.Customer = convertPaymentCustomerToStripe(webhookEvent.Data.Customer)
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
			Invoice:   convertPaymentInvoiceToStripe(webhookEvent.Data.Invoice),
		}
		
		if err := subscriptionService.ProcessWebhookEvent(eventData); err != nil {
			log.Printf("Error processing invoice webhook: %v", err)
			// Don't return error to Stripe - we've received the event
		}

	case "checkout.session.completed":
		// Process checkout session completion - this often triggers subscription creation
		if webhookEvent.Data.CheckoutSession != nil {
			log.Printf("Checkout session completed: %s", webhookEvent.Data.CheckoutSession.ID)
			
			// Send checkout session data to subscription service for processing
			eventData := subscription.WebhookEventData{
				EventType:       webhookEvent.Type,
				CheckoutSession: convertPaymentCheckoutSessionToStripe(webhookEvent.Data.CheckoutSession),
			}
			
			if err := subscriptionService.ProcessWebhookEvent(eventData); err != nil {
				log.Printf("Error processing checkout session webhook: %v", err)
				// Don't return error to Stripe - we've received the event
			}
		} else {
			log.Printf("Checkout session completed but no session data provided")
		}

	default:
		log.Printf("Unhandled webhook event type: %s", webhookEvent.Type)
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// Helper function to convert payment.Subscription to stripe.Subscription format expected by subscription service
// This is a temporary bridge until we refactor the subscription service to use payment types
func convertPaymentSubscriptionToStripe(sub *Subscription) *stripe.Subscription {
	stripeSub := &stripe.Subscription{
		ID:                 sub.ID,
		Customer:           &stripe.Customer{ID: sub.CustomerID},
		Status:             convertToStripeStatus(sub.Status),
		CurrentPeriodStart: sub.CurrentPeriodStart.Unix(),
		CurrentPeriodEnd:   sub.CurrentPeriodEnd.Unix(),
		CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
		Metadata:           sub.Metadata,
	}

	// Handle optional fields
	if sub.CanceledAt != nil {
		stripeSub.CanceledAt = sub.CanceledAt.Unix()
	}
	
	if sub.TrialEnd != nil {
		stripeSub.TrialEnd = sub.TrialEnd.Unix()
	}

	// Create subscription items with price
	if sub.PriceID != "" {
		stripeSub.Items = &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{
					Price: &stripe.Price{ID: sub.PriceID},
				},
			},
		}
	}

	return stripeSub
}

// Helper function to convert payment.Invoice to stripe.Invoice format
func convertPaymentInvoiceToStripe(invoice *Invoice) *stripe.Invoice {
	stripeInvoice := &stripe.Invoice{
		ID:       invoice.ID,
		Customer: &stripe.Customer{ID: invoice.CustomerID},
		Status:   stripe.InvoiceStatus(invoice.Status),
		Total:    invoice.Total,
		Currency: stripe.Currency(invoice.Currency),
		Metadata: invoice.Metadata,
	}

	if invoice.SubscriptionID != nil {
		stripeInvoice.Subscription = &stripe.Subscription{ID: *invoice.SubscriptionID}
	}

	if invoice.PaidAt != nil {
		stripeInvoice.StatusTransitions = &stripe.InvoiceStatusTransitions{
			PaidAt: invoice.PaidAt.Unix(),
		}
	}

	return stripeInvoice
}

// Helper function to convert payment.SubscriptionStatus to stripe.SubscriptionStatus
func convertToStripeStatus(status SubscriptionStatus) stripe.SubscriptionStatus {
	switch status {
	case SubscriptionStatusActive:
		return stripe.SubscriptionStatusActive
	case SubscriptionStatusCanceled:
		return stripe.SubscriptionStatusCanceled
	case SubscriptionStatusIncomplete:
		return stripe.SubscriptionStatusIncomplete
	case SubscriptionStatusIncompleteExpired:
		return stripe.SubscriptionStatusIncompleteExpired
	case SubscriptionStatusPastDue:
		return stripe.SubscriptionStatusPastDue
	case SubscriptionStatusTrialing:
		return stripe.SubscriptionStatusTrialing
	case SubscriptionStatusUnpaid:
		return stripe.SubscriptionStatusUnpaid
	default:
		return stripe.SubscriptionStatusActive
	}
}

// Helper function to convert payment.Customer to stripe.Customer format
func convertPaymentCustomerToStripe(customer *Customer) *stripe.Customer {
	return &stripe.Customer{
		ID:       customer.ID,
		Email:    customer.Email,
		Name:     customer.Name,
		Metadata: customer.Metadata,
	}
}

// Helper function to convert payment.CheckoutSession to stripe.CheckoutSession format
func convertPaymentCheckoutSessionToStripe(session *CheckoutSession) *stripe.CheckoutSession {
	return &stripe.CheckoutSession{
		ID:       session.ID,
		URL:      session.URL,
		Customer: &stripe.Customer{ID: session.CustomerID},
		Status:   stripe.CheckoutSessionStatus(session.Status),
		Metadata: session.Metadata,
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