package api_internal

import "time"

// FriendRequestCreateRequest はフレンド申請作成リクエストです。
type FriendRequestCreateRequest struct {
	Username string `json:"username" validate:"required,username"`
}

// FriendshipUserResponse はフレンド・申請一覧の相手ユーザー概要です。
type FriendshipUserResponse struct {
	Username    string     `json:"username"`
	PlayerLevel *int       `json:"player_level"`
	PlayerName  *string    `json:"player_name"`
	Rating      *float64   `json:"rating"`
	IsPrivate   bool       `json:"is_private"`
	RequestedAt time.Time  `json:"requested_at"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
}

// FriendshipListResponse はフレンド・申請一覧レスポンスです。
type FriendshipListResponse struct {
	Items []*FriendshipUserResponse `json:"items"`
}
