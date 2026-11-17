package repository

import (
	"context"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

type courseStatsRepository struct {
	db *sqlx.DB
}

// NewCourseStatsRepository は CourseStatsRepository の実装を生成します。
func NewCourseStatsRepository(db *sqlx.DB) domainrepo.CourseStatsRepository {
	return &courseStatsRepository{db: db}
}

// FindByCourseID は指定コースのレーティング帯別統計を返します。
func (r *courseStatsRepository) FindByCourseID(ctx context.Context, exec domainrepo.Executor, courseID int) ([]*entity.CourseStatsByRatingBand, error) {
	const query = `
		SELECT course_id, rating_band_id, combo_none, combo_fc, combo_aj, clear_failed, clear_clear,
			average_score, median_score, player_count
		FROM course_stats_by_rating_band
		WHERE course_id = ?
		ORDER BY rating_band_id
	`
	var rows []models.CourseStatsByRatingBandModel
	if err := exec.SelectContext(ctx, &rows, query, courseID); err != nil {
		return nil, fmt.Errorf("%w: find course stats: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	results := make([]*entity.CourseStatsByRatingBand, 0, len(rows))
	for _, row := range rows {
		results = append(results, row.ToEntity())
	}
	return results, nil
}
