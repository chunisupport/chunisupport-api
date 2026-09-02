package repository

import (
	"context"
	"database/sql"
	"fmt"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/jmoiron/sqlx"
)

var _ domainrepo.ChartStatsExportQueryService = (*ChartStatsExportQueryService)(nil)

// ChartStatsExportQueryService は譜面マスタとALL帯統計を一括結合します。
type ChartStatsExportQueryService struct {
	db *sqlx.DB
}

// NewChartStatsExportQueryService は公開統計用の読み取りサービスを生成します。
func NewChartStatsExportQueryService(db *sqlx.DB) *ChartStatsExportQueryService {
	return &ChartStatsExportQueryService{db: db}
}

type chartStatsExportRow struct {
	SongDisplayID  string                      `db:"song_display_id"`
	SongTitle      string                      `db:"song_title"`
	Difficulty     string                      `db:"difficulty"`
	ChartConst     chartconstant.ChartConstant `db:"chart_const"`
	IsConstUnknown bool                        `db:"is_const_unknown"`
	PlayerCount    int                         `db:"player_count"`
	RankAAAL       int                         `db:"rank_aaal"`
	RankS          int                         `db:"rank_s"`
	RankSP         int                         `db:"rank_sp"`
	RankSS         int                         `db:"rank_ss"`
	RankSSP        int                         `db:"rank_ssp"`
	RankSSS        int                         `db:"rank_sss"`
	RankSSSP       int                         `db:"rank_sssp"`
	RankMax        int                         `db:"rank_max"`
	ComboNone      int                         `db:"combo_none"`
	ComboFC        int                         `db:"combo_fc"`
	ComboAJ        int                         `db:"combo_aj"`
	ComboAJC       int                         `db:"combo_ajc"`
}

type worldsendChartStatsExportRow struct {
	SongDisplayID string  `db:"song_display_id"`
	SongTitle     string  `db:"song_title"`
	LevelStar     *int    `db:"level_star"`
	Attribute     *string `db:"attribute"`
	PlayerCount   int     `db:"player_count"`
	RankAAAL      int     `db:"rank_aaal"`
	RankS         int     `db:"rank_s"`
	RankSP        int     `db:"rank_sp"`
	RankSS        int     `db:"rank_ss"`
	RankSSP       int     `db:"rank_ssp"`
	RankSSS       int     `db:"rank_sss"`
	RankSSSP      int     `db:"rank_sssp"`
	RankMax       int     `db:"rank_max"`
	ComboNone     int     `db:"combo_none"`
	ComboFC       int     `db:"combo_fc"`
	ComboAJ       int     `db:"combo_aj"`
	ComboAJC      int     `db:"combo_ajc"`
}

// Get は統計行が存在しない譜面も0件として返します。
func (q *ChartStatsExportQueryService) Get(ctx context.Context) (*domainrepo.ChartStatsExportSnapshot, error) {
	tx, err := q.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("%w: begin chart stats export snapshot: %v", domainrepo.ErrRepositoryOperationFailed, err)
	}
	defer tx.Rollback()

	charts, err := q.getCharts(ctx, tx)
	if err != nil {
		return nil, err
	}
	worldsendCharts, err := q.getWorldsendCharts(ctx, tx)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("%w: commit chart stats export snapshot: %v", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return &domainrepo.ChartStatsExportSnapshot{Charts: charts, WorldsendCharts: worldsendCharts}, nil
}

func (q *ChartStatsExportQueryService) getCharts(ctx context.Context, exec domainrepo.Executor) ([]domainrepo.ChartStatsExportItem, error) {
	const query = `
		SELECT
			s.display_id AS song_display_id,
			s.title AS song_title,
			d.name AS difficulty,
			c.const AS chart_const,
			c.is_const_unknown,
			COALESCE(stats.player_count, 0) AS player_count,
			COALESCE(stats.rank_aaal, 0) AS rank_aaal,
			COALESCE(stats.rank_s, 0) AS rank_s,
			COALESCE(stats.rank_sp, 0) AS rank_sp,
			COALESCE(stats.rank_ss, 0) AS rank_ss,
			COALESCE(stats.rank_ssp, 0) AS rank_ssp,
			COALESCE(stats.rank_sss, 0) AS rank_sss,
			COALESCE(stats.rank_sssp, 0) AS rank_sssp,
			COALESCE(stats.rank_max, 0) AS rank_max,
			COALESCE(stats.combo_none, 0) AS combo_none,
			COALESCE(stats.combo_fc, 0) AS combo_fc,
			COALESCE(stats.combo_aj, 0) AS combo_aj,
			COALESCE(stats.combo_ajc, 0) AS combo_ajc
		FROM charts c
		INNER JOIN songs s ON s.id = c.song_id
		INNER JOIN difficulties d ON d.id = c.difficulty_id
		LEFT JOIN chart_stats_by_rating_band stats
		  ON stats.chart_id = c.id
		 AND stats.rating_band_id = ?
		WHERE s.is_worldsend = 0
		  AND s.is_deleted = 0
		ORDER BY d.id, s.id, c.id
	`
	var rows []chartStatsExportRow
	if err := exec.SelectContext(ctx, &rows, query, info.AllRatingBandID); err != nil {
		return nil, fmt.Errorf("%w: list chart stats export rows: %v", domainrepo.ErrRepositoryOperationFailed, err)
	}
	items := make([]domainrepo.ChartStatsExportItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, domainrepo.ChartStatsExportItem{
			SongDisplayID:  row.SongDisplayID,
			SongTitle:      row.SongTitle,
			Difficulty:     row.Difficulty,
			ChartConst:     row.ChartConst,
			IsConstUnknown: row.IsConstUnknown,
			PlayerCount:    row.PlayerCount,
			Rank:           rankFromExportRow(row.RankAAAL, row.RankS, row.RankSP, row.RankSS, row.RankSSP, row.RankSSS, row.RankSSSP, row.RankMax),
			Combo:          comboFromExportRow(row.ComboNone, row.ComboFC, row.ComboAJ, row.ComboAJC),
		})
	}
	return items, nil
}

func (q *ChartStatsExportQueryService) getWorldsendCharts(ctx context.Context, exec domainrepo.Executor) ([]domainrepo.WorldsendChartStatsExportItem, error) {
	const query = `
		SELECT
			s.display_id AS song_display_id,
			s.title AS song_title,
			wc.level_star,
			wc.attribute,
			COALESCE(stats.player_count, 0) AS player_count,
			COALESCE(stats.rank_aaal, 0) AS rank_aaal,
			COALESCE(stats.rank_s, 0) AS rank_s,
			COALESCE(stats.rank_sp, 0) AS rank_sp,
			COALESCE(stats.rank_ss, 0) AS rank_ss,
			COALESCE(stats.rank_ssp, 0) AS rank_ssp,
			COALESCE(stats.rank_sss, 0) AS rank_sss,
			COALESCE(stats.rank_sssp, 0) AS rank_sssp,
			COALESCE(stats.rank_max, 0) AS rank_max,
			COALESCE(stats.combo_none, 0) AS combo_none,
			COALESCE(stats.combo_fc, 0) AS combo_fc,
			COALESCE(stats.combo_aj, 0) AS combo_aj,
			COALESCE(stats.combo_ajc, 0) AS combo_ajc
		FROM worldsend_charts wc
		INNER JOIN songs s ON s.id = wc.song_id
		LEFT JOIN worldsend_chart_stats_by_rating_band stats
		  ON stats.worldsend_chart_id = wc.id
		 AND stats.rating_band_id = ?
		WHERE s.is_worldsend = 1
		  AND s.is_deleted = 0
		ORDER BY s.id, wc.id
	`
	var rows []worldsendChartStatsExportRow
	if err := exec.SelectContext(ctx, &rows, query, info.AllRatingBandID); err != nil {
		return nil, fmt.Errorf("%w: list worldsend chart stats export rows: %v", domainrepo.ErrRepositoryOperationFailed, err)
	}
	items := make([]domainrepo.WorldsendChartStatsExportItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, domainrepo.WorldsendChartStatsExportItem{
			SongDisplayID: row.SongDisplayID,
			SongTitle:     row.SongTitle,
			LevelStar:     row.LevelStar,
			Attribute:     row.Attribute,
			PlayerCount:   row.PlayerCount,
			Rank:          rankFromExportRow(row.RankAAAL, row.RankS, row.RankSP, row.RankSS, row.RankSSP, row.RankSSS, row.RankSSSP, row.RankMax),
			Combo:         comboFromExportRow(row.ComboNone, row.ComboFC, row.ComboAJ, row.ComboAJC),
		})
	}
	return items, nil
}

func rankFromExportRow(aaal, s, sp, ss, ssp, sss, sssp, max int) domainrepo.ChartStatsExportRank {
	return domainrepo.ChartStatsExportRank{AAAL: aaal, S: s, SP: sp, SS: ss, SSP: ssp, SSS: sss, SSSP: sssp, Max: max}
}

func comboFromExportRow(none, fc, aj, ajc int) domainrepo.ChartStatsExportCombo {
	return domainrepo.ChartStatsExportCombo{None: none, FC: fc, AJ: aj, AJC: ajc}
}
