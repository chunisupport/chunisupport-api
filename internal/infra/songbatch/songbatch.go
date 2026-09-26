// Package songbatch は楽曲データ収集バッチのインフラ実装を組み立てます。
package songbatch

import (
	apirepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	domainsongbatch "github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/consolidation"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/datasource"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/importer"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/registry"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/transaction"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/jmoiron/sqlx"
)

// NewSongBatchUsecase は CLI と API で共通の楽曲バッチユースケースを組み立てます。
func NewSongBatchUsecase(database *sqlx.DB, wikiBaseURL string) *usecase.SongBatchUsecase {
	return usecase.NewSongBatchUsecase(
		registry.Resolver{},
		datasource.NewSongBatchDownloader,
		importer.NewSourceImporter(),
		consolidation.NewBatchRunner(
			database,
			transaction.NewTransactionManager(database),
			repository.NewDifficultyRepository(database),
			repository.NewGenreRepository(database),
			func(executor apirepo.Executor) domainsongbatch.CourseRepository {
				return repository.NewCourseRepository(executor)
			},
			wikiBaseURL,
		),
	)
}
