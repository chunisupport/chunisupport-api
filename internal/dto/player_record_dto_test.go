package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
)

func TestToPlayerRecordDTO_OverpowerPercent(t *testing.T) {
	// Given
	recordScore, err := score.NewScore(1009000)
	require.NoError(t, err)

	record := &entity.PlayerRecord{
		Score:       recordScore,
		ComboLampID: 3,
		Chart: &entity.Chart{
			Const: chartconstant.ChartConstant(14.0),
		},
	}

	// When
	actual := ToPlayerRecordDTO(record)

	// Then
	require.NotNil(t, actual)
	assert.Equal(t, service.CalcSingleOverpowerPercent(1009000, 14.0, 3), actual.OverpowerPercent)
}

func TestToPlayerRecordDTO_IsOPTarget(t *testing.T) {
	// Given
	recordScore, err := score.NewScore(1009000)
	require.NoError(t, err)

	record := &entity.PlayerRecord{
		Score:      recordScore,
		IsOPTarget: true,
	}

	// When
	actual := ToPlayerRecordDTO(record)

	// Then
	require.NotNil(t, actual)
	assert.True(t, actual.IsOPTarget)
}

func TestToPlayerRecordDTO_JusticeCount(t *testing.T) {
	tests := []struct {
		name        string
		score       uint32
		comboLampID int
		notes       *int
		expected    *int
	}{
		{
			name:        "ALL JUSTICEかつノーツ数ありの場合JUSTICE数を計算する",
			score:       1009975,
			comboLampID: 3,
			notes:       intPtr(200),
			expected:    intPtr(1),
		},
		{
			name:        "理論値の場合ノーツ数不明でも0を返す",
			score:       1010000,
			comboLampID: 0,
			notes:       nil,
			expected:    intPtr(0),
		},
		{
			name:        "ALL JUSTICEでない場合nilを返す",
			score:       1009975,
			comboLampID: 2,
			notes:       intPtr(200),
			expected:    nil,
		},
		{
			name:        "ノーツ数不明の場合nilを返す",
			score:       1009975,
			comboLampID: 3,
			notes:       nil,
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			recordScore, err := score.NewScore(tt.score)
			require.NoError(t, err)

			record := &entity.PlayerRecord{
				Score:       recordScore,
				ComboLampID: tt.comboLampID,
				Chart:       &entity.Chart{},
			}
			if tt.notes != nil {
				notesValue, err := notes.NewNotes(*tt.notes)
				require.NoError(t, err)
				record.Chart.Notes = &notesValue
			}

			// When
			actual := ToPlayerRecordDTO(record)

			// Then
			require.NotNil(t, actual)
			assert.Equal(t, tt.expected, actual.JusticeCount)
		})
	}
}

func intPtr(value int) *int {
	return &value
}
