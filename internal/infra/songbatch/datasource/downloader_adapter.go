package datasource

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

// NewSongBatchDownloader は実行専用ディレクトリへ取得する usecase.SongBatchDownloader を生成します。
func NewSongBatchDownloader(outputDir string) usecase.SongBatchDownloader {
	return downloaderAdapter{inner: NewDownloader(outputDir)}
}

type downloaderAdapter struct {
	inner *Downloader
}

func (a downloaderAdapter) DownloadAll(ctx context.Context, datasources []usecase.SongBatchDatasourceRef) ([]usecase.SongBatchDownloadResult, error) {
	converted := make([]Datasource, len(datasources))
	for i, ds := range datasources {
		converted[i] = Datasource{Type: string(ds.Type), URL: ds.URL, Params: ds.Params}
	}
	results, err := a.inner.DownloadAll(ctx, converted)
	if err != nil {
		return nil, err
	}
	out := make([]usecase.SongBatchDownloadResult, len(results))
	for i, result := range results {
		out[i] = usecase.SongBatchDownloadResult{
			Type:    songbatch.DataSourceType(result.Type),
			Success: result.Success,
			Path:    result.Path,
			Error:   result.Error,
		}
	}
	return out, nil
}
