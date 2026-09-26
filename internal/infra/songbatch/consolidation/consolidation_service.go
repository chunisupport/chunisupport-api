// Package consolidation は取り込み済みの楽曲データソースを SQLite ワークスペースで統合し、MySQL へ同期します。
package consolidation

import (
	"context"
	"fmt"
	"log/slog"

	apirepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/entity"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/importer"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/songchart"
)

// ConsolidationOptions は統合処理の挙動を制御します。
type ConsolidationOptions struct {
	MajorUpdate            bool
	FillMissingReleaseDate bool // 特定フラグ有効時、データソース・MySQL両方に日付のない新規楽曲へ実行日(JST)を補完
}

// ConsolidationSources は統合に利用するソースデータを保持します。
type ConsolidationSources struct {
	Official        *importer.OfficialData
	AdditionalSongs *importer.AdditionalSongsData
	St1027          *importer.St1027Data
	Mainframe       *importer.MainframeData
	OtogeDb         *importer.OtogeDbData
}

// ConsolidationService はデータソースの統合処理を管理します。
type ConsolidationService struct {
	db             apirepo.Executor
	difficultyRepo songbatch.DifficultyRepository
	genreRepo      songbatch.GenreRepository
	wikiBaseURL    string
	datasources    []songbatch.DataSourceType
	opts           ConsolidationOptions
	sources        ConsolidationSources
}

// NewConsolidationService は新しいConsolidationServiceのインスタンスを生成します。
func NewConsolidationService(
	db apirepo.Executor,
	difficultyRepo songbatch.DifficultyRepository,
	genreRepo songbatch.GenreRepository,
	wikiBaseURL string,
	datasources []songbatch.DataSourceType,
	opts ConsolidationOptions,
	sources ConsolidationSources,
) *ConsolidationService {
	return &ConsolidationService{
		db:             db,
		difficultyRepo: difficultyRepo,
		genreRepo:      genreRepo,
		wikiBaseURL:    wikiBaseURL,
		datasources:    datasources,
		opts:           opts,
		sources:        sources,
	}
}

// BuildWorkspace は対象データソースを SQLite ワークスペースに取り込みます。
func (s *ConsolidationService) BuildWorkspace(ctx context.Context) (*songchart.SongChartWorkspace, error) {
	target := make([]songbatch.DataSourceType, 0, len(s.datasources))
	for _, sourceName := range s.datasources {
		if s.opts.MajorUpdate && !songbatch.RunModeMajorUpdate.IsRequired(sourceName) {
			slog.Info("Skipping datasource in major update mode", "type", sourceName)
			continue
		}
		target = append(target, sourceName)
	}

	if len(target) == 0 {
		slog.Warn("No datasources to consolidate")
		return nil, nil
	}

	workspace, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create song chart workspace: %w", err)
	}

	for _, sourceName := range target {
		if err := s.consolidateSource(ctx, workspace, sourceName); err != nil {
			_ = workspace.Close()
			return nil, err
		}
	}

	return workspace, nil
}

// SyncWorkspace は準備済みワークスペースを MySQL に同期します。
func (s *ConsolidationService) SyncWorkspace(ctx context.Context, workspace *songchart.SongChartWorkspace, mysql apirepo.Executor, courseRepo songbatch.CourseRepository) error {
	syncOpts := songchart.SyncOptions{
		MajorUpdate:            s.opts.MajorUpdate,
		FillMissingReleaseDate: s.opts.FillMissingReleaseDate,
	}

	if err := workspace.SyncToMySQL(ctx, mysql, syncOpts); err != nil {
		return fmt.Errorf("failed to sync workspace to MySQL: %w", err)
	}
	return s.syncCourses(ctx, courseRepo)
}

func (s *ConsolidationService) syncCourses(ctx context.Context, courseRepo songbatch.CourseRepository) error {
	if s.sources.AdditionalSongs == nil || len(s.sources.AdditionalSongs.Courses) == 0 {
		return nil
	}

	courses := make([]entity.Course, 0, len(s.sources.AdditionalSongs.Courses))
	for _, course := range s.sources.AdditionalSongs.Courses {
		entityCourse, err := entity.NewCourse(course.ID, course.Title, course.Class)
		if err != nil {
			return fmt.Errorf("failed to create course %q: %w", course.ID, err)
		}
		courses = append(courses, entityCourse)
	}
	if err := courseRepo.SaveAll(ctx, courses); err != nil {
		return fmt.Errorf("failed to sync courses to MySQL: %w", err)
	}
	slog.Info("Synchronized courses to MySQL", "count", len(courses))
	return nil
}

func (s *ConsolidationService) consolidateSource(ctx context.Context, workspace *songchart.SongChartWorkspace, name songbatch.DataSourceType) error {
	switch name {
	case songbatch.DataSourceOfficial:
		if s.sources.Official == nil {
			slog.Warn("Skipping official consolidation due to missing data")
			return nil
		}
		consolidator := NewOfficialConsolidator(s.db, s.difficultyRepo, s.genreRepo, workspace, s.sources.Official)
		return consolidator.Consolidate(ctx)
	case songbatch.DataSourceAdditionalSongs:
		if s.sources.AdditionalSongs == nil {
			slog.Warn("Skipping additional_songs consolidation due to missing data")
			return nil
		}
		consolidator := NewAdditionalSongsConsolidator(s.db, s.difficultyRepo, s.genreRepo, workspace, s.sources.AdditionalSongs)
		return consolidator.Consolidate(ctx)
	case songbatch.DataSourceSt1027:
		if s.sources.St1027 == nil {
			slog.Warn("Skipping st1027 consolidation due to missing data")
			return nil
		}
		consolidator := NewSt1027Consolidator(workspace, s.sources.St1027)
		return consolidator.Consolidate(ctx)
	case songbatch.DataSourceMainframe:
		if s.sources.Mainframe == nil {
			slog.Warn("Skipping mainframe consolidation due to missing data")
			return nil
		}
		consolidator := NewMainframeConsolidator(workspace, s.sources.Mainframe)
		return consolidator.Consolidate(ctx)
	case songbatch.DataSourceOtogeDb:
		if s.sources.OtogeDb == nil {
			slog.Warn("Skipping otoge-db consolidation due to missing data")
			return nil
		}
		consolidator := NewOtogeDbConsolidator(workspace, s.sources.OtogeDb, s.wikiBaseURL)
		return consolidator.Consolidate(ctx)
	default:
		slog.Warn("Unknown datasource requested for consolidation", "type", name)
		return nil
	}
}

// assignSource は取り込み済みデータをデータソースの種類に応じた型で保持します。
func assignSource(sources *ConsolidationSources, source songbatch.ImportedSource) error {
	switch source.Type {
	case songbatch.DataSourceOfficial:
		typed, ok := source.Data.(*importer.OfficialData)
		if !ok {
			return fmt.Errorf("unexpected data type for official datasource: %T", source.Data)
		}
		sources.Official = typed
	case songbatch.DataSourceAdditionalSongs:
		typed, ok := source.Data.(*importer.AdditionalSongsData)
		if !ok {
			return fmt.Errorf("unexpected data type for additional_songs datasource: %T", source.Data)
		}
		sources.AdditionalSongs = typed
	case songbatch.DataSourceSt1027:
		typed, ok := source.Data.(*importer.St1027Data)
		if !ok {
			return fmt.Errorf("unexpected data type for st1027 datasource: %T", source.Data)
		}
		sources.St1027 = typed
	case songbatch.DataSourceMainframe:
		typed, ok := source.Data.(*importer.MainframeData)
		if !ok {
			return fmt.Errorf("unexpected data type for mainframe datasource: %T", source.Data)
		}
		sources.Mainframe = typed
	case songbatch.DataSourceOtogeDb:
		typed, ok := source.Data.(*importer.OtogeDbData)
		if !ok {
			return fmt.Errorf("unexpected data type for otoge-db datasource: %T", source.Data)
		}
		sources.OtogeDb = typed
	default:
		return fmt.Errorf("unsupported datasource type: %s", source.Type)
	}
	return nil
}
