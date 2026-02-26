package usecase

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubGoalRepo struct {
	count     int
	goal      *entity.Goal
	updateErr error
	stats     *repository.GoalTargetStats
}

func (s *stubGoalRepo) ListByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.Goal, error) {
	return []*entity.Goal{s.goal}, nil
}
func (s *stubGoalRepo) FindByIDAndUserID(ctx context.Context, exec repository.Executor, id uint32, userID int) (*entity.Goal, error) {
	if s.goal == nil || s.goal.ID != id {
		return nil, sql.ErrNoRows
	}
	return s.goal, nil
}
func (s *stubGoalRepo) Create(ctx context.Context, exec repository.Executor, goal *entity.Goal) error {
	goal.ID = 1
	goal.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.goal = goal
	return nil
}
func (s *stubGoalRepo) Update(ctx context.Context, exec repository.Executor, goal *entity.Goal) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.goal = goal
	return nil
}
func (s *stubGoalRepo) DeleteByIDAndUserID(ctx context.Context, exec repository.Executor, id uint32, userID int) error {
	if s.goal == nil || s.goal.ID != id {
		return sql.ErrNoRows
	}
	s.goal = nil
	return nil
}
func (s *stubGoalRepo) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	return s.count, nil
}
func (s *stubGoalRepo) LockUserByID(ctx context.Context, exec repository.Executor, userID int) error {
	return nil
}
func (s *stubGoalRepo) GetTargetStats(ctx context.Context, exec repository.Executor, filter repository.GoalTargetFilter) (*repository.GoalTargetStats, error) {
	if s.stats == nil {
		return &repository.GoalTargetStats{ChartCount: 1000, TotalChartConst: 17000}, nil
	}
	return s.stats, nil
}

type stubTM struct{}

func (s *stubTM) Transactional(ctx context.Context, f func(tx repository.Executor) error) error {
	return f(nil)
}

type stubGoalMasterProvider struct{}

type stubMissingTypeMasterProvider struct{}

type stubNilGoalMasterProvider struct{}

func (s *stubNilGoalMasterProvider) GoalMasters() *domainmasterdata.GoalMasters {
	return nil
}

func (s *stubGoalMasterProvider) GoalMasters() *domainmasterdata.GoalMasters {
	return &domainmasterdata.GoalMasters{
		AchievementTypesByCode: map[string]domainmasterdata.Item{
			"score_count":       {ID: 2, Name: "score_count"},
			"rank_count":        {ID: 1, Name: "rank_count"},
			"total_score":       {ID: 6, Name: "total_score"},
			"overpower_value":   {ID: 7, Name: "overpower_value"},
			"overpower_percent": {ID: 8, Name: "overpower_percent"},
		},
		AchievementTypesByID: map[int]string{1: "rank_count", 2: "score_count", 6: "total_score", 7: "overpower_value", 8: "overpower_percent"},
		DifficultyNamesByID:  map[int]string{3: "EXPERT", 4: "MASTER"},
		GenreNamesByID:       map[int]string{1: "POPS & ANIME", 2: "niconico"},
		VersionsByID:         map[int]domainmasterdata.Version{20: {ID: 20, Name: "VERSE", ReleasedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}, 21: {ID: 21, Name: "VERSE EP. II", ReleasedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}},
	}
}

func (s *stubMissingTypeMasterProvider) GoalMasters() *domainmasterdata.GoalMasters {
	m := (&stubGoalMasterProvider{}).GoalMasters()
	delete(m.AchievementTypesByID, 2)
	return m
}

func TestGoalUsecase_Create(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	in := &GoalInput{Title: "  test  ", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{"diff":4,"genre":1,"ver":20}`)}
	out, err := u.Create(context.Background(), 1, in)
	require.NoError(t, err)
	assert.Equal(t, "test", out.Title)
	assert.Equal(t, "score_count", out.AchievementType)
}

func TestGoalUsecase_CreateRejectsTitleOver30Runes(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "1234567890123456789012345678901",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidGoalTitle))
}

func TestGoalUsecase_CreateAcceptsTitleAt30Runes(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "123456789012345678901234567890",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{}`),
	})
	require.NoError(t, err)
}

func TestGoalUsecase_CreateLimitExceeded(t *testing.T) {
	repo := &stubGoalRepo{count: 100}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{Title: "test", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)})
	assert.True(t, errors.Is(err, ErrGoalLimitExceeded))
}

func TestGoalUsecase_Delete(t *testing.T) {
	repo := &stubGoalRepo{goal: &entity.Goal{ID: 1, UserID: 1}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	err := u.Delete(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Nil(t, repo.goal)
}

func TestGoalUsecase_DeleteNotFound(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	err := u.Delete(context.Background(), 1, 999)
	assert.True(t, errors.Is(err, ErrGoalNotFound))
}

func TestGoalUsecase_UpdateNotFoundOnSave(t *testing.T) {
	repo := &stubGoalRepo{goal: &entity.Goal{ID: 1, UserID: 1}, updateErr: sql.ErrNoRows}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Update(context.Background(), 1, 1, &GoalInput{
		Title:             "updated",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"diff":4,"genre":1,"ver":20}`),
	})
	assert.True(t, errors.Is(err, ErrGoalNotFound))
}

func TestGoalUsecase_CreateMasterDataUnavailable(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubNilGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{}`),
	})
	assert.True(t, errors.Is(err, ErrInternalError))
}

func TestGoalUsecase_CreateInvalidDifficultyAttribute(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"diff":5,"genre":1,"ver":20}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidGoalAttributes))
}

func TestGoalUsecase_CreateAcceptsDifficultyAsSingleElementArrayAndNormalizesToScalar(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"diff":[4]}`),
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"diff": float64(4)}, out.Attributes)
}

func TestGoalUsecase_CreateAcceptsDifficultyAsMultiArray(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"diff":[3,4]}`),
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"diff": []any{float64(3), float64(4)}}, out.Attributes)
}

func TestGoalUsecase_CreateAcceptsGenreAsMultiArray(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"genre":[1,2]}`),
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"genre": []any{float64(1), float64(2)}}, out.Attributes)
}

func TestGoalUsecase_CreateAcceptsVersionAsMultiArray(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"ver":[20,21]}`),
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"ver": []any{float64(20), float64(21)}}, out.Attributes)
}

func TestGoalUsecase_CreateNormalizesDuplicateDifficultyArrayToScalar(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"diff":[4,4]}`),
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"diff": float64(4)}, out.Attributes)
}

func TestGoalUsecase_CreateRejectsDifficultyArrayIncludingUnknownID(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"diff":[4,99]}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidGoalAttributes))
}

func TestGoalUsecase_CreateRejectsEmptyDifficultyArray(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"diff":[]}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidGoalAttributes))
}

func TestGoalUsecase_CreateConstAttributeWithOmittedMinUsesDefault(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"const":{"max":15.9}}`),
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"const": map[string]any{"min": float64(1), "max": float64(15.9)}}, out.Attributes)
}

func TestGoalUsecase_CreateConstAttributeRejectsMoreThanOneDecimal(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"const":{"min":1.23,"max":15.9}}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidGoalAttributes))
}

func TestGoalUsecase_CreateOverpowerPercentRejectsOver100(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "overpower_percent",
		AchievementParams: []byte(`{"total":123.456}`),
		Attributes:        []byte(`{}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidAchievementParam))
}

func TestGoalUsecase_CreateRejectsControlCharacterInTitle(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "bad\ntitle",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidGoalTitle))
}

func TestGoalUsecase_CreateRejectsUnknownAttributeKey(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"unknown":1}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidGoalAttributes))
}

func TestGoalUsecase_CreateRejectsCountOverDynamicUpperBound(t *testing.T) {
	repo := &stubGoalRepo{stats: &repository.GoalTargetStats{ChartCount: 2, TotalChartConst: 20.0}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":3}`),
		Attributes:        []byte(`{}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidAchievementParam))
}

func TestGoalUsecase_CreateRejectsTotalScoreOverDynamicUpperBound(t *testing.T) {
	repo := &stubGoalRepo{stats: &repository.GoalTargetStats{ChartCount: 2, TotalChartConst: 20.0}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "total_score",
		AchievementParams: []byte(`{"total":2020001}`),
		Attributes:        []byte(`{}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidAchievementParam))
}

func TestGoalUsecase_CreateRejectsOverpowerValueOverDynamicUpperBound(t *testing.T) {
	repo := &stubGoalRepo{stats: &repository.GoalTargetStats{ChartCount: 2, TotalChartConst: 10.0}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "overpower_value",
		AchievementParams: []byte(`{"total":100.0}`),
		Attributes:        []byte(`{}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidAchievementParam))
}

func TestGoalUsecase_ListReturnsInternalErrorWhenAchievementTypeMissing(t *testing.T) {
	repo := &stubGoalRepo{goal: &entity.Goal{ID: 1, UserID: 1, Title: "test", AchievementTypeID: 2, AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubMissingTypeMasterProvider{})
	_, err := u.List(context.Background(), 1)
	assert.True(t, errors.Is(err, ErrInternalError))
}
