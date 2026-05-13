package masterdata

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/stretchr/testify/assert"
)

func TestChartStatsMasterProviderAdapter_RatingBands_呼び出し側で変更しても内部キャッシュへ波及しない(t *testing.T) {
	t.Parallel()

	cache := &StaticCache{
		RatingBands: []*ratingband.RatingBand{
			{ID: 1, Label: "A", SortOrder: 1},
			{ID: 2, Label: "B", SortOrder: 2},
		},
	}
	adapter := NewChartStatsMasterProviderAdapter(cache)

	got := adapter.RatingBands()
	got[0].Label = "CHANGED"

	assert.Equal(t, "A", cache.RatingBands[0].Label)
}
