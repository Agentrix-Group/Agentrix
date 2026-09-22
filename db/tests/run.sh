#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DB_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
: "${PGHOST:?set PGHOST explicitly}"
: "${PGPORT:?set PGPORT explicitly}"
: "${PGUSER:?set PGUSER explicitly}"
: "${PGDATABASE:?set PGDATABASE explicitly}"
[[ "$PGDATABASE" =~ ^agentrix_test_[A-Za-z0-9_]+$ ]] || {
    printf 'tests require an existing empty database named agentrix_test_*\n' >&2
    exit 1
}
[[ "${AGENTRIX_CONFIRM_TEST:-}" == "$PGDATABASE" ]] || {
    printf 'set AGENTRIX_CONFIRM_TEST exactly to %s\n' "$PGDATABASE" >&2
    exit 1
}

PSQL=(psql -X -v ON_ERROR_STOP=1)
"$DB_DIR/setup.sh"
if "$DB_DIR/setup.sh" >/dev/null 2>&1; then
    printf 'second installation unexpectedly succeeded\n' >&2
    exit 1
fi
"${PSQL[@]}" -1 -f "$DB_DIR/seed/test_fixtures.sql"
"${PSQL[@]}" -f "$SCRIPT_DIR/00_helpers.sql"

SUITES=(
    01_install_catalog.sql
    02_identity_teams_submissions.sql
    03_artifacts_builds_immutability.sql
    04_batches_matches.sql
    05_execution_fencing_replay.sql
    06_judgements_verification.sql
    07_scoring_policies_equivalence.sql
    08_rejudge_atomicity.sql
    09_freeze_publications.sql
    10_indexes_history_privileges.sql
)
ASSERTIONS=0
for suite in "${SUITES[@]}"; do
    "${PSQL[@]}" -f "$SCRIPT_DIR/$suite"
    COUNT=$(grep -Ec 'test_support\.(assert_true|expect_error)' "$SCRIPT_DIR/$suite" || true)
    ASSERTIONS=$((ASSERTIONS + COUNT))
done
"$SCRIPT_DIR/concurrency.sh"

printf '%s SQL suites plus 2 concurrency suites passed (%s explicit assertions).\n' "${#SUITES[@]}" "$ASSERTIONS"
