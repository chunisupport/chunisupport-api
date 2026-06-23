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
}

// AdminUserAccountTypeResponse はADMINによるユーザー権限変更後のレスポンスです。
type AdminUserAccountTypeResponse struct {
	ID          int       `json:"id"`
	UserName    string    `json:"username"`
	AccountType string    `json:"account_type"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UpdateUserAccountTypeRequest はユーザー権限変更APIのリクエストです。
type UpdateUserAccountTypeRequest struct {
	AccountType string "json:\"account_type\" validate:\"required,oneof=PLAYER EDITOR ADMIN\""
}
