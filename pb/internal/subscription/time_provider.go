package subscription

import "time"

// TimeProvider interface allows for time manipulation in tests
type TimeProvider interface {
	Now() time.Time
}

// RealTimeProvider implements TimeProvider using real system time
type RealTimeProvider struct{}

func (r *RealTimeProvider) Now() time.Time {
	return time.Now()
}

// MockTimeProvider implements TimeProvider for testing with controllable time
type MockTimeProvider struct {
	current time.Time
}

// NewMockTimeProvider creates a new MockTimeProvider with the current time
func NewMockTimeProvider() *MockTimeProvider {
	return &MockTimeProvider{
		current: time.Now(),
	}
}

// NewMockTimeProviderAt creates a new MockTimeProvider starting at a specific time
func NewMockTimeProviderAt(t time.Time) *MockTimeProvider {
	return &MockTimeProvider{
		current: t,
	}
}

func (m *MockTimeProvider) Now() time.Time {
	return m.current
}

// SetTime sets the current time to a specific value
func (m *MockTimeProvider) SetTime(t time.Time) {
	m.current = t
}

// AddTime adds a duration to the current time
func (m *MockTimeProvider) AddTime(d time.Duration) {
	m.current = m.current.Add(d)
}

// AddDays adds the specified number of days to the current time
func (m *MockTimeProvider) AddDays(days int) {
	m.current = m.current.AddDate(0, 0, days)
}

// AddMonths adds the specified number of months to the current time
func (m *MockTimeProvider) AddMonths(months int) {
	m.current = m.current.AddDate(0, months, 0)
}

// FastForwardTo sets the time to a future date (must be after current time)
func (m *MockTimeProvider) FastForwardTo(t time.Time) {
	if t.After(m.current) {
		m.current = t
	}
}

// FastForwardPastPeriodEnd advances time past a subscription's period end
func (m *MockTimeProvider) FastForwardPastPeriodEnd(periodEnd time.Time) {
	if periodEnd.After(m.current) {
		// Go 1 day past the period end to ensure it's truly over
		m.current = periodEnd.AddDate(0, 0, 1)
	}
}