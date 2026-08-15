package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type scoreHistoryRepository struct {
	db *sqlx.DB
}

// NewScoreHistoryRepository はスコア履歴Repositoryを生成します。
func NewScoreHistoryRepository(db *sqlx.DB) domainrepo.ScoreHistoryRepository {
	return &scoreHistoryRepository{db: db}
}

func (r *scoreHistoryRepository) BulkInsertStandard(ctx context.Context, exec domainrepo.Executor, rows []domainrepo.PlayerRecordHistory) error {
	if len(rows) == 0 {
		return nil
	}
	historyModels := make([]models.PlayerRecordHistoryModel, 0, len(rows))
	for _, row := range rows {
		historyModels = append(historyModels, models.PlayerRecordHistoryModelFromEntity(row))
	}
	const query = `INSERT INTO player_record_histories
		(player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at)
		VALUES (:player_id, :chart_id, :score, :clear_lamp_id, :combo_lamp_id, :full_chain_id, :updated_at)`
	return bulkInsertScoreHistories(ctx, exec, query, historyModels)
}

func (r *scoreHistoryRepository) BulkInsertWorldsend(ctx context.Context, exec domainrepo.Executor, rows []domainrepo.PlayerWorldsendRecordHistory) error {
	if len(rows) == 0 {
		return nil
	}
	historyModels := make([]models.PlayerWorldsendRecordHistoryModel, 0, len(rows))
	for _, row := range rows {
		historyModels = append(historyModels, models.PlayerWorldsendRecordHistoryModelFromEntity(row))
	}
	const query = `INSERT INTO player_worldsend_record_histories
		(player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at)
		VALUES (:player_id, :worldsend_chart_id, :score, :clear_lamp_id, :combo_lamp_id, :full_chain_id, :updated_at)`
	return bulkInsertScoreHistories(ctx, exec, query, historyModels)
}

func bulkInsertScoreHistories[T any](ctx context.Context, exec domainrepo.Executor, query string, rows []T) error {
	for start := 0; start < len(rows); start += info.BulkInsertChunkSize {
		end := min(start+info.BulkInsertChunkSize, len(rows))
		if _, err := exec.NamedExecContext(ctx, query, rows[start:end]); err != nil {
			return wrapScoreHistoryInsertError(err)
		}
	}
	return nil
}

func wrapScoreHistoryInsertError(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntryErrorNumber {
		return fmt.Errorf("%w: %v", domainrepo.ErrScoreHistoryTimestampConflict, err)
	}
	return fmt.Errorf("failed to insert score histories: %w", err)
}

func (r *scoreHistoryRepository) PruneStandardOverLimit(ctx context.Context, exec domainrepo.Executor, playerID int, chartIDs []int) error {
	return pruneHistories(ctx, exec, r.db.DriverName(), "player_record_histories", "chart_id", playerID, chartIDs)
}

func (r *scoreHistoryRepository) PruneWorldsendOverLimit(ctx context.Context, exec domainrepo.Executor, playerID int, chartIDs []int) error {
	return pruneHistories(ctx, exec, r.db.DriverName(), "player_worldsend_record_histories", "worldsend_chart_id", playerID, chartIDs)
}

func pruneHistories(ctx context.Context, exec domainrepo.Executor, driverName, table, chartColumn string, playerID int, chartIDs []int) error {
	if len(chartIDs) == 0 {
		return nil
	}
	query := fmt.Sprintf(`DELETE history
		FROM %s AS history
		INNER JOIN (
			SELECT player_id, chart_id, updated_at
			FROM (
				SELECT player_id, %s AS chart_id, updated_at,
					ROW_NUMBER() OVER (
						PARTITION BY player_id, %s
						ORDER BY updated_at DESC
					) AS history_rank
				FROM %s
				WHERE player_id = ? AND %s IN (?)
			) AS ranked
			WHERE history_rank > ?
		) AS expired
			ON expired.player_id = history.player_id
			AND expired.chart_id = history.%s
			AND expired.updated_at = history.updated_at`,
		table, chartColumn, chartColumn, table, chartColumn, chartColumn)
	if driverName == "sqlite" {
		query = fmt.Sprintf(`DELETE FROM %s
			WHERE (player_id, %s, updated_at) IN (
				SELECT player_id, chart_id, updated_at
				FROM (
					SELECT player_id, %s AS chart_id, updated_at,
						ROW_NUMBER() OVER (
							PARTITION BY player_id, %s
							ORDER BY updated_at DESC
						) AS history_rank
					FROM %s
					WHERE player_id = ? AND %s IN (?)
				) AS ranked
				WHERE history_rank > ?
			)`, table, chartColumn, chartColumn, chartColumn, table, chartColumn)
	}
	query, args, err := sqlx.In(query, playerID, chartIDs, info.MaxScoreHistoryEntriesPerChart)
	if err != nil {
		return fmt.Errorf("failed to build score history prune query: %w", err)
	}
	if _, err := exec.ExecContext(ctx, exec.Rebind(query), args...); err != nil {
		return fmt.Errorf("failed to prune score histories: %w", err)
	}
	return nil
}

func (r *scoreHistoryRepository) FindStandardTimeline(ctx context.Context, playerID, chartID int) ([]entity.ScoreHistoryEntry, error) {
	const query = `SELECT score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at
		FROM (
			SELECT score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at, 1 AS is_current
			FROM player_records
			WHERE player_id = ? AND chart_id = ?
			UNION ALL
			SELECT history.score, history.clear_lamp_id, history.combo_lamp_id,
				history.full_chain_id, history.updated_at, 0 AS is_current
			FROM player_record_histories AS history
			WHERE history.player_id = ? AND history.chart_id = ?
				AND EXISTS (
					SELECT 1 FROM player_records AS current
					WHERE current.player_id = ? AND current.chart_id = ?
				)
		) AS timeline
		ORDER BY is_current DESC, updated_at DESC
		LIMIT ?`
	return r.findTimeline(ctx, query, playerID, chartID, playerID, chartID, playerID, chartID, info.MaxScoreHistoryEntriesPerChart+1)
}

func (r *scoreHistoryRepository) FindWorldsendTimeline(ctx context.Context, playerID, worldsendChartID int) ([]entity.ScoreHistoryEntry, error) {
	const query = `SELECT score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at
		FROM (
			SELECT score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at, 1 AS is_current
			FROM player_worldsend_records
			WHERE player_id = ? AND worldsend_chart_id = ?
			UNION ALL
			SELECT history.score, history.clear_lamp_id, history.combo_lamp_id,
				history.full_chain_id, history.updated_at, 0 AS is_current
			FROM player_worldsend_record_histories AS history
			WHERE history.player_id = ? AND history.worldsend_chart_id = ?
				AND EXISTS (
					SELECT 1 FROM player_worldsend_records AS current
					WHERE current.player_id = ? AND current.worldsend_chart_id = ?
				)
		) AS timeline
		ORDER BY is_current DESC, updated_at DESC
		LIMIT ?`
	return r.findTimeline(ctx, query, playerID, worldsendChartID, playerID, worldsendChartID, playerID, worldsendChartID, info.MaxScoreHistoryEntriesPerChart+1)
}

func (r *scoreHistoryRepository) findTimeline(ctx context.Context, query string, args ...any) ([]entity.ScoreHistoryEntry, error) {
	rows := make([]models.ScoreHistoryTimelineModel, 0, info.MaxScoreHistoryEntriesPerChart+1)
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to find score history timeline: %w", err)
	}
	entries := make([]entity.ScoreHistoryEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, row.ToEntity())
	}
	return entries, nil
}
