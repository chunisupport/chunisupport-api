package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/chartstatsbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/jmoiron/sqlx"
)

// 集計元の読み取りクエリです。
// 中央値の計算で保持するスコアを1譜面分に抑えるため、譜面統計の記録は譜面ID順に返します。
const (
	chartStatsBatchChartRecordQuery = `
		SELECT pr.chart_id, p.best_average_rating, pr.score, pr.clear_lamp_id, pr.combo_lamp_id
		FROM players p
		INNER JOIN users u ON u.id = p.user_id
		INNER JOIN player_records pr ON pr.player_id = p.id
		WHERE p.best_average_rating IS NOT NULL
		ORDER BY pr.chart_id, p.id
	`
	chartStatsBatchWorldsendChartRecordQuery = `
		SELECT pwr.worldsend_chart_id, p.best_average_rating, pwr.score, pwr.clear_lamp_id, pwr.combo_lamp_id
		FROM players p
		INNER JOIN users u ON u.id = p.user_id
		INNER JOIN player_worldsend_records pwr ON pwr.player_id = p.id
		WHERE p.best_average_rating IS NOT NULL
		ORDER BY pwr.worldsend_chart_id, p.id
	`
	chartStatsBatchEligiblePlayerQuery = `
		SELECT p.best_average_rating
		FROM players p
		INNER JOIN users u ON u.id = p.user_id
		WHERE p.best_average_rating IS NOT NULL
		  AND u.is_suspicious = 0
	`
	chartStatsBatchBestSlotRecordQuery = `
		SELECT pr.chart_id, p.best_average_rating
		FROM players p
		INNER JOIN users u ON u.id = p.user_id
		INNER JOIN player_records pr ON pr.player_id = p.id
		INNER JOIN slots sl ON sl.id = pr.slot_id
		WHERE p.best_average_rating IS NOT NULL
		  AND u.is_suspicious = 0
		  AND sl.name = 'best'
	`
	// ベスト枠採用率は削除済み楽曲と WORLD'S END を除く通常譜面だけを対象にします。
	chartStatsBatchBestSlotChartQuery = `
		SELECT c.id
		FROM charts c
		INNER JOIN songs s ON s.id = c.song_id
		WHERE s.is_deleted = 0
		  AND s.is_worldsend = 0
		ORDER BY c.id
	`
)

// 統計テーブルへの挿入文です。
const (
	chartStatsInsertColumns = `
			rank_aaal, rank_s, rank_sp, rank_ss, rank_ssp, rank_sss, rank_sssp, rank_max,
			combo_none, combo_fc, combo_aj, combo_ajc,
			clear_failed, clear_clear, clear_hard, clear_brave, clear_absolute, clear_catastrophy,
			average_score, median_score, player_count
		) VALUES `
	chartStatsInsertPlaceholder     = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	chartStatsInsertPrefix          = `INSERT INTO chart_stats_by_rating_band (chart_id, rating_band_id,` + chartStatsInsertColumns
	worldsendChartStatsInsertPrefix = `INSERT INTO worldsend_chart_stats_by_rating_band (worldsend_chart_id, rating_band_id,` +
		chartStatsInsertColumns
	bestSlotStatsInsertPrefix = `
		INSERT INTO chart_best_slot_stats_by_rating_band (
			chart_id, rating_band_id, best_player_count, eligible_player_count, best_player_percentage
		) VALUES `
	bestSlotStatsInsertPlaceholder = "(?, ?, ?, ?, ?)"
)

type chartStatsBatchRepository struct {
	db *sqlx.DB
}

// NewChartStatsBatchRepository は譜面統計バッチリポジトリを生成します。
func NewChartStatsBatchRepository(db *sqlx.DB) domainrepo.ChartStatsBatchRepository {
	return &chartStatsBatchRepository{db: db}
}

// FindRatingBands はレーティング帯を表示順に返します。
func (r *chartStatsBatchRepository) FindRatingBands(ctx context.Context) ([]*ratingband.RatingBand, error) {
	return findRatingBands(ctx, r.db)
}

// StreamSource は集計元を読み取り専用の REPEATABLE READ トランザクション内で読み取ります。
// 譜面統計とベスト枠採用率が同じ時点のプレイヤー状態を基にするよう、すべての読み取りを1つのスナップショットで行います。
func (r *chartStatsBatchRepository) StreamSource(ctx context.Context, callbacks domainrepo.ChartStatsBatchSourceCallbacks) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return fmt.Errorf("begin chart stats source snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := streamChartStatsBatchChartRecords(ctx, tx, chartStatsBatchChartRecordQuery, callbacks.ChartRecord); err != nil {
		return fmt.Errorf("stream chart records: %w", err)
	}
	if err := streamChartStatsBatchChartRecords(ctx, tx, chartStatsBatchWorldsendChartRecordQuery, callbacks.WorldsendChartRecord); err != nil {
		return fmt.Errorf("stream worldsend chart records: %w", err)
	}
	if err := streamChartStatsBatchRows(ctx, tx, chartStatsBatchEligiblePlayerQuery, func(rows *sqlx.Rows) error {
		var player chartstatsbatch.EligiblePlayer
		if err := rows.Scan(&player.BestAverageRating); err != nil {
			return err
		}
		return callbacks.EligiblePlayer(player)
	}); err != nil {
		return fmt.Errorf("stream eligible players: %w", err)
	}
	if err := streamChartStatsBatchRows(ctx, tx, chartStatsBatchBestSlotRecordQuery, func(rows *sqlx.Rows) error {
		var record chartstatsbatch.BestSlotRecord
		if err := rows.Scan(&record.ChartID, &record.BestAverageRating); err != nil {
			return err
		}
		return callbacks.BestSlotRecord(record)
	}); err != nil {
		return fmt.Errorf("stream best-slot records: %w", err)
	}
	if err := streamChartStatsBatchRows(ctx, tx, chartStatsBatchBestSlotChartQuery, func(rows *sqlx.Rows) error {
		var chartID int
		if err := rows.Scan(&chartID); err != nil {
			return err
		}
		return callbacks.BestSlotChart(chartID)
	}); err != nil {
		return fmt.Errorf("stream best-slot charts: %w", err)
	}
	return tx.Commit()
}

func streamChartStatsBatchChartRecords(ctx context.Context, tx *sqlx.Tx, query string, callback func(chartstatsbatch.ChartRecord) error) error {
	return streamChartStatsBatchRows(ctx, tx, query, func(rows *sqlx.Rows) error {
		var record chartstatsbatch.ChartRecord
		if err := rows.Scan(&record.ChartID, &record.BestAverageRating, &record.Score, &record.ClearLampID, &record.ComboLampID); err != nil {
			return err
		}
		return callback(record)
	})
}

// streamChartStatsBatchRows は結果を全件メモリへ載せずに1行ずつ処理します。
func streamChartStatsBatchRows(ctx context.Context, tx *sqlx.Tx, query string, handle func(*sqlx.Rows) error) error {
	rows, err := tx.QueryxContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := handle(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// ReplaceAll は3つの統計テーブルを削除してから snapshot を挿入し、単一トランザクションでコミットします。
// コミットまでは他のセッションから直前の統計が見え続け、失敗やキャンセル時はロールバックされます。
func (r *chartStatsBatchRepository) ReplaceAll(ctx context.Context, snapshot *chartstatsbatch.Snapshot) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin chart stats replacement: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	for _, table := range []string{
		"chart_stats_by_rating_band",
		"worldsend_chart_stats_by_rating_band",
		"chart_best_slot_stats_by_rating_band",
	} {
		if _, err = tx.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("delete %s: %w", table, err)
		}
	}
	if err = insertChartStatsBatchRows(ctx, tx, chartStatsInsertPrefix, chartStatsInsertPlaceholder, chartStatsRows(snapshot.ChartStats)); err != nil {
		return fmt.Errorf("insert chart stats: %w", err)
	}
	if err = insertChartStatsBatchRows(ctx, tx, worldsendChartStatsInsertPrefix, chartStatsInsertPlaceholder, chartStatsRows(snapshot.WorldsendChartStats)); err != nil {
		return fmt.Errorf("insert worldsend chart stats: %w", err)
	}
	if err = insertChartStatsBatchRows(ctx, tx, bestSlotStatsInsertPrefix, bestSlotStatsInsertPlaceholder, bestSlotStatsRows(snapshot.BestSlotStats)); err != nil {
		return fmt.Errorf("insert best-slot stats: %w", err)
	}
	return nil
}

func chartStatsRows(stats []*entity.ChartStatsByRatingBand) [][]any {
	rows := make([][]any, 0, len(stats))
	for _, s := range stats {
		rows = append(rows, []any{
			s.ChartID, s.RatingBandID,
			s.Rank.AAAL, s.Rank.S, s.Rank.SP, s.Rank.SS, s.Rank.SSP, s.Rank.SSS, s.Rank.SSSP, s.Rank.Max,
			s.Combo.None, s.Combo.FC, s.Combo.AJ, s.Combo.AJC,
			s.Clear.Failed, s.Clear.Clear, s.Clear.Hard, s.Clear.Brave, s.Clear.Absolute, s.Clear.Catastrophy,
			s.AverageScore, s.MedianScore, s.PlayerCount,
		})
	}
	return rows
}

func bestSlotStatsRows(stats []*entity.ChartBestSlotStatsByRatingBand) [][]any {
	rows := make([][]any, 0, len(stats))
	for _, s := range stats {
		rows = append(rows, []any{s.ChartID, s.RatingBandID, s.BestPlayerCount, s.EligiblePlayerCount, s.BestPlayerPercentage})
	}
	return rows
}

// insertChartStatsBatchRows は複数行 VALUES で info.ChartStatsBatchInsertChunkSize 行ずつ挿入します。
func insertChartStatsBatchRows(ctx context.Context, tx *sqlx.Tx, prefix, placeholder string, rows [][]any) error {
	for start := 0; start < len(rows); start += info.ChartStatsBatchInsertChunkSize {
		chunk := rows[start:min(start+info.ChartStatsBatchInsertChunkSize, len(rows))]
		query := prefix + strings.TrimSuffix(strings.Repeat(placeholder+",", len(chunk)), ",")
		args := make([]any, 0, len(chunk)*len(chunk[0]))
		for _, row := range chunk {
			args = append(args, row...)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

var _ domainrepo.ChartStatsBatchRepository = (*chartStatsBatchRepository)(nil)
