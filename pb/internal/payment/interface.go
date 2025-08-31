package payment

import (
	"time"
)

// Provider represents a payment service provider (Stripe, Paddle, Polar.sh, etc.)
type Provider interface {
	// Checkout operations
	CreateCheckoutSession(params CheckoutSessionParams) (*CheckoutSession, error)
	CreateBillingPortalLink(customerID string, returnURL string) (*PortalLink, error)
	
	// Subscription management
	ChangeSubscriptionPlan(subscriptionID string, newPriceID string, prorationBehavior string) (*Subscription, error)
	CancelSubscription(subscriptionID string, cancelAtPeriodEnd bool) (*Subscription, error)
	
	// Customer management
	CreateCustomer(params CustomerParams) (*Customer, error)
	GetCustomer(customerID string) (*Customer, error)
	HasValidPaymentMethod(customerID string) (*PaymentMethodStatus, error)
	
	// Webhook handling
	ParseWebhookEvent(payload []byte, signature string) (*WebhookEvent, error)
	
	// Provider identification
	GetProviderName() string
	GetProviderType() ProviderType
}

// ProviderType represents different payment providers
type ProviderType string

const (
	ProviderStripe   ProviderType = "stripe"
	ProviderPaddle   ProviderType = "paddle"
	ProviderPolarSh  ProviderType = "polar"
)

// CheckoutSessionParams represents parameters for creating a checkout session
type CheckoutSessionParams struct {
	CustomerID     string
	PriceID        string
	Quantity       int64
	SuccessURL     string
	CancelURL      string
	Mode           string // "subscription", "payment", "setup"
	UserID         string // For metadata
	PlanID         string // For metadata
	AllowPromoCodes bool
}

// CheckoutSession represents a payment checkout session
type CheckoutSession struct {
	ID         string
	URL        string
	CustomerID string
	Status     string
	Metadata   map[string]string
}

// PortalLink represents a billing portal/management link
type PortalLink struct {
	URL string
}

// CustomerParams represents parameters for creating a customer
type CustomerParams struct {
	Email    string
	Name     string
	UserID   string // Internal user ID for mapping
	Metadata map[string]string
}

// Customer represents a payment provider customer
type Customer struct {
	ID       string
	Email    string
	Name     string
	Created  time.Time
	Metadata map[string]string
}

// Subscription represents a subscription from the payment provider
type Subscription struct {
	ID                   string
	CustomerID           string
	Status               SubscriptionStatus
	CurrentPeriodStart   time.Time
	CurrentPeriodEnd     time.Time
	CanceledAt           *time.Time
	PriceID              string
	Metadata             map[string]string
}

// SubscriptionStatus represents subscription status across providers
type SubscriptionStatus string

const (
	SubscriptionStatusActive         SubscriptionStatus = "active"
	SubscriptionStatusCanceled       SubscriptionStatus = "canceled"
	SubscriptionStatusIncomplete     SubscriptionStatus = "incomplete"
	SubscriptionStatusIncompleteExpired SubscriptionStatus = "incomplete_expired"
	SubscriptionStatusPastDue        SubscriptionStatus = "past_due"
	SubscriptionStatusTrialing       SubscriptionStatus = "trialing"
	SubscriptionStatusUnpaid         SubscriptionStatus = "unpaid"
)

// WebhookEvent represents a webhook event from a payment provider
type WebhookEvent struct {
	ID           string
	Type         string
	Created      time.Time
	Data         WebhookEventData
	ProviderType ProviderType
}

// WebhookEventData contains the actual event data
type WebhookEventData struct {
	Subscription    *Subscription
	Invoice         *Invoice
	Customer        *Customer
	CheckoutSession *CheckoutSession
}

// Invoice represents an invoice from the payment provider
type Invoice struct {
	ID             string
	CustomerID     string
	SubscriptionID *string
	Status         string
	Total          int64
	Currency       string
	PaidAt         *time.Time
	Metadata       map[string]string
}

// PaymentMethodStatus represents the status of a customer's payment methods
type PaymentMethodStatus struct {
	HasValidPaymentMethod bool      `json:"has_valid_payment_method"`
	PaymentMethods        int       `json:"payment_methods_count"`
	DefaultPaymentMethod  *string   `json:"default_payment_method,omitempty"`
	LastUsed              *time.Time `json:"last_used,omitempty"`
	RequiresUpdate        bool      `json:"requires_update"`
	CanProcessPayments    bool      `json:"can_process_payments"`
}

// Config represents payment provider configuration
type Config struct {
	ProviderType ProviderType
	SecretKey    string
	WebhookSecret string
	PublicKey     string // For client-side usage
}

// Service handles payment operations with provider abstraction
type Service struct {
	provider Provider
	config   Config
}

// NewService creates a new payment service with the specified provider
func NewService(provider Provider, config Config) *Service {
	return &Service{
		provider: provider,
		config:   config,
	}
}

// Delegate methods to the provider
func (s *Service) CreateCheckoutSession(params CheckoutSessionParams) (*CheckoutSession, error) {
	return s.provider.CreateCheckoutSession(params)
}

func (s *Service) CreateBillingPortalLink(customerID string, returnURL string) (*PortalLink, error) {
	return s.provider.CreateBillingPortalLink(customerID, returnURL)
}

func (s *Service) ChangeSubscriptionPlan(subscriptionID string, newPriceID string, prorationBehavior string) (*Subscription, error) {
	return s.provider.ChangeSubscriptionPlan(subscriptionID, newPriceID, prorationBehavior)
}

func (s *Service) CancelSubscription(subscriptionID string, cancelAtPeriodEnd bool) (*Subscription, error) {
	return s.provider.CancelSubscription(subscriptionID, cancelAtPeriodEnd)
}

func (s *Service) CreateCustomer(params CustomerParams) (*Customer, error) {
	return s.provider.CreateCustomer(params)
}

func (s *Service) GetCustomer(customerID string) (*Customer, error) {
	return s.provider.GetCustomer(customerID)
}

func (s *Service) HasValidPaymentMethod(customerID string) (*PaymentMethodStatus, error) {
	return s.provider.HasValidPaymentMethod(customerID)
}

func (s *Service) ParseWebhookEvent(payload []byte, signature string) (*WebhookEvent, error) {
	return s.provider.ParseWebhookEvent(payload, signature)
}

func (s *Service) GetProviderName() string {
	return s.provider.GetProviderName()
}

func (s *Service) GetProviderType() ProviderType {
	return s.provider.GetProviderType()
}