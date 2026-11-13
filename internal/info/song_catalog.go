package info

import "time"

const (
	// SongCatalogRevisionPollInterval は別プロセス（CLIのsong-batch）による楽曲データ変更を検知する間隔です。
	// 楽曲カタログキャッシュへの反映遅延の上限（＋再構築時間）になります。
	SongCatalogRevisionPollInterval = 10 * time.Second
	// SongCatalogRefreshTimeout は保存成功後の同期的な再構築にかける時間の上限です。
	// 再構築は直列化されるため、DBの応答停止で後続の保存応答が待ち続けないよう打ち切ります。
	SongCatalogRefreshTimeout = 30 * time.Second
)
