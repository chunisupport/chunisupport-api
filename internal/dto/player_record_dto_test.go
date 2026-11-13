package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
)

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
			notes:       new(200),
			expected:    new(1),
		},
		{
			// 200×24÷10000=0.48は0へ丸める。常時切り上げへの退行を検知する。
			name:        "丸め境界の切り捨て側（1009976・ノーツ200）は0になる",
			score:       1009976,
			comboLampID: 3,
			notes:       new(200),
			expected:    new(0),
		},
		{
			// 200×26÷10000=0.52は1へ丸め、丸め境界の両側を通常譜面側で保証する。
			name:        "丸め境界の切り上げ側（1009974・ノーツ200）は1になる",
			score:       1009974,
			comboLampID: 3,
			notes:       new(200),
			expected:    new(1),
		},
		{
			name:        "理論値の場合ノーツ数不明でも0を返す",
			score:       1010000,
			comboLampID: 0,
			notes:       nil,
			expected:    new(0),
		},
		{
			// 理論値の直前には特例を適用せず、計算条件を満たさないためnilを返す。
			name:        "理論値直前（1009999）はノーツ数不明かつALL JUSTICEでない場合nilを返す",
			score:       1009999,
			comboLampID: 0,
			notes:       nil,
			expected:    nil,
		},
		{
			name:        "ALL JUSTICEでない場合nilを返す",
			score:       1009975,
			comboLampID: 2,
			notes:       new(200),
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

func TestToPlayerRecordDTO_譜面がない場合は定数による計算を行わない(t *testing.T) {
	recordScore, err := score.NewScore(1010000)
	require.NoError(t, err)
	result := ToPlayerRecordDTO(&entity.PlayerRecord{Score: recordScore, ComboLampID: 3})
	require.NotNil(t, result)
	assert.Zero(t, result.Const)
	assert.Zero(t, result.Rating)
	assert.Zero(t, result.Overpower)
	assert.Zero(t, result.OverpowerPercent)
}
