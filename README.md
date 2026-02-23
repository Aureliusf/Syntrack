# Syntrack

CLI tool to track [Synthetic.new](https://synthetic.new) usage and leftover requests.

Collects quota data every 30 minutes, stores in SQLite, and provides CLI queries and charts as well as a web dashboard for analysis.

## Features

- **Auto-collection**: Cron job fetches quota every 30 minutes
- **Historical tracking**: Store usage data beyond Synthetic's 4-hour window
- **Burn rate analysis**: Predict when you'll run out of requests
- **ASCII charts**: Visualize usage directly in terminal
- **JSON output**: Agent-friendly for automation
- **Web dashboard**: HTMX-powered UI with SVG charts

## Installation

```bash
git clone git@github.com:Aureliusf/Syntrack.git
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

## Usage

### Collect Data

Fetch quota from Synthetic API and store snapshot:

```bash
./syntrack collect
# Output: Collected: 89/135 used (46 leftover)
```

### Cron Setup

Install 30-minute collection:

```bash
# Make script executable (first time only)
chmod +x ./scripts/install-cron.sh

# Run installer
./scripts/install-cron.sh
```

Or manually add to crontab:

```bash
*/30 * * * * /path/to/syntrack collect >> /tmp/syntrack.log 2>&1
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

### Background Server (Silent Mode)

Start the web dashboard in the background and exit the CLI:

```bash
./syntrack serve --silent       # Start server in background
./syntrack serve --silent -p 3000  # Custom port
```

This will:
1. Start the server as a detached background process
2. Wait for the server to respond (up to ~7 seconds)
3. Display the server URL and process ID
4. Exit the CLI while keeping the server running

To stop the background server:
```bash
kill <PID>  # Use the process ID shown when starting
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

Dashboard features:
- **Current quota status** (auto-refreshes every 5min)
- **Usage chart** over time (SVG, server-rendered)
- **Burn rate** estimates
- **Daily/weekly** tables
- **History** view
- **Token authentication** for remote access (see Deployment section)

### Dashboard Authentication

When accessing remotely (non-localhost), authentication is required:

**Quick access with token in URL:**
```
http://your-server:8080/?token=syntrack_token_abc123...
```

**Or use the auth modal:**
1. Click "🔒 Authenticate" in the navbar
2. Enter your token
3. Token saves to browser storage
4. All requests automatically authenticated

**Security notes:**
- Localhost access (127.0.0.1) requires no authentication
- Remote access requires valid token
- Tokens persist in browser until logout
- Use `--bind-all` flag only with `--auth-token` configured


## Database

SQLite stored at `usage.db` (gitignored). Contains:

- `usage_snapshots`: Raw data points every 30min
- `daily_usage` (view): Daily aggregations
- `weekly_usage` (view): Weekly aggregations

Query directly:

```bash
sqlite3 usage.db "SELECT * FROM daily_usage;"
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

## Security

### Dependency Scanning

Scan for known vulnerabilities in Go dependencies:

```bash
# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Run vulnerability scan
govulncheck ./...
```

**Nix users**: This is automatically available in the dev shell via `flake.nix`.

## Deployment

### Nix Deployment

Build and run directly with Nix:

```bash
# Build the package
nix build .

# Run without installing
nix run . -- serve

# Install to profile
nix profile install .
```

For NixOS systems, add to your `configuration.nix`:

```nix
# Option 1: Import as flake input
{
  inputs.syntrack.url = "github:yourusername/syntrack";
  
  environment.systemPackages = [ inputs.syntrack.packages.${pkgs.system}.default ];
}

# Option 2: Use the systemd service (recommended)
# Create a systemd service in your configuration:
systemd.services.syntrack = {
  description = "Syntrack dashboard server";
  after = [ "network.target" ];
  wantedBy = [ "multi-user.target" ];
  
  serviceConfig = {
    Type = "simple";
    User = "syntrack";
    Group = "syntrack";
    WorkingDirectory = "/var/lib/syntrack";
    Environment = [
      "SYNTHETIC_API_KEY_FILE=/var/lib/syntrack/.env"
      "DATABASE_PATH=/var/lib/syntrack/usage.db"
    ];
    ExecStart = "${pkgs.syntrack}/bin/syntrack serve --tailscale -p 8080";
    Restart = "on-failure";
    RestartSec = 5;
    
    # Security hardening
    NoNewPrivileges = true;
    PrivateTmp = true;
    ProtectSystem = "strict";
    ProtectHome = true;
    ReadWritePaths = [ "/var/lib/syntrack" ];
  };
};

# Create user and directory
users.users.syntrack = {
  isSystemUser = true;
  group = "syntrack";
  home = "/var/lib/syntrack";
  createHome = true;
};
users.groups.syntrack = {};
```

### Non-Nix Linux Deployment

#### Build from Source

```bash
# Prerequisites: Go 1.21+
git clone <repo>
cd syntrack
go build -o syntrack .

# Install to system
sudo cp syntrack /usr/local/bin/
sudo mkdir -p /var/lib/syntrack
sudo chmod 750 /var/lib/syntrack
```

#### Systemd Service

Create `/etc/systemd/system/syntrack.service`:

```ini
[Unit]
Description=Syntrack dashboard server
After=network.target

[Service]
Type=simple
User=syntrack
Group=syntrack
WorkingDirectory=/var/lib/syntrack

# Security: Load API key from file
Environment="SYNTHETIC_API_KEY_FILE=/var/lib/syntrack/.env"
Environment="DATABASE_PATH=/var/lib/syntrack/usage.db"

# Use --tailscale for Tailscale network access (auto-detects IP)
# Use --bind-all only with --auth-token (see Security section below)
ExecStart=/usr/local/bin/syntrack serve --tailscale -p 8080

Restart=on-failure
RestartSec=5

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/syntrack

# Logging to journald
StandardOutput=journal
StandardError=journal
SyslogIdentifier=syntrack

[Install]
WantedBy=multi-user.target
```

Create user and enable service:

```bash
# Create syntrack user
sudo useradd --system --create-home --home-dir /var/lib/syntrack syntrack

# Set up environment
sudo mkdir -p /var/lib/syntrack
sudo chown syntrack:syntrack /var/lib/syntrack
sudo chmod 750 /var/lib/syntrack

# Create .env file with restricted permissions
sudo tee /var/lib/syntrack/.env << 'EOF'
SYNTHETIC_API_KEY=your_api_key_here
EOF
sudo chmod 600 /var/lib/syntrack/.env
sudo chown syntrack:syntrack /var/lib/syntrack/.env

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable syntrack
sudo systemctl start syntrack

# View logs
sudo journalctl -u syntrack -f
```

#### Cron Job for Data Collection

The install script automatically sets up cron. For systemd-based timers (recommended):

Create `/etc/systemd/system/syntrack-collect.service`:

```ini
[Unit]
Description=Syntrack data collection

[Service]
Type=oneshot
User=syntrack
Group=syntrack
WorkingDirectory=/var/lib/syntrack
Environment="SYNTHETIC_API_KEY_FILE=/var/lib/syntrack/.env"
Environment="DATABASE_PATH=/var/lib/syntrack/usage.db"
ExecStart=/usr/local/bin/syntrack collect
StandardOutput=journal
StandardError=journal
```

Create `/etc/systemd/system/syntrack-collect.timer`:

```ini
[Unit]
Description=Run Syntrack collection every 30 minutes

[Timer]
OnCalendar=*:0/30
Persistent=true

[Install]
WantedBy=timers.target
```

Enable:

```bash
sudo systemctl daemon-reload
sudo systemctl enable syntrack-collect.timer
sudo systemctl start syntrack-collect.timer

# Check status
sudo systemctl list-timers syntrack-collect.timer
sudo journalctl -u syntrack-collect -f
```

### Security Configuration

**⚠️ CRITICAL: Never expose the dashboard without authentication.**

#### Token Generation

Generate authentication tokens:

```bash
# Generate and save token
syntrack token generate --save
# Output: syntrack_token_abc123...
# Saved to ~/.syntrack/tokens

# Or generate without saving
syntrack token generate
# Copy the token for use in URL or browser
```

#### Network Binding Modes

**Mode 1: Localhost Only (Default - Most Secure)**
```bash
syntrack serve -p 8080
# Only accessible from localhost
# No authentication required for local access
```

**Mode 2: Tailscale Network**
```bash
syntrack serve --tailscale -p 8080
# Auto-detects Tailscale IP (100.x.x.x)
# WARNING: Token authentication is strongly recommended
# Use: http://your-tailscale-ip:8080/?token=your-token
```

**Mode 3: All Interfaces (Requires Auth Token)**
```bash
# MUST provide auth token when binding to all interfaces
syntrack serve --bind-all --auth-token syntrack_token_abc123... -p 8080

# Or set via environment
export SYNTRACK_AUTH_TOKENS=syntrack_token_abc123...
syntrack serve --bind-all -p 8080
```

#### Token Storage Options

**Option A: Environment Variable (Recommended for systemd)**
```bash
export SYNTRACK_AUTH_TOKENS=token1,token2,token3
```

**Option B: Token File**
```bash
# Tokens stored in ~/.syntrack/tokens (one per line)
echo "syntrack_token_abc123..." >> ~/.syntrack/tokens
chmod 600 ~/.syntrack/tokens
```

#### Web Dashboard Authentication

When accessing from non-localhost:

1. **Via URL (one-time setup):**
   ```
   http://localhost:8080/?token=syntrack_token_abc123...
   ```
   Token automatically saved to browser localStorage.

2. **Via Auth Modal:**
   - Click "🔒 Authenticate" button in navbar
   - Enter token in the password field
   - Click "Authenticate"
   - Token persists in browser until logout

3. **Logout:**
   - Click "🔓 Authenticated" button
   - Token cleared from browser

### Tailscale Integration

**Auto-detect (Recommended):**
```bash
syntrack serve --tailscale -p 8080
# Output: Detected Tailscale IP: 100.87.201.23
# Server binds to 100.87.201.23:8080
```

**Manual IP (if auto-detect fails):**
```bash
syntrack serve --tailscale-ip 100.87.201.23 -p 8080
```

Access from other Tailscale devices:
```
http://100.87.201.23:8080/?token=your-token
```

### Database Location (Production)

**For production deployments, move database outside project directory:**

```bash
# Create secure directory
sudo mkdir -p /var/lib/syntrack
sudo chown syntrack:syntrack /var/lib/syntrack
sudo chmod 700 /var/lib/syntrack

# Set environment variable
export DATABASE_PATH=/var/lib/syntrack/usage.db

# Or in systemd service
Environment="DATABASE_PATH=/var/lib/syntrack/usage.db"
```

**Permissions:**
- Directory: `0700` (owner read/write/execute only)
- Database file: `0600` (owner read/write only)
- Logs directory: `0700` (owner only)

**Note:** The application automatically enforces these permissions on startup.

## License

MIT
