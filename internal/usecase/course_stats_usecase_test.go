package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type courseStatsCourseRepository struct {
	repository.CourseRepository
	course         *entity.Course
	includeDeleted []bool
}

func (r *courseStatsCourseRepository) FindByDisplayID(_ context.Context, _ repository.Executor, _ string, includeDeleted bool) (*entity.Course, error) {
	r.includeDeleted = append(r.includeDeleted, includeDeleted)
	if r.course == nil || (r.course.IsDeleted && !includeDeleted) {
		return nil, repository.ErrCourseNotFound
	}
	return r.course, nil
}

type courseStatsRepositoryStub struct {
	stats     []*entity.CourseStatsByRatingBand
	err       error
	courseIDs []int
}

func (r *courseStatsRepositoryStub) FindByCourseID(_ context.Context, _ repository.Executor, courseID int) ([]*entity.CourseStatsByRatingBand, error) {
	r.courseIDs = append(r.courseIDs, courseID)
	return r.stats, r.err
}

type courseStatsMasterProvider struct{ bands []*ratingband.RatingBand }

func (p courseStatsMasterProvider) RatingBands() []*ratingband.RatingBand { return p.bands }

func newCourseStatsTestCourse(t *testing.T, deleted bool) *entity.Course {
	t.Helper()
	id, err := displayid.NewDisplayID("0123456789abcdef")
	require.NoError(t, err)
	return &entity.Course{ID: 501, DisplayID: id, IsDeleted: deleted}
}

func TestCourseStatsUsecase_GetByDisplayID_レーティング帯の表示順で返す(t *testing.T) {
	// Given: 保存順（帯ID順）と表示順が異なり、ALL は表示順の先頭にある。
	bands := courseStatsMasterProvider{bands: []*ratingband.RatingBand{
		{ID: 0, Label: "ALL", SortOrder: 0}, {ID: 30, Label: "15.0", SortOrder: 2}, {ID: 5, Label: "17.0", SortOrder: 22},
	}}
	statsRepo := &courseStatsRepositoryStub{stats: []*entity.CourseStatsByRatingBand{
		{CourseID: 501, RatingBandID: 5, PlayerCount: 1},
		{CourseID: 501, RatingBandID: 30, PlayerCount: 2},
		{CourseID: 501, RatingBandID: 0, PlayerCount: 3},
	}}
	courseRepo := &courseStatsCourseRepository{course: newCourseStatsTestCourse(t, false)}
	uc := NewCourseStatsUsecase(courseRepo, statsRepo, bands, &MockExecutor{})

	// When
	got, err := uc.GetByDisplayID(context.Background(), "0123456789abcdef")

	// Then
	require.NoError(t, err)
	assert.Equal(t, "0123456789abcdef", got.CourseDisplayID)
	require.Len(t, got.Stats, 3)
	assert.Equal(t, []int{0, 30, 5}, []int{got.Stats[0].RatingBandID, got.Stats[1].RatingBandID, got.Stats[2].RatingBandID})
	assert.Equal(t, []int{501}, statsRepo.courseIDs)
	assert.Equal(t, []bool{false}, courseRepo.includeDeleted)
}

func TestCourseStatsUsecase_GetByDisplayID_未集計とエラー(t *testing.T) {
	statsErr := errors.New("query failed")
	tests := []struct {
		name string
		// Given
		course   *entity.Course
		statsErr error
		// Then
		wantErr       error
		wantStatsRead bool
	}{
		{name: "未集計のコースは空の統計を返す", course: newCourseStatsTestCourse(t, false), wantStatsRead: true},
		{name: "存在しないコースは統計を読まない", wantErr: repository.ErrCourseNotFound},
		{name: "削除済みコースは存在しないものとして扱う", course: newCourseStatsTestCourse(t, true), wantErr: repository.ErrCourseNotFound},
		{name: "統計の読み取り失敗を返す", course: newCourseStatsTestCourse(t, false), statsErr: statsErr, wantErr: statsErr, wantStatsRead: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			statsRepo := &courseStatsRepositoryStub{stats: []*entity.CourseStatsByRatingBand{}, err: tt.statsErr}
			uc := NewCourseStatsUsecase(&courseStatsCourseRepository{course: tt.course}, statsRepo, courseStatsMasterProvider{}, &MockExecutor{})

			// When
			got, err := uc.GetByDisplayID(context.Background(), "0123456789abcdef")

			// Then
			assert.Equal(t, tt.wantStatsRead, len(statsRepo.courseIDs) == 1)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Empty(t, got.Stats)
		})
	}
}
