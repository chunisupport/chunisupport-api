package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
)

func TestToWorldsendRecordDTO_JusticeCount(t *testing.T) {
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

			record := &entity.PlayerWorldsendRecord{
				Score:          recordScore,
				ComboLampID:    tt.comboLampID,
				WorldsendChart: &entity.WorldsendChart{},
			}
			if tt.notes != nil {
				notesValue, err := notes.NewNotes(*tt.notes)
				require.NoError(t, err)
				record.WorldsendChart.Notes = &notesValue
			}

			// When
			actual := ToWorldsendRecordDTO(record)

			// Then
			require.NotNil(t, actual)
			assert.Equal(t, tt.expected, actual.JusticeCount)
		})
	}
}
