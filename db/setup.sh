#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODE="${1:-install}"

fail() {
    printf 'Agentrix DB setup: %s\n' "$1" >&2
    exit 1
}

command -v psql >/dev/null 2>&1 || fail "psql is required"
: "${PGHOST:?set PGHOST explicitly}"
: "${PGPORT:?set PGPORT explicitly}"
: "${PGUSER:?set PGUSER explicitly}"
: "${PGDATABASE:?set PGDATABASE explicitly}"

case "$PGDATABASE" in
    postgres|template0|template1) fail "refusing system database $PGDATABASE" ;;
esac

PSQL=(psql -X -v ON_ERROR_STOP=1)

if [[ "$MODE" == "--rebuild-local-dev" ]]; then
    case "$PGHOST" in
        localhost|127.0.0.1|::1|/var/run/postgresql|/run/postgresql) ;;
        *) fail "local rebuild requires a local PostgreSQL host" ;;
    esac
    [[ "$PGDATABASE" =~ ^agentrix_dev_[A-Za-z0-9_]+$ ]] ||
        fail "local rebuild database must match agentrix_dev_*"
    [[ "${AGENTRIX_CONFIRM_REBUILD:-}" == "$PGDATABASE" ]] ||
        fail "set AGENTRIX_CONFIRM_REBUILD exactly to $PGDATABASE"
    printf 'The following user relations in %s will be removed; CASCADE also removes routines and types in schemas agentrix/public:\n' "$PGDATABASE"
    "${PSQL[@]}" -Atc "
        SELECT n.nspname || '.' || c.relname || ' (' || c.relkind || ')'
          FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname NOT LIKE 'pg_%' AND n.nspname <> 'information_schema'
         ORDER BY 1"
    "${PSQL[@]}" -1 -c "DROP SCHEMA IF EXISTS agentrix CASCADE; DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
elif [[ "$MODE" != "install" ]]; then
    fail "usage: db/setup.sh [install|--rebuild-local-dev]"
fi

"${PSQL[@]}" -f "$SCRIPT_DIR/contracts/empty_database_preflight.sql"

SQL_FILES=(
    "$SCRIPT_DIR/schema/00_foundation.sql"
    "$SCRIPT_DIR/schema/01_identity_teams.sql"
    "$SCRIPT_DIR/schema/02_artifacts_games_policies.sql"
    "$SCRIPT_DIR/schema/03_contests_tasks.sql"
    "$SCRIPT_DIR/schema/04_submissions_builds.sql"
    "$SCRIPT_DIR/schema/05_batches_matches.sql"
    "$SCRIPT_DIR/schema/06_judgements_rejudges.sql"
    "$SCRIPT_DIR/schema/07_scoring_publications.sql"
    "$SCRIPT_DIR/schema/08_communications_awards.sql"
    "$SCRIPT_DIR/functions/00_integrity_triggers.sql"
    "$SCRIPT_DIR/functions/01_submissions.sql"
    "$SCRIPT_DIR/functions/02_batches_matches.sql"
    "$SCRIPT_DIR/functions/03_judgements_rejudges.sql"
    "$SCRIPT_DIR/functions/04_scoring_publication.sql"
    "$SCRIPT_DIR/views/00_scoreboards.sql"
    "$SCRIPT_DIR/functions/05_access_contracts.sql"
)

PSQL_FILES=()
for file in "${SQL_FILES[@]}"; do
    [[ -f "$file" ]] || fail "missing canonical module: $file"
    PSQL_FILES+=(-f "$file")
done

"${PSQL[@]}" -1 "${PSQL_FILES[@]}"
printf 'Agentrix schema installed atomically in %s. Demo fixtures were not loaded.\n' "$PGDATABASE"
