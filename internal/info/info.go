package info

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
)

const (
	Name                                        = "chunisupport-api"
	ConfigDir                                   = ".config/"
	UsernameForbiddenWordsFile                  = "username_forbidden_words.json"
	ResourceDir                                 = ".resources/"
	MigrationDir                                = "migration/mysql/"
	BulkInsertChunkSize                         = 3000 // 2GB RAM以上を想定。1GB以下なら1000に下げる
	BulkSelectChunkSize                         = 1000 // IN句のプレースホルダ上限を避けるための分割数
	DefaultUserListLimit                        = 100
	AdminUserStatisticsActivePeriod             = 30 * 24 * time.Hour
	DefaultSongListLimit                        = 100
	DefaultBestSlotRankingLimit                 = 50
	MaxBestSlotRankingLimit                     = 100
	GoalMaxPerUser                              = constants.GoalMaxPerUser
	GoalGroupMaxPerUser                         = constants.GoalGroupMaxPerUser
	RecordFilterMaxPerUser                      = constants.RecordFilterMaxPerUser
	RecordFilterNameMaxLength                   = constants.RecordFilterNameMaxLength
	RecordFilterMaxPayloadBytes                 = constants.RecordFilterMaxPayloadBytes
	APITokenMaxPerUser                          = 10
	APITokenRandomByteLength                    = 32
	APITokenPrefixLength                        = 5
	APITokenLastUsedUpdateInterval              = time.Hour
	MaxScoreHistoryEntriesPerChart              = constants.MaxScoreHistoryEntriesPerChart
	MaxOfficialRating                           = constants.MaxOfficialRating
	MaxOfficialOverpower                        = constants.MaxOfficialOverpower
	MaxOfficialOverpowerPercent                 = constants.MaxOfficialOverpowerPercent
	OfficialMetricDecimalScale                  = constants.OfficialMetricDecimalScale
	OfficialMetricDecimalTolerance              = constants.OfficialMetricDecimalTolerance
	ChartConstMin                               = constants.ChartConstMin
	ChartConstMax                               = constants.ChartConstMax
	GoalChartTargetOP                           = "OP_TARGET"
	RainbowRequiredDifficultyMinID              = 1
	RainbowRequiredDifficultyMaxID              = 4
	RainbowRequiredDifficultyCount              = 4
	RandomFavoriteHonorTitle                    = "お気に入りからランダム"
	UnknownSPHonorRegisteredEvent               = "unknown_sp_honor_registered"
	PlayerDataBatchLockName                     = "chunisupport:recalculate-player-data"
	SongSnapshotExportBatchLockName             = "chunisupport:export-song-snapshots"
	SongSnapshotObjectKey                       = "v1/songs.json"
	WorldsendSongSnapshotObjectKey              = "v1/worldsend-songs.json"
	ChunirecSongSnapshotObjectKey               = "compat/chunirec/2.0/music/showall.json"
	ReiwaSongSnapshotObjectKey                  = "compat/reiwa/1/chunithm_record/original.json"
	PlayerLatestUpdateSchemaVersion             = 3
	PlayerLatestUpdateMinSupportedSchemaVersion = 1
	PlayerLatestUpdateMetricDiffSchemaVersion   = 2
	PlayerLatestUpdateOPPercentSchemaVersion    = 3
	PlayerLatestUpdateMaxPayloadBytes           = 1024 * 1024
	MaintenanceRetryAfterSeconds                = 60

	// Goal関連の理論値計算定数
	TheoreticalScore            = constants.TheoreticalScore
	TheoreticalOverpowerBaseAdd = 2.0
	TheoreticalOverpowerScale   = 5.0
	TheoreticalOverpowerBonus   = 5.0

	// レートリミット設定: 外部API v1
	APIRateLimitRequests       = 150              // 一般ユーザーのリクエスト制限（15分間）
	APIRateLimitEditorRequests = 3000             // EDITOR/EXTDEV共用のリクエスト制限（15分間）
	APIRateLimitAdminRequests  = 150000           // ADMINユーザーのリクエスト制限（15分間）
	APIRateLimitWindow         = 15 * time.Minute // レートリミットのウィンドウ期間

	// レートリミット設定: 認証エンドポイント（IPベース）
	LoginRateLimitRequests          = 10              // ログインエンドポイントのリクエスト制限（1分間）
	LoginRateLimitWindow            = 1 * time.Minute // ログインレートリミットのウィンドウ期間
	RegisterRateLimitRequests       = 5               // 登録エンドポイントのリクエスト制限（1分間）
	RegisterRateLimitWindow         = 1 * time.Minute // 登録レートリミットのウィンドウ期間
	InternalPublicRateLimitRequests = 60
	InternalPublicRateLimitWindow   = 1 * time.Minute
	RegisterDataRateLimitRequests   = 1
	RegisterDataRateLimitWindow     = 30 * time.Second
	RecentSignInMaxAge              = 5 * time.Minute
	RecentSignInFutureAllowance     = 1 * time.Minute

	TempDataTTL                  = 5 * time.Minute
	TempDataMaxCompressedBytes   = 512000
	TempDataMaxUncompressedBytes = 5 * 1024 * 1024
	TempDataMaxEntriesPerIP      = 3
	DefaultTempDataMaxTotalMB    = 64
	TempDataRateLimitPerMin      = 30
	TempDataRateLimitWindow      = 1 * time.Minute
	ExternalCORSAllowOrigin      = "https://new.chunithm-net.com"

	// アカウントタイプ定数
	AccountTypePlayer = 1 // 一般ユーザー
	AccountTypeEditor = 2 // 編集者
	AccountTypeAdmin  = 3 // 管理者
	AccountTypeExtDev = 4 // 外部API開発者

	// リクエストボディサイズ上限
	RequestBodyLimit                      = 5 * 1024 * 1024
	DataTransferFormat                    = "chunisupport-user-transfer"
	DataTransferSchemaVersion             = 2
	DataTransferMinSupportedSchemaVersion = 1
	DataTransferHMACSecretMinBytes        = 32
	DataTransferEnvelopeMaxBytes          = 32 * 1024 * 1024
	DataTransferCompressedPayloadMaxBytes = 24 * 1024 * 1024
	DataTransferPayloadMaxBytes           = 128 * 1024 * 1024
	DataTransferRateLimitRequests         = 5
	DataTransferRateLimitWindow           = time.Minute

	// DBコネクションプールのデフォルト設定
	DefaultDBMaxOpenConns       = 25
	DefaultDBMaxIdleConns       = 25
	DefaultDBConnMaxLifetimeSec = 300
	DefaultDBConnMaxIdleTimeSec = 60
	DefaultDBStartupMaxWaitSec  = 120
	DefaultDBStartupIntervalSec = 5

	// プレイヤーお気に入り楽曲
	PlayerFavoriteSongMaxCount = constants.PlayerFavoriteSongMaxCount

	// フレンド機能
	FriendshipMaxOutgoingActive = 100
)

var (
	BuildDate = "dev"  // ビルド日: YYYYMMDD
	Revision  = "none" // Git短縮ハッシュ: a1b2c3d。開発起動時はnone
)

var (
	knownAccountTypes       = make(map[int]struct{})
	roleAllowedAccountTypes = map[int]map[int]struct{}{
		AccountTypePlayer: {
			AccountTypePlayer: {},
			AccountTypeEditor: {},
			AccountTypeAdmin:  {},
			AccountTypeExtDev: {},
		},
		AccountTypeEditor: {
			AccountTypeEditor: {},
			AccountTypeAdmin:  {},
		},
		AccountTypeAdmin: {
			AccountTypeAdmin: {},
		},
		// EXTDEVはPLAYERの操作を利用できる一方、専用ゲートではEXTDEV自身だけを許可する。
		AccountTypeExtDev: {
			AccountTypeExtDev: {},
		},
	}
)

func init() {
	for roleID := range roleAllowedAccountTypes {
		knownAccountTypes[roleID] = struct{}{}
	}
}

// IsKnownAccountType は account_type_id が既知ロールかを判定します。
func IsKnownAccountType(accountTypeID int) bool {
	_, ok := knownAccountTypes[accountTypeID]
	return ok
}

// HasRole は account_type_id が requiredRoleID を満たすかを判定します。
// 未知ロールIDは常に拒否します。
func HasRole(accountTypeID, requiredRoleID int) bool {
	allowedAccountTypes, ok := roleAllowedAccountTypes[requiredRoleID]
	if !ok {
		return false
	}

	_, ok = allowedAccountTypes[accountTypeID]
	return ok
}

// HardLampAbbrevToName はAPI略称→マスタ名（clear_lamp_types.name）への変換テーブルです。
var HardLampAbbrevToName = map[string]string{
	"HRD": "HARD",
	"BRV": "BRAVE",
	"ABS": "ABSOLUTE",
	"CTS": "CATASTROPHY",
}

// HardLampNameToAbbrev はマスタ名→API略称への逆引き変換テーブルです。
var HardLampNameToAbbrev = map[string]string{
	"HARD":        "HRD",
	"BRAVE":       "BRV",
	"ABSOLUTE":    "ABS",
	"CATASTROPHY": "CTS",
}

// ComboLampAbbrevToName はAPI略称→マスタ名（combo_lamp_types.name）への変換テーブルです。
var ComboLampAbbrevToName = map[string]string{
	"FC": "FULL COMBO",
	"AJ": "ALL JUSTICE",
}

// ComboLampNameToAbbrev はマスタ名→API略称への逆引き変換テーブルです。
var ComboLampNameToAbbrev = map[string]string{
	"FULL COMBO":  "FC",
	"ALL JUSTICE": "AJ",
}

// CalcTheoreticalOverpowerTotal は対象譜面群の理論値OVER POWER合計を計算します。
func CalcTheoreticalOverpowerTotal(totalChartConst float64, chartCount int) float64 {
	return (totalChartConst+float64(chartCount)*TheoreticalOverpowerBaseAdd)*TheoreticalOverpowerScale + float64(chartCount)*TheoreticalOverpowerBonus
}
