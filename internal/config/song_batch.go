package config

import (
	"os"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/info"
)

// SongBatchConfig は楽曲データ収集バッチの設定です。
// データソースのURLやシートIDは実行ごとに環境変数から解決するため、ここでは起動時に一度だけ必要な値を保持します。
type SongBatchConfig struct {
	// WikiBaseURL はotoge-dbのwikiwiki_urlからページタイトルを取り出す際に除去するベースURLです。
	// 未設定の場合はWikiページタイトルの補完をスキップします。
	WikiBaseURL string
}

// LoadSongBatchConfigFromEnv は楽曲データ収集バッチの設定を環境変数から読み込みます。
// Wikiページタイトルは補完用途のため、未設定でもバッチ全体は止めません。
func LoadSongBatchConfigFromEnv() SongBatchConfig {
	return SongBatchConfig{
		WikiBaseURL: strings.TrimSpace(os.Getenv(info.SongBatchEnvWikiBaseURL)),
	}
}
