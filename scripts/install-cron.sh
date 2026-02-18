#!/bin/bash

SYNTRACK_BIN="$(pwd)/syntrack"
CRON_JOB="*/30 * * * * $SYNTRACK_BIN collect >> /tmp/syntrack.log 2>&1"

echo "Installing syntrack cron job..."
echo "Binary: $SYNTRACK_BIN"

# Check if cron job already exists
if crontab -l 2>/dev/null | grep -q "syntrack collect"; then
    echo "Cron job already exists. Removing old one..."
    crontab -l 2>/dev/null | grep -v "syntrack collect" | crontab -
fi

# Add new cron job
(crontab -l 2>/dev/null; echo "$CRON_JOB") | crontab -

echo "Done! Cron job installed:"
crontab -l | grep syntrack
