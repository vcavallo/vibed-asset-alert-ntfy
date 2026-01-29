package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level configuration
type Config struct {
	Ntfy          NtfyConfig    `yaml:"ntfy"`
	CheckInterval string        `yaml:"check_interval"`
	Alerts        []AlertConfig `yaml:"alerts"`
}

// NtfyConfig holds ntfy server configuration
type NtfyConfig struct {
	Server   string `yaml:"server"`
	Topic    string `yaml:"topic"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"token"`
	Priority int    `yaml:"priority"`
}

// AlertConfig represents an alert for a specific ticker
type AlertConfig struct {
	Ticker     string            `yaml:"ticker"`
	Name       string            `yaml:"name"`
	Conditions []ConditionConfig `yaml:"conditions"`
}

// ConditionConfig represents a single alert condition
type ConditionConfig struct {
	Type    string  `yaml:"type"`    // "above", "below", "percent_change"
	Value   float64 `yaml:"value"`   // threshold price or percentage
	Period  string  `yaml:"period"`  // for percent_change: "24h", "1h", etc.
	Message string  `yaml:"message"` // custom alert message (optional)
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Expand environment variables
	content := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Set defaults
	if cfg.Ntfy.Priority == 0 {
		cfg.Ntfy.Priority = 3
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// expandEnvVars replaces ${VAR} patterns with environment variable values
func expandEnvVars(content string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		varName := match[2 : len(match)-1] // strip ${ and }
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match // keep original if env var not set
	})
}

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	if c.Ntfy.Server == "" {
		return fmt.Errorf("ntfy.server is required")
	}
	if c.Ntfy.Topic == "" {
		return fmt.Errorf("ntfy.topic is required")
	}
	if c.Ntfy.Priority < 1 || c.Ntfy.Priority > 5 {
		return fmt.Errorf("ntfy.priority must be between 1 and 5")
	}

	if len(c.Alerts) == 0 {
		return fmt.Errorf("at least one alert is required")
	}

	for i, alert := range c.Alerts {
		if alert.Ticker == "" {
			return fmt.Errorf("alerts[%d].ticker is required", i)
		}
		if len(alert.Conditions) == 0 {
			return fmt.Errorf("alerts[%d].conditions is required", i)
		}

		for j, cond := range alert.Conditions {
			if err := validateCondition(cond); err != nil {
				return fmt.Errorf("alerts[%d].conditions[%d]: %w", i, j, err)
			}
		}
	}

	return nil
}

func validateCondition(c ConditionConfig) error {
	validTypes := map[string]bool{
		"above":          true,
		"below":          true,
		"percent_change": true,
	}

	if !validTypes[c.Type] {
		return fmt.Errorf("invalid type %q (must be above, below, or percent_change)", c.Type)
	}

	if c.Value <= 0 {
		return fmt.Errorf("value must be positive")
	}

	if c.Type == "percent_change" && c.Period == "" {
		return fmt.Errorf("period is required for percent_change conditions")
	}

	return nil
}

// GetUniqueTickers returns a deduplicated list of all tickers in the config
func (c *Config) GetUniqueTickers() []string {
	seen := make(map[string]bool)
	var tickers []string

	for _, alert := range c.Alerts {
		ticker := strings.ToUpper(alert.Ticker)
		if !seen[ticker] {
			seen[ticker] = true
			tickers = append(tickers, ticker)
		}
	}

	return tickers
}
