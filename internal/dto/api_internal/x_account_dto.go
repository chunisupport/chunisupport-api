package api_internal

// XAccountDTO は連携中の X アカウントの識別情報です。
type XAccountDTO struct {
	// ID は X の数値ユーザーIDです。表示側が現在の連携先と同じアカウントか照合するために返します。
	ID string `json:"id"`
	// Username は @ を除いた X のユーザー名です。
	Username string `json:"username"`
}

// XAccountResponse は GET /internal/me/x-account のレスポンスです。
type XAccountResponse struct {
	// XAccount は一度も取得していない場合 null です。
	XAccount *XAccountDTO `json:"x_account"`
}

// SyncXAccountRequest は X ログイン直後に得た OAuth 1.0a アクセストークンです。
type SyncXAccountRequest struct {
	AccessToken       string `json:"access_token"`
	AccessTokenSecret string `json:"access_token_secret"`
}
