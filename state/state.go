package state

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// State tracks prices and alert states across runs
type State struct {
	// Prices maps ticker -> current price info
	Prices map[string]PriceRecord `json:"prices"`

	// TriggeredAlerts tracks which alerts have been triggered
	// Key format: "ticker:type:value" (e.g., "BTC-USD:above:100000")
	TriggeredAlerts map[string]bool `json:"triggered_alerts"`

	// PriceHistory stores historical prices for percent change calculations
	// Key format: "ticker" -> list of price records
	PriceHistory map[string][]PriceRecord `json:"price_history"`

	path string
}

// PriceRecord represents a price at a point in time
type PriceRecord struct {
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

// Load reads state from a JSON file, or creates new state if file doesn't exist
func Load(path string) (*State, error) {
	s := &State{
		Prices:          make(map[string]PriceRecord),
		TriggeredAlerts: make(map[string]bool),
		PriceHistory:    make(map[string][]PriceRecord),
		path:            path,
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	// Handle empty file
	if len(data) == 0 {
		return s, nil
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}

	s.path = path
	return s, nil
}

// Save writes state to the JSON file
func (s *State) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}

	return nil
}

// UpdatePrice records a new price for a ticker
func (s *State) UpdatePrice(ticker string, price float64) {
	record := PriceRecord{
		Price:     price,
		Timestamp: time.Now(),
	}

	s.Prices[ticker] = record

	// Add to history
	s.PriceHistory[ticker] = append(s.PriceHistory[ticker], record)

	// Prune old history (keep last 7 days)
	s.pruneHistory(ticker, 7*24*time.Hour)
}

// GetLastPrice returns the last known price for a ticker
func (s *State) GetLastPrice(ticker string) (float64, bool) {
	record, ok := s.Prices[ticker]
	if !ok {
		return 0, false
	}
	return record.Price, true
}

// GetPriceAtTime returns the price closest to the given duration ago
func (s *State) GetPriceAtTime(ticker string, ago time.Duration) (float64, bool) {
	history, ok := s.PriceHistory[ticker]
	if !ok || len(history) == 0 {
		return 0, false
	}

	targetTime := time.Now().Add(-ago)

	// Find the price record closest to but before the target time
	var closest *PriceRecord
	for i := range history {
		record := &history[i]
		if record.Timestamp.Before(targetTime) || record.Timestamp.Equal(targetTime) {
			if closest == nil || record.Timestamp.After(closest.Timestamp) {
				closest = record
			}
		}
	}

	if closest == nil {
		// No historical price old enough, use the oldest we have
		return history[0].Price, true
	}

	return closest.Price, true
}

// AlertKey generates a unique key for an alert condition
func AlertKey(ticker, alertType string, value float64) string {
	return fmt.Sprintf("%s:%s:%.2f", ticker, alertType, value)
}

// IsAlertTriggered checks if an alert has already been triggered
func (s *State) IsAlertTriggered(key string) bool {
	return s.TriggeredAlerts[key]
}

// SetAlertTriggered marks an alert as triggered
func (s *State) SetAlertTriggered(key string, triggered bool) {
	s.TriggeredAlerts[key] = triggered
}

// pruneHistory removes price records older than maxAge
func (s *State) pruneHistory(ticker string, maxAge time.Duration) {
	history, ok := s.PriceHistory[ticker]
	if !ok {
		return
	}

	cutoff := time.Now().Add(-maxAge)
	var pruned []PriceRecord

	for _, record := range history {
		if record.Timestamp.After(cutoff) {
			pruned = append(pruned, record)
		}
	}

	s.PriceHistory[ticker] = pruned
}
