package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestDatabase_FreshMigrationRollbackAndReapply(t *testing.T) {
	db := freshDB(t)
	ctx := context.Background()
	require.NoError(t, database.CheckSchemaCompatible(ctx, db))
	var legacy int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public'
		AND (column_name IN ('role_id', 'participant_id', 'active', 'run_id', 'replay_id') AND table_name IN
		('users', 'matches', 'contests', 'agents', 'submissions'))`).Scan(&legacy))
	require.Zero(t, legacy, "legacy columns must not exist")
	var participantRole int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM roles WHERE id = 'participant'`).Scan(&participantRole))
	require.Zero(t, participantRole)

	require.NoError(t, database.Reset(db))
	require.ErrorIs(t, database.CheckSchemaCompatible(ctx, db), database.ErrPendingMigrations)
	require.NoError(t, database.Migrate(db))
	require.NoError(t, database.CheckSchemaCompatible(ctx, db))
}

func TestDatabase_SchemaCheckDetectsMissingConstraint(t *testing.T) {
	db := freshDB(t)
	ctx := context.Background()
	_, err := db.Exec(`ALTER TABLE submissions DROP CONSTRAINT uq_submissions_agent_version CASCADE`)
	require.NoError(t, err)
	err = database.CheckSchemaCompatible(ctx, db)
	require.ErrorIs(t, err, database.ErrSchemaDrift)
	require.ErrorContains(t, err, "uq_submissions_agent_version")

	db2 := freshDB(t)
	_, err = db2.Exec(`ALTER TABLE match_runs DISABLE TRIGGER trg_match_runs_transition`)
	require.NoError(t, err)
	require.ErrorIs(t, database.CheckSchemaCompatible(ctx, db2), database.ErrSchemaDrift)
}

// Capability catalog: the Go constants and the seeded rows are identical.
func TestDatabase_CapabilityCatalogMatchesCode(t *testing.T) {
	db := freshDB(t)
	rows, err := db.Query(`SELECT id FROM capabilities ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()
	var seeded []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		seeded = append(seeded, id)
	}
	var code []string
	for _, c := range model.Capabilities {
		code = append(code, string(c))
	}
	sort.Strings(code)
	require.Equal(t, code, seeded)
	var roles []string
	rrows, err := db.Query(`SELECT id FROM roles ORDER BY id`)
	require.NoError(t, err)
	defer rrows.Close()
	for rrows.Next() {
		var id string
		require.NoError(t, rrows.Scan(&id))
		roles = append(roles, id)
	}
	expected := append([]string(nil), model.Roles...)
	sort.Strings(expected)
	require.Equal(t, expected, roles)
}

// Integrity rules hold even for writes that bypass the service.
func TestDatabase_ConstraintsRejectInvalidRows(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	w := e.worker("w1")
	comp := e.competition(w)
	ticket := schedule(t, comp.organizer, comp.matchID)
	exec := func(q string, args ...any) error {
		_, err := e.db.Exec(q, args...)
		return err
	}
	sub := comp.subs[0]
	agentOfEntry := comp.players[0].expect(http.StatusOK, "GET", "/api/v1/submissions/"+sub, nil)["agent_id"].(string)
	v2 := comp.players[0].expect(http.StatusAccepted, "POST", "/api/v1/agents/"+agentOfEntry+"/submissions",
		bundleUpload(t, "v2", referenceBot(t, "bot_random.py")))["id"].(string)
	var agentID, userID string
	require.NoError(t, e.db.QueryRow(`SELECT agent_id, created_by FROM submissions WHERE id = $1`, sub).Scan(&agentID, &userID))

	cases := map[string]error{
		"duplicate submission version": exec(`INSERT INTO submissions (id, agent_id, version, status, runtime, entrypoint, artifact_key,
			artifact_sha256, size_bytes, manifest, created_by, created_at, updated_at)
			SELECT gen_random_uuid(), agent_id, version, 'validating', runtime, entrypoint, artifact_key, artifact_sha256, size_bytes,
			manifest, created_by, NOW(), NOW() FROM submissions WHERE id = $1`, sub),
		"non-positive version": exec(`INSERT INTO submissions (id, agent_id, version, status, runtime, entrypoint, artifact_key,
			artifact_sha256, size_bytes, manifest, created_by, created_at, updated_at)
			SELECT gen_random_uuid(), agent_id, 0, 'validating', runtime, entrypoint, artifact_key, artifact_sha256, size_bytes,
			manifest, created_by, NOW(), NOW() FROM submissions WHERE id = $1`, sub),
		"submission immutable": exec(`UPDATE submissions SET artifact_sha256 = repeat('f', 64),
			artifact_key = 'submissions/sha256/' || repeat('f', 64) || '.py' WHERE id = $1`, sub),
		"contest window":    exec(`UPDATE contests SET starts_at = NOW(), ends_at = NOW() - interval '1 day' WHERE id = $1`, comp.contestID),
		"contest backwards": exec(`UPDATE contests SET state = 'draft' WHERE id = $1`, comp.contestID),
		"roster frozen":     exec(`UPDATE contest_entries SET submission_id = $2 WHERE id = $1`, comp.entries[0], v2),
		"slot immutable":    exec(`UPDATE match_slots SET slot_index = 5 WHERE match_id = $1`, comp.matchID),
		"slot delete":       exec(`DELETE FROM match_slots WHERE match_id = $1`, comp.matchID),
		"duplicate attempt": exec(`INSERT INTO match_runs (id, match_id, attempt, state, execution_spec, execution_spec_hash, engine_sha256, created_at)
			SELECT gen_random_uuid(), match_id, attempt, 'created', execution_spec, execution_spec_hash, engine_sha256, NOW()
			FROM match_runs WHERE id = $1`, ticket["run_id"]),
		"second active job": exec(`INSERT INTO match_runs (id, match_id, attempt, state, execution_spec, execution_spec_hash, engine_sha256, created_at)
			SELECT '00000000-0000-0000-0000-000000000001', match_id, 99, 'created', execution_spec, execution_spec_hash, engine_sha256, NOW()
			FROM match_runs WHERE id = $1;
			INSERT INTO match_jobs (id, match_id, run_id, engine_sha256, state, available_at, created_at, updated_at)
			SELECT gen_random_uuid(), match_id, '00000000-0000-0000-0000-000000000001', engine_sha256, 'pending', NOW(), NOW(), NOW()
			FROM match_runs WHERE id = $1`, ticket["run_id"]),
		"spec immutable":        exec(`UPDATE match_runs SET execution_spec_hash = repeat('0', 64) WHERE id = $1`, ticket["run_id"]),
		"match skips states":    exec(`UPDATE matches SET state = 'finished' WHERE id = $1`, comp.matchID),
		"history not deletable": exec(`DELETE FROM matches WHERE id = $1`, comp.matchID),
		"audit append-only":     exec(`DELETE FROM audit_log`),
		"legacy role":           exec(`INSERT INTO user_roles (user_id, role_id, granted_at) VALUES ($1, 'participant', NOW())`, userID),
		"unnormalized email":    exec(`UPDATE users SET email = 'MIXED@Example.com' WHERE id = $1`, userID),
	}
	for name, err := range cases {
		require.Error(t, err, name)
	}
	_ = agentID
}

// Concurrent uploads to the same agent get distinct consecutive versions.
func TestDatabase_ConcurrentSubmissionVersioning(t *testing.T) {
	e := newEnv(t, envOptions{})
	p := e.user("versioner", model.RolePlayer)
	agent := p.expect(http.StatusCreated, "POST", "/api/v1/agents", map[string]string{"game_id": "starfighter", "name": "V"})
	id := agent["id"].(string)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res := p.do("POST", "/api/v1/agents/"+id+"/submissions", bundleUpload(t, "v", []byte(fmt.Sprintf("print(%d)\n", i))))
			if res.Status != http.StatusAccepted {
				errs <- errors.New(string(res.Body))
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, 8, e.count(`SELECT COUNT(DISTINCT version) FROM submissions WHERE agent_id = $1`, id))
	require.Equal(t, 8, e.count(`SELECT MAX(version) FROM submissions WHERE agent_id = $1`, id))
}
