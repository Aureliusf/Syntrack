#!/usr/bin/env bash

# Install syntrack cron job with proper .env sourcing

SYNTRACK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
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
        BASH_PATH=$(command -v bash)
    fi
fi

if [ -z "$BASH_PATH" ] || [ ! -x "$BASH_PATH" ]; then
    echo "Warning: bash not found. Falling back to /bin/sh (may have issues with .env)"
    BASH_PATH="/bin/sh"
fi

# Build the cron job
# Use 'set -a' to export all variables from .env automatically
CRON_JOB="SHELL=$BASH_PATH"
CRON_JOB="${CRON_JOB}
*/30 * * * * cd $SYNTRACK_DIR && set -a && source $ENV_FILE 2>/dev/null && set +a && $SYNTRACK_BIN collect >> $LOG_FILE 2>&1"

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

# Remove old cron job
if crontab -l 2>/dev/null | grep -q "syntrack collect"; then
    echo "Removing old cron job..."
    crontab -l 2>/dev/null | grep -v "syntrack" | crontab -
fi

# Add new cron job with SHELL directive
EXISTING_CRON=$(crontab -l 2>/dev/null | grep -v "syntrack")
if [ -n "$EXISTING_CRON" ]; then
    # Preserve existing crontab, just add our job
    echo "$EXISTING_CRON" > /tmp/crontab.tmp
    echo "" >> /tmp/crontab.tmp
    echo "# Syntrack - runs every 30 minutes" >> /tmp/crontab.tmp
    echo "$CRON_JOB" >> /tmp/crontab.tmp
    crontab /tmp/crontab.tmp
    rm /tmp/crontab.tmp
else
    # New crontab
    echo "# Syntrack - runs every 30 minutes" > /tmp/crontab.tmp
    echo "$CRON_JOB" >> /tmp/crontab.tmp
    crontab /tmp/crontab.tmp
    rm /tmp/crontab.tmp
fi

echo ""
echo "Done! Cron job installed:"
crontab -l | grep -A2 "Syntrack"
echo ""
echo "Note: The cron job sources variables from $ENV_FILE"
echo "To verify it's working:"
echo "  1. Wait 30 minutes for the next run"
echo "  2. Check logs: cat $LOG_FILE"
echo "  3. View history: ./syntrack history"
