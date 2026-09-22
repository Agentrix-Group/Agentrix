SET search_path = agentrix, pg_catalog;

CREATE VIEW jury_current_score_rows AS
SELECT sr.contest_id, sr.id AS revision_id, sr.revision_number,
       r.division_id, r.contest_entry_id, r.rank, r.total_points,
       r.solved_count, r.total_penalty_seconds, r.total_wins, r.tie_break_value
  FROM score_revisions sr
  JOIN score_rows r ON r.revision_id = sr.id
 WHERE sr.state = 'complete'
   AND sr.revision_number = (
       SELECT max(sr2.revision_number) FROM score_revisions sr2
        WHERE sr2.contest_id = sr.contest_id AND sr2.state = 'complete'
   );

CREATE VIEW latest_published_score_rows AS
SELECT p.contest_id, p.id AS publication_id, p.audience, p.kind,
       p.publication_number, p.cutoff_submitted_at, r.division_id,
       r.contest_entry_id, r.rank, r.total_points, r.solved_count,
       r.total_penalty_seconds, r.total_wins, r.tie_break_value
  FROM scoreboard_publications p
  JOIN scoreboard_publication_rows r ON r.publication_id = p.id
 WHERE p.publication_number = (
       SELECT max(p2.publication_number) FROM scoreboard_publications p2
        WHERE p2.contest_id = p.contest_id AND p2.audience = p.audience
   );

CREATE VIEW claimable_match_jobs AS
SELECT mj.id, mj.match_id, mj.priority, mj.available_at, mj.attempt_count,
       mj.max_attempts, mj.fencing_token
  FROM match_jobs mj
 WHERE (mj.state = 'available' AND mj.available_at <= clock_timestamp())
    OR (mj.state = 'leased' AND mj.lease_expires_at <= clock_timestamp()
        AND mj.attempt_count < mj.max_attempts);

