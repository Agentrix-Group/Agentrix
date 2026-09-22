-- Concrete acceptance-test data. These are not product defaults and are never
-- loaded by db/setup.sh.
SET search_path = agentrix, pg_catalog;

INSERT INTO users(id, username, email, status, created_at, updated_at) VALUES
('00000000-0000-0000-0000-000000000001', 'Alice', 'alice@example.test', 'active', now() - interval '30 days', now()),
('00000000-0000-0000-0000-000000000002', 'Bob', 'bob@example.test', 'active', now() - interval '30 days', now()),
('00000000-0000-0000-0000-000000000003', 'Juror', 'juror@example.test', 'active', now() - interval '30 days', now()),
('00000000-0000-0000-0000-000000000004', 'Outsider', 'outsider@example.test', 'active', now() - interval '30 days', now()),
('00000000-0000-0000-0000-000000000005', 'Disabled', 'disabled@example.test', 'disabled', now() - interval '30 days', now(), now());

INSERT INTO user_credentials(id, user_id, password_hash, algorithm, algorithm_parameters, created_at) VALUES
('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', decode(repeat('aa', 32), 'hex'), 'argon2id', '{"memory_kib":65536,"iterations":3,"parallelism":1}', now() - interval '10 days');

INSERT INTO roles(id, key, scope_kind, description) VALUES
('00000000-0000-0000-0000-000000000021', 'organizer', 'both', 'Contest and platform organizer'),
('00000000-0000-0000-0000-000000000022', 'jury', 'contest', 'Contest jury'),
('00000000-0000-0000-0000-000000000023', 'competitor', 'contest', 'Competitor'),
('00000000-0000-0000-0000-000000000024', 'spectator', 'contest', 'Spectator'),
('00000000-0000-0000-0000-000000000025', 'worker_service', 'global', 'Technical worker identity');
INSERT INTO permissions(id, key, description) VALUES
('00000000-0000-0000-0000-000000000031', 'contest.judge', 'Verify judgements'),
('00000000-0000-0000-0000-000000000032', 'contest.rejudge', 'Apply reviewed rejudges'),
('00000000-0000-0000-0000-000000000033', 'contest.publish_scoreboard', 'Publish scoreboard'),
('00000000-0000-0000-0000-000000000034', 'submission.create', 'Submit for an active team'),
('00000000-0000-0000-0000-000000000035', 'match.execute', 'Lease and complete match jobs');
INSERT INTO role_permissions(role_id, permission_id) VALUES
('00000000-0000-0000-0000-000000000022', '00000000-0000-0000-0000-000000000031'),
('00000000-0000-0000-0000-000000000022', '00000000-0000-0000-0000-000000000032'),
('00000000-0000-0000-0000-000000000021', '00000000-0000-0000-0000-000000000033'),
('00000000-0000-0000-0000-000000000023', '00000000-0000-0000-0000-000000000034'),
('00000000-0000-0000-0000-000000000025', '00000000-0000-0000-0000-000000000035');

INSERT INTO teams(id, slug, display_name, status, created_by_user_id, created_at) VALUES
('00000000-0000-0000-0000-000000000101', 'alpha', 'Alpha', 'active', '00000000-0000-0000-0000-000000000001', now() - interval '20 days'),
('00000000-0000-0000-0000-000000000102', 'beta', 'Beta', 'active', '00000000-0000-0000-0000-000000000002', now() - interval '20 days');
INSERT INTO team_memberships(id, team_id, user_id, member_role, joined_at, granted_by_user_id) VALUES
('00000000-0000-0000-0000-000000000111', '00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000001', 'owner', now() - interval '20 days', '00000000-0000-0000-0000-000000000001'),
('00000000-0000-0000-0000-000000000112', '00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000002', 'owner', now() - interval '20 days', '00000000-0000-0000-0000-000000000002');

INSERT INTO artifacts(id, kind, publication_state, sha256, size_bytes, media_type, storage_key, created_at, verified_at) VALUES
('00000000-0000-0000-0000-000000000201', 'manifest', 'ready', decode(repeat('01',32),'hex'), 101, 'application/json', 'fixtures/manifest', now()-interval '5 days', now()-interval '5 days'),
('00000000-0000-0000-0000-000000000202', 'engine', 'ready', decode(repeat('02',32),'hex'), 102, 'application/octet-stream', 'fixtures/engine', now()-interval '5 days', now()-interval '5 days'),
('00000000-0000-0000-0000-000000000203', 'rules', 'ready', decode(repeat('03',32),'hex'), 103, 'application/octet-stream', 'fixtures/rules', now()-interval '5 days', now()-interval '5 days'),
('00000000-0000-0000-0000-000000000204', 'baseline', 'ready', decode(repeat('04',32),'hex'), 104, 'application/octet-stream', 'fixtures/baseline', now()-interval '5 days', now()-interval '5 days'),
('00000000-0000-0000-0000-000000000205', 'source', 'ready', decode(repeat('05',32),'hex'), 105, 'text/plain', 'fixtures/source-a', now()-interval '2 hours', now()-interval '2 hours'),
('00000000-0000-0000-0000-000000000206', 'source', 'ready', decode(repeat('06',32),'hex'), 106, 'text/plain', 'fixtures/source-b', now()-interval '2 hours', now()-interval '2 hours'),
('00000000-0000-0000-0000-000000000207', 'executable', 'ready', decode(repeat('07',32),'hex'), 107, 'application/octet-stream', 'fixtures/exe-a', now()-interval '1 hour', now()-interval '1 hour'),
('00000000-0000-0000-0000-000000000208', 'executable', 'ready', decode(repeat('08',32),'hex'), 108, 'application/octet-stream', 'fixtures/exe-b', now()-interval '1 hour', now()-interval '1 hour'),
('00000000-0000-0000-0000-000000000209', 'replay', 'ready', decode(repeat('09',32),'hex'), 109, 'application/x-agentrix-replay', 'fixtures/replay', now()-interval '30 minutes', now()-interval '30 minutes'),
('00000000-0000-0000-0000-000000000211', 'source', 'ready', decode(repeat('0b',32),'hex'), 111, 'text/plain', 'fixtures/source-c', now()-interval '10 minutes', now()-interval '10 minutes');
INSERT INTO artifacts(id, kind, publication_state, created_at) VALUES
('00000000-0000-0000-0000-000000000210', 'source', 'pending', now());

INSERT INTO games(id, slug, display_name, description, status, created_at) VALUES
('00000000-0000-0000-0000-000000000301', 'starfighter', 'Starfighter', 'Fixture game', 'active', now()-interval '10 days');
INSERT INTO game_releases(id, game_id, version, protocol_version, min_players, max_players,
    manifest_artifact_id, engine_artifact_id, rules_artifact_id,
    manifest_digest, engine_digest, rules_digest, published_at) VALUES
('00000000-0000-0000-0000-000000000302', '00000000-0000-0000-0000-000000000301', 'fixture-1', '1', 2, 4,
 '00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000203',
 decode(repeat('01',32),'hex'), decode(repeat('02',32),'hex'), decode(repeat('03',32),'hex'), now()-interval '4 days');

INSERT INTO evaluation_policy_versions(id, name, kind, algorithm_version, parameters,
    parameters_digest, cpu_limit_ms, memory_limit_bytes, tick_limit,
    max_match_attempts, lease_duration_seconds, created_at) VALUES
('00000000-0000-0000-0000-000000000401', 'fixed fixture', 'fixed_cases', 1,
 '{"suite_version":"fixture-v1"}', decode(repeat('11',32),'hex'), 10000, 268435456, 10000, 3, 60, now()-interval '4 days');
INSERT INTO scoring_policy_versions(id, name, kind, algorithm_version, parameters,
    parameters_digest, score_unit, disqualification_task_points, tie_method, created_at) VALUES
('00000000-0000-0000-0000-000000000402', 'best fixture', 'best_score', 1,
 '{"higher_is_better":true}', decode(repeat('12',32),'hex'), 'task_points', 0, 'rank', now()-interval '4 days'),
('00000000-0000-0000-0000-000000000403', 'icpc fixture', 'icpc_pass_fail', 1,
 '{"wrong_submission_penalty_minutes":20,"higher_is_better":true}', decode(repeat('13',32),'hex'), 'task_points', 0, 'rank', now()-interval '4 days'),
('00000000-0000-0000-0000-000000000404', 'aggregate fixture', 'aggregate_points', 1,
 '{"higher_is_better":true,"group_aggregation":"weighted_sum"}', decode(repeat('14',32),'hex'), 'task_points', 0, 'dense_rank', now()-interval '4 days'),
('00000000-0000-0000-0000-000000000405', 'league fixture', 'league_points', 1,
 '{"win_points":3,"draw_points":1,"loss_points":0}', decode(repeat('15',32),'hex'), 'league_points', 0, 'rank', now()-interval '4 days'),
('00000000-0000-0000-0000-000000000406', 'best fixture v2', 'best_score', 2,
 '{"higher_is_better":true}', decode(repeat('16',32),'hex'), 'task_points', 0, 'rank', now()-interval '1 day');

INSERT INTO languages(id, key, display_name, source_size_limit_bytes, active) VALUES
('00000000-0000-0000-0000-000000000501', 'python', 'Python fixture', 1048576, true);
INSERT INTO toolchains(id, language_id, version, image_digest, command_contract_digest, active) VALUES
('00000000-0000-0000-0000-000000000502', '00000000-0000-0000-0000-000000000501', 'fixture', decode(repeat('21',32),'hex'), decode(repeat('22',32),'hex'), true);
INSERT INTO baseline_programs(id, game_release_id, name, version, executable_artifact_id,
    executable_digest, protocol_version, published_at) VALUES
('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000302', 'reference', '1',
 '00000000-0000-0000-0000-000000000204', decode(repeat('04',32),'hex'), '1', now()-interval '3 days');

INSERT INTO contests(id, slug, title, state, registration_opens_at, registration_closes_at,
    submission_opens_at, starts_at, freeze_at, submission_closes_at, ends_at, unfreeze_at,
    late_registration_policy, entry_uniqueness_policy, default_submission_limit_per_task,
    code_visibility, private_feedback_policy, rank_method, created_by_user_id, created_at) VALUES
('00000000-0000-0000-0000-000000000701', 'fixture-cup', 'Fixture Cup', 'running',
 now()-interval '10 days', now()-interval '2 days', now()-interval '1 day', now()-interval '12 hours',
 now()+interval '2 hours', now()+interval '1 day', now()+interval '2 days', now()+interval '2 days',
 'reject', 'one_per_team', 10, 'jury_only', 'verdict_only', 'rank',
 '00000000-0000-0000-0000-000000000003', now()-interval '10 days'),
('00000000-0000-0000-0000-000000000702', 'other-cup', 'Other Cup', 'running',
 now()-interval '10 days', now()-interval '2 days', now()-interval '1 day', now()-interval '12 hours',
 NULL, now()+interval '1 day', now()+interval '2 days', NULL,
 'reject', 'one_per_team', NULL, 'team_only', 'hidden', 'dense_rank',
 '00000000-0000-0000-0000-000000000003', now()-interval '10 days');
INSERT INTO contest_divisions(id, contest_id, key, title, sort_order, award_eligible) VALUES
('00000000-0000-0000-0000-000000000711', '00000000-0000-0000-0000-000000000701', 'open', 'Open', 1, true),
('00000000-0000-0000-0000-000000000713', '00000000-0000-0000-0000-000000000702', 'open', 'Open', 1, true);
INSERT INTO contest_entries(id, contest_id, team_id, division_id, status, eligibility, registered_at, accepted_at) VALUES
('00000000-0000-0000-0000-000000000721', '00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000711', 'accepted', 'eligible', now()-interval '3 days', now()-interval '3 days'),
('00000000-0000-0000-0000-000000000722', '00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000711', 'accepted', 'eligible', now()-interval '3 days', now()-interval '3 days'),
('00000000-0000-0000-0000-000000000723', '00000000-0000-0000-0000-000000000702', '00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000713', 'accepted', 'eligible', now()-interval '3 days', now()-interval '3 days');
INSERT INTO contest_user_roles(contest_id, user_id, role_id, granted_by_user_id, granted_at) VALUES
('00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000022', '00000000-0000-0000-0000-000000000003', now()-interval '5 days');

INSERT INTO contest_tasks(id, contest_id, code, title, position, game_release_id,
    evaluation_policy_version_id, scoring_policy_version_id, max_task_points,
    submission_limit, allow_submit, allow_judge, requires_verified_replay, visibility, published_at) VALUES
('00000000-0000-0000-0000-000000000731', '00000000-0000-0000-0000-000000000701', 'A', 'Best score', 1, '00000000-0000-0000-0000-000000000302', '00000000-0000-0000-0000-000000000401', '00000000-0000-0000-0000-000000000402', 100, NULL, true, true, true, 'public', now()-interval '2 days'),
('00000000-0000-0000-0000-000000000732', '00000000-0000-0000-0000-000000000701', 'B', 'ICPC', 2, '00000000-0000-0000-0000-000000000302', '00000000-0000-0000-0000-000000000401', '00000000-0000-0000-0000-000000000403', 1, NULL, true, true, false, 'registered', now()-interval '2 days'),
('00000000-0000-0000-0000-000000000733', '00000000-0000-0000-0000-000000000702', 'A', 'Other task', 1, '00000000-0000-0000-0000-000000000302', '00000000-0000-0000-0000-000000000401', '00000000-0000-0000-0000-000000000404', 100, NULL, true, true, false, 'public', now()-interval '2 days');
INSERT INTO task_toolchains(contest_id, task_id, toolchain_id, enabled) VALUES
('00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000731', '00000000-0000-0000-0000-000000000502', true),
('00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000732', '00000000-0000-0000-0000-000000000502', true),
('00000000-0000-0000-0000-000000000702', '00000000-0000-0000-0000-000000000733', '00000000-0000-0000-0000-000000000502', true);
INSERT INTO test_groups(id, contest_id, task_id, key, weight, aggregation, visibility) VALUES
('00000000-0000-0000-0000-000000000741', '00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000731', 'main', 1, 'sum', 'private');
INSERT INTO test_cases(id, task_id, group_id, key, seed, weight, visibility) VALUES
('00000000-0000-0000-0000-000000000751', '00000000-0000-0000-0000-000000000731', '00000000-0000-0000-0000-000000000741', 'case-1', 42, 1, 'private');
INSERT INTO task_baselines(task_id, baseline_program_id, purpose) VALUES
('00000000-0000-0000-0000-000000000731', '00000000-0000-0000-0000-000000000601', 'opponent');

INSERT INTO submissions(id, contest_id, contest_entry_id, contest_task_id, submitted_by_user_id,
    source_artifact_id, toolchain_id, submitted_at, received_at, idempotency_key, disposition) VALUES
('00000000-0000-0000-0000-000000000801', '00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000721', '00000000-0000-0000-0000-000000000731', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000205', '00000000-0000-0000-0000-000000000502', now()-interval '90 minutes', now()-interval '90 minutes', 'fixture-a', 'eligible'),
('00000000-0000-0000-0000-000000000802', '00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000722', '00000000-0000-0000-0000-000000000731', '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000206', '00000000-0000-0000-0000-000000000502', now()-interval '80 minutes', now()-interval '80 minutes', 'fixture-b', 'eligible');
INSERT INTO build_attempts(id, submission_id, attempt_number, toolchain_id, state,
    executable_artifact_id, started_at, ended_at, created_at) VALUES
('00000000-0000-0000-0000-000000000811', '00000000-0000-0000-0000-000000000801', 1, '00000000-0000-0000-0000-000000000502', 'succeeded', '00000000-0000-0000-0000-000000000207', now()-interval '70 minutes', now()-interval '69 minutes', now()-interval '70 minutes'),
('00000000-0000-0000-0000-000000000812', '00000000-0000-0000-0000-000000000802', 1, '00000000-0000-0000-0000-000000000502', 'succeeded', '00000000-0000-0000-0000-000000000208', now()-interval '70 minutes', now()-interval '69 minutes', now()-interval '70 minutes');

INSERT INTO evaluation_batches(id, contest_id, contest_task_id, game_release_id,
    evaluation_policy_version_id, scoring_policy_version_id, purpose, reference_kind,
    state, created_by_user_id, created_at) VALUES
('00000000-0000-0000-0000-000000000901', '00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000731', '00000000-0000-0000-0000-000000000302', '00000000-0000-0000-0000-000000000401', '00000000-0000-0000-0000-000000000402', 'official', 'sealed_roster', 'draft', '00000000-0000-0000-0000-000000000003', now()-interval '20 minutes');
INSERT INTO batch_roster(id, batch_id, contest_id, contest_task_id, source_kind,
    contest_entry_id, submission_id, build_attempt_id, baseline_program_id,
    executable_artifact_id, executable_digest, added_at) VALUES
('00000000-0000-0000-0000-000000000911', '00000000-0000-0000-0000-000000000901', '00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000731', 'submission', '00000000-0000-0000-0000-000000000721', '00000000-0000-0000-0000-000000000801', '00000000-0000-0000-0000-000000000811', NULL, '00000000-0000-0000-0000-000000000207', decode(repeat('07',32),'hex'), now()-interval '19 minutes'),
('00000000-0000-0000-0000-000000000912', '00000000-0000-0000-0000-000000000901', '00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000731', 'submission', '00000000-0000-0000-0000-000000000722', '00000000-0000-0000-0000-000000000802', '00000000-0000-0000-0000-000000000812', NULL, '00000000-0000-0000-0000-000000000208', decode(repeat('08',32),'hex'), now()-interval '19 minutes'),
('00000000-0000-0000-0000-000000000913', '00000000-0000-0000-0000-000000000901', '00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000731', 'baseline', NULL, NULL, NULL, '00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000204', decode(repeat('04',32),'hex'), now()-interval '19 minutes');
INSERT INTO matches(id, batch_id, contest_id, contest_task_id, game_release_id,
    test_case_id, purpose, seed, schedule_key, specification_digest,
    allow_duplicate_programs, state, created_at) VALUES
('00000000-0000-0000-0000-000000000921', '00000000-0000-0000-0000-000000000901', '00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000731', '00000000-0000-0000-0000-000000000302', '00000000-0000-0000-0000-000000000751', 'official', 42, 'fixture-match', decode(repeat('31',32),'hex'), false, 'planned', now()-interval '18 minutes');
INSERT INTO match_seats(id, match_id, batch_id, roster_item_id, seat_index, created_at) VALUES
('00000000-0000-0000-0000-000000000931', '00000000-0000-0000-0000-000000000921', '00000000-0000-0000-0000-000000000901', '00000000-0000-0000-0000-000000000911', 0, now()-interval '17 minutes'),
('00000000-0000-0000-0000-000000000932', '00000000-0000-0000-0000-000000000921', '00000000-0000-0000-0000-000000000901', '00000000-0000-0000-0000-000000000912', 1, now()-interval '17 minutes');
