package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/jmoiron/sqlx"
)

var _ domainrepo.AdminChartRankingQueryService = (*AdminChartRankingQueryService)(nil)

type AdminChartRankingQueryService struct {
	db *sqlx.DB
}

func NewAdminChartRankingQueryService(db *sqlx.DB) *AdminChartRankingQueryService {
	return &AdminChartRankingQueryService{db: db}
}

type adminChartRankingChartRow struct {
	SongDisplayID  string                      `db:"song_display_id"`
	SongTitle      string                      `db:"song_title"`
	SongArtist     string                      `db:"song_artist"`
	Difficulty     string                      `db:"difficulty"`
	Const          chartconstant.ChartConstant `db:"chart_const"`
	IsConstUnknown bool                        `db:"is_const_unknown"`
	LevelStar      *int                        `db:"level_star"`
	Attribute      *string                     `db:"attribute"`
	ChartID        int                         `db:"chart_id"`
	IsWorldsend    bool                        `db:"is_worldsend"`
}

type adminChartRankingRecordRow struct {
	Username   string      `db:"username"`
	PlayerName string      `db:"player_name"`
	Score      score.Score `db:"score"`
	ClearLamp  string      `db:"clear_lamp"`
	ComboLamp  string      `db:"combo_lamp"`
	FullChain  string      `db:"full_chain"`
	UpdatedAt  time.Time   `db:"updated_at"`
}

func (q *AdminChartRankingQueryService) GetStandard(ctx context.Context, displayID string, difficulty string, limit int) (*domainrepo.AdminChartRankingData, error) {
	const query = `
		SELECT
			s.display_id AS song_display_id,
			s.title AS song_title,
			s.artist AS song_artist,
			d.name AS difficulty,
			c.const AS chart_const,
			c.is_const_unknown AS is_const_unknown,
			NULL AS level_star,
			NULL AS attribute,
			c.id AS chart_id,
			0 AS is_worldsend
		FROM songs s
		INNER JOIN charts c ON c.song_id = s.id
		INNER JOIN difficulties d ON d.id = c.difficulty_id
		WHERE s.display_id = ?
		  AND d.name = ?
		  AND s.is_worldsend = 0
		  AND s.is_deleted = 0
	`
	var row adminChartRankingChartRow
	if err := q.db.GetContext(ctx, &row, query, displayID, difficulty); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapAdminChartRankingQueryError("find chart", err)
	}
	records, total, err := listAdminChartRankingRecords(ctx, q.db, standardAdminChartRankingCountQuery, standardAdminChartRankingListQuery, row.ChartID, limit)
	if err != nil {
		return nil, err
	}
	return &domainrepo.AdminChartRankingData{Chart: toAdminChartRankingChart(row), Records: records, Total: total}, nil
}

func (q *AdminChartRankingQueryService) GetWorldsend(ctx context.Context, displayID string, limit int) (*domainrepo.AdminChartRankingData, error) {
	const query = `
		SELECT
			s.display_id AS song_display_id,
			s.title AS song_title,
			s.artist AS song_artist,
			'WORLD''S END' AS difficulty,
			0 AS chart_const,
			1 AS is_const_unknown,
			wc.level_star AS level_star,
			wc.attribute AS attribute,
			wc.id AS chart_id,
			1 AS is_worldsend
		FROM songs s
		INNER JOIN worldsend_charts wc ON wc.song_id = s.id
		WHERE s.display_id = ?
		  AND s.is_worldsend = 1
		  AND s.is_deleted = 0
	`
	var row adminChartRankingChartRow
	if err := q.db.GetContext(ctx, &row, query, displayID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapAdminChartRankingQueryError("find worldsend chart", err)
	}
	records, total, err := listAdminChartRankingRecords(ctx, q.db, worldsendAdminChartRankingCountQuery, worldsendAdminChartRankingListQuery, row.ChartID, limit)
	if err != nil {
		return nil, err
	}
	return &domainrepo.AdminChartRankingData{Chart: toAdminChartRankingChart(row), Records: records, Total: total}, nil
}

const standardAdminChartRankingCountQuery = `
		SELECT COUNT(*)
		FROM player_records
		WHERE chart_id = ?
	`

const standardAdminChartRankingListQuery = `
		SELECT
			u.username AS username,
			p.player_name AS player_name,
			pr.score AS score,
			cl.name AS clear_lamp,
			co.name AS combo_lamp,
			fc.name AS full_chain,
			pr.updated_at AS updated_at
		FROM player_records pr
		INNER JOIN players p ON p.id = pr.player_id
		INNER JOIN users u ON u.player_id = p.id
		INNER JOIN clear_lamp_types cl ON cl.id = pr.clear_lamp_id
		INNER JOIN combo_lamp_types co ON co.id = pr.combo_lamp_id
		INNER JOIN full_chain_types fc ON fc.id = pr.full_chain_id
		WHERE pr.chart_id = ?
		ORDER BY pr.score DESC, pr.updated_at DESC, u.username ASC
		LIMIT ?
	`

const worldsendAdminChartRankingCountQuery = `
		SELECT COUNT(*)
		FROM player_worldsend_records
		WHERE worldsend_chart_id = ?
	`

const worldsendAdminChartRankingListQuery = `
		SELECT
			u.username AS username,
			p.player_name AS player_name,
			pwr.score AS score,
			cl.name AS clear_lamp,
			co.name AS combo_lamp,
			fc.name AS full_chain,
			pwr.updated_at AS updated_at
		FROM player_worldsend_records pwr
		INNER JOIN players p ON p.id = pwr.player_id
		INNER JOIN users u ON u.player_id = p.id
		INNER JOIN clear_lamp_types cl ON cl.id = pwr.clear_lamp_id
		INNER JOIN combo_lamp_types co ON co.id = pwr.combo_lamp_id
		INNER JOIN full_chain_types fc ON fc.id = pwr.full_chain_id
		WHERE pwr.worldsend_chart_id = ?
		ORDER BY pwr.score DESC, pwr.updated_at DESC, u.username ASC
		LIMIT ?
	`

func listAdminChartRankingRecords(ctx context.Context, db *sqlx.DB, countQuery string, listQuery string, chartID int, limit int) ([]*domainrepo.AdminChartRankingRecord, int, error) {
	var total int
	if err := db.GetContext(ctx, &total, countQuery, chartID); err != nil {
		return nil, 0, wrapAdminChartRankingQueryError("count records", err)
	}

	var rows []adminChartRankingRecordRow
	if err := db.SelectContext(ctx, &rows, listQuery, chartID, limit); err != nil {
		return nil, 0, wrapAdminChartRankingQueryError("list records", err)
	}

	records := make([]*domainrepo.AdminChartRankingRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, &domainrepo.AdminChartRankingRecord{
			Username:   row.Username,
			PlayerName: row.PlayerName,
			Score:      uint32(row.Score),
			ClearLamp:  row.ClearLamp,
			ComboLamp:  row.ComboLamp,
			FullChain:  row.FullChain,
			UpdatedAt:  row.UpdatedAt,
		})
	}

	return records, total, nil
}

func toAdminChartRankingChart(row adminChartRankingChartRow) *domainrepo.AdminChartRankingChart {
	return &domainrepo.AdminChartRankingChart{
		SongDisplayID:  row.SongDisplayID,
		SongTitle:      row.SongTitle,
		SongArtist:     row.SongArtist,
		Difficulty:     row.Difficulty,
		Const:          row.Const,
		IsConstUnknown: row.IsConstUnknown,
		LevelStar:      row.LevelStar,
		Attribute:      row.Attribute,
		IsWorldsend:    row.IsWorldsend,
	}
}

func wrapAdminChartRankingQueryError(operation string, err error) error {
	return fmt.Errorf("%w: %s: %v", domainrepo.ErrRepositoryOperationFailed, operation, err)
}
