package masterdata

import (
	"context"
	"fmt"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/jmoiron/sqlx"
)

// StaticCache は統計DB（SQLite）から起動時にプリロードされるマスタのセットです。
type StaticCache struct {
	RatingBands        []*entity.RatingBand
	RatingBandsByID    map[int]*entity.RatingBand
	RatingBandsByLabel map[string]*entity.RatingBand
}

// PreloadStatic は統計DBから固定値が INSERT されているマスタを読み込み、キャッシュを構築します。
func PreloadStatic(ctx context.Context, staticDB *sqlx.DB) (*StaticCache, error) {
	ratingBands, err := loadRatingBands(ctx, staticDB)
	if err != nil {
		return nil, fmt.Errorf("failed to preload rating_bands: %w", err)
	}

	ratingBandsByID := make(map[int]*entity.RatingBand, len(ratingBands))
	ratingBandsByLabel := make(map[string]*entity.RatingBand, len(ratingBands))
	for _, band := range ratingBands {
		ratingBandsByID[band.ID] = band
		ratingBandsByLabel[band.Label] = band
	}

	return &StaticCache{
		RatingBands:        ratingBands,
		RatingBandsByID:    ratingBandsByID,
		RatingBandsByLabel: ratingBandsByLabel,
	}, nil
}

func loadRatingBands(ctx context.Context, db *sqlx.DB) ([]*entity.RatingBand, error) {
	const query = `
		SELECT id, label, min_inclusive, max_exclusive, sort_order
		FROM rating_bands
		ORDER BY sort_order
	`

	type ratingBandRow struct {
		ID           int      `db:"id"`
		Label        string   `db:"label"`
		MinInclusive *float64 `db:"min_inclusive"`
		MaxExclusive *float64 `db:"max_exclusive"`
		SortOrder    int      `db:"sort_order"`
	}

	var rows []ratingBandRow
	if err := db.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}

	results := make([]*entity.RatingBand, 0, len(rows))
	for _, row := range rows {
		results = append(results, &entity.RatingBand{
			ID:           row.ID,
			Label:        row.Label,
			MinInclusive: row.MinInclusive,
			MaxExclusive: row.MaxExclusive,
			SortOrder:    row.SortOrder,
		})
	}

	return results, nil
}

// GetRatingBandByID はIDからRatingBandを取得します。
// 見つからない場合はnilを返します。
func (c *StaticCache) GetRatingBandByID(id int) *entity.RatingBand {
	if c == nil {
		return nil
	}
	return c.RatingBandsByID[id]
}

// GetRatingBandByLabel はラベルからRatingBandを取得します。
// 見つからない場合はnilを返します。
func (c *StaticCache) GetRatingBandByLabel(label string) *entity.RatingBand {
	if c == nil {
		return nil
	}
	return c.RatingBandsByLabel[label]
}
