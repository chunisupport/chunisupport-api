package handler

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildChartsMap_難易度IDが非連番でも通常譜面キーを初期化する(t *testing.T) {
	difficultyNames := map[int]string{
		10: "BASIC",
		30: "ADVANCED",
		50: "EXPERT",
		70: "MASTER",
		90: "ULTIMA",
	}
	charts := []*entity.Chart{
		{DifficultyID: 50},
		{DifficultyID: 70},
	}

	chartsMap := BuildChartsMap(charts, difficultyNames, func(c *entity.Chart) int {
		return c.DifficultyID
	})

	require.Len(t, chartsMap, 5)
	assert.Equal(t, 0, chartsMap["BASIC"])
	assert.Equal(t, 0, chartsMap["ADVANCED"])
	assert.Equal(t, 50, chartsMap["EXPERT"])
	assert.Equal(t, 70, chartsMap["MASTER"])
	assert.Equal(t, 0, chartsMap["ULTIMA"])
}

func TestBuildChartsMap_通常譜面以外の難易度は初期化対象外(t *testing.T) {
	difficultyNames := map[int]string{
		10: "BASIC",
		20: "ADVANCED",
		30: "EXPERT",
		40: "MASTER",
		50: "ULTIMA",
		99: "WORLD'S END",
	}
	charts := []*entity.Chart{
		{DifficultyID: 99},
	}

	chartsMap := BuildChartsMap(charts, difficultyNames, func(c *entity.Chart) int {
		return c.DifficultyID
	})

	require.Len(t, chartsMap, 5)
	_, exists := chartsMap["WORLD'S END"]
	assert.False(t, exists)
}
