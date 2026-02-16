#!/bin/bash
set -e

# Ensure the data directory is writable by the bbs user
# This handles the case where Docker creates the bind mount as root
if [ -d /opt/bbs/data ]; then
    chown -R bbs:bbs /opt/bbs/data
fi

# Drop privileges and run the application as bbs user
exec gosu bbs "$@"
