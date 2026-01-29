package ntfy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vcavallo/asset-alerts/config"
)

// Sender sends notifications to ntfy
type Sender struct {
	cfg        config.NtfyConfig
	httpClient *http.Client
}

// notification represents the JSON payload for ntfy
type notification struct {
	Topic    string   `json:"topic"`
	Message  string   `json:"message"`
	Title    string   `json:"title,omitempty"`
	Priority int      `json:"priority,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

// NewSender creates a new ntfy sender
func NewSender(cfg config.NtfyConfig) *Sender {
	return &Sender{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send sends a notification to ntfy
func (s *Sender) Send(title, message string, tags []string) error {
	notif := notification{
		Topic:    s.cfg.Topic,
		Message:  message,
		Title:    title,
		Priority: s.cfg.Priority,
		Tags:     tags,
	}

	body, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("marshaling notification: %w", err)
	}

	url := s.cfg.Server
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication
	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy returned status %d", resp.StatusCode)
	}

	return nil
}

// addAuth adds authentication headers based on configuration
func (s *Sender) addAuth(req *http.Request) {
	// Token auth takes precedence
	if s.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.Token)
		return
	}

	// Basic auth
	if s.cfg.Username != "" && s.cfg.Password != "" {
		req.SetBasicAuth(s.cfg.Username, s.cfg.Password)
	}
}

// SendAlert sends an alert notification with appropriate formatting
func (s *Sender) SendAlert(ticker, name, message string, price float64) error {
	title := fmt.Sprintf("💰 %s Alert", name)
	if name == "" {
		title = fmt.Sprintf("💰 %s Alert", ticker)
	}

	// Add price to message if not already included
	fullMessage := fmt.Sprintf("%s\n\nCurrent price: $%.2f", message, price)

	// Use emoji tags for visual identification
	tags := []string{"chart_with_upwards_trend", ticker}

	return s.Send(title, fullMessage, tags)
}
