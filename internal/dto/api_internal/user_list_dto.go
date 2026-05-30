package api_internal

import "time"

// AdminUserListResponse はADMIN用のユーザー一覧APIのレスポンスです。
// プライベートなユーザーも含まれます。
type AdminUserListResponse struct {
	UserName       string    `json:"username"`
	AccountType    string    `json:"account_type"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	PlayerName     *string   `json:"player_name"`
	Rating         *float64  `json:"rating"`
	OverPowerValue *float64  `json:"overpower_value"`
	IsSuspicious   bool      `json:"is_suspicious"`
	IsPrivate      bool      `json:"is_private"`
	FirebaseUID    *string   `json:"firebase_uid"`
	Email          *string   `json:"email"`
}
