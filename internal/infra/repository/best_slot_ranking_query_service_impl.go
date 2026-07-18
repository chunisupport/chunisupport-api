package repository

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/jmoiron/sqlx"
)

var _ domainrepo.BestSlotRankingQueryService = (*BestSlotRankingQueryService)(nil)

// BestSlotRankingQueryService は統計行と通常譜面マスターを一括結合します。
type BestSlotRankingQueryService struct {
	db *sqlx.DB
}

// NewBestSlotRankingQueryService はランキング読み取りサービスを生成します。
func NewBestSlotRankingQueryService(db *sqlx.DB) *BestSlotRankingQueryService {
	return &BestSlotRankingQueryService{db: db}
}

type bestSlotRankingStatsRow struct {
	ChartID              int      `db:"chart_id"`
	BestPlayerCount      int      `db:"best_player_count"`
	BestPlayerPercentage float64  `db:"best_player_percentage"`
	AverageScore         *float64 `db:"average_score"`
}

type bestSlotRankingChartRow struct {
	ChartID       int                         `db:"chart_id"`
	SongDisplayID string                      `db:"song_display_id"`
	SongTitle     string                      `db:"song_title"`
	Difficulty    string                      `db:"difficulty"`
	ChartConst    chartconstant.ChartConstant `db:"chart_const"`
	IsUnknown     bool                        `db:"is_const_unknown"`
}

func (q *BestSlotRankingQueryService) List(ctx context.Context, ratingBandID int, cursor *domainrepo.BestSlotRankingCursor, limit int) (*domainrepo.BestSlotRankingPage, error) {
	eligibleCount, err := findEligiblePlayerCount(ctx, q.db, ratingBandID)
	if err != nil {
		return nil, err
	}

	rows, err := findBestSlotRankingRows(ctx, q.db, ratingBandID)
	if err != nil {
		return nil, err
	}

	charts, err := findBestSlotRankingCharts(ctx, q.db)
	if err != nil {
		return nil, err
	}
	chartByID := make(map[int]bestSlotRankingChartRow, len(charts))
	for _, chart := range charts {
		chartByID[chart.ChartID] = chart
	}

	items := make([]domainrepo.BestSlotRankingItem, 0, len(rows))
	for _, row := range rows {
		chart, ok := chartByID[row.ChartID]
		if !ok {
			continue
		}
		items = append(items, domainrepo.BestSlotRankingItem{
			SongDisplayID:        chart.SongDisplayID,
			SongTitle:            chart.SongTitle,
			Difficulty:           chart.Difficulty,
			ChartConst:           chart.ChartConst,
			IsConstUnknown:       chart.IsUnknown,
			BestPlayerCount:      row.BestPlayerCount,
			BestPlayerPercentage: row.BestPlayerPercentage,
			AverageScore:         row.AverageScore,
		})
	}
	slices.SortFunc(items, compareBestSlotRankingItems)
	for i := range items {
		items[i].Rank = i + 1
	}
	if cursor != nil {
		items = slices.DeleteFunc(items, func(item domainrepo.BestSlotRankingItem) bool {
			return compareBestSlotRankingItemToCursor(item, cursor) <= 0
		})
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	page := &domainrepo.BestSlotRankingPage{EligiblePlayerCount: eligibleCount, Items: items}
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		page.NextCursor = &domainrepo.BestSlotRankingCursor{
			BestPlayerPercentage: last.BestPlayerPercentage,
			BestPlayerCount:      last.BestPlayerCount,
			SongDisplayID:        last.SongDisplayID,
			Difficulty:           last.Difficulty,
		}
	}
	return page, nil
}

func findEligiblePlayerCount(ctx context.Context, exec domainrepo.Executor, ratingBandID int) (int, error) {
	var count int
	err := exec.GetContext(ctx, &count, `
		SELECT eligible_player_count
		FROM chart_best_slot_stats_by_rating_band
		WHERE rating_band_id = ?
		ORDER BY chart_id
		LIMIT 1
	`, ratingBandID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("%w: find eligible player count: %v", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return count, nil
}

func findBestSlotRankingRows(ctx context.Context, exec domainrepo.Executor, ratingBandID int) ([]bestSlotRankingStatsRow, error) {
	const query = `
		SELECT
			best.chart_id,
			best.best_player_count,
			best.best_player_percentage,
			stats.average_score
		FROM chart_best_slot_stats_by_rating_band best
		LEFT JOIN chart_stats_by_rating_band stats
		  ON stats.chart_id = best.chart_id
		 AND stats.rating_band_id = best.rating_band_id
		WHERE best.rating_band_id = ?
		  AND best.best_player_percentage IS NOT NULL
		  AND best.best_player_count > 0
	`
	var rows []bestSlotRankingStatsRow
	if err := exec.SelectContext(ctx, &rows, query, ratingBandID); err != nil {
		return nil, fmt.Errorf("%w: list best-slot ranking stats: %v", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return rows, nil
}

func findBestSlotRankingCharts(ctx context.Context, exec domainrepo.Executor) ([]bestSlotRankingChartRow, error) {
	const query = `
		SELECT
			c.id AS chart_id,
			s.display_id AS song_display_id,
			s.title AS song_title,
			d.name AS difficulty,
			c.const AS chart_const,
			c.is_const_unknown AS is_const_unknown
		FROM charts c
		INNER JOIN songs s ON s.id = c.song_id
		INNER JOIN difficulties d ON d.id = c.difficulty_id
		WHERE s.is_worldsend = 0
		  AND s.is_deleted = 0
	`
	var rows []bestSlotRankingChartRow
	if err := exec.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("%w: list best-slot ranking charts: %v", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return rows, nil
}

func compareBestSlotRankingItems(a, b domainrepo.BestSlotRankingItem) int {
	if result := cmp.Compare(b.BestPlayerPercentage, a.BestPlayerPercentage); result != 0 {
		return result
	}
	if result := cmp.Compare(b.BestPlayerCount, a.BestPlayerCount); result != 0 {
		return result
	}
	if result := cmp.Compare(a.SongDisplayID, b.SongDisplayID); result != 0 {
		return result
	}
	return cmp.Compare(a.Difficulty, b.Difficulty)
}

func compareBestSlotRankingItemToCursor(item domainrepo.BestSlotRankingItem, cursor *domainrepo.BestSlotRankingCursor) int {
	return compareBestSlotRankingItems(item, domainrepo.BestSlotRankingItem{
		SongDisplayID:        cursor.SongDisplayID,
		Difficulty:           cursor.Difficulty,
		BestPlayerCount:      cursor.BestPlayerCount,
		BestPlayerPercentage: cursor.BestPlayerPercentage,
	})
}
