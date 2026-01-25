package info

import "time"

const (
	Name                 = "chunisupport-api"
	Version              = "0.0.1"
	ConfigDir            = ".config/"
	ResourceDir          = ".resources/"
	MigrationDir         = "migration/mysql/"
	BulkInsertChunkSize  = 3000 // 2GB RAM以上を想定。1GB以下なら1000に下げる
	DefaultUserListLimit = 100
	DefaultSongListLimit = 100
	// CHUNITHMの理論値スコア（score_sumのinvert表示で使用）
	TheoreticalMaxScore = 1010000

	// レートリミット設定: 外部API v1
	APIRateLimitRequests      = 150              // 一般ユーザーのリクエスト制限（15分間）
	APIRateLimitAdminRequests = 150000           // ADMINユーザーのリクエスト制限（15分間）
	APIRateLimitWindow        = 15 * time.Minute // レートリミットのウィンドウ期間

	// レートリミット設定: 認証エンドポイント（IPベース）
	LoginRateLimitRequests          = 10              // ログインエンドポイントのリクエスト制限（1分間）
	LoginRateLimitWindow            = 1 * time.Minute // ログインレートリミットのウィンドウ期間
	RegisterRateLimitRequests       = 5               // 登録エンドポイントのリクエスト制限（1分間）
	RegisterRateLimitWindow         = 1 * time.Minute // 登録レートリミットのウィンドウ期間
	InternalPublicRateLimitRequests = 10
	InternalPublicRateLimitWindow   = 1 * time.Minute

	// リカバリーコード設定
	RecoveryCodeCount             = 10
	RecoveryCodeSegmentLength     = 4
	RecoveryCodeSegmentCount      = 3
	RecoveryCodeRateLimitRequests = 5
	RecoveryCodeRateLimitWindow   = 1 * time.Minute
	RecoveryCodeCharset           = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// パスワード設定
	PasswordMinLength = 8
	PasswordMaxLength = 128

	// リクエストボディサイズ上限
	RequestBodyLimit = "5M"
)
