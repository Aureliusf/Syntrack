# Syntrack

CLI tool to track [Synthetic.ai](https://synthetic.ai) usage and leftover requests.

Collects quota data every 30 minutes, stores in SQLite, and provides CLI queries + web dashboard for analysis.

## Features

- **Auto-collection**: Cron job fetches quota every 30 minutes
- **Historical tracking**: Store usage data beyond Synthetic's 4-hour window
- **Burn rate analysis**: Predict when you'll run out of requests
- **ASCII charts**: Visualize usage directly in terminal
- **JSON output**: Agent-friendly for automation
- **Web dashboard**: HTMX-powered UI with SVG charts

## Installation

```bash
git clone <repo>
cd syntrack

# Enter dev shell (Go + SQLite)
nix develop

# Build
go build -o syntrack .
```

## Configuration

Create `.env` in project directory:

```bash
SYNTHETIC_API_KEY=your_api_key_here
DATABASE_PATH=usage.db
```

Or use environment variables:

```bash
export SYNTHETIC_API_KEY=your_key
./syntrack collect
```

## Usage

### Collect Data

Fetch quota from Synthetic API and store snapshot:

```bash
./syntrack collect
# Output: Collected: 89/135 used (46 leftover)
```

### Check Status

```bash
./syntrack status
```

### View History

```bash
./syntrack history              # Last 7 days
./syntrack history -d 14        # Last 14 days
./syntrack history -c           # ASCII chart view
```

### Statistics

```bash
./syntrack stats                # Human-readable
./syntrack stats -c             # With inline charts
```

### ASCII Charts

```bash
./syntrack chart                # Usage over time (line chart)
./syntrack chart -t daily       # Daily consumption bars
./syntrack chart -t weekly      # Weekly consumption bars
./syntrack chart -d 30          # Last 30 days
```

### JSON Queries (for agents/scripts)

```bash
./syntrack query current        # Current status
./syntrack query today          # Today's summary
./syntrack query yesterday      # Yesterday's summary
./syntrack query week           # This week's summary
./syntrack query burn-rate      # Rate + predictions
./syntrack query history -d 3   # Recent snapshots
./syntrack query daily -d 7     # Daily breakdown
./syntrack query weekly -w 4    # Weekly breakdown
```

## Web Dashboard

Start HTTP server:

```bash
./syntrack serve -p 8080
# Open http://localhost:8080
```

Dashboard includes:
- Current quota status (auto-refreshes every 5min)
- Usage chart over time (SVG, server-rendered)
- Burn rate estimates
- Daily/weekly tables
- History view

## Cron Setup

Install 30-minute collection:

```bash
./scripts/install-cron.sh
```

Or manually add to crontab:

```bash
*/30 * * * * /path/to/syntrack collect >> /tmp/syntrack.log 2>&1
```

## Database

SQLite stored at `usage.db` (gitignored). Contains:

- `usage_snapshots`: Raw data points every 30min
- `daily_usage` (view): Daily aggregations
- `weekly_usage` (view): Weekly aggregations

Query directly:

```bash
sqlite3 usage.db "SELECT * FROM daily_usage;"
```

## Development

```bash
nix develop  # Enter dev shell with Go

# Build
make build

# Run tests
make test

# Run server
make serve
```

## Project Structure

```
├── cmd/              # CLI commands
│   ├── root.go
│   ├── collect.go
│   ├── status.go
│   ├── history.go
│   ├── stats.go
│   ├── query.go
│   ├── chart.go
│   └── serve.go
├── internal/
│   ├── api/          # Synthetic API client
│   ├── db/           # SQLite layer
│   ├── models/       # Data structures
│   └── config/       # Config loading
├── web/              # Dashboard templates
├── scripts/
│   └── install-cron.sh
├── flake.nix         # Nix dev shell
├── Makefile
└── README.md
```

## License

MIT
