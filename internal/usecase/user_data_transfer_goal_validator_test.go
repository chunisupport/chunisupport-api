package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/goalgroupname"
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
	// Given: 難易度をマスタ名ではなく内部IDで指定している
	masters := &domainmasterdata.GoalMasters{DifficultyNamesByID: map[int]string{4: "MASTER"}}

	// When: 移行用の属性を内部IDへ戻す
	_, err := internalizeTransferredGoalAttributes(json.RawMessage("{\"diff\":4}"), masters)

	// Then: 内部IDの指定であることが分かる理由で拒否する
	require.Error(t, err)
	assert.Contains(t, err.Error(), "diff")
	assert.Contains(t, err.Error(), "must contain master names")
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

func TestValidateTransferredRatingCountUsesReachableChartFilter(t *testing.T) {
	repo := &stubGoalRepo{stats: &repository.GoalTargetStats{ChartCount: 1}}
	validator := &goalUsecase{goalRepo: repo, masterProvider: &stubGoalMasterProvider{}}
	goals := entity.UserDataTransferGoals{Ungrouped: []entity.UserDataTransferGoal{
		{Title: "単曲レート18", AchievementType: "rating_count", AchievementParams: json.RawMessage(`{"rating":18.00,"count":1}`), Attributes: json.RawMessage(`{}`)},
	}}

	err := validator.ValidateTransferredGoals(context.Background(), goals)

	require.NoError(t, err)
	require.NotNil(t, repo.lastFilter.MinTheoreticalRatingHundredths)
	assert.Equal(t, int64(1800), *repo.lastFilter.MinTheoreticalRatingHundredths)
}

func TestValidateTransferredRatingCountRejectsNoReachableCharts(t *testing.T) {
	repo := &stubGoalRepo{stats: &repository.GoalTargetStats{ChartCount: 0}}
	validator := &goalUsecase{goalRepo: repo, masterProvider: &stubGoalMasterProvider{}}
	goals := entity.UserDataTransferGoals{Ungrouped: []entity.UserDataTransferGoal{
		{Title: "単曲レート18", AchievementType: "rating_count", AchievementParams: json.RawMessage(`{"rating":18.00}`), Attributes: json.RawMessage(`{}`)},
	}}

	err := validator.ValidateTransferredGoals(context.Background(), goals)

	assert.ErrorIs(t, err, ErrDataTransferInvalidData)
}

func TestValidateTransferredGoals_不正な目標は原因エラーを保持する(t *testing.T) {
	// Given
	validator := &goalUsecase{goalRepo: &stubGoalRepo{}, masterProvider: &stubGoalMasterProvider{}}
	goals := entity.UserDataTransferGoals{Ungrouped: []entity.UserDataTransferGoal{
		{Title: "", AchievementType: "score_count", AchievementParams: json.RawMessage(`{"score":1000000,"count":1}`), Attributes: json.RawMessage(`{}`)},
	}}

	// When
	err := validator.ValidateTransferredGoals(context.Background(), goals)

	// Then
	assert.ErrorIs(t, err, ErrDataTransferInvalidData)
	assert.ErrorIs(t, err, ErrInvalidGoalTitle)
}

func TestValidateTransferredGoals_不正なスタート位置を拒否する(t *testing.T) {
	// Given: 件数系の目標に小数のスタート位置が入っている
	validator := &goalUsecase{goalRepo: &stubGoalRepo{}, masterProvider: &stubGoalMasterProvider{}}
	goals := entity.UserDataTransferGoals{Ungrouped: []entity.UserDataTransferGoal{
		{Title: "100万点を1譜面", AchievementType: "score_count", AchievementParams: json.RawMessage(`{"score":1000000,"count":1}`), Attributes: json.RawMessage(`{}`), GaugeStart: 0.5},
	}}

	// When
	err := validator.ValidateTransferredGoals(context.Background(), goals)

	// Then
	assert.ErrorIs(t, err, ErrDataTransferInvalidData)
	assert.ErrorIs(t, err, ErrInvalidGoalGaugeStart)
}

func TestValidateTransferredGoals_譜面定数の入力モードを検証する(t *testing.T) {
	tests := []struct {
		name       string
		attributes json.RawMessage
		wantErr    error
	}{
		{name: "入力モードを含む目標を受け付ける", attributes: json.RawMessage(`{"diff":"MASTER","const":{"min":14.0,"max":14.9},"const_filter_mode":"level"}`)},
		{name: "入力モードを含まない目標を受け付ける", attributes: json.RawMessage(`{"diff":"MASTER","const":{"min":13.7,"max":14.2}}`)},
		{name: "不正な入力モードを拒否する", attributes: json.RawMessage(`{"diff":"MASTER","const":{"min":14.0,"max":14.9},"const_filter_mode":"range"}`), wantErr: ErrInvalidGoalAttributes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			validator := &goalUsecase{goalRepo: &stubGoalRepo{}, masterProvider: &stubGoalMasterProvider{}}
			goals := entity.UserDataTransferGoals{Ungrouped: []entity.UserDataTransferGoal{
				{Title: "MASTER 14", AchievementType: "score_count", AchievementParams: json.RawMessage(`{"score":1000000,"count":1}`), Attributes: tt.attributes},
			}}

			// When
			err := validator.ValidateTransferredGoals(context.Background(), goals)

			// Then
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, ErrDataTransferInvalidData)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateTransferredGoals_グループの目標だけが不正でも拒否する(t *testing.T) {
	// Given: 未所属の目標は妥当で、グループ内の目標だけ題名が空である
	groupName, err := goalgroupname.NewGoalGroupName("グループ01")
	require.NoError(t, err)
	validator := &goalUsecase{goalRepo: &stubGoalRepo{}, masterProvider: &stubGoalMasterProvider{}}
	goals := entity.UserDataTransferGoals{
		Ungrouped: []entity.UserDataTransferGoal{
			{Title: "妥当な目標", AchievementType: "score_count", AchievementParams: json.RawMessage(`{"score":1000000,"count":1}`), Attributes: json.RawMessage(`{}`)},
		},
		Groups: []entity.UserDataTransferGoalGroup{{
			Name:      groupName,
			SortOrder: 1,
			CreatedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			Goals: []entity.UserDataTransferGoal{
				{Title: "", AchievementType: "score_count", AchievementParams: json.RawMessage(`{"score":1000000,"count":1}`), Attributes: json.RawMessage(`{}`)},
			},
		}},
	}

	// When
	err = validator.ValidateTransferredGoals(context.Background(), goals)

	// Then: グループ内の不正が理由で取り込みを拒否する
	assert.ErrorIs(t, err, ErrDataTransferInvalidData)
	assert.ErrorIs(t, err, ErrInvalidGoalTitle)
}

func TestValidateTransferredGoals_属性に未知のマスタ名があれば拒否する(t *testing.T) {
	// Given: 題名と達成条件は妥当で、ジャンル名だけ存在しない値である
	validator := &goalUsecase{goalRepo: &stubGoalRepo{}, masterProvider: &stubGoalMasterProvider{}}
	goals := entity.UserDataTransferGoals{Ungrouped: []entity.UserDataTransferGoal{
		{Title: "妥当な目標", AchievementType: "score_count", AchievementParams: json.RawMessage(`{"score":1000000,"count":1}`), Attributes: json.RawMessage(`{"genre":"存在しないジャンル"}`)},
	}}

	// When
	err := validator.ValidateTransferredGoals(context.Background(), goals)

	// Then: 未知のマスタ名が理由で取り込みを拒否する
	assert.ErrorIs(t, err, ErrDataTransferInvalidData)
	assert.Contains(t, err.Error(), "unknown master name")
}

// mismatchedStatsGoalRepo は統計の件数を目標の件数と合わなく返す検証用のリポジトリです。
// 内部不整合の分岐を確かめるために使います。
type mismatchedStatsGoalRepo struct {
	*stubGoalRepo
}

func (s *mismatchedStatsGoalRepo) GetTargetStatsBatch(context.Context, repository.Executor, []repository.GoalTargetFilter) ([]repository.GoalTargetStats, error) {
	return []repository.GoalTargetStats{}, nil
}

func TestValidateTransferredGoals_統計の件数が合わなければ内部エラーにする(t *testing.T) {
	// Given: 目標自体は妥当だが、統計が0件しか返らない状態にする
	validator := &goalUsecase{goalRepo: &mismatchedStatsGoalRepo{stubGoalRepo: &stubGoalRepo{}}, masterProvider: &stubGoalMasterProvider{}}
	goals := entity.UserDataTransferGoals{Ungrouped: []entity.UserDataTransferGoal{
		{Title: "100万点を1譜面", AchievementType: "score_count", AchievementParams: json.RawMessage(`{"score":1000000,"count":1}`), Attributes: json.RawMessage(`{}`)},
	}}

	// When
	err := validator.ValidateTransferredGoals(context.Background(), goals)

	// Then: 内部不整合として報告する
	assert.ErrorIs(t, err, ErrInternalError)
}
