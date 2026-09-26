package registry

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

// Resolver は登録済みプロバイダーでデータソース定義を解決します。
type Resolver struct{}

var _ usecase.SongBatchDatasourceResolver = Resolver{}

// Resolve は指定データソースの定義を環境変数から組み立てます。
func (Resolver) Resolve(sourceType songbatch.DataSourceType) (usecase.SongBatchDatasourceRef, error) {
	ds, err := Resolve(string(sourceType))
	if err != nil {
		return usecase.SongBatchDatasourceRef{}, err
	}
	return usecase.SongBatchDatasourceRef{Type: songbatch.DataSourceType(ds.Type), URL: ds.URL, Params: ds.Params}, nil
}
