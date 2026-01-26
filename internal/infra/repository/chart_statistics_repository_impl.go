package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/info"
	"github.com/Qman110101/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

// chartStatisticsRepository は ChartStatisticsRepository の実装です。
type chartStatisticsRepository struct {
	db *sqlx.DB
}

// NewChartStatisticsRepository は ChartStatisticsRepository の実装を生成します。
func NewChartStatisticsRepository(db *sqlx.DB) repository.ChartStatisticsRepository {
	return &chartStatisticsRepository{db: db}
}

// FindByChartID は指定された譜面IDの統計を全レーティング帯分取得します。
func (r *chartStatisticsRepository) FindByChartID(ctx context.Context, exec repository.Executor, chartID int) ([]*entity.ChartStatistics, error) {
	query := `
		SELECT
			chart_id, rating_tier,
			rank_s_count, rank_s_plus_count, rank_ss_count,
			rank_ss_plus_count, rank_sss_count, rank_sss_plus_count,
			lamp_aj_count, lamp_fc_count, lamp_other_count,
			total_count, updated_at
		FROM chart_statistics
		WHERE chart_id = ?
		ORDER BY rating_tier
	`

	executor := r.getExecutor(exec)
	var rows []*models.ChartStatistics
	if err := sqlx.SelectContext(ctx, executor, &rows, query, chartID); err != nil {
		return nil, fmt.Errorf("failed to find chart statistics by chart_id=%d: %w", chartID, err)
	}

	entities := make([]*entity.ChartStatistics, len(rows))
	for i, row := range rows {
		entities[i] = row.ToEntity()
	}

	return entities, nil
}

// FindByChartIDs は指定された譜面IDリストの統計を一括取得します（N+1問題回避）。
func (r *chartStatisticsRepository) FindByChartIDs(ctx context.Context, exec repository.Executor, chartIDs []int) ([]*entity.ChartStatistics, error) {
	if len(chartIDs) == 0 {
		return []*entity.ChartStatistics{}, nil
	}

	chunkSize := info.BulkSelectChunkSize
	if chunkSize <= 0 {
		chunkSize = info.BulkInsertChunkSize
	}

	executor := r.getExecutor(exec)
	entities := make([]*entity.ChartStatistics, 0)
	for start := 0; start < len(chartIDs); start += chunkSize {
		end := min(start+chunkSize, len(chartIDs))
		chunkIDs := chartIDs[start:end]

		placeholders := make([]string, len(chunkIDs))
		args := make([]any, len(chunkIDs))
		for i, id := range chunkIDs {
			placeholders[i] = "?"
			args[i] = id
		}

		query := fmt.Sprintf(`
			SELECT
				chart_id, rating_tier,
				rank_s_count, rank_s_plus_count, rank_ss_count,
				rank_ss_plus_count, rank_sss_count, rank_sss_plus_count,
				lamp_aj_count, lamp_fc_count, lamp_other_count,
				total_count, updated_at
			FROM chart_statistics
			WHERE chart_id IN (%s)
			ORDER BY chart_id, rating_tier
		`, strings.Join(placeholders, ","))

		var rows []*models.ChartStatistics
		if err := sqlx.SelectContext(ctx, executor, &rows, query, args...); err != nil {
			return nil, fmt.Errorf("failed to find chart statistics by chart_ids: %w", err)
		}

		for _, row := range rows {
			entities = append(entities, row.ToEntity())
		}
	}

	return entities, nil
}

// Save は統計データを保存または更新します（UPSERT）。
func (r *chartStatisticsRepository) Save(ctx context.Context, exec repository.Executor, stats *entity.ChartStatistics) error {
	if !stats.IsValidRatingTier() {
		return fmt.Errorf("invalid rating tier: %d", stats.RatingTier)
	}

	model := models.FromEntity(stats)

	query := `
		INSERT INTO chart_statistics (
			chart_id, rating_tier,
			rank_s_count, rank_s_plus_count, rank_ss_count,
			rank_ss_plus_count, rank_sss_count, rank_sss_plus_count,
			lamp_aj_count, lamp_fc_count, lamp_other_count,
			total_count, updated_at
		) VALUES (
			:chart_id, :rating_tier,
			:rank_s_count, :rank_s_plus_count, :rank_ss_count,
			:rank_ss_plus_count, :rank_sss_count, :rank_sss_plus_count,
			:lamp_aj_count, :lamp_fc_count, :lamp_other_count,
			:total_count, :updated_at
		)
		ON DUPLICATE KEY UPDATE
			rank_s_count = VALUES(rank_s_count),
			rank_s_plus_count = VALUES(rank_s_plus_count),
			rank_ss_count = VALUES(rank_ss_count),
			rank_ss_plus_count = VALUES(rank_ss_plus_count),
			rank_sss_count = VALUES(rank_sss_count),
			rank_sss_plus_count = VALUES(rank_sss_plus_count),
			lamp_aj_count = VALUES(lamp_aj_count),
			lamp_fc_count = VALUES(lamp_fc_count),
			lamp_other_count = VALUES(lamp_other_count),
			total_count = VALUES(total_count),
			updated_at = VALUES(updated_at)
	`

	executor := r.getExecutor(exec)
	if _, err := sqlx.NamedExecContext(ctx, executor, query, model); err != nil {
		return fmt.Errorf("failed to save chart statistics (chart_id=%d, tier=%d): %w", stats.ChartID, stats.RatingTier, err)
	}

	return nil
}

// BulkSave は統計データを一括保存します（バッチ処理用）。
func (r *chartStatisticsRepository) BulkSave(ctx context.Context, exec repository.Executor, statsList []*entity.ChartStatistics) error {
	if len(statsList) == 0 {
		return nil
	}

	// バリデーション
	for _, stats := range statsList {
		if !stats.IsValidRatingTier() {
			return fmt.Errorf("invalid rating tier: %d", stats.RatingTier)
		}
	}

	query := `
		INSERT INTO chart_statistics (
			chart_id, rating_tier,
			rank_s_count, rank_s_plus_count, rank_ss_count,
			rank_ss_plus_count, rank_sss_count, rank_sss_plus_count,
			lamp_aj_count, lamp_fc_count, lamp_other_count,
			total_count, updated_at
		) VALUES (
			:chart_id, :rating_tier,
			:rank_s_count, :rank_s_plus_count, :rank_ss_count,
			:rank_ss_plus_count, :rank_sss_count, :rank_sss_plus_count,
			:lamp_aj_count, :lamp_fc_count, :lamp_other_count,
			:total_count, :updated_at
		)
		ON DUPLICATE KEY UPDATE
			rank_s_count = VALUES(rank_s_count),
			rank_s_plus_count = VALUES(rank_s_plus_count),
			rank_ss_count = VALUES(rank_ss_count),
			rank_ss_plus_count = VALUES(rank_ss_plus_count),
			rank_sss_count = VALUES(rank_sss_count),
			rank_sss_plus_count = VALUES(rank_sss_plus_count),
			lamp_aj_count = VALUES(lamp_aj_count),
			lamp_fc_count = VALUES(lamp_fc_count),
			lamp_other_count = VALUES(lamp_other_count),
			total_count = VALUES(total_count),
			updated_at = VALUES(updated_at)
	`

	executor := r.getExecutor(exec)
	chunkSize := info.BulkInsertChunkSize
	if chunkSize <= 0 {
		chunkSize = len(statsList)
	}
	for start := 0; start < len(statsList); start += chunkSize {
		end := min(start+chunkSize, len(statsList))
		chunkStats := statsList[start:end]

		modelsSlice := make([]*models.ChartStatistics, len(chunkStats))
		for i, e := range chunkStats {
			modelsSlice[i] = models.FromEntity(e)
		}

		if _, err := sqlx.NamedExecContext(ctx, executor, query, modelsSlice); err != nil {
			return fmt.Errorf("failed to bulk save chart statistics: %w", err)
		}
	}

	return nil
}

// DeleteByChartID は指定された譜面IDの統計を全レーティング帯分削除します。
func (r *chartStatisticsRepository) DeleteByChartID(ctx context.Context, exec repository.Executor, chartID int) error {
	query := `DELETE FROM chart_statistics WHERE chart_id = ?`

	executor := r.getExecutor(exec)
	if _, err := executor.ExecContext(ctx, query, chartID); err != nil {
		return fmt.Errorf("failed to delete chart statistics by chart_id=%d: %w", chartID, err)
	}

	return nil
}

// getExecutor は Executor から実際の実行オブジェクトを取得します。
func (r *chartStatisticsRepository) getExecutor(exec repository.Executor) sqlx.ExtContext {
	if exec != nil {
		if tx, ok := exec.(*sqlx.Tx); ok {
			return tx
		}
	}
	return r.db
}
