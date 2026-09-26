package consolidation

import (
	"context"
	"fmt"

	apirepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

// BatchRunner は統合と MySQL 同期を実行します。
type BatchRunner struct {
	db             apirepo.Executor
	tm             usecase.TransactionManager
	difficultyRepo songbatch.DifficultyRepository
	genreRepo      songbatch.GenreRepository
	newCourseRepo  func(apirepo.Executor) songbatch.CourseRepository
	wikiBaseURL    string
}

var _ usecase.SongBatchConsolidator = (*BatchRunner)(nil)

// NewBatchRunner は BatchRunner を生成します。
func NewBatchRunner(
	db apirepo.Executor,
	tm usecase.TransactionManager,
	difficultyRepo songbatch.DifficultyRepository,
	genreRepo songbatch.GenreRepository,
	newCourseRepo func(apirepo.Executor) songbatch.CourseRepository,
	wikiBaseURL string,
) *BatchRunner {
	return &BatchRunner{
		db:             db,
		tm:             tm,
		difficultyRepo: difficultyRepo,
		genreRepo:      genreRepo,
		newCourseRepo:  newCourseRepo,
		wikiBaseURL:    wikiBaseURL,
	}
}

// Consolidate は必須条件を満たしたソースをワークスペース経由で同期します。
// ワークスペースの構築は MySQL トランザクションの外で行い、長時間のトランザクションを避けます。
func (r *BatchRunner) Consolidate(ctx context.Context, imported []songbatch.ImportedSource, req songbatch.RunRequest) error {
	var sources ConsolidationSources
	names := make([]songbatch.DataSourceType, 0, len(imported))
	for _, source := range imported {
		if err := assignSource(&sources, source); err != nil {
			return err
		}
		names = append(names, source.Type)
	}

	opts := ConsolidationOptions{
		MajorUpdate:            req.Mode == songbatch.RunModeMajorUpdate,
		FillMissingReleaseDate: req.FillMissingReleaseDate,
	}
	svc := NewConsolidationService(r.db, r.difficultyRepo, r.genreRepo, r.wikiBaseURL, names, opts, sources)
	workspace, err := svc.BuildWorkspace(ctx)
	if err != nil {
		return err
	}
	if workspace == nil {
		return fmt.Errorf("no datasources to consolidate")
	}
	defer workspace.Close()

	return r.tm.Transactional(ctx, func(tx apirepo.Executor) error {
		return svc.SyncWorkspace(ctx, workspace, tx, r.newCourseRepo(tx))
	})
}
