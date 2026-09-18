package api_v1

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewV1SongDTO(t *testing.T) {
	masterDesigner := "譜面作者"
	song := &entity.Song{
		DisplayID:            "v1test1234567890",
		OpTargetDifficultyID: 4,
		Charts: []*entity.Chart{
			{DifficultyID: 2, Const: 9.0},
			{DifficultyID: 4, Const: 13.7, NotesDesigner: &masterDesigner},
		},
	}
	difficultyNamesByID := map[int]string{
		1: "BASIC",
		2: "ADVANCED",
		3: "EXPERT",
		4: "MASTER",
		5: "ULTIMA",
	}
	maxOPCalculated := false

	dto := NewV1SongDTO(song, nil, difficultyNamesByID, func(got *entity.Song) float64 {
		assert.Same(t, song, got)
		maxOPCalculated = true
		return 90
	})

	require.NotNil(t, dto)
	assert.True(t, maxOPCalculated)
	assert.Equal(t, 90.0, dto.MaxOP)
	require.NotNil(t, dto.OpTargetDifficulty)
	assert.Equal(t, "MASTER", *dto.OpTargetDifficulty)
	require.NotNil(t, dto.Charts["ADVANCED"])
	assert.Equal(t, 9.0, dto.Charts["ADVANCED"].Const.Float64())
	require.NotNil(t, dto.Charts["MASTER"])
	assert.Equal(t, masterDesigner, *dto.Charts["MASTER"].NotesDesigner)
	assert.Nil(t, dto.Charts["BASIC"])
	assert.Nil(t, dto.Charts["EXPERT"])
	assert.Nil(t, dto.Charts["ULTIMA"])
}
