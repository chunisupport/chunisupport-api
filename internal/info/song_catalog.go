package info

import "time"

const (
	// SongCatalogRevisionPollInterval は別プロセス（CLIのsong-batch）による楽曲データ変更を検知する間隔です。
	// 楽曲カタログキャッシュへの反映遅延の上限（＋再構築時間）になります。
	SongCatalogRevisionPollInterval = 10 * time.Second
	// SongCatalogRefreshTimeout は1回の再構築（先行する再構築の待機を含む）にかける時間の上限です。
	// 保存後の同期再構築と定期確認の両方に適用し、DBの応答停止で保存応答が待ち続けないよう打ち切ります。
	SongCatalogRefreshTimeout = 30 * time.Second
)
