package usecase

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPlayerLockedSongMasterProvider struct {
	masters *masterdata.SongMasters
}

func (s *stubPlayerLockedSongMasterProvider) SongMasters() *masterdata.SongMasters {
	return s.masters
}

func TestPlayerLockedSongUltimaDifficulty(t *testing.T) {
	// Given
	expected := master.ChartDifficulty{ID: 99, Name: difficultyNameUltima, SortOrder: 4}
	u := &playerLockedSongUsecase{
		masterProvider: &stubPlayerLockedSongMasterProvider{
			masters: &masterdata.SongMasters{
				Difficulties: map[string]master.ChartDifficulty{
					difficultyNameUltima: expected,
				},
			},
		},
	}

	// When
	actual, err := u.ultimaDifficulty()

	// Then
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPlayerLockedSongDifficultyChart(t *testing.T) {
	tests := []struct {
		name       string
		chartIDs   []int
		difficulty master.ChartDifficulty
		expected   bool
	}{
		{
			name:       "対象難易度の譜面が存在する場合true",
			chartIDs:   []int{10, 20, 30},
			difficulty: master.ChartDifficulty{ID: 20, Name: difficultyNameUltima, SortOrder: 4},
			expected:   true,
		},
		{
			name:       "対象難易度の譜面が存在しない場合false",
			chartIDs:   []int{10, 20, 30},
			difficulty: master.ChartDifficulty{ID: 40, Name: difficultyNameUltima, SortOrder: 4},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			song := &entity.Song{Charts: make([]*entity.Chart, 0, len(tt.chartIDs))}
			for _, chartID := range tt.chartIDs {
				song.Charts = append(song.Charts, &entity.Chart{DifficultyID: chartID})
			}

			// When
			actual := hasDifficultyChart(song, tt.difficulty)

			// Then
			assert.Equal(t, tt.expected, actual)
		})
	}
}
