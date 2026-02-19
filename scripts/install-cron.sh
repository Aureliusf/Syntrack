#!/bin/bash

SYNTRACK_BIN="$(pwd)/syntrack"

# Secure logging: use private directory with restricted permissions
# instead of world-readable /tmp
LOG_DIR="$HOME/.syntrack/logs"
LOG_FILE="$LOG_DIR/syntrack.log"

# Create log directory with secure permissions (0700 = owner only)
if [ ! -d "$LOG_DIR" ]; then
    echo "Creating secure log directory: $LOG_DIR"
    mkdir -p "$LOG_DIR"
    chmod 0700 "$LOG_DIR"
fi

# Verify log directory exists and is secure
if [ ! -d "$LOG_DIR" ]; then
    echo "Error: Failed to create log directory: $LOG_DIR"
    exit 1
fi

CRON_JOB="*/30 * * * * $SYNTRACK_BIN collect >> $LOG_FILE 2>&1"

echo "Installing syntrack cron job..."
echo "Binary: $SYNTRACK_BIN"
echo "Log file: $LOG_FILE"

# Check if cron job already exists
if crontab -l 2>/dev/null | grep -q "syntrack collect"; then
    echo "Cron job already exists. Removing old one..."
    crontab -l 2>/dev/null | grep -v "syntrack collect" | crontab -
fi

# Add new cron job
(crontab -l 2>/dev/null; echo "$CRON_JOB") | crontab -

echo "Done! Cron job installed:"
crontab -l | grep syntrack
