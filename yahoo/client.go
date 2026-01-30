package yahoo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	baseURL    = "https://query1.finance.yahoo.com/v8/finance/chart"
	userAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
	timeoutSec = 10
)

// Client fetches quotes from Yahoo Finance
type Client struct {
	httpClient *http.Client
}

// Quote represents price data for a ticker
type Quote struct {
	Ticker        string
	Price         float64
	PreviousClose float64
	Timestamp     time.Time
}

// chartResponse represents the Yahoo Finance API response
type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol             string  `json:"symbol"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				PreviousClose      float64 `json:"previousClose"`
				RegularMarketTime  int64   `json:"regularMarketTime"`
			} `json:"meta"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// NewClient creates a new Yahoo Finance client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeoutSec * time.Second,
		},
	}
}

// GetQuote fetches the current price for a ticker
func (c *Client) GetQuote(ticker string) (*Quote, error) {
	url := fmt.Sprintf("%s/%s", baseURL, ticker)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching quote: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var chartResp chartResponse
	if err := json.NewDecoder(resp.Body).Decode(&chartResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if chartResp.Chart.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s",
			chartResp.Chart.Error.Code,
			chartResp.Chart.Error.Description)
	}

	if len(chartResp.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data returned for ticker %s", ticker)
	}

	meta := chartResp.Chart.Result[0].Meta

	return &Quote{
		Ticker:        meta.Symbol,
		Price:         meta.RegularMarketPrice,
		PreviousClose: meta.PreviousClose,
		Timestamp:     time.Unix(meta.RegularMarketTime, 0),
	}, nil
}

// GetQuotes fetches prices for multiple tickers
// Continues on individual failures, only returns error if all tickers fail
func (c *Client) GetQuotes(tickers []string) (map[string]*Quote, error) {
	quotes := make(map[string]*Quote)
	var lastErr error

	for _, ticker := range tickers {
		quote, err := c.GetQuote(ticker)
		if err != nil {
			lastErr = fmt.Errorf("fetching %s: %w", ticker, err)
			// Log but continue with other tickers
			fmt.Printf("Warning: failed to fetch %s: %v\n", ticker, err)
			continue
		}
		quotes[ticker] = quote
	}

	if len(quotes) == 0 && lastErr != nil {
		return nil, fmt.Errorf("all tickers failed, last error: %w", lastErr)
	}

	return quotes, nil
}
