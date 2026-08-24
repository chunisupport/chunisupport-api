package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type playerMetricHistoryQueryService struct {
	db *sqlx.DB
}

// NewPlayerMetricHistoryQueryService はプレイヤー公式指標履歴QueryServiceを生成します。
func NewPlayerMetricHistoryQueryService(db *sqlx.DB) domainrepo.PlayerMetricHistoryQueryService {
	return &playerMetricHistoryQueryService{db: db}
}

func insertPlayerMetricHistory(ctx context.Context, exec domainrepo.Executor, entry entity.PlayerMetricHistoryEntry) error {
	model := models.PlayerMetricHistoryModelFromEntity(entry)
	const query = `INSERT INTO player_metric_histories
		(player_id, official_rating, official_overpower, official_overpower_percent, data_collected_at)
		VALUES (:player_id, :official_rating, :official_overpower, :official_overpower_percent, :data_collected_at)`
	if _, err := exec.NamedExecContext(ctx, query, model); err != nil {
		return wrapPlayerMetricHistoryInsertError(err)
	}
	return nil
}

func wrapPlayerMetricHistoryInsertError(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntryErrorNumber {
		return fmt.Errorf("%w: %v", domainrepo.ErrPlayerMetricHistoryTimestampConflict, err)
	}
	return fmt.Errorf("failed to insert player metric history: %w", err)
}

func (r *playerMetricHistoryQueryService) FindTimeline(ctx context.Context, playerID int) ([]entity.PlayerMetricHistoryEntry, error) {
	const query = `SELECT player_id, official_rating, official_overpower, official_overpower_percent, data_collected_at
		FROM (
			SELECT id AS player_id, official_player_rating AS official_rating,
				official_overpower, official_overpower_percent, data_collected_at, 1 AS is_current
			FROM players
			WHERE id = ? AND data_collected_at IS NOT NULL
			UNION ALL
			SELECT history.player_id, history.official_rating, history.official_overpower, history.official_overpower_percent,
				history.data_collected_at, 0 AS is_current
			FROM player_metric_histories AS history
			WHERE history.player_id = ?
				AND EXISTS (SELECT 1 FROM players AS current WHERE current.id = ?)
		) AS timeline
		ORDER BY is_current DESC, data_collected_at DESC`
	rows := make([]models.PlayerMetricHistoryModel, 0)
	if err := r.db.SelectContext(ctx, &rows, query, playerID, playerID, playerID); err != nil {
		return nil, fmt.Errorf("failed to find player metric history timeline: %w", err)
	}
	entries := make([]entity.PlayerMetricHistoryEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, row.ToEntity())
	}
	return entries, nil
}
