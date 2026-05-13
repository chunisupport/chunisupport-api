package masterdata

import (
	"context"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/jmoiron/sqlx"
)

// StaticCache は統計DB（SQLite）から起動時にプリロードされるマスタのセットです。
type StaticCache struct {
	RatingBands        []*ratingband.RatingBand
	RatingBandsByID    map[int]*ratingband.RatingBand
	RatingBandsByLabel map[string]*ratingband.RatingBand
}

// PreloadStatic は統計DBから固定値が INSERT されているマスタを読み込み、キャッシュを構築します。
func PreloadStatic(ctx context.Context, staticDB *sqlx.DB) (*StaticCache, error) {
	ratingBands, err := loadRatingBands(ctx, staticDB)
	if err != nil {
		return nil, fmt.Errorf("failed to preload rating_bands: %w", err)
	}

	ratingBandsByID := make(map[int]*ratingband.RatingBand, len(ratingBands))
	ratingBandsByLabel := make(map[string]*ratingband.RatingBand, len(ratingBands))
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

func loadRatingBands(ctx context.Context, db *sqlx.DB) ([]*ratingband.RatingBand, error) {
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

// RatingBandsSnapshot はレーティング帯の防衛的コピーを返します。
func (c *StaticCache) RatingBandsSnapshot() []*ratingband.RatingBand {
	if c == nil || len(c.RatingBands) == 0 {
		return []*ratingband.RatingBand{}
	}

	res := make([]*ratingband.RatingBand, len(c.RatingBands))
	for i, band := range c.RatingBands {
		if band == nil {
			continue
		}
		copied := *band
		// Deep copy pointer fields to prevent mutation of cached values
		if band.MinInclusive != nil {
			minVal := *band.MinInclusive
			copied.MinInclusive = &minVal
		}
		if band.MaxExclusive != nil {
			maxVal := *band.MaxExclusive
			copied.MaxExclusive = &maxVal
		}
		res[i] = &copied
	}

	return res
}

// GetRatingBandByID はIDからRatingBandを取得します。
// 見つからない場合はnilを返します。
func (c *StaticCache) GetRatingBandByID(id int) *ratingband.RatingBand {
	if c == nil {
		return nil
	}
	return c.RatingBandsByID[id]
}

// GetRatingBandByLabel はラベルからRatingBandを取得します。
// 見つからない場合はnilを返します。
func (c *StaticCache) GetRatingBandByLabel(label string) *ratingband.RatingBand {
	if c == nil {
		return nil
	}
	return c.RatingBandsByLabel[label]
}
