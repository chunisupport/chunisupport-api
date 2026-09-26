package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
)

// SongBatchDatasourceRef は解決済みデータソースの取得に必要な情報です。
// Params は取得方式ごとの追加情報で、ダウンローダーだけが解釈します。
type SongBatchDatasourceRef struct {
	Type   songbatch.DataSourceType
	URL    string
	Params any
}

// SongBatchDownloadResult はデータソース単位の取得結果です。
type SongBatchDownloadResult struct {
	Type    songbatch.DataSourceType
	Success bool
	Path    string
	Error   string
}

// SongBatchDatasourceResolver はデータソース定義を解決します。
type SongBatchDatasourceResolver interface {
	Resolve(sourceType songbatch.DataSourceType) (SongBatchDatasourceRef, error)
}

// SongBatchDownloader は渡されたディレクトリへデータソースを取得します。
type SongBatchDownloader interface {
	DownloadAll(ctx context.Context, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error)
}

// SongBatchDownloaderFactory は実行専用ディレクトリ向けの SongBatchDownloader を生成します。
type SongBatchDownloaderFactory func(outputDir string) SongBatchDownloader

// SongBatchSourceImporter はファイルからデータソースを読み込みます。
type SongBatchSourceImporter interface {
	Import(sourceType songbatch.DataSourceType, filePath string) (*songbatch.ImportedSource, error)
}

// SongBatchConsolidator はインポート済みソースを統合し、MySQL へ同期します。
// sources は統合順に並んでいる前提です。
type SongBatchConsolidator interface {
	Consolidate(ctx context.Context, sources []songbatch.ImportedSource, req songbatch.RunRequest) error
}
