package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInternalizeTransferredGoalAttributes(t *testing.T) {
	masters := &domainmasterdata.GoalMasters{
		DifficultyNamesByID: map[int]string{4: "MASTER"},
		GenreNamesByID:      map[int]string{2: "POPS&ANIME"},
		VersionsByID:        map[int]domainmasterdata.Version{7: {ID: 7, Name: "VERSE", ReleasedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}},
	}

	got, err := internalizeTransferredGoalAttributes(json.RawMessage("{\"diff\":\"MASTER\",\"genre\":[\"POPS&ANIME\"],\"ver\":\"VERSE\"}"), masters)

	require.NoError(t, err)
	assert.JSONEq(t, "{\"diff\":4,\"genre\":[2],\"ver\":7}", string(got))
}

func TestInternalizeTransferredGoalAttributesRejectsInternalIDs(t *testing.T) {
	masters := &domainmasterdata.GoalMasters{DifficultyNamesByID: map[int]string{4: "MASTER"}}

	_, err := internalizeTransferredGoalAttributes(json.RawMessage("{\"diff\":4}"), masters)

	assert.Error(t, err)
}

func TestValidateTransferredGoalsFetchesDynamicStatsInOneBatch(t *testing.T) {
	repo := &stubGoalRepo{}
	validator := &goalUsecase{goalRepo: repo, masterProvider: &stubGoalMasterProvider{}}
	goals := entity.UserDataTransferGoals{Ungrouped: []entity.UserDataTransferGoal{
		{Title: "100万点を1譜面", AchievementType: "score_count", AchievementParams: json.RawMessage(`{"score":1000000,"count":1}`), Attributes: json.RawMessage(`{}`)},
		{Title: "100万点を2譜面", AchievementType: "score_count", AchievementParams: json.RawMessage(`{"score":1000000,"count":2}`), Attributes: json.RawMessage(`{"diff":"MASTER"}`)},
	}}

	err := validator.ValidateTransferredGoals(context.Background(), goals)

	require.NoError(t, err)
	assert.Equal(t, 1, repo.statsBatchCalls)
}
