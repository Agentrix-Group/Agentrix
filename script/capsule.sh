#!/bin/bash

# Determine project directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

CAPSULE_SQL="$SCRIPT_DIR/capsule.sql"

echo "Assembling database migration script at $CAPSULE_SQL..."

cat > "$CAPSULE_SQL" << 'EOF'
SET FOREIGN_KEY_CHECKS = 0;
EOF

cat "$SCRIPT_DIR/database/user.sql" >> "$CAPSULE_SQL"
cat "$SCRIPT_DIR/database/database.sql" >> "$CAPSULE_SQL"
cat "$SCRIPT_DIR/database/tables.sql" >> "$CAPSULE_SQL"
cat "$SCRIPT_DIR/database/views.sql" >> "$CAPSULE_SQL"

cat "$SCRIPT_DIR/data/permissions.sql" >> "$CAPSULE_SQL"
cat "$SCRIPT_DIR/data/roles.sql" >> "$CAPSULE_SQL"
cat "$SCRIPT_DIR/data/game.sql" >> "$CAPSULE_SQL"
cat "$SCRIPT_DIR/data/reference_agents.sql" >> "$CAPSULE_SQL"

cat >> "$CAPSULE_SQL" << 'EOF'
SET FOREIGN_KEY_CHECKS = 1;
EOF

echo "Migration script assembled successfully ($CAPSULE_SQL)."

if command -v mysql >/dev/null 2>&1; then
    DB_USER="${DB_USER:-root}"
    if [ -n "$DB_PASSWORD" ]; then
        mysql -u "$DB_USER" -p"$DB_PASSWORD" < "$CAPSULE_SQL" 2>/dev/null || echo "Note: MySQL setup skipped or credentials required."
    else
        mysql -u "$DB_USER" < "$CAPSULE_SQL" 2>/dev/null || echo "Note: MySQL setup skipped or credentials required."
    fi
else
    echo "mysql command not found; assembled $CAPSULE_SQL for manual import."
fi
