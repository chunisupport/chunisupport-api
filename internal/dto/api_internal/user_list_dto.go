package api_internal

// AdminUserListResponse はADMIN用のユーザー一覧APIのレスポンスです。
// プライベートなユーザーや削除済みユーザーも含まれます。
type AdminUserListResponse struct {
	UserName       string   `json:"username"`
	PlayerName     string   `json:"player_name"`
	Rating         *float64 `json:"rating"`
	OverPowerValue *float64 `json:"overpower_value"`
	IsPrivate      bool     `json:"is_private"`
	IsDeleted      bool     `json:"is_deleted"`
}
