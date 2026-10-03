package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/jmoiron/sqlx"
)

var _ domainrepo.ChunirecQueryService = (*ChunirecQueryService)(nil)

type ChunirecQueryService struct {
	db *sqlx.DB
}

func NewChunirecQueryService(db *sqlx.DB) *ChunirecQueryService {
	return &ChunirecQueryService{db: db}
}

type chunirecProfileRow struct {
	Name              string         `db:"name"`
	Level             int            `db:"level"`
	Rating            *float64       `db:"rating"`
	ClassEmblemID     *int           `db:"class_emblem_id"`
	ClassEmblemBaseID *int           `db:"class_emblem_base_id"`
	Title             sql.NullString `db:"title"`
	TitleRarity       sql.NullString `db:"title_rarity"`
	UpdatedAt         time.Time      `db:"updated_at"`
}

type chunirecRecordRow struct {
	ID             string                      `db:"id"`
	Title          string                      `db:"title"`
	Difficulty     string                      `db:"difficulty"`
	Genre          string                      `db:"genre"`
	Const          chartconstant.ChartConstant `db:"const"`
	IsConstUnknown bool                        `db:"is_const_unknown"`
	Score          uint32                      `db:"score"`
	ClearLamp      *string                     `db:"clear_lamp"`
	ComboLamp      *string                     `db:"combo_lamp"`
	FullChain      *string                     `db:"full_chain"`
	UpdatedAt      time.Time                   `db:"updated_at"`
}

func (q *ChunirecQueryService) FindProfileByPlayerID(ctx context.Context, playerID int) (*domainrepo.ChunirecProfile, error) {
	const query = `
		SELECT
			p.player_name AS name,
			p.player_level AS level,
			p.calculated_player_rating AS rating,
			p.class_emblem_id AS class_emblem_id,
			p.class_emblem_base_id AS class_emblem_base_id,
			h.name AS title,
			ht.name AS title_rarity,
			p.updated_at AS updated_at
		FROM players p
		LEFT JOIN player_honors ph ON ph.player_id = p.id AND ph.slot = 1
		LEFT JOIN honors h ON h.id = ph.honor_id
		LEFT JOIN honor_types ht ON ht.id = h.honor_type_id
		WHERE p.id = ?
		LIMIT 1
	`

	var row chunirecProfileRow
	if err := q.db.GetContext(ctx, &row, query, playerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapChunirecQueryError("find profile", err)
	}

	profile := &domainrepo.ChunirecProfile{
		Name:              row.Name,
		Level:             row.Level,
		Rating:            row.Rating,
		ClassEmblemID:     row.ClassEmblemID,
		ClassEmblemBaseID: row.ClassEmblemBaseID,
		UpdatedAt:         row.UpdatedAt,
	}
	if row.Title.Valid && row.TitleRarity.Valid {
		profile.Title = &row.Title.String
		profile.TitleRarity = &row.TitleRarity.String
	}
	return profile, nil
}

func (q *ChunirecQueryService) ListRecordsByPlayerID(ctx context.Context, playerID int) ([]*domainrepo.ChunirecRecord, error) {
	const query = `
		SELECT
			s.display_id AS id,
			s.title AS title,
			d.name AS difficulty,
			COALESCE(g.name, '') AS genre,
			c.const AS const,
			c.is_const_unknown AS is_const_unknown,
			pr.score AS score,
			cl.name AS clear_lamp,
			co.name AS combo_lamp,
			fc.name AS full_chain,
			pr.updated_at AS updated_at
		FROM player_records pr
		INNER JOIN players p ON p.id = pr.player_id
		INNER JOIN charts c ON c.id = pr.chart_id
		INNER JOIN songs s ON s.id = c.song_id
		INNER JOIN difficulties d ON d.id = c.difficulty_id
		INNER JOIN slots sl ON sl.id = pr.slot_id
		LEFT JOIN genres g ON g.id = s.genre_id
		LEFT JOIN clear_lamp_types cl ON cl.id = pr.clear_lamp_id
		LEFT JOIN combo_lamp_types co ON co.id = pr.combo_lamp_id
		LEFT JOIN full_chain_types fc ON fc.id = pr.full_chain_id
		WHERE pr.player_id = ?
		  AND s.is_deleted = 0
		  AND s.is_worldsend = 0
		ORDER BY s.id ASC, d.sort_order ASC
	`

	var rows []chunirecRecordRow
	if err := q.db.SelectContext(ctx, &rows, query, playerID); err != nil {
		return nil, wrapChunirecQueryError("list records", err)
	}

	records := make([]*domainrepo.ChunirecRecord, 0, len(rows))
	for _, row := range rows {
		if row.UpdatedAt.IsZero() {
			continue
		}
		records = append(records, &domainrepo.ChunirecRecord{
			ID:             row.ID,
			Title:          row.Title,
			Difficulty:     row.Difficulty,
			Genre:          row.Genre,
			Const:          row.Const,
			IsConstUnknown: row.IsConstUnknown,
			Score:          row.Score,
			ClearLamp:      chunirecLamp(row.ClearLamp),
			ComboLamp:      chunirecLamp(row.ComboLamp),
			FullChain:      chunirecLamp(row.FullChain),
			UpdatedAt:      row.UpdatedAt,
		})
	}
	return records, nil
}

// NONEはランプの取得を示さないため、互換レスポンスでは未取得として扱う。
func chunirecLamp(lamp *string) *string {
	if lamp == nil || *lamp == "" || *lamp == "NONE" || *lamp == "none" {
		return nil
	}
	return lamp
}

func wrapChunirecQueryError(operation string, err error) error {
	return fmt.Errorf("%w: chunirec %s: %w", domainrepo.ErrRepositoryOperationFailed, operation, err)
}
