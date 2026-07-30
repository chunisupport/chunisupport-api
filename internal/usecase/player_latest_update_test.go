package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	playerdataresult "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPlayerLatestUpdate_skippedRecordsを除外して登録結果を圧縮する(t *testing.T) {
	// Given
	importedAt := time.Date(2026, 7, 16, 2, 3, 4, 0, time.UTC)
	sourceUpdatedAt := importedAt.Add(-time.Minute)
	result := &playerdataresult.Result{
		PlayerID:   12,
		AppVersion: "1.2.3",
		ImportedAt: importedAt,
		Profile: playerdataresult.Profile{
			PlayerID: 12,
			Name:     "TEST",
			Level:    99,
		},
		Summary: playerdataresult.Summary{Name: "TEST", Level: 99},
		MetricDiffs: playerdataresult.MetricDiffs{
			Rating:         playerdataresult.Float64Diff{Before: float64Pointer(16.42), After: float64Pointer(16.45), Delta: float64Pointer(0.03)},
			OverpowerValue: playerdataresult.Float64Diff{Before: float64Pointer(96120.123), After: float64Pointer(96123.91), Delta: float64Pointer(3.787)},
		},
		Counts: playerdataresult.Counts{FullRecordsActuallyChanged: 2, FullRecordsSkipped: 1},
		Changes: []playerdataresult.RecordChange{{
			RecordType: "standard",
			ChangeType: "updated",
			Idx:        "100",
			Diff:       "MASTER",
			After:      playerdataresult.RecordState{Score: 1000000},
		}},
		SkippedRecords: []playerdataresult.SkippedRecord{{RecordType: "standard", Reason: "unknown", Details: "secret"}},
	}

	// When
	update, err := buildPlayerLatestUpdate(result, sourceUpdatedAt, "body-hash")

	// Then
	require.NoError(t, err)
	assert.Equal(t, playerLatestUpdateSchemaVersion, update.SchemaVersion())
	assert.Equal(t, sourceUpdatedAt, update.SourceUpdatedAt())
	assert.Equal(t, importedAt, update.ImportedAt())

	raw, err := gunzipBytes(update.ResultGzip(), 1024*1024)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	assert.Equal(t, float64(playerLatestUpdateSchemaVersion), payload["schema_version"])
	assert.Equal(t, "1.2.3", payload["app_ver"])
	assert.Contains(t, payload, "profile")
	assert.Contains(t, payload, "summary")
	assert.Contains(t, payload, "metric_diffs")
	assert.Contains(t, payload, "statistics")
	assert.Contains(t, payload, "counts")
	assert.Contains(t, payload, "changes")
	assert.NotContains(t, payload, "skipped_records")
	assert.NotContains(t, string(raw), "secret")
}

func TestBuildPlayerDataFloat64Diff_登録前後がある場合だけ差分を返す(t *testing.T) {
	tests := []struct {
		name   string
		before *float64
		after  *float64
		want   playerdataresult.Float64Diff
	}{
		{
			name:   "登録前後がある場合はdeltaを計算する",
			before: float64Pointer(16.42),
			after:  float64Pointer(16.45),
			want:   playerdataresult.Float64Diff{Before: float64Pointer(16.42), After: float64Pointer(16.45), Delta: float64Pointer(0.03)},
		},
		{
			name:  "初回登録ではdeltaを返さない",
			after: float64Pointer(16.45),
			want:  playerdataresult.Float64Diff{After: float64Pointer(16.45)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := buildPlayerDataFloat64Diff(tt.before, tt.after)

			// Then
			assert.Equal(t, tt.want.Before, got.Before)
			assert.Equal(t, tt.want.After, got.After)
			if tt.want.Delta == nil {
				assert.Nil(t, got.Delta)
			} else {
				require.NotNil(t, got.Delta)
				assert.InDelta(t, *tt.want.Delta, *got.Delta, 1e-9)
			}
		})
	}
}

func float64Pointer(value float64) *float64 {
	return &value
}

func TestPlayerDataUsecase_GetLatestUpdate(t *testing.T) {
	// Given
	playerID := 12
	result := &playerdataresult.Result{
		PlayerID:   playerID,
		AppVersion: "1.2.3",
		ImportedAt: time.Date(2026, 7, 16, 2, 3, 4, 0, time.UTC),
		Changes:    []playerdataresult.RecordChange{},
	}
	update, err := buildPlayerLatestUpdate(result, result.ImportedAt.Add(-time.Minute), "body-hash")
	require.NoError(t, err)
	repo := &stubPlayerDataRepositoryForApplyScoresTest{latestUpdate: update}
	u := &playerDataUsecase{playerDataRepo: repo}

	// When
	raw, err := u.GetLatestUpdate(context.Background(), &entity.User{PlayerID: &playerID})

	// Then
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	assert.Equal(t, float64(playerLatestUpdateSchemaVersion), payload["schema_version"])
	assert.Equal(t, "1.2.3", payload["app_ver"])
	assert.NotContains(t, payload, "skipped_records")
}

func TestPlayerDataUsecase_GetLatestUpdate_schema1の保存結果も返す(t *testing.T) {
	// Given
	playerID := 12
	result := &playerdataresult.Result{
		PlayerID:   playerID,
		AppVersion: "1.2.3",
		ImportedAt: time.Date(2026, 7, 16, 2, 3, 4, 0, time.UTC),
		Changes:    []playerdataresult.RecordChange{},
	}
	payload := playerLatestUpdatePayload(result)
	payload["schema_version"] = info.PlayerLatestUpdateMinSupportedSchemaVersion
	delete(payload, "metric_diffs")
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	compressed, err := gzipBytes(raw)
	require.NoError(t, err)
	update, err := entity.NewPlayerLatestUpdate(
		playerID,
		info.PlayerLatestUpdateMinSupportedSchemaVersion,
		compressed,
		result.ImportedAt.Add(-time.Minute),
		result.ImportedAt,
		"body-hash",
	)
	require.NoError(t, err)
	repo := &stubPlayerDataRepositoryForApplyScoresTest{latestUpdate: update}
	u := &playerDataUsecase{playerDataRepo: repo}

	// When
	got, err := u.GetLatestUpdate(context.Background(), &entity.User{PlayerID: &playerID})

	// Then
	require.NoError(t, err)
	var gotPayload map[string]any
	require.NoError(t, json.Unmarshal(got, &gotPayload))
	assert.Equal(t, float64(info.PlayerLatestUpdateMinSupportedSchemaVersion), gotPayload["schema_version"])
	assert.NotContains(t, gotPayload, "metric_diffs")
}

func TestPlayerDataUsecase_GetLatestUpdate_未連携と未保存を区別する(t *testing.T) {
	playerID := 12
	tests := []struct {
		name    string
		user    *entity.User
		repoErr error
		wantErr error
	}{
		{name: "プレイヤー未連携", user: &entity.User{}, wantErr: ErrPlayerNotLinked},
		{name: "最新結果未保存", user: &entity.User{PlayerID: &playerID}, repoErr: repository.ErrPlayerLatestUpdateNotFound, wantErr: ErrPlayerLatestUpdateNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			u := &playerDataUsecase{playerDataRepo: &stubPlayerDataRepositoryForApplyScoresTest{latestUpdateErr: tt.repoErr}}

			// When
			_, err := u.GetLatestUpdate(context.Background(), tt.user)

			// Then
			assert.True(t, errors.Is(err, tt.wantErr))
		})
	}
}

func TestPlayerDataUsecase_GetLatestUpdate_不正な保存内容は内部エラーにする(t *testing.T) {
	playerID := 12
	validResult := &playerdataresult.Result{
		PlayerID:   playerID,
		AppVersion: "1.0.0",
		ImportedAt: time.Now().UTC(),
		Changes:    []playerdataresult.RecordChange{},
	}
	validUpdate, err := buildPlayerLatestUpdate(validResult, time.Now().UTC(), "hash")
	require.NoError(t, err)
	missingMetricPayload := playerLatestUpdatePayload(validResult)
	delete(missingMetricPayload, "metric_diffs")
	missingMetricRaw, err := json.Marshal(missingMetricPayload)
	require.NoError(t, err)
	missingMetricGzip, err := gzipBytes(missingMetricRaw)
	require.NoError(t, err)
	missingMetricUpdate, err := entity.NewPlayerLatestUpdate(
		playerID,
		playerLatestUpdateSchemaVersion,
		missingMetricGzip,
		time.Now().UTC(),
		time.Now().UTC(),
		"missing-metric-hash",
	)
	require.NoError(t, err)
	missingFieldGzip, err := gzipBytes([]byte(`{"schema_version":1}`))
	require.NoError(t, err)
	mismatchResult := *validResult
	mismatchResult.PlayerID = playerID + 1
	mismatchUpdate, err := buildPlayerLatestUpdate(&mismatchResult, time.Now().UTC(), "other-hash")
	require.NoError(t, err)
	mismatchPlayerUpdate, err := entity.NewPlayerLatestUpdate(playerID, playerLatestUpdateSchemaVersion, mismatchUpdate.ResultGzip(), mismatchUpdate.SourceUpdatedAt(), mismatchUpdate.ImportedAt(), mismatchUpdate.BodyHash())
	require.NoError(t, err)
	schemaMismatchUpdate, err := entity.NewPlayerLatestUpdate(playerID, playerLatestUpdateSchemaVersion+1, validUpdate.ResultGzip(), validUpdate.SourceUpdatedAt(), validUpdate.ImportedAt(), validUpdate.BodyHash())
	require.NoError(t, err)
	missingFieldUpdate, err := entity.NewPlayerLatestUpdate(playerID, playerLatestUpdateSchemaVersion, missingFieldGzip, time.Now().UTC(), time.Now().UTC(), "missing-hash")
	require.NoError(t, err)
	brokenGzipUpdate, err := entity.NewPlayerLatestUpdate(playerID, playerLatestUpdateSchemaVersion, []byte("broken-gzip"), time.Now().UTC(), time.Now().UTC(), "broken-hash")
	require.NoError(t, err)

	tests := []struct {
		name   string
		update *entity.PlayerLatestUpdate
	}{
		{name: "gzipが壊れている", update: brokenGzipUpdate},
		{name: "必須フィールドがない", update: missingFieldUpdate},
		{name: "schema 2でメトリクス差分がない", update: missingMetricUpdate},
		{name: "DBとJSONのスキーマが一致しない", update: schemaMismatchUpdate},
		{name: "DBとJSONのプレイヤーIDが一致しない", update: mismatchPlayerUpdate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			u := &playerDataUsecase{playerDataRepo: &stubPlayerDataRepositoryForApplyScoresTest{latestUpdate: tt.update}}

			// When
			_, err := u.GetLatestUpdate(context.Background(), &entity.User{PlayerID: &playerID})

			// Then
			assert.True(t, errors.Is(err, ErrInternalError))
		})
	}
}
