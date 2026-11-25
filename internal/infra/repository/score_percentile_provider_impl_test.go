package repository

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/chartstatsbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/scorepercentile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScorePercentileProvider_SnapshotAndRefresh(t *testing.T) {
	// Given: 初回の境界データを投入
	db := setupChartStatsBatchRepositorySQLite(t)
	batchRepo := NewChartStatsBatchRepository(db)
	provider := NewScorePercentileProvider(db)
	ctx := context.Background()

	initial := &chartstatsbatch.Snapshot{
		ChartPercentiles: []scorepercentile.Entry{
			{TargetID: 10, Thresholds: scorepercentile.BuildThresholds([]int{1010000, 1000000})},
		},
		WorldsendChartPercentiles: []scorepercentile.Entry{
			{TargetID: 20, Thresholds: scorepercentile.BuildThresholds([]int{1005000, 990000, 980000, 970000})},
		},
		CoursePercentiles: []scorepercentile.Entry{
			{TargetID: 30, Thresholds: scorepercentile.BuildThresholds([]int{3030000, 2900000})},
		},
	}
	require.NoError(t, batchRepo.ReplaceAll(ctx, initial))

	// When: 初回 Snapshot で DB から読み込まれる
	snap1, err := provider.Snapshot(ctx)
	require.NoError(t, err)

	// Then
	assert.Equal(t, uint64(1), snap1.Revision())
	chartTop := snap1.LookupChart(10, 1010000)
	require.NotNil(t, chartTop)
	assert.Equal(t, 50.00, *chartTop)

	weTop := snap1.LookupWorldsendChart(20, 1005000)
	require.NotNil(t, weTop)
	assert.Equal(t, 25.00, *weTop)

	courseLow := snap1.LookupCourse(30, 2900000)
	require.NotNil(t, courseLow)
	assert.Equal(t, 100.00, *courseLow)

	// When: 世代が変わっていない状態で Refresh してもスナップショットは差し替わらない
	require.NoError(t, provider.Refresh(ctx))
	snapSame, err := provider.Snapshot(ctx)
	require.NoError(t, err)
	assert.Same(t, snap1, snapSame)

	// When: バッチで再構築して世代を進めると Refresh で新しいスナップショットへ差し替わる
	updated := &chartstatsbatch.Snapshot{
		ChartPercentiles: []scorepercentile.Entry{
			{TargetID: 10, Thresholds: scorepercentile.BuildThresholds([]int{1010000, 1009000, 1008000, 1007000})},
		},
	}
	require.NoError(t, batchRepo.ReplaceAll(ctx, updated))
	require.NoError(t, provider.Refresh(ctx))

	snap2, err := provider.Snapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), snap2.Revision())
	updatedTop := snap2.LookupChart(10, 1010000)
	require.NotNil(t, updatedTop)
	assert.Equal(t, 25.00, *updatedTop)
	assert.Nil(t, snap2.LookupWorldsendChart(20, 1005000))
	assert.Nil(t, snap2.LookupCourse(30, 3030000))
}
