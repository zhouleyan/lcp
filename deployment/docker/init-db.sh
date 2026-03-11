#!/bin/bash
set -e

# Apply schema
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f /schema.sql

echo "Database initialized: schema applied"
