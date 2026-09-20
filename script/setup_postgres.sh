#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-agentrix}"

echo "=== Agentrix PostgreSQL Setup ==="
echo "Target: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"

# Check if postgres server is accessible
if ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" >/dev/null 2>&1; then
    echo "Error: PostgreSQL server is not responding at $DB_HOST:$DB_PORT"
    exit 1
fi

# Create database if not exists
if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -lqt | cut -d \| -f 1 | grep -qw "$DB_NAME"; then
    echo "Creating database '$DB_NAME'..."
    createdb -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" "$DB_NAME"
    echo "Database '$DB_NAME' created."
else
    echo "Database '$DB_NAME' already exists."
fi

# Apply migrations
echo "Applying canonical database schema via Goose..."
(cd "$SCRIPT_DIR/.." && go run ./cmd/migrate up)

# Apply Seeds
echo "Applying seed data (00_seeds_postgresql.sql)..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$SCRIPT_DIR/data/00_seeds_postgresql.sql"

echo "=== Agentrix PostgreSQL setup complete! ==="
