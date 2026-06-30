package entity

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSongChangeChartConstant(t *testing.T) {
	// Given
	original, err := chartconstant.NewChartConstant(14.5)
	require.NoError(t, err)
	updated, err := chartconstant.NewChartConstant(14.7)
	require.NoError(t, err)
	song := &Song{Charts: []*Chart{{
		DifficultyID:   4,
		Const:          original,
		IsConstUnknown: true,
	}}}

	// When
	err = song.ChangeChartConstant(4, updated)

	// Then
	require.NoError(t, err)
	assert.Equal(t, updated, song.Charts[0].Const)
	assert.False(t, song.Charts[0].IsConstUnknown)
}

func TestSongChangeChartConstant_対象譜面がなければエラー(t *testing.T) {
	// Given
	constant, err := chartconstant.NewChartConstant(14.7)
	require.NoError(t, err)
	song := &Song{Charts: []*Chart{}}

	// When
	err = song.ChangeChartConstant(4, constant)

	// Then
	require.ErrorIs(t, err, ErrChartNotFound)
}
