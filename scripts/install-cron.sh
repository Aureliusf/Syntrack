#!/bin/sh

# Install syntrack cron job with proper .env sourcing
# Works with both bash and POSIX sh

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SYNTRACK_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SYNTRACK_BIN="$SYNTRACK_DIR/syntrack"
ENV_FILE="$SYNTRACK_DIR/.env"

# Secure logging: use private directory with restricted permissions
LOG_DIR="$HOME/.syntrack/logs"
LOG_FILE="$LOG_DIR/syntrack.log"

# Create log directory with secure permissions (0700 = owner only)
if [ ! -d "$LOG_DIR" ]; then
    echo "Creating secure log directory: $LOG_DIR"
    mkdir -p "$LOG_DIR"
    chmod 0700 "$LOG_DIR"
fi

# Check for bash availability
BASH_PATH="/bin/bash"
if [ ! -x "$BASH_PATH" ]; then
    BASH_PATH="/usr/bin/bash"
    if [ ! -x "$BASH_PATH" ]; then
        BASH_PATH="$(command -v bash 2>/dev/null)"
    fi
fi

if [ -z "$BASH_PATH" ] || [ ! -x "$BASH_PATH" ]; then
    echo "Warning: bash not found. Falling back to /bin/sh (may have issues with .env)"
    BASH_PATH="/bin/sh"
fi

echo "Installing syntrack cron job..."
echo "Binary: $SYNTRACK_BIN"
echo "Log file: $LOG_FILE"
echo "Env file: $ENV_FILE"
echo "Shell: $BASH_PATH"

# Verify binary exists
if [ ! -x "$SYNTRACK_BIN" ]; then
    echo "Error: Binary not found or not executable: $SYNTRACK_BIN"
    echo "Did you run 'go build -o syntrack .' first?"
    exit 1
fi

# Check if .env file exists
if [ ! -f "$ENV_FILE" ]; then
    echo "Warning: .env file not found at $ENV_FILE"
    echo "The cron job may fail if SYNTHETIC_API_KEY is not set."
fi

# Build the cron job command with timestamps
# The job changes to the project directory, sources the .env file, and logs with timestamps
CRON_CMD="cd $SYNTRACK_DIR && set -a && . $ENV_FILE 2>/dev/null && set +a && echo \"[\$(date '+%Y-%m-%d %H:%M:%S')] Starting syntrack collection\" >> $LOG_FILE && $SYNTRACK_BIN collect >> $LOG_FILE 2>&1 && echo \"[\$(date '+%Y-%m-%d %H:%M:%S')] Collection complete\" >> $LOG_FILE"

# Remove old syntrack entries from crontab
OLD_CRON=$(crontab -l 2>/dev/null | grep -v "syntrack" || true)

# Build new crontab with SHELL directive
NEW_CRON=""
if [ -n "$OLD_CRON" ]; then
    NEW_CRON="$OLD_CRON"
fi

# Add syntrack section
if [ -n "$NEW_CRON" ]; then
    NEW_CRON="$NEW_CRON

"
fi

NEW_CRON="${NEW_CRON}# Syntrack - runs every 30 minutes
SHELL=$BASH_PATH
*/30 * * * * $CRON_CMD"

# Install new crontab
echo "$NEW_CRON" | crontab -

echo ""
echo "Done! Cron job installed:"
crontab -l | grep -A2 "Syntrack"
echo ""
echo "Note: The cron job sources variables from $ENV_FILE"
echo "To verify it's working:"
echo "  1. Wait 30 minutes for the next run"
echo "  2. Check logs: cat $LOG_FILE"
echo "  3. View history: ./syntrack history"
