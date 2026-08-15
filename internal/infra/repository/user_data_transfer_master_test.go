package repository

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferGoalAttributesRoundTrip(t *testing.T) {
	masters := &transferMasterData{
		difficultyIDs:   map[string]int{"MASTER": 4},
		difficultyNames: map[int]string{4: "MASTER"},
		genreIDs:        map[string]int{"POPS&ANIME": 2},
		genreNames:      map[int]string{2: "POPS&ANIME"},
		versionIDs:      map[string]int{"VERSE": 7},
		versionNames:    map[int]string{7: "VERSE"},
	}
	internal := json.RawMessage("{\"diff\":4,\"genre\":[2],\"ver\":7,\"const\":{\"min\":14.0,\"max\":15.0}}")

	external, err := masters.externalizeGoalAttributes(internal)
	require.NoError(t, err)
	assert.JSONEq(t, "{\"diff\":\"MASTER\",\"genre\":[\"POPS&ANIME\"],\"ver\":\"VERSE\",\"const\":{\"min\":14.0,\"max\":15.0}}", string(external))

	restored, err := masters.internalizeGoalAttributes(external)
	require.NoError(t, err)
	assert.JSONEq(t, string(internal), string(restored))
}

func TestFindTransferUnresolvedReferencesCollectsAndSorts(t *testing.T) {
	name, err := playername.NewPlayerName("テスト")
	require.NoError(t, err)
	parsedScore, err := score.NewScore(1000000)
	require.NoError(t, err)
	snapshot := &entity.UserDataTransferSnapshot{
		Player:                   entity.UserDataTransferPlayer{Name: name, Level: 1, CreatedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		Records:                  []entity.UserDataTransferRecord{{SongOfficialIdx: "999", Difficulty: "MASTER", Score: parsedScore, ClearLampName: "NONE", ComboLampName: "NONE", FullChainName: "NONE", SlotName: "none", UpdatedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)}},
		RecordHistories:          []entity.UserDataTransferRecordHistory{},
		WorldsendRecords:         []entity.UserDataTransferWorldsendRecord{},
		WorldsendRecordHistories: []entity.UserDataTransferWorldsendRecordHistory{},
		MetricHistories:          []entity.UserDataTransferMetricHistory{},
		CourseRecords:            []entity.UserDataTransferCourseRecord{},
		Honors:                   []entity.UserDataTransferHonor{},
		FavoriteSongs:            []entity.UserDataTransferFavoriteSong{},
		LockedSongs:              []entity.UserDataTransferLockedSong{},
		Goals:                    entity.UserDataTransferGoals{Groups: []entity.UserDataTransferGoalGroup{}, Ungrouped: []entity.UserDataTransferGoal{}},
		RecordFilters:            []entity.UserDataTransferRecordFilter{},
	}
	masters := &transferMasterData{
		songIDs: map[string]int{}, charts: map[string]transferChartMaster{}, worldsendChartIDs: map[string]int{}, courseIDs: map[string]int{},
		clearLampIDs: map[string]int{"NONE": 1}, comboLampIDs: map[string]int{"NONE": 1}, fullChainIDs: map[string]int{"NONE": 1}, slotIDs: map[string]int{"none": 1},
		classEmblemIDs: map[string]int{}, classEmblemBaseIDs: map[string]int{}, honorIDsByImage: map[string]int{}, honorIDsByNameAndType: map[string]int{},
		achievementTypeIDs: map[string]int{}, difficultyIDs: map[string]int{"MASTER": 4}, difficultyNames: map[int]string{4: "MASTER"},
		genreIDs: map[string]int{}, genreNames: map[int]string{}, versionIDs: map[string]int{}, versionNames: map[int]string{},
	}

	got := findTransferUnresolvedReferences(snapshot, masters)

	assert.Equal(t, []string{"chart:999/MASTER"}, got)
}
func TestCalculateTransferDerivedMetricsExcludesLockedAndDeletedSongsFromOverpower(t *testing.T) {
	name, err := playername.NewPlayerName("テスト")
	require.NoError(t, err)
	parsedScore, err := score.NewScore(1010000)
	require.NoError(t, err)
	updatedAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	snapshot := &entity.UserDataTransferSnapshot{
		Player: entity.UserDataTransferPlayer{Name: name, Level: 1, CreatedAt: updatedAt},
		Records: []entity.UserDataTransferRecord{
			{SongOfficialIdx: "1", Difficulty: "MASTER", Score: parsedScore, ComboLampName: "ALL JUSTICE", SlotName: "best", UpdatedAt: updatedAt},
			{SongOfficialIdx: "2", Difficulty: "MASTER", Score: parsedScore, ComboLampName: "ALL JUSTICE", SlotName: "none", UpdatedAt: updatedAt},
			{SongOfficialIdx: "3", Difficulty: "MASTER", Score: parsedScore, ComboLampName: "ALL JUSTICE", SlotName: "none", UpdatedAt: updatedAt},
		},
		LockedSongs: []entity.UserDataTransferLockedSong{{SongOfficialIdx: "2", IsUltima: false}},
	}
	masters := &transferMasterData{
		charts: map[string]transferChartMaster{
			transferChartKey("1", "MASTER"): {ID: 1, SongID: 1, OfficialIdx: "1", Difficulty: "MASTER", ChartConst: 15},
			transferChartKey("2", "MASTER"): {ID: 2, SongID: 2, OfficialIdx: "2", Difficulty: "MASTER", ChartConst: 15},
			transferChartKey("3", "MASTER"): {ID: 3, SongID: 3, OfficialIdx: "3", Difficulty: "MASTER", ChartConst: 15, IsDeleted: true},
		},
		comboLampIDs: map[string]int{"ALL JUSTICE": 3},
	}

	_, _, _, got := calculateTransferDerivedMetrics(snapshot, masters)
	want := service.CalcSingleOverpower(1010000, 15, 3)

	assert.Equal(t, want, got)
}
