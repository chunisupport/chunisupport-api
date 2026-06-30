package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSongRepositoryToChartEntity_不正な譜面定数ならエラーを返す(t *testing.T) {
	// Given
	repo := &songRepository{}
	row := &chartRow{
		ID:    1,
		Const: 13.71,
	}

	// When
	chart, err := repo.toChartEntity(row)

	// Then
	require.Error(t, err)
	assert.Nil(t, chart)
	assert.ErrorContains(t, err, "chart constant")
}
