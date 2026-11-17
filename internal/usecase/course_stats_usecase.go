package usecase

import (
	"cmp"
	"context"
	"slices"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// CourseStatsUsecase はコース統計の取得ユースケースを提供します。
type CourseStatsUsecase interface {
	// GetByDisplayID は指定コースのレーティング帯別統計を、レーティング帯の表示順で返します。
	// 削除済みコースは統計の集計対象外のため、存在しないコースと同じく repository.ErrCourseNotFound を返します。
	GetByDisplayID(ctx context.Context, displayID string) (*entity.CourseStats, error)
}

type courseStatsUsecase struct {
	courseRepo     repository.CourseRepository
	statsRepo      repository.CourseStatsRepository
	masterProvider repository.ChartStatsMasterProvider
	executor       repository.Executor
}

// NewCourseStatsUsecase は CourseStatsUsecase の実装を生成します。
func NewCourseStatsUsecase(
	courseRepo repository.CourseRepository,
	statsRepo repository.CourseStatsRepository,
	masterProvider repository.ChartStatsMasterProvider,
	executor repository.Executor,
) CourseStatsUsecase {
	return &courseStatsUsecase{courseRepo: courseRepo, statsRepo: statsRepo, masterProvider: masterProvider, executor: executor}
}

func (u *courseStatsUsecase) GetByDisplayID(ctx context.Context, displayID string) (*entity.CourseStats, error) {
	course, err := u.courseRepo.FindByDisplayID(ctx, u.executor, displayID, false)
	if err != nil {
		return nil, err
	}
	stats, err := u.statsRepo.FindByCourseID(ctx, u.executor, course.ID)
	if err != nil {
		return nil, err
	}

	ratingBands := u.masterProvider.RatingBands()
	bandOrder := make(map[int]int, len(ratingBands))
	for _, band := range ratingBands {
		bandOrder[band.ID] = band.SortOrder
	}
	slices.SortFunc(stats, func(a, b *entity.CourseStatsByRatingBand) int {
		return cmp.Compare(bandOrder[a.RatingBandID], bandOrder[b.RatingBandID])
	})
	return &entity.CourseStats{CourseDisplayID: course.DisplayID.String(), Stats: stats}, nil
}
