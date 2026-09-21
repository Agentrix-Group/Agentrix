package model

// SnapshotRow is the frozen, public representation of one ranking row inside
// a published snapshot. Its JSON form is hashed, so field order matters.
type SnapshotRow struct {
	Rank              int    `json:"rank"`
	EntryID           string `json:"entry_id"`
	AgentID           string `json:"agent_id"`
	AgentName         string `json:"agent_name"`
	UserID            string `json:"user_id"`
	Username          string `json:"username"`
	Points            int    `json:"points"`
	MatchesPlayed     int    `json:"matches_played"`
	Wins              int    `json:"wins"`
	Draws             int    `json:"draws"`
	Losses            int    `json:"losses"`
	Disqualifications int    `json:"disqualifications"`
	ScoreFor          int    `json:"score_for"`
	ScoreAgainst      int    `json:"score_against"`
}

func (r Ranking) SnapshotRow() SnapshotRow {
	return SnapshotRow{Rank: r.Rank, EntryID: r.EntryID, AgentID: r.AgentID, AgentName: r.AgentName, UserID: r.UserID,
		Username: r.Username, Points: r.Points, MatchesPlayed: r.MatchesPlayed, Wins: r.Wins, Draws: r.Draws, Losses: r.Losses,
		Disqualifications: r.Disqualifications, ScoreFor: r.ScoreFor, ScoreAgainst: r.ScoreAgainst}
}

func (s SnapshotRow) Ranking(contestID string) Ranking {
	return Ranking{ContestID: contestID, EntryID: s.EntryID, AgentID: s.AgentID, AgentName: s.AgentName, UserID: s.UserID,
		Username: s.Username, Rank: s.Rank, Points: s.Points, MatchesPlayed: s.MatchesPlayed, Wins: s.Wins, Draws: s.Draws,
		Losses: s.Losses, Disqualifications: s.Disqualifications, ScoreFor: s.ScoreFor, ScoreAgainst: s.ScoreAgainst}
}
