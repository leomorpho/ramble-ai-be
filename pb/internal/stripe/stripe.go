// Package stripe provides Stripe payment processing integration for PocketBase.
//
// This package includes:
// - Webhook handlers for processing Stripe events
// - API endpoints for creating checkout sessions and billing portal links
// - Helper functions for customer and subscription management
//
// The package automatically syncs Stripe data (products, prices, customers, subscriptions)
// with PocketBase collections via webhook events.
package stripe