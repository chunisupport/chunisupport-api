package usecase

import "time"

// UserOutput は認証・アカウント操作で返すユーザー情報です。
// API固有の表現を持たず、Handlerが公開DTOへ変換します。
type UserOutput struct {
	Username        string
	AccountType     string
	IsPrivate       bool
	LastScoreUpdate *time.Time
}
