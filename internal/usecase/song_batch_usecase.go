package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// SongBatchResult は楽曲バッチ1回分の実行結果です。
type SongBatchResult struct {
	// WarningCount は利用できず除外した補完データソースの件数です。
	WarningCount int
}

// SongBatchUsecase は楽曲バッチの取得・検証・統合を実行します。
type SongBatchUsecase struct {
	resolver      SongBatchDatasourceResolver
	newDownloader SongBatchDownloaderFactory
	importer      SongBatchSourceImporter
	consolidator  SongBatchConsolidator
}

type songBatchSourceInput struct {
	Type songbatch.DataSourceType
	Path string
}

// NewSongBatchUsecase は SongBatchUsecase を生成します。
func NewSongBatchUsecase(
	resolver SongBatchDatasourceResolver,
	newDownloader SongBatchDownloaderFactory,
	sourceImporter SongBatchSourceImporter,
	consolidator SongBatchConsolidator,
) *SongBatchUsecase {
	return &SongBatchUsecase{
		resolver:      resolver,
		newDownloader: newDownloader,
		importer:      sourceImporter,
		consolidator:  consolidator,
	}
}

// Execute はモードに応じてデータソースを取得し、必須条件を満たせば同期します。
// 取得は毎回実行専用の一時ディレクトリへ行い、前回実行時のファイルを誤って読み込まないようにします。
func (u *SongBatchUsecase) Execute(ctx context.Context, req songbatch.RunRequest) (SongBatchResult, error) {
	tempDir, err := os.MkdirTemp("", info.SongBatchTempDirPrefix)
	if err != nil {
		return SongBatchResult{}, fmt.Errorf("failed to create temp dir: %w", err)
	}
	slog.Info("using execution temp dir", "path", tempDir)
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			slog.Warn("failed to remove temp dir", "path", tempDir, "error", removeErr)
		}
	}()

	inputs, warningCount, err := u.inputsFromDownload(ctx, req.Mode, tempDir)
	if err != nil {
		return SongBatchResult{WarningCount: warningCount}, err
	}

	sources, importWarnings, err := u.importInputs(req.Mode, inputs)
	warningCount += importWarnings
	if err != nil {
		return SongBatchResult{WarningCount: warningCount}, err
	}

	if err := u.consolidator.Consolidate(ctx, sources, req); err != nil {
		return SongBatchResult{WarningCount: warningCount}, err
	}

	if warningCount > 0 {
		slog.Warn("Data Import Batch Completed with Warnings", "warning_count", warningCount)
	} else {
		slog.Info("Data Import Batch Completed Successfully")
	}
	return SongBatchResult{WarningCount: warningCount}, nil
}

func (u *SongBatchUsecase) inputsFromDownload(ctx context.Context, mode songbatch.RunMode, tempDir string) ([]songBatchSourceInput, int, error) {
	targets := mode.TargetSources()
	resolved := make(map[songbatch.DataSourceType]SongBatchDatasourceRef, len(targets))
	toDownload := make([]SongBatchDatasourceRef, 0, len(targets))
	warningCount := 0

	for _, name := range targets {
		ds, err := u.resolver.Resolve(name)
		if err != nil {
			if mode.IsRequired(name) {
				return nil, warningCount, fmt.Errorf("required datasource %s failed at resolve stage: %w", name, err)
			}
			logComplementaryFailure(name, "resolve", err.Error())
			warningCount++
			continue
		}
		if ds.Type == "" {
			ds.Type = name
		}
		resolved[name] = ds
		toDownload = append(toDownload, ds)
	}

	results, err := u.newDownloader(tempDir).DownloadAll(ctx, toDownload)
	if err != nil {
		return nil, warningCount, fmt.Errorf("datasource download failed: %w", err)
	}
	resultsByType := make(map[songbatch.DataSourceType]SongBatchDownloadResult, len(results))
	for _, result := range results {
		resultsByType[result.Type] = result
	}

	inputs := make([]songBatchSourceInput, 0, len(targets))
	for _, name := range targets {
		ds, ok := resolved[name]
		if !ok {
			continue
		}
		result, ok := resultsByType[ds.Type]
		if ok && result.Success {
			inputs = append(inputs, songBatchSourceInput{Type: ds.Type, Path: result.Path})
			continue
		}

		reason := "download result was not returned"
		if ok && result.Error != "" {
			reason = result.Error
		}
		if mode.IsRequired(name) {
			return nil, warningCount, fmt.Errorf("required datasource %s failed at download stage: %s", name, reason)
		}
		logComplementaryFailure(name, "download", reason)
		warningCount++
	}

	return inputs, warningCount, nil
}

func (u *SongBatchUsecase) importInputs(mode songbatch.RunMode, inputs []songBatchSourceInput) ([]songbatch.ImportedSource, int, error) {
	sources := make([]songbatch.ImportedSource, 0, len(inputs))
	warningCount := 0

	for _, in := range inputs {
		result, err := u.importer.Import(in.Type, in.Path)
		if err == nil && (result == nil || result.Data == nil) {
			err = errors.New("no data")
		}
		if err != nil {
			stage := importFailureStage(err)
			if mode.IsRequired(in.Type) {
				return nil, warningCount, fmt.Errorf("required datasource %s failed at %s stage: %w", in.Type, stage, err)
			}
			logComplementaryFailure(in.Type, stage, err.Error())
			warningCount++
			continue
		}
		sources = append(sources, songbatch.ImportedSource{Type: in.Type, Data: result.Data})
	}

	for _, required := range mode.RequiredSources() {
		if !slices.ContainsFunc(sources, func(source songbatch.ImportedSource) bool { return source.Type == required }) {
			return nil, warningCount, fmt.Errorf("required datasource %s is missing after validation", required)
		}
	}

	return sources, warningCount, nil
}

func importFailureStage(err error) string {
	if errors.Is(err, songbatch.ErrSourceValidation) {
		return "validation"
	}
	return "parse"
}

func logComplementaryFailure(source songbatch.DataSourceType, stage, reason string) {
	slog.Warn("excluding complementary datasource", "source", source, "stage", stage, "reason", reason)
}
