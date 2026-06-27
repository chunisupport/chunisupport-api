package usecase

import (
	"context"
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
	count      int
	goal       *entity.Goal
	updateErr  error
	stats      *repository.GoalTargetStats
	lastFilter repository.GoalTargetFilter
}

func (s *stubGoalRepo) ListByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.Goal, error) {
	return []*entity.Goal{s.goal}, nil
}
func (s *stubGoalRepo) FindByIDAndUserID(ctx context.Context, exec repository.Executor, id uint32, userID int) (*entity.Goal, error) {
	if s.goal == nil || s.goal.ID != id {
		return nil, repository.ErrGoalNotFound
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
		return repository.ErrGoalNotFound
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
	s.lastFilter = filter
	if s.stats == nil {
		return &repository.GoalTargetStats{ChartCount: 1000, TotalChartConst: 17000}, nil
	}
	return s.stats, nil
}

type stubTM struct{}

func (s *stubTM) Transactional(ctx context.Context, f func(tx repository.Executor) error) error {
	return f(nil)
}

// trackingTM は Transactional の呼び出しを記録し、渡された executor を closure に渡す。
type trackingTM struct {
	called      bool
	passedFloor repository.Executor
}

func (s *trackingTM) Transactional(ctx context.Context, f func(tx repository.Executor) error) error {
	s.called = true
	return f(s.passedFloor)
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
	repo := &stubGoalRepo{goal: &entity.Goal{ID: 1, UserID: 1}, updateErr: repository.ErrGoalNotFound}
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

func TestGoalUsecase_CreateAttributeIntOrSliceNormalization(t *testing.T) {
	tests := []struct {
		name               string
		attributes         []byte
		expectedAttributes map[string]any
		expectError        bool
	}{
		{
			name:               "難易度を単一要素配列で指定するとスカラーに正規化される",
			attributes:         []byte(`{"diff":[4]}`),
			expectedAttributes: map[string]any{"diff": float64(4)},
			expectError:        false,
		},
		{
			name:               "難易度を複数配列で指定できる",
			attributes:         []byte(`{"diff":[3,4]}`),
			expectedAttributes: map[string]any{"diff": []any{float64(3), float64(4)}},
			expectError:        false,
		},
		{
			name:               "ジャンルを複数配列で指定できる",
			attributes:         []byte(`{"genre":[1,2]}`),
			expectedAttributes: map[string]any{"genre": []any{float64(1), float64(2)}},
			expectError:        false,
		},
		{
			name:               "バージョンを複数配列で指定できる",
			attributes:         []byte(`{"ver":[20,21]}`),
			expectedAttributes: map[string]any{"ver": []any{float64(20), float64(21)}},
			expectError:        false,
		},
		{
			name:               "難易度配列の重複は除去されスカラーに正規化される",
			attributes:         []byte(`{"diff":[4,4]}`),
			expectedAttributes: map[string]any{"diff": float64(4)},
			expectError:        false,
		},
		{
			name:               "存在しない難易度IDを含む配列はエラーになる",
			attributes:         []byte(`{"diff":[4,99]}`),
			expectedAttributes: nil,
			expectError:        true,
		},
		{
			name:               "空の難易度配列はエラーになる",
			attributes:         []byte(`{"diff":[]}`),
			expectedAttributes: nil,
			expectError:        true,
		},
		{
			name:               "diff が null の場合はエラーになる",
			attributes:         []byte(`{"diff":null}`),
			expectedAttributes: nil,
			expectError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := &stubGoalRepo{}
			u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})

			// When
			out, err := u.Create(context.Background(), 1, &GoalInput{
				Title:             "test",
				AchievementType:   "score_count",
				AchievementParams: []byte(`{"score":1000000,"count":1}`),
				Attributes:        tt.attributes,
			})

			// Then
			if tt.expectError {
				assert.True(t, errors.Is(err, ErrInvalidGoalAttributes))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedAttributes, out.Attributes)
		})
	}
}

func TestGoalUsecase_CreateAcceptsOPTargetAttribute(t *testing.T) {
	// Given
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})

	// When
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "OP対象",
		AchievementType:   "overpower_value",
		AchievementParams: []byte(`{"total":10.0}`),
		Attributes:        []byte(`{"chart_target":"OP_TARGET"}`),
	})

	// Then
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"chart_target": "OP_TARGET"}, out.Attributes)
	assert.True(t, repo.lastFilter.OPTargetOnly)
}

func TestGoalUsecase_CreateRejectsOPTargetWithDifficulty(t *testing.T) {
	// Given
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})

	// When
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "OP対象",
		AchievementType:   "overpower_value",
		AchievementParams: []byte(`{"total":10.0}`),
		Attributes:        []byte(`{"chart_target":"OP_TARGET","diff":[3,4]}`),
	})

	// Then
	assert.True(t, errors.Is(err, ErrInvalidGoalAttributes))
}

func TestGoalUsecase_CreateRejectsUnknownChartTarget(t *testing.T) {
	// Given
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})

	// When
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "OP対象",
		AchievementType:   "overpower_value",
		AchievementParams: []byte(`{"total":10.0}`),
		Attributes:        []byte(`{"chart_target":"MASTER_ULTIMA"}`),
	})

	// Then
	assert.True(t, errors.Is(err, ErrInvalidGoalAttributes))
}

func TestGoalUsecase_CreateConstAttributeWithOmittedMinUsesDefault(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"const":{"max":16.0}}`),
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"const": map[string]any{"min": float64(1), "max": float64(16)}}, out.Attributes)
}

func TestGoalUsecase_CreateConstAttributeRejectsMoreThanOneDecimal(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"const":{"min":1.23,"max":16.0}}`),
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

func TestGoalUsecase_CreateRejectsUnknownKeyInScoreCountParams(t *testing.T) {
	repo := &stubGoalRepo{stats: &repository.GoalTargetStats{ChartCount: 2, TotalChartConst: 20.0}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"coutn":1}`),
		Attributes:        []byte(`{}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidAchievementParam))
}

func TestGoalUsecase_CreateRejectsUnknownKeyInTotalScoreParams(t *testing.T) {
	repo := &stubGoalRepo{stats: &repository.GoalTargetStats{ChartCount: 2, TotalChartConst: 20.0}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "total_score",
		AchievementParams: []byte(`{"totla":100}`),
		Attributes:        []byte(`{}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidAchievementParam))
}

func TestGoalUsecase_CreateRejectsUnknownKeyInOverpowerValueParams(t *testing.T) {
	repo := &stubGoalRepo{stats: &repository.GoalTargetStats{ChartCount: 2, TotalChartConst: 20.0}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "overpower_value",
		AchievementParams: []byte(`{"totla":12.345}`),
		Attributes:        []byte(`{}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidAchievementParam))
}

func TestGoalUsecase_CreateAcceptsOmittedCountForScoreCount(t *testing.T) {
	repo := &stubGoalRepo{stats: &repository.GoalTargetStats{ChartCount: 2, TotalChartConst: 20.0}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000}`),
		Attributes:        []byte(`{}`),
	})
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestGoalUsecase_CreateAcceptsNullCountForScoreCount(t *testing.T) {
	repo := &stubGoalRepo{stats: &repository.GoalTargetStats{ChartCount: 2, TotalChartConst: 20.0}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":null}`),
		Attributes:        []byte(`{}`),
	})
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestGoalUsecase_CreateAcceptsOmittedTotalForTotalScore(t *testing.T) {
	repo := &stubGoalRepo{stats: &repository.GoalTargetStats{ChartCount: 2, TotalChartConst: 20.0}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "total_score",
		AchievementParams: []byte(`{}`),
		Attributes:        []byte(`{}`),
	})
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestGoalUsecase_CreateAcceptsNullTotalForOverpowerValue(t *testing.T) {
	repo := &stubGoalRepo{stats: &repository.GoalTargetStats{ChartCount: 2, TotalChartConst: 20.0}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "overpower_value",
		AchievementParams: []byte(`{"total":null}`),
		Attributes:        []byte(`{}`),
	})
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestGoalUsecase_CreateRemainingValidation(t *testing.T) {
	tests := []struct {
		name            string
		achievementType string
		params          []byte
		stats           *repository.GoalTargetStats
		wantErr         bool
	}{
		{
			name:            "件数系の整数remainingを受理する",
			achievementType: "score_count",
			params:          []byte(`{"score":1000000,"remaining":1}`),
		},
		{
			name:            "件数系の小数remainingを拒否する",
			achievementType: "score_count",
			params:          []byte(`{"score":1000000,"remaining":1.5}`),
			wantErr:         true,
		},
		{
			name:            "合計スコアの整数remainingを受理する",
			achievementType: "total_score",
			params:          []byte(`{"remaining":1000000}`),
		},
		{
			name:            "合計スコアの小数remainingを拒否する",
			achievementType: "total_score",
			params:          []byte(`{"remaining":1.5}`),
			wantErr:         true,
		},
		{
			name:            "OP値の小数remainingを受理する",
			achievementType: "overpower_value",
			params:          []byte(`{"remaining":1.234}`),
		},
		{
			name:            "動的上限を超えるremainingを拒否する",
			achievementType: "score_count",
			params:          []byte(`{"score":1000000,"remaining":3}`),
			stats:           &repository.GoalTargetStats{ChartCount: 2, TotalChartConst: 20},
			wantErr:         true,
		},
		{
			name:            "絶対値とremainingの同時指定を拒否する",
			achievementType: "total_score",
			params:          []byte(`{"total":1000000,"remaining":100}`),
			wantErr:         true,
		},
		{
			name:            "nullの件数とremainingの同時指定を受理する",
			achievementType: "score_count",
			params:          []byte(`{"score":1000000,"count":null,"remaining":1}`),
		},
		{
			name:            "nullの合計値とremainingの同時指定を受理する",
			achievementType: "total_score",
			params:          []byte(`{"total":null,"remaining":100}`),
		},
		{
			name:            "nullの合計値とremainingの同時指定を受理する_OP値",
			achievementType: "overpower_value",
			params:          []byte(`{"total":null,"remaining":1.234}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := &stubGoalRepo{stats: tt.stats}
			u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})

			// When
			out, err := u.Create(context.Background(), 1, &GoalInput{
				Title:             "test",
				AchievementType:   tt.achievementType,
				AchievementParams: tt.params,
				Attributes:        []byte(`{}`),
			})

			// Then
			if tt.wantErr {
				assert.True(t, errors.Is(err, ErrInvalidAchievementParam))
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, out)
		})
	}
}

func TestGoalUsecase_CreatePercentValidation(t *testing.T) {
	tests := []struct {
		name            string
		achievementType string
		params          []byte
		wantErr         bool
	}{
		{
			name:            "件数系の割合を受理する",
			achievementType: "score_count",
			params:          []byte(`{"score":1000000,"percent":50.5}`),
		},
		{
			name:            "0パーセントを受理する",
			achievementType: "total_score",
			params:          []byte(`{"percent":0}`),
		},
		{
			name:            "100パーセントを受理する",
			achievementType: "overpower_value",
			params:          []byte(`{"percent":100}`),
		},
		{
			name:            "100を超える割合を拒否する",
			achievementType: "score_count",
			params:          []byte(`{"score":1000000,"percent":100.1}`),
			wantErr:         true,
		},
		{
			name:            "負の割合を拒否する",
			achievementType: "total_score",
			params:          []byte(`{"percent":-0.1}`),
			wantErr:         true,
		},
		{
			name:            "件数と割合の同時指定を拒否する",
			achievementType: "score_count",
			params:          []byte(`{"score":1000000,"count":1,"percent":50}`),
			wantErr:         true,
		},
		{
			name:            "OP値の小数第4位の割合を拒否する",
			achievementType: "overpower_value",
			params:          []byte(`{"percent":50.1234}`),
			wantErr:         true,
		},
		{
			name:            "nullの件数と割合の同時指定を受理する",
			achievementType: "score_count",
			params:          []byte(`{"score":1000000,"count":null,"percent":50}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := &stubGoalRepo{}
			u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})

			// When
			out, err := u.Create(context.Background(), 1, &GoalInput{
				Title:             "test",
				AchievementType:   tt.achievementType,
				AchievementParams: tt.params,
				Attributes:        []byte(`{}`),
			})

			// Then
			if tt.wantErr {
				assert.True(t, errors.Is(err, ErrInvalidAchievementParam))
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, out)
		})
	}
}

func TestGoalUsecase_Update(t *testing.T) {
	// Given: 既存の Goal が存在する状態
	repo := &stubGoalRepo{goal: &entity.Goal{ID: 1, UserID: 1, AchievementTypeID: 2, AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)}}
	tm := &trackingTM{passedFloor: nil}
	u := NewGoalUsecase(nil, tm, repo, &stubGoalMasterProvider{})

	// When: Update を呼び出す
	out, err := u.Update(context.Background(), 1, 1, &GoalInput{
		Title:             "updated",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"diff":4,"genre":1,"ver":20}`),
	})

	// Then: トランザクションマネージャが使用され、更新された内容が返却される
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "updated", out.Title)
	assert.Equal(t, "score_count", out.AchievementType)
	assert.True(t, tm.called, "Update は Transactional を経由すべき")
}

func TestGoalUsecase_UpdateReturnsNotFoundWhenGoalMissing(t *testing.T) {
	// Given: 対象 Goal が存在しない
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})

	// When
	_, err := u.Update(context.Background(), 1, 999, &GoalInput{
		Title:             "updated",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{}`),
	})

	// Then
	assert.True(t, errors.Is(err, ErrGoalNotFound))
}

func TestGoalUsecase_ListReturnsInternalErrorWhenAchievementTypeMissing(t *testing.T) {
	repo := &stubGoalRepo{goal: &entity.Goal{ID: 1, UserID: 1, Title: "test", AchievementTypeID: 2, AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubMissingTypeMasterProvider{})
	_, err := u.List(context.Background(), 1)
	assert.True(t, errors.Is(err, ErrInternalError))
}
