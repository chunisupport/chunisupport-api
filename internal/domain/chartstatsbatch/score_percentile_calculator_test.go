package chartstatsbatch

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/scorepercentile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScorePercentileCalculator(t *testing.T) {
	// Given: 譜面10に未クリア含む3件、譜面11に1件
	calc := NewScorePercentileCalculator()
	calc.Add(10, 1009000)
	calc.Add(10, 950000) // FAILED でもそのまま集計対象になる
	calc.Add(10, 1010000)
	calc.Add(11, 1000000)

	// When
	results := calc.Results()

	// Then
	require.Len(t, results, 2)
	assert.Equal(t, 10, results[0].TargetID)
	assert.Equal(t, 11, results[1].TargetID)

	snapshot := scorepercentile.NewSnapshot(1, results, nil, nil)
	top := snapshot.LookupChart(10, 1010000)
	require.NotNil(t, top)
	assert.Equal(t, 33.33, *top)

	low := snapshot.LookupChart(10, 950000)
	require.NotNil(t, low)
	assert.Equal(t, 100.00, *low)
}
