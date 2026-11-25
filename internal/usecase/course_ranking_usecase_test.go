package usecase

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRankCourseRecords_同点を同順位にし本人の行を識別する(t *testing.T) {
	// Given
	records := []repository.CourseRankingRecord{
		{UserID: 2, Score: 3_020_000, ComboLamp: "ALL JUSTICE"},
		{UserID: 1, Score: 3_020_000, ComboLamp: "FULL COMBO"},
		{UserID: 3, Score: 2_800_000, ComboLamp: "NONE"},
	}

	// When
	ranking := rankCourseRecords(records, 1)

	// Then
	require.Len(t, ranking, 3)
	assert.Equal(t, []int{1, 1, 3}, []int{ranking[0].Rank, ranking[1].Rank, ranking[2].Rank})
	assert.Equal(t, []bool{false, true, false}, []bool{ranking[0].IsSelf, ranking[1].IsSelf, ranking[2].IsSelf})
	assert.Nil(t, ranking[2].ComboLamp)
}
