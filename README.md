# Asset Price Alert System

A self-hosted Go application that monitors asset prices (stocks + crypto) via Yahoo Finance and sends alerts via ntfy when prices cross configured thresholds or change by a percentage.

## Features

- **Multiple asset types:** Stocks (AAPL, MSFT) and crypto (BTC-USD, ETH-USD)
- **Flexible alert conditions:**
  - Price above threshold
  - Price below threshold
  - Percent change over time period
  - Absolute dollar change over time period
- **ntfy notifications:** Supports self-hosted ntfy servers with authentication
- **Smart alerting:** Prevents duplicate alerts with hysteresis (alerts reset when price crosses back)
- **Cron-friendly:** Runs as a single binary, checks prices, sends alerts, exits

## Installation

### From Source

```bash
git clone https://github.com/vcavallo/asset-alerts.git
cd asset-alerts
go build -o asset-alerts
```

### Docker

```bash
docker build -t asset-alerts .
```

## Configuration

Copy the example configuration and customize:

```bash
cp config.example.yaml config.yaml
```

### Configuration Options

```yaml
ntfy:
  server: "https://your-ntfy-server.com"  # your self-hosted instance
  topic: "asset-alerts"
  # Authentication (all optional, use what your server requires)
  username: "your-username"
  password: "your-password"
  # OR use access token instead of username/password:
  # token: "tk_your_access_token"
  # Optional: priority (1-5, default 3)
  priority: 3

# Reference for cron setup (not used by the application)
check_interval: "*/5 * * * *"

alerts:
  - ticker: "BTC-USD"
    name: "Bitcoin"
    conditions:
      - type: "above"
        value: 100000
        message: "BTC crossed $100k!"
      - type: "below"
        value: 90000
        message: "BTC dropped below $90k"
      - type: "percent_change"
        value: 5
        period: "24h"
        message: "BTC moved 5% in 24h"

  - ticker: "SI=F"
    name: "Silver"
    conditions:
      - type: "absolute_change"
        value: 10
        period: "24h"
        # Auto-generates: "Silver moved $10.50 up in 24h (currently $125.64)"

  - ticker: "AAPL"
    name: "Apple"
    conditions:
      - type: "above"
        value: 200
      - type: "below"
        value: 150
```

### Alert Types

| Type | Description | Parameters |
|------|-------------|------------|
| `above` | Triggers when price goes above threshold | `value` (price) |
| `below` | Triggers when price goes below threshold | `value` (price) |
| `percent_change` | Triggers when price changes by X% | `value` (percentage), `period` (e.g., "24h", "1h") |
| `absolute_change` | Triggers when price changes by $X | `value` (dollar amount), `period` (e.g., "24h", "1h") |

For `above`/`below` alerts, you can set both on the same value to get notified when price crosses in either direction.

### ntfy Authentication

The application supports multiple authentication methods:

- **No auth:** Leave username/password/token empty
- **Basic auth:** Set `username` and `password`
- **Token auth:** Set `token` (for access tokens like `tk_...`)

For sensitive credentials, use environment variables:

```yaml
ntfy:
  server: "https://your-server.com"
  topic: "alerts"
  username: "${NTFY_USER}"
  password: "${NTFY_PASS}"
```

## Usage

### Manual Run

```bash
./asset-alerts --config config.yaml
```

### With Cron (recommended)

Add to your crontab (`crontab -e`):

```bash
# Check every 5 minutes
*/5 * * * * /path/to/asset-alerts --config /path/to/config.yaml

# Check every minute for more responsive alerts
* * * * * /path/to/asset-alerts --config /path/to/config.yaml
```

### Docker

```bash
# Build the image
docker build -t asset-alerts .

# Create state file (required before first run)
echo '{}' > state.json

# Run once
docker run --rm \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/state.json:/app/state.json \
  asset-alerts

# Run with verbose output
docker run --rm \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/state.json:/app/state.json \
  asset-alerts --config /app/config.yaml -v
```

### Docker with Cron

Add to your crontab (`crontab -e`):

```bash
*/5 * * * * docker run --rm -v /path/to/config.yaml:/app/config.yaml -v /path/to/state.json:/app/state.json asset-alerts
```

Use absolute paths in crontab (not `$(pwd)`).

## State Management

The application maintains state in `state.json` (same directory as config by default):

- Tracks last known price per ticker
- Records which alert conditions have been triggered
- Stores historical prices for percent change calculations

This prevents duplicate alerts and enables smart threshold crossing detection.

## How It Works

1. Load configuration and state
2. Fetch current prices from Yahoo Finance for all configured tickers
3. For each alert condition:
   - **Threshold alerts:** Check if price crossed the threshold since last check. Only alert once per crossing.
   - **Percent/absolute change:** Compare to historical price from the specified period. Alert if change exceeds threshold.
4. Send ntfy notifications for triggered alerts
5. Update state file
6. Exit

## Yahoo Finance API

Uses the free, unauthenticated Yahoo Finance v8 endpoint:
- No API key required
- Works for stocks and crypto
- Rate limit friendly for personal use

## Troubleshooting

### No notifications received

1. Verify ntfy server URL and topic are correct
2. Check authentication credentials
3. Run with verbose output to see what's happening
4. Verify the ticker symbol is valid (test at finance.yahoo.com)

### Duplicate alerts

Delete `state.json` to reset alert states, then run again.

### Testing

Create a test alert that will definitely trigger:

```yaml
alerts:
  - ticker: "BTC-USD"
    name: "Bitcoin Test"
    conditions:
      - type: "above"
        value: 1  # BTC is definitely above $1
        message: "Test alert works!"
```

Run the application and verify you receive the notification.

## License

MIT
