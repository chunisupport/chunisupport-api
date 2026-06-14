package info

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
)

const (
	Name                 = "chunisupport-api"
	ConfigDir            = ".config/"
	ResourceDir          = ".resources/"
	MigrationDir         = "migration/mysql/"
	StaticDBFilename     = "static.db"
	BulkInsertChunkSize  = 3000 // 2GB RAM以上を想定。1GB以下なら1000に下げる
	BulkSelectChunkSize  = 1000 // IN句のプレースホルダ上限を避けるための分割数
	DefaultUserListLimit = 100
	DefaultSongListLimit = 100
	GoalMaxPerUser       = 100
	ChartConstMin        = constants.ChartConstMin
	ChartConstMax        = constants.ChartConstMax

	// Goal関連の理論値計算定数
	TheoreticalScore            = constants.TheoreticalScore
	TheoreticalOverpowerBaseAdd = 2.0
	TheoreticalOverpowerScale   = 5.0
	TheoreticalOverpowerBonus   = 5.0

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
	RegisterDataRateLimitRequests   = 1
	RegisterDataRateLimitWindow     = 30 * time.Second
	RecentSignInMaxAge              = 5 * time.Minute
	RecentSignInFutureAllowance     = 1 * time.Minute

	TempDataTTL                  = 5 * time.Minute
	TempDataMaxCompressedBytes   = 512000
	TempDataMaxUncompressedBytes = 512000
	TempDataMaxEntriesPerIP      = 3
	DefaultTempDataMaxTotalMB    = 64
	TempDataRateLimitPerMin      = 30
	TempDataRateLimitWindow      = 1 * time.Minute
	ExternalCORSAllowOrigin      = "https://new.chunithm-net.com"

	// アカウントタイプ定数
	AccountTypePlayer = 1 // 一般ユーザー
	AccountTypeEditor = 2 // 編集者
	AccountTypeAdmin  = 3 // 管理者

	// リクエストボディサイズ上限
	RequestBodyLimit = "5M"

	// DBコネクションプールのデフォルト設定
	DefaultDBMaxOpenConns       = 25
	DefaultDBMaxIdleConns       = 25
	DefaultDBConnMaxLifetimeSec = 300
	DefaultDBConnMaxIdleTimeSec = 60
	DefaultDBStartupMaxWaitSec  = 120
	DefaultDBStartupIntervalSec = 5
)

var (
	BuildDate = "dev" // ビルド日: YYYYMMDD
	Revision  = "dev" // Git短縮ハッシュ: a1b2c3d
)

// 対応アプリバージョン設定
// プレイヤーデータ登録時に、このリストに含まれるバージョンのみ受け付ける
// NOTE: ユーザーが設定ファイルで変更できるようにする必要があれば、example.setting.jsonに追加してください
var SupportedAppVersions = []string{"0.1.0"}

var (
	knownAccountTypes       = make(map[int]struct{})
	roleAllowedAccountTypes = map[int]map[int]struct{}{
		AccountTypePlayer: {
			AccountTypePlayer: {},
			AccountTypeEditor: {},
			AccountTypeAdmin:  {},
		},
		AccountTypeEditor: {
			AccountTypeEditor: {},
			AccountTypeAdmin:  {},
		},
		AccountTypeAdmin: {
			AccountTypeAdmin: {},
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
