package info

import "time"

// 楽曲データ収集バッチ（song-batch）で使用する定数です。
const (
	// SongBatchLockName は song-batch の全起動経路（CLI・管理画面）で共有する MySQL アドバイザリロック名です。
	SongBatchLockName = "chunisupport:song-batch"
	// SongBatchTempDirPrefix は実行専用一時ディレクトリの MkdirTemp パターンです。
	SongBatchTempDirPrefix = "chunisupport-song-batch-*"
	// SongBatchBulkInsertChunkSize は楽曲・譜面の一括書き込みで1文に含める行数です。
	// SQLite ワークスペースと MySQL の双方で使うため、API 本体の BulkInsertChunkSize とは別に小さめに保ちます。
	SongBatchBulkInsertChunkSize = 500
	// SongBatchSQLiteCompoundSelectLimit は SQLite の UNION ALL 制約を考慮した上限です。
	SongBatchSQLiteCompoundSelectLimit = 400
	// SongBatchJobHistoryLimit は管理画面へ返す実行履歴の件数です。
	SongBatchJobHistoryLimit = 20
	// SongBatchJobFinalizeTimeout は実行結果の記録とロック解放に使う猶予です。
	// 停止シグナルで実行がキャンセルされた後も、中断を記録できるよう独立したタイムアウトを使います。
	SongBatchJobFinalizeTimeout = 10 * time.Second
)

// song-batch が参照する環境変数名です。
// 統合前の song-batch と同じ名前を維持し、既存サーバーの環境設定をそのまま使えるようにしています。
const (
	songBatchEnvPrefix = "CHUNISUPPORT_BATCH_"

	// SongBatchEnvOfficialURL は公式データソースのURLを指す環境変数名です。
	SongBatchEnvOfficialURL = songBatchEnvPrefix + "OFFICIAL_URL"
	// SongBatchEnvOtogeDbURL はotoge-dbデータソースのURLを指す環境変数名です。
	SongBatchEnvOtogeDbURL = songBatchEnvPrefix + "OTOGE_DB_URL"
	// SongBatchEnvGoogleCloudAPIKey はGoogle Cloud APIキーを指す環境変数名です。
	SongBatchEnvGoogleCloudAPIKey = songBatchEnvPrefix + "GOOGLE_CLOUD_API_KEY"
	// SongBatchEnvGoogleSheetID はmainframeのGoogle Sheet IDを指す環境変数名です。
	SongBatchEnvGoogleSheetID = songBatchEnvPrefix + "GOOGLE_SHEET_ID"
	// SongBatchEnvGoogleSpreadsheetBaseURL はGoogle Spreadsheet APIのベースURLを指す環境変数名です。
	SongBatchEnvGoogleSpreadsheetBaseURL = songBatchEnvPrefix + "GOOGLE_SPREADSHEET_BASE_URL"
	// SongBatchEnvAdditionalSongsSheetID は追加楽曲用Google Sheet IDを指す環境変数名です。
	SongBatchEnvAdditionalSongsSheetID = songBatchEnvPrefix + "ADDITIONAL_SONGS_SHEET_ID"
	// SongBatchEnvSt1027URL はst1027データソースのURLを指す環境変数名です。
	SongBatchEnvSt1027URL = songBatchEnvPrefix + "ST1027_URL"
	// SongBatchEnvWikiBaseURL はwikiwiki_urlからページタイトルを取り出す際に除去するWikiのベースURLを指す環境変数名です。
	SongBatchEnvWikiBaseURL = songBatchEnvPrefix + "WIKI_BASE_URL"
)
