#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PSQL=(psql -X -v ON_ERROR_STOP=1)
LOG_ONE="$(mktemp /tmp/agentrix-concurrency-one.XXXXXX)"
LOG_TWO="$(mktemp /tmp/agentrix-concurrency-two.XXXXXX)"
trap 'rm -f "$LOG_ONE" "$LOG_TWO"' EXIT

# Prepare exactly one job. Worker one holds the row lock; SKIP LOCKED makes
# worker two return the explicit no-claimable-job error rather than double claim.
"${PSQL[@]}" -c "SET search_path=agentrix,pg_catalog; SELECT seal_evaluation_batch('00000000-0000-0000-0000-000000000901',decode(repeat('41',32),'hex'),now()); SELECT seal_match('00000000-0000-0000-0000-000000000921',now()); SELECT enqueue_match('00000000-0000-0000-0000-000000000941','00000000-0000-0000-0000-000000000921',10,now());"
("${PSQL[@]}" -c "BEGIN; SET search_path=agentrix,pg_catalog; SELECT claim_match_job('90000000-0000-0000-0000-000000000101','worker-one'); SELECT pg_sleep(2); ROLLBACK;" >"$LOG_ONE" 2>&1) &
FIRST_PID=$!
sleep 0.4
if "${PSQL[@]}" -c "SET search_path=agentrix,pg_catalog; SELECT claim_match_job('90000000-0000-0000-0000-000000000102','worker-two');" >"$LOG_TWO" 2>&1; then
    printf 'concurrency failure: two workers claimed the only job\n' >&2
    wait "$FIRST_PID" || true
    exit 1
fi
grep -q 'no claimable match job' "$LOG_TWO"
wait "$FIRST_PID"

# Prepare two reviewed rejudges against the same effective judgement. The
# second transaction waits on the locked submission and must then fail stale.
"${PSQL[@]}" -f "$SCRIPT_DIR/concurrency_rejudge_prepare.sql"
("${PSQL[@]}" -c "BEGIN; SET search_path=agentrix,pg_catalog; SELECT apply_rejudge_batch('90000000-0000-0000-0000-000000000010',now()); SELECT pg_sleep(2); COMMIT;" >"$LOG_ONE" 2>&1) &
FIRST_PID=$!
sleep 0.4
if "${PSQL[@]}" -c "SET search_path=agentrix,pg_catalog; SELECT apply_rejudge_batch('90000000-0000-0000-0000-000000000011',now());" >"$LOG_TWO" 2>&1; then
    printf 'concurrency failure: two rejudges replaced the same effective judgement\n' >&2
    wait "$FIRST_PID" || true
    exit 1
fi
grep -q 'previous judgement is no longer effective' "$LOG_TWO"
wait "$FIRST_PID"
printf '2 concurrency suites passed\n'
