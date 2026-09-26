package usecase

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubResolver struct {
	sources map[songbatch.DataSourceType]SongBatchDatasourceRef
	errs    map[songbatch.DataSourceType]error
	called  []songbatch.DataSourceType
}

func (s *stubResolver) Resolve(name songbatch.DataSourceType) (SongBatchDatasourceRef, error) {
	s.called = append(s.called, name)
	if err, ok := s.errs[name]; ok {
		return SongBatchDatasourceRef{}, err
	}
	if ds, ok := s.sources[name]; ok {
		return ds, nil
	}
	return SongBatchDatasourceRef{}, errors.New("unresolved")
}

type stubDownloader struct {
	outputDir string
	fn        func(outputDir string, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error)
}

func (s *stubDownloader) DownloadAll(_ context.Context, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error) {
	return s.fn(s.outputDir, datasources)
}

type stubImporter struct {
	fn func(sourceType songbatch.DataSourceType, filePath string) (*songbatch.ImportedSource, error)
}

func (s *stubImporter) Import(sourceType songbatch.DataSourceType, filePath string) (*songbatch.ImportedSource, error) {
	return s.fn(sourceType, filePath)
}

type stubConsolidator struct {
	called  bool
	sources []songbatch.ImportedSource
	req     songbatch.RunRequest
	err     error
}

func (s *stubConsolidator) Consolidate(_ context.Context, sources []songbatch.ImportedSource, req songbatch.RunRequest) error {
	s.called = true
	s.sources = sources
	s.req = req
	return s.err
}

// names は統合に渡されたデータソース名を統合順に返します。
func (s *stubConsolidator) names() []string {
	names := make([]string, len(s.sources))
	for i, source := range s.sources {
		names[i] = string(source.Type)
	}
	return names
}

func (s *stubConsolidator) hasSource(sourceType songbatch.DataSourceType) bool {
	for _, source := range s.sources {
		if source.Type == sourceType {
			return true
		}
	}
	return false
}

func successfulImport(sourceType songbatch.DataSourceType) *songbatch.ImportedSource {
	return &songbatch.ImportedSource{Type: sourceType, Data: struct{}{}}
}

func allResolved() map[songbatch.DataSourceType]SongBatchDatasourceRef {
	names := songbatch.SupportedDataSources()
	sources := make(map[songbatch.DataSourceType]SongBatchDatasourceRef, len(names))
	for _, name := range names {
		sources[name] = SongBatchDatasourceRef{Type: name, URL: "https://example.invalid/" + string(name)}
	}
	return sources
}

func successfulDownloads(t *testing.T, outputDir string, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error) {
	t.Helper()
	results := make([]SongBatchDownloadResult, 0, len(datasources))
	for _, ds := range datasources {
		path := filepath.Join(outputDir, string(ds.Type)+".json")
		if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
			t.Fatalf("write download: %v", err)
		}
		results = append(results, SongBatchDownloadResult{Type: ds.Type, Success: true, Path: path})
	}
	return results, nil
}

func newUsecase(
	resolver *stubResolver,
	downloadFn func(outputDir string, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error),
	importFn func(sourceType songbatch.DataSourceType, filePath string) (*songbatch.ImportedSource, error),
	consolidator *stubConsolidator,
	tempDir *string,
) *SongBatchUsecase {
	return NewSongBatchUsecase(
		resolver,
		func(outputDir string) SongBatchDownloader {
			*tempDir = outputDir
			return &stubDownloader{outputDir: outputDir, fn: downloadFn}
		},
		&stubImporter{fn: importFn},
		consolidator,
	)
}

func TestExecuteRequiredDatasourceFailuresDoNotConsolidate(t *testing.T) {
	tests := []struct {
		name       string
		resolver   *stubResolver
		downloadFn func(string, []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error)
		importFn   func(songbatch.DataSourceType, string) (*songbatch.ImportedSource, error)
		wantStage  string
	}{
		{
			name:     "resolve",
			resolver: &stubResolver{sources: allResolved(), errs: map[songbatch.DataSourceType]error{"official": errors.New("missing env")}},
			downloadFn: func(string, []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error) {
				t.Fatal("download must not run")
				return nil, nil
			},
			importFn:  func(songbatch.DataSourceType, string) (*songbatch.ImportedSource, error) { return nil, nil },
			wantStage: "resolve",
		},
		{
			name:     "download",
			resolver: &stubResolver{sources: allResolved()},
			downloadFn: func(outputDir string, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error) {
				results, _ := successfulDownloads(t, outputDir, datasources)
				for i := range results {
					if results[i].Type == "official" {
						results[i] = SongBatchDownloadResult{Type: "official", Error: "http 500"}
					}
				}
				return results, nil
			},
			importFn: func(sourceType songbatch.DataSourceType, _ string) (*songbatch.ImportedSource, error) {
				return successfulImport(sourceType), nil
			},
			wantStage: "download",
		},
		{
			name:     "parse",
			resolver: &stubResolver{sources: allResolved()},
			downloadFn: func(outputDir string, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error) {
				return successfulDownloads(t, outputDir, datasources)
			},
			importFn: func(sourceType songbatch.DataSourceType, _ string) (*songbatch.ImportedSource, error) {
				if sourceType == "official" {
					return nil, errors.New("invalid JSON")
				}
				return successfulImport(sourceType), nil
			},
			wantStage: "parse",
		},
		{
			name:     "validation",
			resolver: &stubResolver{sources: allResolved()},
			downloadFn: func(outputDir string, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error) {
				return successfulDownloads(t, outputDir, datasources)
			},
			importFn: func(sourceType songbatch.DataSourceType, _ string) (*songbatch.ImportedSource, error) {
				if sourceType == "official" {
					return nil, fmtValidationError("official is empty")
				}
				return successfulImport(sourceType), nil
			},
			wantStage: "validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consolidator := &stubConsolidator{}
			var tempDir string
			uc := newUsecase(tt.resolver, tt.downloadFn, tt.importFn, consolidator, &tempDir)

			_, err := uc.Execute(context.Background(), songbatch.NewRunRequest(false, false))
			if err == nil || !strings.Contains(err.Error(), "official") || !strings.Contains(err.Error(), tt.wantStage+" stage") {
				t.Fatalf("unexpected error: %v", err)
			}
			if consolidator.called {
				t.Fatal("consolidator must not run")
			}
			if tempDir != "" {
				if _, statErr := os.Stat(tempDir); !os.IsNotExist(statErr) {
					t.Fatalf("temp dir should be removed: %v", statErr)
				}
			}
		})
	}
}

func fmtValidationError(message string) error {
	return errors.Join(songbatch.ErrSourceValidation, errors.New(message))
}

func TestExecuteExcludesUnavailableComplementarySourcesAndLogsWarningResult(t *testing.T) {
	resolver := &stubResolver{sources: allResolved(), errs: map[songbatch.DataSourceType]error{"st1027": errors.New("missing env")}}
	consolidator := &stubConsolidator{}
	var tempDir string
	uc := newUsecase(
		resolver,
		func(outputDir string, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error) {
			results, _ := successfulDownloads(t, outputDir, datasources)
			for i := range results {
				if results[i].Type == "otoge_db" {
					results[i] = SongBatchDownloadResult{Type: "otoge_db", Error: "timeout"}
				}
			}
			return results, nil
		},
		func(sourceType songbatch.DataSourceType, _ string) (*songbatch.ImportedSource, error) {
			return successfulImport(sourceType), nil
		},
		consolidator,
		&tempDir,
	)

	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	if _, err := uc.Execute(context.Background(), songbatch.NewRunRequest(false, false)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !consolidator.called {
		t.Fatal("valid sources must still be consolidated")
	}
	wantNames := []string{"official", "additional_songs", "mainframe"}
	if strings.Join(consolidator.names(), ",") != strings.Join(wantNames, ",") {
		t.Fatalf("names=%v, want=%v", consolidator.names(), wantNames)
	}
	output := logs.String()
	for _, want := range []string{"source=st1027 stage=resolve", "source=otoge_db stage=download", "Completed with Warnings", "warning_count=2"} {
		if !strings.Contains(output, want) {
			t.Fatalf("log does not contain %q:\n%s", want, output)
		}
	}
}

func TestExecuteExcludesComplementarySourceOnValidationFailure(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	var tempDir string
	uc := newUsecase(
		resolver,
		func(outputDir string, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error) {
			return successfulDownloads(t, outputDir, datasources)
		},
		func(sourceType songbatch.DataSourceType, _ string) (*songbatch.ImportedSource, error) {
			if sourceType == "st1027" {
				return nil, fmtValidationError("missing songs")
			}
			return successfulImport(sourceType), nil
		},
		consolidator,
		&tempDir,
	)

	if _, err := uc.Execute(context.Background(), songbatch.NewRunRequest(false, false)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if consolidator.hasSource(songbatch.DataSourceSt1027) {
		t.Fatal("invalid complementary source must be excluded")
	}
	if !consolidator.hasSource(songbatch.DataSourceOtogeDb) {
		t.Fatal("other valid complementary source must be retained")
	}
}

func TestExecuteLogsSuccessWithoutWarnings(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	var tempDir string
	uc := newUsecase(
		resolver,
		func(outputDir string, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error) {
			return successfulDownloads(t, outputDir, datasources)
		},
		func(sourceType songbatch.DataSourceType, _ string) (*songbatch.ImportedSource, error) {
			return successfulImport(sourceType), nil
		},
		consolidator,
		&tempDir,
	)

	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	if _, err := uc.Execute(context.Background(), songbatch.NewRunRequest(false, false)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	output := logs.String()
	if !strings.Contains(output, "Completed Successfully") || strings.Contains(output, "Completed with Warnings") {
		t.Fatalf("unexpected completion log:\n%s", output)
	}
}

func TestExecuteDoesNotReadExistingDatasourceDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(".datasources", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".datasources", "official.json"), []byte(`stale`), 0644); err != nil {
		t.Fatal(err)
	}

	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	var tempDir string
	var importedPaths []string
	uc := newUsecase(
		resolver,
		func(outputDir string, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error) {
			return successfulDownloads(t, outputDir, datasources)
		},
		func(sourceType songbatch.DataSourceType, filePath string) (*songbatch.ImportedSource, error) {
			importedPaths = append(importedPaths, filePath)
			return successfulImport(sourceType), nil
		},
		consolidator,
		&tempDir,
	)

	if _, err := uc.Execute(context.Background(), songbatch.NewRunRequest(false, false)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, path := range importedPaths {
		if !strings.HasPrefix(path, tempDir+string(os.PathSeparator)) {
			t.Fatalf("imported non-execution file: %s", path)
		}
	}
}

func TestExecuteMajorUpdateUsesOnlyRequiredSources(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	var tempDir string
	uc := newUsecase(
		resolver,
		func(outputDir string, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error) {
			if len(datasources) != 2 || datasources[0].Type != "official" || datasources[1].Type != "additional_songs" {
				t.Fatalf("unexpected major-update sources: %v", datasources)
			}
			return successfulDownloads(t, outputDir, datasources)
		},
		func(sourceType songbatch.DataSourceType, _ string) (*songbatch.ImportedSource, error) {
			return successfulImport(sourceType), nil
		},
		consolidator,
		&tempDir,
	)

	if _, err := uc.Execute(context.Background(), songbatch.NewRunRequest(true, true)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if consolidator.req.Mode != songbatch.RunModeMajorUpdate || !consolidator.req.FillMissingReleaseDate {
		t.Fatalf("request=%+v", consolidator.req)
	}
	if len(resolver.called) != 2 {
		t.Fatalf("resolved sources=%v", resolver.called)
	}
}

func TestExecute_警告件数と統合順(t *testing.T) {
	tests := []struct {
		name             string
		resolveErrs      map[songbatch.DataSourceType]error
		downloadFailures []songbatch.DataSourceType
		importFailures   []songbatch.DataSourceType
		wantErr          bool
		expectedWarnings int
		expectedNames    []string
	}{
		{
			name:          "全データソースを統合順に渡す",
			expectedNames: []string{"official", "additional_songs", "st1027", "mainframe", "otoge_db"},
		},
		{
			name:             "解決・取得・検証の各段階で除外した補完ソースを1件ずつ数える",
			resolveErrs:      map[songbatch.DataSourceType]error{songbatch.DataSourceSt1027: errors.New("missing env")},
			downloadFailures: []songbatch.DataSourceType{songbatch.DataSourceOtogeDb},
			expectedWarnings: 2,
			expectedNames:    []string{"official", "additional_songs", "mainframe"},
		},
		{
			name:             "必須ソースの失敗時もそれまでの警告件数を返す",
			resolveErrs:      map[songbatch.DataSourceType]error{songbatch.DataSourceSt1027: errors.New("missing env")},
			importFailures:   []songbatch.DataSourceType{songbatch.DataSourceOtogeDb, songbatch.DataSourceMainframe},
			wantErr:          true,
			expectedWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			consolidator := &stubConsolidator{}
			var tempDir string
			uc := newUsecase(
				&stubResolver{sources: allResolved(), errs: tt.resolveErrs},
				func(outputDir string, datasources []SongBatchDatasourceRef) ([]SongBatchDownloadResult, error) {
					results, _ := successfulDownloads(t, outputDir, datasources)
					for i := range results {
						if slices.Contains(tt.downloadFailures, results[i].Type) {
							results[i] = SongBatchDownloadResult{Type: results[i].Type, Error: "timeout"}
						}
					}
					return results, nil
				},
				func(sourceType songbatch.DataSourceType, _ string) (*songbatch.ImportedSource, error) {
					if slices.Contains(tt.importFailures, sourceType) {
						return nil, errors.New("invalid JSON")
					}
					return successfulImport(sourceType), nil
				},
				consolidator,
				&tempDir,
			)

			// When
			result, err := uc.Execute(context.Background(), songbatch.NewRunRequest(false, false))

			// Then
			assert.Equal(t, tt.expectedWarnings, result.WarningCount)
			if tt.wantErr {
				assert.Error(t, err)
				assert.False(t, consolidator.called)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedNames, consolidator.names())
		})
	}
}
