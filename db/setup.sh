#!/bin/bash
set -e

# ==============================================================================
# Agentrix Modular Database Setup & Compilation
# Inspired by Capibara modular layout, adapted natively for PostgreSQL.
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_SQL="$SCRIPT_DIR/init.sql"

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-agentrix}"

echo "=== Compiling modular database files into $OUTPUT_SQL ==="

# 1. Reset / initialize consolidated SQL file
cat > "$OUTPUT_SQL" << 'EOF'
-- ==============================================================================
-- Consolidated Agentrix Database Initialization & Seeds
-- Generated from modular schema files in db/database/ and db/data/
-- ==============================================================================
EOF

# 2. Database & User definitions
cat "$SCRIPT_DIR/database/user.sql" >> "$OUTPUT_SQL"
echo "" >> "$OUTPUT_SQL"
cat "$SCRIPT_DIR/database/database.sql" >> "$OUTPUT_SQL"
echo "" >> "$OUTPUT_SQL"
cat "$SCRIPT_DIR/database/tables.sql" >> "$OUTPUT_SQL"
echo "" >> "$OUTPUT_SQL"
cat "$SCRIPT_DIR/database/views.sql" >> "$OUTPUT_SQL"
echo "" >> "$OUTPUT_SQL"

# 3. Seed data ordered by relational dependencies
DATA_FILES=(
    "roles.sql"
    "permissions.sql"
    "role_permissions.sql"
    "users.sql"
    "user_roles.sql"
    "categories.sql"
    "games.sql"
    "contests.sql"
    "agents.sql"
    "submissions.sql"
    "contest_entries.sql"
    "matches.sql"
    "rankings.sql"
    "results.sql"
    "replays.sql"
)

for file in "${DATA_FILES[@]}"; do
    if [ -f "$SCRIPT_DIR/data/$file" ]; then
        echo "" >> "$OUTPUT_SQL"
        echo "-- Data: $file" >> "$OUTPUT_SQL"
        cat "$SCRIPT_DIR/data/$file" >> "$OUTPUT_SQL"
    fi
done

echo "Compilation complete: $OUTPUT_SQL"

# 4. Optional execution against live PostgreSQL if --apply is passed
if [ "$1" = "--apply" ]; then
    echo "=== Applying to PostgreSQL ($DB_USER@$DB_HOST:$DB_PORT) ==="
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -f "$OUTPUT_SQL"
    echo "=== Database setup applied successfully! ==="
else
    echo "To apply this to your PostgreSQL instance, run:"
    echo "  $0 --apply"
    echo "or directly with psql:"
    echo "  psql -h $DB_HOST -p $DB_PORT -U $DB_USER -f $OUTPUT_SQL"
fi
