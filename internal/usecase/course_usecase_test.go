package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type courseRepositoryStub struct {
	created           *entity.Course
	displayIDCourse   *entity.Course
	searchedDisplayID string
	saved             *entity.Course
	records           []*entity.PlayerCourseRecord
	latestUpdatedAt   *time.Time
}

func (s *courseRepositoryStub) FindAll(context.Context, repository.Executor, bool) ([]*entity.Course, error) {
	return nil, nil
}
func (s *courseRepositoryStub) FindByDisplayID(_ context.Context, _ repository.Executor, value string, _ bool) (*entity.Course, error) {
	s.searchedDisplayID = value
	if s.displayIDCourse == nil {
		return nil, repository.ErrCourseNotFound
	}
	return s.displayIDCourse, nil
}
func (s *courseRepositoryStub) FindByOfficialIdx(context.Context, repository.Executor, string, bool) (*entity.Course, error) {
	if s.created != nil {
		return s.created, nil
	}
	if s.displayIDCourse != nil {
		return s.displayIDCourse, nil
	}
	return nil, repository.ErrCourseNotFound
}
func (s *courseRepositoryStub) FindByOfficialIdxList(context.Context, repository.Executor, []string) (map[string]*entity.Course, error) {
	return nil, nil
}
func (s *courseRepositoryStub) FindClassByName(context.Context, repository.Executor, string) (*entity.CourseClass, error) {
	return &entity.CourseClass{ID: 1, Name: "1"}, nil
}
func (s *courseRepositoryStub) Create(_ context.Context, _ repository.Executor, course *entity.Course) error {
	s.created = course
	return nil
}

func (s *courseRepositoryStub) Save(_ context.Context, _ repository.Executor, course *entity.Course) error {
	s.saved = course
	return nil
}
func (s *courseRepositoryStub) FindRecordsByPlayerID(context.Context, repository.Executor, int, bool, bool) ([]*entity.PlayerCourseRecord, error) {
	return s.records, nil
}

type courseUserRepositoryStub struct{ user *entity.User }

func (s *courseUserRepositoryStub) FindByID(context.Context, repository.Executor, int) (*entity.User, error) {
	return nil, repository.ErrUserNotFound
}
func (s *courseUserRepositoryStub) FindByIDForUpdate(context.Context, repository.Executor, int) (*entity.User, error) {
	return nil, repository.ErrUserNotFound
}
func (s *courseUserRepositoryStub) FindByUsername(context.Context, repository.Executor, string) (*entity.User, error) {
	return s.user, nil
}
func (s *courseUserRepositoryStub) FindAllWithPlayer(context.Context, repository.Executor, int, int, string) ([]entity.UserWithPlayer, error) {
	return nil, nil
}
func (s *courseUserRepositoryStub) FindAllWithPlayerForAdmin(context.Context, repository.Executor, int, int, string) ([]entity.UserWithPlayer, error) {
	return nil, nil
}
func (s *courseUserRepositoryStub) FindByFirebaseUID(context.Context, repository.Executor, string) (*entity.User, error) {
	return nil, repository.ErrUserNotFound
}
func (s *courseUserRepositoryStub) LinkFirebaseUID(context.Context, repository.Executor, int, *string, string, time.Time) error {
	return nil
}
func (s *courseUserRepositoryStub) DeleteByID(context.Context, repository.Executor, int) error {
	return nil
}
func (s *courseUserRepositoryStub) Save(context.Context, repository.Executor, *entity.User) error {
	return nil
}
func (s *courseRepositoryStub) FindRecordStatesByCourseIDs(context.Context, repository.Executor, int, []int) (map[int]repository.CourseRecordState, error) {
	return nil, nil
}
func (s *courseRepositoryStub) UpsertRecords(context.Context, repository.Executor, []repository.CourseRecordForUpsert) error {
	return nil
}
func (s *courseRepositoryStub) FindLatestUpdatedAt(context.Context, repository.Executor) (*time.Time, error) {
	return s.latestUpdatedAt, nil
}

func TestCourseUsecase_Create_DisplayIDを生成して返す(t *testing.T) {
	repo := &courseRepositoryStub{}
	uc := NewCourseUsecase(nil, repo, nil, nil)

	output, err := uc.Create(context.Background(), CreateCourseInput{Idx: "50020", Name: "通常コース", Class: "1"})

	require.NoError(t, err)
	require.NotNil(t, repo.created)
	assert.True(t, repo.created.DisplayID.IsValid())
	assert.Equal(t, repo.created.DisplayID.String(), output.DisplayID)
}

func TestCourseUsecase_Get_DisplayIDで検索する(t *testing.T) {
	displayID, err := displayid.NewDisplayID("0123456789abcdef")
	require.NoError(t, err)
	repo := &courseRepositoryStub{displayIDCourse: &entity.Course{DisplayID: displayID, OfficialIdx: "50020", Name: "通常コース", CourseClassID: 1}}
	uc := NewCourseUsecase(nil, repo, nil, nil)

	output, err := uc.Get(context.Background(), displayID.String(), false)

	require.NoError(t, err)
	assert.Equal(t, displayID.String(), repo.searchedDisplayID)
	assert.Equal(t, displayID.String(), output.DisplayID)
}

func TestCourseUsecase_Update_DisplayIDで検索して保存する(t *testing.T) {
	displayID, err := displayid.NewDisplayID("0123456789abcdef")
	require.NoError(t, err)
	repo := &courseRepositoryStub{displayIDCourse: &entity.Course{DisplayID: displayID, OfficialIdx: "50020", Name: "変更前", CourseClassID: 1}}
	uc := NewCourseUsecase(nil, repo, nil, nil)

	output, err := uc.Update(context.Background(), displayID.String(), UpdateCourseInput{Name: "変更後", Class: "1"})

	require.NoError(t, err)
	assert.Equal(t, displayID.String(), repo.searchedDisplayID)
	require.NotNil(t, repo.saved)
	assert.Equal(t, "変更後", repo.saved.Name)
	assert.Equal(t, displayID.String(), output.DisplayID)
}

func TestCourseUsecase_DeleteとRestore_DisplayIDで検索して削除状態を保存する(t *testing.T) {
	displayID, err := displayid.NewDisplayID("0123456789abcdef")
	require.NoError(t, err)

	tests := []struct {
		name      string
		operation func(CourseUsecase) error
		expected  bool
	}{
		{name: "削除", operation: func(uc CourseUsecase) error { return uc.Delete(context.Background(), displayID.String()) }, expected: true},
		{name: "復元", operation: func(uc CourseUsecase) error { return uc.Restore(context.Background(), displayID.String()) }, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &courseRepositoryStub{displayIDCourse: &entity.Course{DisplayID: displayID, OfficialIdx: "50020", Name: "コース", CourseClassID: 1, IsDeleted: !tt.expected}}
			uc := NewCourseUsecase(nil, repo, nil, nil)

			err := tt.operation(uc)

			require.NoError(t, err)
			assert.Equal(t, displayID.String(), repo.searchedDisplayID)
			require.NotNil(t, repo.saved)
			assert.Equal(t, tt.expected, repo.saved.IsDeleted)
		})
	}
}

func TestCourseUsecase_GetUserRecord_DisplayIDで対象レコードを選ぶ(t *testing.T) {
	displayID, err := displayid.NewDisplayID("0123456789abcdef")
	require.NoError(t, err)
	otherDisplayID, err := displayid.NewDisplayID("fedcba9876543210")
	require.NoError(t, err)
	playerID := 10
	repo := &courseRepositoryStub{records: []*entity.PlayerCourseRecord{
		{Course: &entity.Course{DisplayID: otherDisplayID, OfficialIdx: "50019", Name: "別コース", CourseClassID: 1}},
		{Course: &entity.Course{DisplayID: displayID, OfficialIdx: "50020", Name: "対象コース", CourseClassID: 1}},
	}}
	userRepo := &courseUserRepositoryStub{user: &entity.User{PlayerID: &playerID}}
	uc := NewCourseUsecase(nil, repo, userRepo, nil)

	output, err := uc.GetUserRecord(context.Background(), "player", nil, displayID.String())

	require.NoError(t, err)
	assert.Equal(t, displayID.String(), output.DisplayID)
	assert.Equal(t, "対象コース", output.Name)
}

func TestCourseUsecase_GetUserRecords_metaUpdatedAtがマスタとレコードの新しい方になる(t *testing.T) {
	displayID, err := displayid.NewDisplayID("0123456789abcdef")
	require.NoError(t, err)
	playerID := 10
	masterAt := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	recordAt := time.Date(2026, 7, 10, 8, 0, 0, 0, time.UTC)
	sameAt := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		playerID        *int
		masterUpdatedAt *time.Time
		recordUpdatedAt *time.Time
		wantUpdatedAt   *time.Time
	}{
		{
			name:            "マスタのみ新しい",
			playerID:        &playerID,
			masterUpdatedAt: &masterAt,
			recordUpdatedAt: &recordAt,
			wantUpdatedAt:   &masterAt,
		},
		{
			name:            "レコードのみ新しい",
			playerID:        &playerID,
			masterUpdatedAt: &recordAt,
			recordUpdatedAt: &masterAt,
			wantUpdatedAt:   &masterAt,
		},
		{
			name:            "両方が同じ",
			playerID:        &playerID,
			masterUpdatedAt: &sameAt,
			recordUpdatedAt: &sameAt,
			wantUpdatedAt:   &sameAt,
		},
		{
			name:            "レコードのみ存在する",
			playerID:        &playerID,
			masterUpdatedAt: nil,
			recordUpdatedAt: &recordAt,
			wantUpdatedAt:   &recordAt,
		},
		{
			name:            "マスタのみ存在する_未プレイ",
			playerID:        &playerID,
			masterUpdatedAt: &masterAt,
			recordUpdatedAt: nil,
			wantUpdatedAt:   &masterAt,
		},
		{
			name:            "マスタのみ存在する_未連携",
			playerID:        nil,
			masterUpdatedAt: &masterAt,
			recordUpdatedAt: nil,
			wantUpdatedAt:   &masterAt,
		},
		{
			name:            "どちらも無い",
			playerID:        &playerID,
			masterUpdatedAt: nil,
			recordUpdatedAt: nil,
			wantUpdatedAt:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var records []*entity.PlayerCourseRecord
			if tt.playerID != nil {
				record := &entity.PlayerCourseRecord{
					Course: &entity.Course{DisplayID: displayID, OfficialIdx: "50020", Name: "対象コース", CourseClassID: 1},
				}
				if tt.recordUpdatedAt != nil {
					record.UpdatedAt = *tt.recordUpdatedAt
				}
				records = []*entity.PlayerCourseRecord{record}
			}
			repo := &courseRepositoryStub{
				records:         records,
				latestUpdatedAt: tt.masterUpdatedAt,
			}
			userRepo := &courseUserRepositoryStub{user: &entity.User{PlayerID: tt.playerID}}
			uc := NewCourseUsecase(nil, repo, userRepo, nil)

			// When
			result, err := uc.GetUserRecords(context.Background(), "player", nil, true)

			// Then
			require.NoError(t, err)
			require.NotNil(t, result)
			if tt.wantUpdatedAt == nil {
				assert.Nil(t, result.UpdatedAt)
			} else {
				require.NotNil(t, result.UpdatedAt)
				assert.True(t, tt.wantUpdatedAt.Equal(*result.UpdatedAt))
			}
			if tt.playerID != nil {
				require.Len(t, result.Records, 1)
				if tt.recordUpdatedAt == nil {
					assert.Nil(t, result.Records[0].UpdatedAt)
				} else {
					require.NotNil(t, result.Records[0].UpdatedAt)
					assert.True(t, tt.recordUpdatedAt.Equal(*result.Records[0].UpdatedAt))
				}
			} else {
				assert.Empty(t, result.Records)
			}
		})
	}
}

func TestCourseUsecase_GetCoursesUpdatedAt_リポジトリの値を返す(t *testing.T) {
	// Given
	expected := time.Date(2026, 7, 14, 12, 34, 56, 0, time.UTC)
	repo := &courseRepositoryStub{latestUpdatedAt: &expected}
	uc := NewCourseUsecase(nil, repo, nil, nil)

	// When
	result, err := uc.GetCoursesUpdatedAt(context.Background())

	// Then
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, expected.Equal(*result))
}

func TestCourseUsecase_GetCoursesUpdatedAt_コースが無い場合はnil(t *testing.T) {
	// Given
	repo := &courseRepositoryStub{}
	uc := NewCourseUsecase(nil, repo, nil, nil)

	// When
	result, err := uc.GetCoursesUpdatedAt(context.Background())

	// Then
	require.NoError(t, err)
	assert.Nil(t, result)
}
