package alerts

import (
	"fmt"
	"math"
	"time"

	"github.com/vcavallo/asset-alerts/config"
	"github.com/vcavallo/asset-alerts/state"
	"github.com/vcavallo/asset-alerts/yahoo"
)

// TriggeredAlert represents an alert that should be sent
type TriggeredAlert struct {
	Ticker    string
	Name      string
	Condition config.ConditionConfig
	Price     float64
	Message   string
}

// Evaluator checks alert conditions against prices
type Evaluator struct {
	state *state.State
}

// NewEvaluator creates a new alert evaluator
func NewEvaluator(s *state.State) *Evaluator {
	return &Evaluator{state: s}
}

// Evaluate checks all alert conditions and returns triggered alerts
func (e *Evaluator) Evaluate(alerts []config.AlertConfig, quotes map[string]*yahoo.Quote) []TriggeredAlert {
	var triggered []TriggeredAlert

	for _, alert := range alerts {
		quote, ok := quotes[alert.Ticker]
		if !ok {
			continue
		}

		for _, cond := range alert.Conditions {
			if t := e.evaluateCondition(alert, cond, quote); t != nil {
				triggered = append(triggered, *t)
			}
		}
	}

	return triggered
}

func (e *Evaluator) evaluateCondition(alert config.AlertConfig, cond config.ConditionConfig, quote *yahoo.Quote) *TriggeredAlert {
	switch cond.Type {
	case "above":
		return e.evaluateAbove(alert, cond, quote)
	case "below":
		return e.evaluateBelow(alert, cond, quote)
	case "percent_change":
		return e.evaluatePercentChange(alert, cond, quote)
	}
	return nil
}

func (e *Evaluator) evaluateAbove(alert config.AlertConfig, cond config.ConditionConfig, quote *yahoo.Quote) *TriggeredAlert {
	key := state.AlertKey(alert.Ticker, "above", cond.Value)
	lastPrice, hasLast := e.state.GetLastPrice(alert.Ticker)

	// Check if price is above threshold
	isAbove := quote.Price >= cond.Value

	// Check if we've already triggered this alert
	alreadyTriggered := e.state.IsAlertTriggered(key)

	if isAbove {
		if !alreadyTriggered {
			// Price crossed above threshold - trigger alert
			e.state.SetAlertTriggered(key, true)
			return &TriggeredAlert{
				Ticker:    alert.Ticker,
				Name:      alert.Name,
				Condition: cond,
				Price:     quote.Price,
				Message:   e.formatMessage(alert, cond, quote.Price, "above"),
			}
		}
	} else {
		// Price is below threshold - reset the alert if it was triggered
		// This implements hysteresis: alert can fire again if price drops and rises
		if alreadyTriggered && hasLast && lastPrice >= cond.Value {
			e.state.SetAlertTriggered(key, false)
		}
	}

	return nil
}

func (e *Evaluator) evaluateBelow(alert config.AlertConfig, cond config.ConditionConfig, quote *yahoo.Quote) *TriggeredAlert {
	key := state.AlertKey(alert.Ticker, "below", cond.Value)
	lastPrice, hasLast := e.state.GetLastPrice(alert.Ticker)

	// Check if price is below threshold
	isBelow := quote.Price <= cond.Value

	// Check if we've already triggered this alert
	alreadyTriggered := e.state.IsAlertTriggered(key)

	if isBelow {
		if !alreadyTriggered {
			// Price crossed below threshold - trigger alert
			e.state.SetAlertTriggered(key, true)
			return &TriggeredAlert{
				Ticker:    alert.Ticker,
				Name:      alert.Name,
				Condition: cond,
				Price:     quote.Price,
				Message:   e.formatMessage(alert, cond, quote.Price, "below"),
			}
		}
	} else {
		// Price is above threshold - reset the alert if it was triggered
		if alreadyTriggered && hasLast && lastPrice <= cond.Value {
			e.state.SetAlertTriggered(key, false)
		}
	}

	return nil
}

func (e *Evaluator) evaluatePercentChange(alert config.AlertConfig, cond config.ConditionConfig, quote *yahoo.Quote) *TriggeredAlert {
	duration, err := parseDuration(cond.Period)
	if err != nil {
		return nil
	}

	histPrice, ok := e.state.GetPriceAtTime(alert.Ticker, duration)
	if !ok {
		// Not enough history yet
		return nil
	}

	// Calculate percent change
	percentChange := ((quote.Price - histPrice) / histPrice) * 100
	absChange := math.Abs(percentChange)

	key := state.AlertKey(alert.Ticker, "percent_change", cond.Value)
	alreadyTriggered := e.state.IsAlertTriggered(key)

	if absChange >= cond.Value {
		if !alreadyTriggered {
			e.state.SetAlertTriggered(key, true)
			direction := "up"
			if percentChange < 0 {
				direction = "down"
			}
			return &TriggeredAlert{
				Ticker:    alert.Ticker,
				Name:      alert.Name,
				Condition: cond,
				Price:     quote.Price,
				Message:   e.formatPercentMessage(alert, cond, quote.Price, percentChange, direction),
			}
		}
	} else {
		// Reset if change has decreased below threshold
		if alreadyTriggered {
			e.state.SetAlertTriggered(key, false)
		}
	}

	return nil
}

func (e *Evaluator) formatMessage(alert config.AlertConfig, cond config.ConditionConfig, price float64, direction string) string {
	if cond.Message != "" {
		return cond.Message
	}

	name := alert.Name
	if name == "" {
		name = alert.Ticker
	}

	return fmt.Sprintf("%s is %s $%.2f (currently $%.2f)", name, direction, cond.Value, price)
}

func (e *Evaluator) formatPercentMessage(alert config.AlertConfig, cond config.ConditionConfig, price float64, change float64, direction string) string {
	if cond.Message != "" {
		return cond.Message
	}

	name := alert.Name
	if name == "" {
		name = alert.Ticker
	}

	return fmt.Sprintf("%s moved %.1f%% %s in %s (currently $%.2f)", name, math.Abs(change), direction, cond.Period, price)
}

// parseDuration converts period strings like "24h", "1h", "7d" to time.Duration
func parseDuration(period string) (time.Duration, error) {
	// Handle day suffix
	if len(period) > 1 && period[len(period)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(period, "%dd", &days); err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// Standard Go duration parsing for hours, minutes, etc.
	return time.ParseDuration(period)
}
