package stripe

import (
	"testing"
	"time"

	"github.com/stripe/stripe-go/v79"
)

// TestMapStripeStatus tests the Stripe status mapping function
func TestMapStripeStatus(t *testing.T) {
	tests := []struct {
		input    stripe.SubscriptionStatus
		expected string
	}{
		{stripe.SubscriptionStatusActive, "active"},
		{stripe.SubscriptionStatusCanceled, "cancelled"},
		{stripe.SubscriptionStatusPastDue, "past_due"},
		{stripe.SubscriptionStatusTrialing, "trialing"},
		{stripe.SubscriptionStatusIncompleteExpired, "active"}, // Default fallback
		{stripe.SubscriptionStatusUnpaid, "active"}, // Default fallback  
		{stripe.SubscriptionStatusPaused, "active"}, // Default fallback
	}

	for _, test := range tests {
		result := mapStripeStatus(test.input)
		if result != test.expected {
			t.Errorf("mapStripeStatus(%v) = %s, expected %s", 
				test.input, result, test.expected)
		}
	}
}

// TestValidateTimestamps tests the timestamp validation logic
func TestValidateTimestamps(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name        string
		start       int64
		end         int64
		expectValid bool
	}{
		{
			name:        "valid timestamps",
			start:       now.Unix(),
			end:         now.Add(time.Hour * 24 * 30).Unix(), // 30 days later
			expectValid: true,
		},
		{
			name:        "zero start timestamp (1970)",
			start:       0,
			end:         now.Add(time.Hour * 24 * 30).Unix(),
			expectValid: false,
		},
		{
			name:        "zero end timestamp (1970)",
			start:       now.Unix(),
			end:         0,
			expectValid: false,
		},
		{
			name:        "both zero timestamps",
			start:       0,
			end:         0,
			expectValid: false,
		},
		{
			name:        "end before start",
			start:       now.Unix(),
			end:         now.Add(-time.Hour).Unix(), // 1 hour ago
			expectValid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			startTime, endTime, valid := validateAndFixTimestamps(test.start, test.end)
			
			if valid != test.expectValid {
				t.Errorf("validateAndFixTimestamps() valid = %v, expected %v", valid, test.expectValid)
			}
			
			if test.expectValid {
				// For valid timestamps, check they're reasonable
				if startTime.Year() < 2000 {
					t.Errorf("Start time should not be in 1970s: %v", startTime)
				}
				if endTime.Year() < 2000 {
					t.Errorf("End time should not be in 1970s: %v", endTime)
				}
				if !endTime.After(startTime) {
					t.Errorf("End time should be after start time: start=%v, end=%v", startTime, endTime)
				}
			}
		})
	}
}

// TestExtractPriceFromSubscription tests extracting price from Stripe subscription
func TestExtractPriceFromSubscription(t *testing.T) {
	tests := []struct {
		name         string
		subscription *stripe.Subscription
		expected     string
		expectError  bool
	}{
		{
			name: "subscription with items",
			subscription: &stripe.Subscription{
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{Price: &stripe.Price{ID: "price_123"}},
					},
				},
			},
			expected:    "price_123",
			expectError: false,
		},
		{
			name: "subscription with no items",
			subscription: &stripe.Subscription{
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{},
				},
			},
			expected:    "",
			expectError: true,
		},
		{
			name: "subscription with nil items",
			subscription: &stripe.Subscription{
				Items: nil,
			},
			expected:    "",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := extractPriceFromSubscription(test.subscription)
			
			if test.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != test.expected {
					t.Errorf("extractPriceFromSubscription() = %s, expected %s", result, test.expected)
				}
			}
		})
	}
}