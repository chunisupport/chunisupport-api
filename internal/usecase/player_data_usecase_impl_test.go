package usecase

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidatePlayerDataPayload_AppVersion は、app_verのバリデーションが正しく動作することをテストします
func TestValidatePlayerDataPayload_AppVersion(t *testing.T) {
	// 対応バージョンを動的に取得（テストの脆弱性を回避）
	require.NotEmpty(t, info.SupportedAppVersions, "info.SupportedAppVersions is empty - test cannot proceed")
	supportedVersion := info.SupportedAppVersions[0]

	tests := []struct {
		name       string
		appVersion string
		wantErr    bool
		errType    error
	}{
		{
			name:       "対応バージョン_正常",
			appVersion: supportedVersion, // 動的に取得した対応バージョン
			wantErr:    false,
			errType:    nil,
		},
		{
			name:       "非対応バージョン_空文字列",
			appVersion: "",
			wantErr:    true,
			errType:    ErrAppVersionUnsupported,
		},
		{
			name:       "非対応バージョン_不正な形式",
			appVersion: "invalid_version_string",
			wantErr:    true,
			errType:    ErrAppVersionUnsupported,
		},
		{
			name:       "非対応バージョン_確実に存在しない値",
			appVersion: "definitely_not_supported_version_xyz_12345",
			wantErr:    true,
			errType:    ErrAppVersionUnsupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 最小限のペイロードを作成（スコアは空）
			payload := &PlayerDataPayload{
				AppVersion: tt.appVersion,
				Name:       "テストプレイヤー",
				Level:      1,
				Rating:     ptrFloat64(0.0),
				LastPlayed: "2024/01/01 00:00",
				Overpower: PlayerDataOverpowerPayload{
					Value:      0.0,
					Percentage: 0.0,
				},
				ClassEmblem: PlayerDataClassPayload{
					MedalClass: "none",
					BaseClass:  "none",
				},
				Team: PlayerDataTeamPayload{
					Name:  "none",
					Color: "",
				},
				Honors: map[string]PlayerDataHonorPayload{},
				Scores: PlayerDataScorePayload{
					Full:      []PlayerDataScoreEntry{},
					Worldsend: []PlayerDataScoreEntry{},
				},
				UpdatedAt: "2024-01-01T00:00:00Z",
			}

			err := validatePlayerDataPayload(payload)

			if tt.wantErr {
				require.Error(t, err, "validatePlayerDataPayload() error = nil, want error")
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType, "validatePlayerDataPayload() error = %v, want %v", err, tt.errType)
				}
			} else {
				assert.NoError(t, err, "validatePlayerDataPayload() unexpected error = %v", err)
			}
		})
	}
}

// TestValidatePlayerDataPayload_MultipleVersions は、複数のバージョンが対応リストに含まれる場合のテストです
func TestValidatePlayerDataPayload_MultipleVersions(t *testing.T) {
	// info.SupportedAppVersionsに複数のバージョンが含まれる場合を想定したテスト
	// 実際の値を確認
	t.Logf("Current supported versions: %v", info.SupportedAppVersions)

	// 対応バージョンリストが空でないことを確認
	require.NotEmpty(t, info.SupportedAppVersions, "info.SupportedAppVersions is empty - test cannot proceed")
	supportedVersion := info.SupportedAppVersions[0]

	payload := &PlayerDataPayload{
		AppVersion: supportedVersion,
		Name:       "テストプレイヤー",
		Level:      1,
		Rating:     ptrFloat64(0.0),
		LastPlayed: "2024/01/01 00:00",
		Overpower: PlayerDataOverpowerPayload{
			Value:      0.0,
			Percentage: 0.0,
		},
		ClassEmblem: PlayerDataClassPayload{
			MedalClass: "none",
			BaseClass:  "none",
		},
		Team: PlayerDataTeamPayload{
			Name:  "none",
			Color: "",
		},
		Honors: map[string]PlayerDataHonorPayload{},
		Scores: PlayerDataScorePayload{
			Full:      []PlayerDataScoreEntry{},
			Worldsend: []PlayerDataScoreEntry{},
		},
		UpdatedAt: "2024-01-01T00:00:00Z",
	}

	err := validatePlayerDataPayload(payload)
	assert.NoError(t, err, "validatePlayerDataPayload() with supported version %s should not error", supportedVersion)
}

// TestValidatePlayerDataPayload_NilPayload は、payloadがnilの場合のテストです
func TestValidatePlayerDataPayload_NilPayload(t *testing.T) {
	err := validatePlayerDataPayload(nil)
	require.Error(t, err, "validatePlayerDataPayload(nil) should return error")

	var validationErr *PlayerDataValidationError
	require.ErrorAs(t, err, &validationErr, "validatePlayerDataPayload(nil) should return PlayerDataValidationError")
}

// ptrFloat64 はfloat64のポインタを返すヘルパー関数です
func ptrFloat64(v float64) *float64 {
	return &v
}

func TestCalculateOverpowerSummary_登録対象の通常譜面から合計値と割合を計算する(t *testing.T) {
	tests := []struct {
		name        string
		fullRecords []repository.PlayerRecordForUpsert
		chartsByID  map[int]entity.PlayerDataChart
		wantValue   float64
		wantPercent float64
	}{
		{
			name: "理論値到達時は割合が100になる",
			fullRecords: []repository.PlayerRecordForUpsert{
				{
					ChartID: 1,
					State: repository.PlayerRecordState{
						Score:       1010000,
						ComboLampID: 3,
					},
				},
			},
			chartsByID: map[int]entity.PlayerDataChart{
				1: {ID: 1, SongID: 1, Const: 15.0},
			},
			wantValue:   90,
			wantPercent: 100,
		},
		{
			name: "2譜面の合計OVER_POWERを計算する",
			fullRecords: []repository.PlayerRecordForUpsert{
				{
					ChartID: 1,
					State: repository.PlayerRecordState{
						Score:       1010000,
						ComboLampID: 3,
					},
				},
				{
					ChartID: 2,
					State: repository.PlayerRecordState{
						Score:       1009000,
						ComboLampID: 3,
					},
				},
			},
			chartsByID: map[int]entity.PlayerDataChart{
				1: {ID: 1, SongID: 1, Const: 15.0},
				2: {ID: 2, SongID: 2, Const: 14.5},
			},
			wantValue: func() float64 {
				return service.CalcSingleOverpower(1010000, 15.0, 3) + service.CalcSingleOverpower(1009000, 14.5, 3)
			}(),
			wantPercent: func() float64 {
				total := service.CalcSingleOverpower(1010000, 15.0, 3) + service.CalcSingleOverpower(1009000, 14.5, 3)
				theoretical := info.CalcTheoreticalOverpowerTotal(29.5, 2)
				return roundFloat(total/theoretical*100, 4)
			}(),
		},
		{
			name: "同一曲に複数譜面がある場合は単譜面OP最大の1譜面だけ採用する",
			fullRecords: []repository.PlayerRecordForUpsert{
				{
					ChartID: 1,
					State: repository.PlayerRecordState{
						Score:       1000000,
						ComboLampID: 2,
					},
				},
				{
					ChartID: 2,
					State: repository.PlayerRecordState{
						Score:       1010000,
						ComboLampID: 3,
					},
				},
				{
					ChartID: 3,
					State: repository.PlayerRecordState{
						Score:       1009000,
						ComboLampID: 3,
					},
				},
			},
			chartsByID: map[int]entity.PlayerDataChart{
				1: {ID: 1, SongID: 1, Const: 14.0},
				2: {ID: 2, SongID: 1, Const: 15.0},
				3: {ID: 3, SongID: 2, Const: 14.5},
			},
			wantValue: func() float64 {
				return service.CalcSingleOverpower(1010000, 15.0, 3) + service.CalcSingleOverpower(1009000, 14.5, 3)
			}(),
			wantPercent: func() float64 {
				total := service.CalcSingleOverpower(1010000, 15.0, 3) + service.CalcSingleOverpower(1009000, 14.5, 3)
				theoretical := info.CalcTheoreticalOverpowerTotal(29.5, 2)
				return roundFloat(total/theoretical*100, 4)
			}(),
		},
		{
			name:        "通常譜面がない場合は0を返す",
			fullRecords: []repository.PlayerRecordForUpsert{},
			chartsByID:  map[int]entity.PlayerDataChart{},
			wantValue:   0,
			wantPercent: 0,
		},
		{
			name: "譜面マスタが見つからないレコードは集計から除外する",
			fullRecords: []repository.PlayerRecordForUpsert{
				{
					ChartID: 999,
					State: repository.PlayerRecordState{
						Score:       1010000,
						ComboLampID: 3,
					},
				},
			},
			chartsByID:  map[int]entity.PlayerDataChart{},
			wantValue:   0,
			wantPercent: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateOverpowerSummary(tt.fullRecords, tt.chartsByID)

			require.NotNil(t, got.Value)
			require.NotNil(t, got.Percent)
			assert.InDelta(t, tt.wantValue, *got.Value, 0.0001)
			assert.InDelta(t, tt.wantPercent, *got.Percent, 0.0001)
		})
	}
}
