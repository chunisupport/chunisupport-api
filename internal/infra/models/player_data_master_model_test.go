package models

import (
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

func TestPlayerDataChartModelToEntity(t *testing.T) {
	tests := []struct {
		name          string
		model         PlayerDataChartModel
		expectedConst chartconstant.ChartConstant
		expectedError string
		wantErr       bool
	}{
		{
			name: "正の定数値ならVOに変換される",
			model: PlayerDataChartModel{
				ID:             10,
				SongID:         100,
				DifficultyID:   3,
				Const:          13.5,
				IsConstUnknown: false,
			},
			expectedConst: mustChartConstantForTest(t, 13.5),
			wantErr:       false,
		},
		{
			name: "0の定数値ならVOに変換される",
			model: PlayerDataChartModel{
				ID:             11,
				SongID:         101,
				DifficultyID:   4,
				Const:          0.0,
				IsConstUnknown: true,
			},
			expectedConst: mustChartConstantForTest(t, 0.0),
			wantErr:       false,
		},
		{
			name: "負の定数値ならエラーになる",
			model: PlayerDataChartModel{
				ID:    12,
				Const: -1.0,
			},
			expectedError: "chart_id=12",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			model := tt.model

			// When
			got, err := model.ToEntity()

			// Then
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.expectedError != "" && !assertErrorContains(err, tt.expectedError) {
					t.Fatalf("error = %v, want contain %q", err, tt.expectedError)
				}
				return
			}
			if got.Const != tt.expectedConst {
				t.Fatalf("const = %v, want %v", got.Const, tt.expectedConst)
			}
			if got.ID != model.ID {
				t.Fatalf("id = %d, want %d", got.ID, model.ID)
			}
			if got.SongID != model.SongID {
				t.Fatalf("song_id = %d, want %d", got.SongID, model.SongID)
			}
			if got.DifficultyID != model.DifficultyID {
				t.Fatalf("difficulty_id = %d, want %d", got.DifficultyID, model.DifficultyID)
			}
			if got.IsConstUnknown != model.IsConstUnknown {
				t.Fatalf("is_const_unknown = %v, want %v", got.IsConstUnknown, model.IsConstUnknown)
			}
		})
	}
}

func assertErrorContains(err error, expected string) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), expected)
}

func TestPlayerDataChartEntityToModel(t *testing.T) {
	tests := []struct {
		name          string
		entity        entity.PlayerDataChart
		expectedConst float64
	}{
		{
			name: "ChartConstantからfloat64へ変換される",
			entity: entity.PlayerDataChart{
				ID:             20,
				SongID:         200,
				DifficultyID:   5,
				Const:          mustChartConstantForTest(t, 13.5),
				IsConstUnknown: false,
			},
			expectedConst: 13.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			playerDataChart := tt.entity

			// When
			got := FromPlayerDataChartEntity(&playerDataChart)

			// Then
			if got.Const != tt.expectedConst {
				t.Fatalf("const = %v, want %v", got.Const, tt.expectedConst)
			}
			if got.ID != playerDataChart.ID {
				t.Fatalf("id = %d, want %d", got.ID, playerDataChart.ID)
			}
			if got.SongID != playerDataChart.SongID {
				t.Fatalf("song_id = %d, want %d", got.SongID, playerDataChart.SongID)
			}
			if got.DifficultyID != playerDataChart.DifficultyID {
				t.Fatalf("difficulty_id = %d, want %d", got.DifficultyID, playerDataChart.DifficultyID)
			}
			if got.IsConstUnknown != playerDataChart.IsConstUnknown {
				t.Fatalf("is_const_unknown = %v, want %v", got.IsConstUnknown, playerDataChart.IsConstUnknown)
			}
		})
	}
}

func mustChartConstantForTest(t *testing.T, value float64) chartconstant.ChartConstant {
	t.Helper()

	got, err := chartconstant.NewChartConstant(value)
	if err != nil {
		t.Fatalf("chartconstant.NewChartConstant failed: %v", err)
	}

	return got
}
