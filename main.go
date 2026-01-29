package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vcavallo/asset-alerts/alerts"
	"github.com/vcavallo/asset-alerts/config"
	"github.com/vcavallo/asset-alerts/ntfy"
	"github.com/vcavallo/asset-alerts/state"
	"github.com/vcavallo/asset-alerts/yahoo"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	statePath := flag.String("state", "", "Path to state file (default: same directory as config)")
	verbose := flag.Bool("v", false, "Verbose output")
	dryRun := flag.Bool("dry-run", false, "Check prices but don't send notifications")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *verbose {
		log.Printf("Loaded config with %d alert groups", len(cfg.Alerts))
	}

	// Determine state file path
	stateFile := *statePath
	if stateFile == "" {
		stateFile = filepath.Join(filepath.Dir(*configPath), "state.json")
	}

	// Load state
	st, err := state.Load(stateFile)
	if err != nil {
		log.Fatalf("Failed to load state: %v", err)
	}

	if *verbose {
		log.Printf("Loaded state from %s", stateFile)
	}

	// Get unique tickers
	tickers := cfg.GetUniqueTickers()
	if *verbose {
		log.Printf("Fetching prices for %d tickers: %v", len(tickers), tickers)
	}

	// Fetch quotes
	yahooClient := yahoo.NewClient()
	quotes, err := yahooClient.GetQuotes(tickers)
	if err != nil {
		log.Fatalf("Failed to fetch quotes: %v", err)
	}

	if *verbose {
		for ticker, quote := range quotes {
			log.Printf("%s: $%.2f", ticker, quote.Price)
		}
	}

	// Evaluate alerts
	evaluator := alerts.NewEvaluator(st)
	triggered := evaluator.Evaluate(cfg.Alerts, quotes)

	if *verbose {
		log.Printf("Triggered %d alerts", len(triggered))
	}

	// Send notifications
	if len(triggered) > 0 && !*dryRun {
		sender := ntfy.NewSender(cfg.Ntfy)

		for _, alert := range triggered {
			if *verbose {
				log.Printf("Sending alert: %s - %s", alert.Ticker, alert.Message)
			}

			if err := sender.SendAlert(alert.Ticker, alert.Name, alert.Message, alert.Price); err != nil {
				log.Printf("Failed to send alert for %s: %v", alert.Ticker, err)
			} else {
				fmt.Printf("✓ Alert sent: %s - %s\n", alert.Name, alert.Message)
			}
		}
	} else if len(triggered) > 0 && *dryRun {
		fmt.Println("Dry run - would send the following alerts:")
		for _, alert := range triggered {
			fmt.Printf("  • %s: %s (price: $%.2f)\n", alert.Name, alert.Message, alert.Price)
		}
	} else if *verbose {
		log.Println("No alerts triggered")
	}

	// Update prices in state
	for ticker, quote := range quotes {
		st.UpdatePrice(ticker, quote.Price)
	}

	// Save state
	if err := st.Save(); err != nil {
		log.Fatalf("Failed to save state: %v", err)
	}

	if *verbose {
		log.Printf("State saved to %s", stateFile)
	}

	os.Exit(0)
}
