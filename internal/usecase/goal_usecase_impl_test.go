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
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubGoalRepo struct {
	count     int
	goal      *entity.Goal
	updateErr error
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

type stubSongRepo struct {
	songs []*entity.Song
}

func (s *stubSongRepo) FindAllExcludingWorldsend(ctx context.Context, exec repository.Executor, includeDeleted bool) ([]*entity.Song, error) {
	return s.songs, nil
}
func (s *stubSongRepo) FindByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*entity.Song, error) {
	return nil, errors.New("not implemented")
}
func (s *stubSongRepo) FindByDisplayIDs(ctx context.Context, exec repository.Executor, displayIDs []string) ([]*entity.Song, error) {
	return nil, errors.New("not implemented")
}
func (s *stubSongRepo) DeleteSong(ctx context.Context, exec repository.Executor, displayID string) error {
	return errors.New("not implemented")
}
func (s *stubSongRepo) RestoreSong(ctx context.Context, exec repository.Executor, displayID string) error {
	return errors.New("not implemented")
}
func (s *stubSongRepo) UpdateSongs(ctx context.Context, exec repository.Executor, songs []*entity.Song) error {
	return errors.New("not implemented")
}

type stubTM struct{}

func (s *stubTM) Transactional(ctx context.Context, f func(tx repository.Executor) error) error {
	return f(nil)
}

type stubGoalMasterProvider struct{}
type stubNilGoalMasterProvider struct{}

func (s *stubNilGoalMasterProvider) GoalMasters() *domainmasterdata.GoalMasters { return nil }

func (s *stubGoalMasterProvider) GoalMasters() *domainmasterdata.GoalMasters {
	releasedAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return &domainmasterdata.GoalMasters{
		AchievementTypesByCode: map[string]domainmasterdata.Item{
			"score_count": {ID: 2, Name: "score_count"}, "overpower_percent": {ID: 8, Name: "overpower_percent"},
		},
		AchievementTypesByID: map[int]string{2: "score_count", 8: "overpower_percent"},
		DifficultyNamesByID:  map[int]string{4: "MASTER"},
		GenreNamesByID:       map[int]string{1: "POPS & ANIME"},
		VersionsByID:         map[int]domainmasterdata.Version{20: {ID: 20, Name: "VERSE", ReleasedAt: releasedAt}},
	}
}

func sampleSongs() []*entity.Song {
	genreID := 1
	releasedAt := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	constValue, _ := chartconstant.NewChartConstant(14.0)
	return []*entity.Song{{ID: 1, GenreID: &genreID, ReleasedAt: &releasedAt, Charts: []*entity.Chart{{ID: 10, DifficultyID: 4, Const: constValue}}}}
}

func newGoalUsecaseForTest(repo *stubGoalRepo) GoalUsecase {
	return NewGoalUsecase(nil, &stubTM{}, repo, &stubSongRepo{songs: sampleSongs()}, &stubGoalMasterProvider{})
}

func TestGoalUsecase_Create(t *testing.T) {
	repo := &stubGoalRepo{}
	u := newGoalUsecaseForTest(repo)
	in := &GoalInput{Title: "  test  ", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{"diff":4,"genre":1,"ver":20}`)}
	out, err := u.Create(context.Background(), 1, in)
	require.NoError(t, err)
	assert.Equal(t, "test", out.Title)
	assert.Equal(t, "score_count", out.AchievementType)
}

func TestGoalUsecase_CreateLimitExceeded(t *testing.T) {
	repo := &stubGoalRepo{count: 100}
	u := newGoalUsecaseForTest(repo)
	_, err := u.Create(context.Background(), 1, &GoalInput{Title: "test", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)})
	assert.True(t, errors.Is(err, ErrGoalLimitExceeded))
}

func TestGoalUsecase_Delete(t *testing.T) {
	repo := &stubGoalRepo{goal: &entity.Goal{ID: 1, UserID: 1}}
	u := newGoalUsecaseForTest(repo)
	err := u.Delete(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Nil(t, repo.goal)
}

func TestGoalUsecase_DeleteNotFound(t *testing.T) {
	repo := &stubGoalRepo{}
	u := newGoalUsecaseForTest(repo)
	err := u.Delete(context.Background(), 1, 999)
	assert.True(t, errors.Is(err, ErrGoalNotFound))
}

func TestGoalUsecase_UpdateNotFoundOnSave(t *testing.T) {
	repo := &stubGoalRepo{goal: &entity.Goal{ID: 1, UserID: 1}, updateErr: sql.ErrNoRows}
	u := newGoalUsecaseForTest(repo)
	_, err := u.Update(context.Background(), 1, 1, &GoalInput{Title: "updated", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{"diff":4,"genre":1,"ver":20}`)})
	assert.True(t, errors.Is(err, ErrGoalNotFound))
}

func TestGoalUsecase_CreateMasterDataUnavailable(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubSongRepo{songs: sampleSongs()}, &stubNilGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{Title: "test", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)})
	assert.True(t, errors.Is(err, ErrInternalError))
}

func TestGoalUsecase_CreateInvalidDifficultyAttribute(t *testing.T) {
	repo := &stubGoalRepo{}
	u := newGoalUsecaseForTest(repo)
	_, err := u.Create(context.Background(), 1, &GoalInput{Title: "test", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{"diff":5,"genre":1,"ver":20}`)})
	assert.True(t, errors.Is(err, ErrInvalidGoalAttributes))
}

func TestGoalUsecase_CreateConstAttributeWithOmittedMinUsesDefault(t *testing.T) {
	repo := &stubGoalRepo{}
	u := newGoalUsecaseForTest(repo)
	out, err := u.Create(context.Background(), 1, &GoalInput{Title: "test", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{"const":{"max":15.9}}`)})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"const": map[string]any{"min": float64(1), "max": float64(15.9)}}, out.Attributes)
}

func TestGoalUsecase_CreateConstAttributeWithOmittedMaxUsesDefault(t *testing.T) {
	repo := &stubGoalRepo{}
	u := newGoalUsecaseForTest(repo)
	out, err := u.Create(context.Background(), 1, &GoalInput{Title: "test", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{"const":{"min":1.2}}`)})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"const": map[string]any{"min": float64(1.2), "max": float64(15.9)}}, out.Attributes)
}

func TestGoalUsecase_CreateOverpowerPercentAcceptsRealValue(t *testing.T) {
	repo := &stubGoalRepo{}
	u := newGoalUsecaseForTest(repo)
	_, err := u.Create(context.Background(), 1, &GoalInput{Title: "test", AchievementType: "overpower_percent", AchievementParams: []byte(`{"total":99.999}`), Attributes: []byte(`{}`)})
	require.NoError(t, err)
}

func TestGoalUsecase_CreateOverpowerPercentRejectsOver100(t *testing.T) {
	repo := &stubGoalRepo{}
	u := newGoalUsecaseForTest(repo)
	_, err := u.Create(context.Background(), 1, &GoalInput{Title: "test", AchievementType: "overpower_percent", AchievementParams: []byte(`{"total":100.001}`), Attributes: []byte(`{}`)})
	assert.True(t, errors.Is(err, ErrInvalidAchievementParam))
}

func TestGoalUsecase_CreateOverpowerPercentRejectsMoreThan3Decimals(t *testing.T) {
	repo := &stubGoalRepo{}
	u := newGoalUsecaseForTest(repo)
	_, err := u.Create(context.Background(), 1, &GoalInput{Title: "test", AchievementType: "overpower_percent", AchievementParams: []byte(`{"total":12.3456}`), Attributes: []byte(`{}`)})
	assert.True(t, errors.Is(err, ErrInvalidAchievementParam))
}

func TestGoalUsecase_CreateRejectsTitleWithLineBreak(t *testing.T) {
	repo := &stubGoalRepo{}
	u := newGoalUsecaseForTest(repo)
	_, err := u.Create(context.Background(), 1, &GoalInput{Title: "test\nabc", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)})
	assert.True(t, errors.Is(err, ErrInvalidGoalTitle))
}

func TestGoalUsecase_CreateRejectsConstAttributeWithTwoDecimals(t *testing.T) {
	repo := &stubGoalRepo{}
	u := newGoalUsecaseForTest(repo)
	_, err := u.Create(context.Background(), 1, &GoalInput{Title: "test", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{"const":{"min":12.34,"max":15.9}}`)})
	assert.True(t, errors.Is(err, ErrInvalidGoalAttributes))
}

func TestGoalUsecase_CreateRejectsCountOverDynamicMax(t *testing.T) {
	repo := &stubGoalRepo{}
	u := newGoalUsecaseForTest(repo)
	_, err := u.Create(context.Background(), 1, &GoalInput{Title: "test", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":2}`), Attributes: []byte(`{}`)})
	assert.True(t, errors.Is(err, ErrInvalidAchievementParam))
}

func TestGoalUsecase_CreateAcceptsCountAtDynamicMax(t *testing.T) {
	repo := &stubGoalRepo{}
	u := newGoalUsecaseForTest(repo)
	_, err := u.Create(context.Background(), 1, &GoalInput{Title: "test", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)})
	require.NoError(t, err)
}

func TestGoalUsecase_ListReturnsInternalErrorWhenAchievementTypeMasterMissing(t *testing.T) {
	repo := &stubGoalRepo{goal: &entity.Goal{ID: 1, UserID: 1, AchievementTypeID: 99, AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)}}
	u := newGoalUsecaseForTest(repo)
	_, err := u.List(context.Background(), 1)
	assert.True(t, errors.Is(err, ErrInternalError))
}
