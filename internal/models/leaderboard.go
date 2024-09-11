package models

type LeaderboardItem struct {
	Username string  `json:"username"`
	UserId   int64   `json:"user_id"`
	Score    float64 `json:"score"`
	Rank     int     `json:"rank,omitempty"`
	Avatar   *string `json:"avatar"`
}

type LeaderboardResponse struct {
	Leaderboard []*LeaderboardItem `json:"leaderboard"`
	Me          *LeaderboardItem   `json:"me"`
}
