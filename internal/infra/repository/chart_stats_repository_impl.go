package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/jmoiron/sqlx"
)

// chartStatsRepository は ChartStatsRepository の実装です。
type chartStatsRepository struct {
	db *sqlx.DB
}

// NewChartStatsRepository は ChartStatsRepository の実装を生成します。
func NewChartStatsRepository(db *sqlx.DB) repository.ChartStatsRepository {
	return &chartStatsRepository{db: db}
}

type ratingBandRow struct {
	ID           int      `db:"id"`
	Label        string   `db:"label"`
	MinInclusive *float64 `db:"min_inclusive"`
	MaxExclusive *float64 `db:"max_exclusive"`
	SortOrder    int      `db:"sort_order"`
}

type chartStatsRow struct {
	ChartID          int      `db:"chart_id"`
	RatingBandID     int      `db:"rating_band_id"`
	RankAAAL         int      `db:"rank_aaal"`
	RankS            int      `db:"rank_s"`
	RankSP           int      `db:"rank_sp"`
	RankSS           int      `db:"rank_ss"`
	RankSSP          int      `db:"rank_ssp"`
	RankSSS          int      `db:"rank_sss"`
	RankSSSP         int      `db:"rank_sssp"`
	RankMax          int      `db:"rank_max"`
	ComboNone        int      `db:"combo_none"`
	ComboFC          int      `db:"combo_fc"`
	ComboAJ          int      `db:"combo_aj"`
	ComboAJC         int      `db:"combo_ajc"`
	ClearFailed      int      `db:"clear_failed"`
	ClearClear       int      `db:"clear_clear"`
	ClearHard        int      `db:"clear_hard"`
	ClearBrave       int      `db:"clear_brave"`
	ClearAbsolute    int      `db:"clear_absolute"`
	ClearCatastrophy int      `db:"clear_catastrophy"`
	AverageScore     *float64 `db:"average_score"`
	MedianScore      *float64 `db:"median_score"`
	PlayerCount      int      `db:"player_count"`
}

func (r *chartStatsRepository) findChartStatsByChartIDs(ctx context.Context, exec repository.Executor, query string, chartIDs []int) ([]*entity.ChartStatsByRatingBand, error) {
	if len(chartIDs) == 0 {
		return []*entity.ChartStatsByRatingBand{}, nil
	}

	boundQuery, args, err := sqlx.In(query, chartIDs)
	if err != nil {
		return nil, err
	}
	boundQuery = exec.Rebind(boundQuery)

	var rows []chartStatsRow
	if err := exec.SelectContext(ctx, &rows, boundQuery, args...); err != nil {
		return nil, err
	}

	results := make([]*entity.ChartStatsByRatingBand, 0, len(rows))
	for _, row := range rows {
		results = append(results, &entity.ChartStatsByRatingBand{
			ChartID:      row.ChartID,
			RatingBandID: row.RatingBandID,
			Rank: entity.ChartRankStats{
				AAAL: row.RankAAAL,
				S:    row.RankS,
				SP:   row.RankSP,
				SS:   row.RankSS,
				SSP:  row.RankSSP,
				SSS:  row.RankSSS,
				SSSP: row.RankSSSP,
				Max:  row.RankMax,
			},
			Combo: entity.ChartComboStats{
				None: row.ComboNone,
				FC:   row.ComboFC,
				AJ:   row.ComboAJ,
				AJC:  row.ComboAJC,
			},
			Clear: entity.ChartClearStats{
				Failed:      row.ClearFailed,
				Clear:       row.ClearClear,
				Hard:        row.ClearHard,
				Brave:       row.ClearBrave,
				Absolute:    row.ClearAbsolute,
				Catastrophy: row.ClearCatastrophy,
			},
			AverageScore: row.AverageScore,
			MedianScore:  row.MedianScore,
			PlayerCount:  row.PlayerCount,
		})
	}

	return results, nil
}

// FindRatingBands はレーティング帯マスタ一覧を返します。
func (r *chartStatsRepository) FindRatingBands(ctx context.Context, exec repository.Executor) ([]*ratingband.RatingBand, error) {
	const query = `
		SELECT id, label, min_inclusive, max_exclusive, sort_order
		FROM rating_bands
		ORDER BY sort_order
	`

	var rows []ratingBandRow
	if err := exec.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}

	results := make([]*ratingband.RatingBand, 0, len(rows))
	for _, row := range rows {
		results = append(results, &ratingband.RatingBand{
			ID:           row.ID,
			Label:        row.Label,
			MinInclusive: row.MinInclusive,
			MaxExclusive: row.MaxExclusive,
			SortOrder:    row.SortOrder,
		})
	}
	return results, nil
}

// FindChartStatsByChartIDs は譜面ID一覧に対する統計を返します。
func (r *chartStatsRepository) FindChartStatsByChartIDs(ctx context.Context, exec repository.Executor, chartIDs []int) ([]*entity.ChartStatsByRatingBand, error) {
	return r.findChartStatsByChartIDs(ctx, exec, `
		SELECT
			chart_id,
			rating_band_id,
			rank_aaal,
			rank_s,
			rank_sp,
			rank_ss,
			rank_ssp,
			rank_sss,
			rank_sssp,
			rank_max,
			combo_none,
			combo_fc,
			combo_aj,
			combo_ajc,
			clear_failed,
			clear_clear,
			clear_hard,
			clear_brave,
			clear_absolute,
			clear_catastrophy,
			average_score,
			median_score,
			player_count
		FROM chart_stats_by_rating_band
		WHERE chart_id IN (?)
		ORDER BY chart_id, rating_band_id
	`, chartIDs)
}

// FindChartBestSlotStatsByChartIDs は通常譜面のベスト枠採用統計を一括取得します。
func (r *chartStatsRepository) FindChartBestSlotStatsByChartIDs(ctx context.Context, exec repository.Executor, chartIDs []int) ([]*entity.ChartBestSlotStatsByRatingBand, error) {
	if len(chartIDs) == 0 {
		return []*entity.ChartBestSlotStatsByRatingBand{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT chart_id, rating_band_id, best_player_count, eligible_player_count, best_player_percentage
		FROM chart_best_slot_stats_by_rating_band
		WHERE chart_id IN (?)
		ORDER BY chart_id, rating_band_id
	`, chartIDs)
	if err != nil {
		return nil, err
	}
	query = exec.Rebind(query)

	type row struct {
		ChartID              int      `db:"chart_id"`
		RatingBandID         int      `db:"rating_band_id"`
		BestPlayerCount      int      `db:"best_player_count"`
		EligiblePlayerCount  int      `db:"eligible_player_count"`
		BestPlayerPercentage *float64 `db:"best_player_percentage"`
	}
	var rows []row
	if err := exec.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}

	results := make([]*entity.ChartBestSlotStatsByRatingBand, 0, len(rows))
	for _, item := range rows {
		results = append(results, &entity.ChartBestSlotStatsByRatingBand{
			ChartID:              item.ChartID,
			RatingBandID:         item.RatingBandID,
			BestPlayerCount:      item.BestPlayerCount,
			EligiblePlayerCount:  item.EligiblePlayerCount,
			BestPlayerPercentage: item.BestPlayerPercentage,
		})
	}
	return results, nil
}

// FindWorldsendChartStatsByChartIDs はWORLD'S END譜面ID一覧に対する統計を返します。
func (r *chartStatsRepository) FindWorldsendChartStatsByChartIDs(ctx context.Context, exec repository.Executor, chartIDs []int) ([]*entity.ChartStatsByRatingBand, error) {
	return r.findChartStatsByChartIDs(ctx, exec, `
		SELECT
			worldsend_chart_id AS chart_id,
			rating_band_id,
			rank_aaal,
			rank_s,
			rank_sp,
			rank_ss,
			rank_ssp,
			rank_sss,
			rank_sssp,
			rank_max,
			combo_none,
			combo_fc,
			combo_aj,
			combo_ajc,
			clear_failed,
			clear_clear,
			clear_hard,
			clear_brave,
			clear_absolute,
			clear_catastrophy,
			average_score,
			median_score,
			player_count
		FROM worldsend_chart_stats_by_rating_band
		WHERE worldsend_chart_id IN (?)
		ORDER BY worldsend_chart_id, rating_band_id
	`, chartIDs)
}
