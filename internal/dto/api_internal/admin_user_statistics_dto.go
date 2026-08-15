package api_internal

// AdminUserStatisticsResponse は管理者向けユーザー集計APIのレスポンスです。
type AdminUserStatisticsResponse struct {
	TotalUsers                 int `json:"total_users"`
	UsersWithPlayerData        int `json:"users_with_player_data"`
	ActivePlayerDataLast30Days int `json:"active_player_data_last_30_days"`
}
