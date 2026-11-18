package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// CourseStatsRepository はコース統計の参照を扱うリポジトリです。
type CourseStatsRepository interface {
	// FindByCourseID は指定コースのレーティング帯別統計を返します。未集計の場合は空です。
	FindByCourseID(ctx context.Context, exec Executor, courseID int) ([]*entity.CourseStatsByRatingBand, error)
}
