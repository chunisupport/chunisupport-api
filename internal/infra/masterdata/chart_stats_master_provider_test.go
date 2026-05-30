package masterdata

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/stretchr/testify/assert"
)

func TestChartStatsMasterProviderAdapter_RatingBands_呼び出し側で変更しても内部キャッシュへ波及しない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		buildCache       func() *StaticCache
		mutate           func(got []*ratingband.RatingBand)
		wantLabel        string
		wantMinInclusive *float64
		wantMaxExclusive *float64
	}{
		{
			name: "Labelを書き換えても内部キャッシュの値は保持される",
			buildCache: func() *StaticCache {
				return &StaticCache{RatingBands: []*ratingband.RatingBand{{ID: 1, Label: "A", SortOrder: 1}}}
			},
			mutate: func(got []*ratingband.RatingBand) {
				got[0].Label = "CHANGED"
			},
			wantLabel: "A",
		},
		{
			name: "ポインタフィールドを書き換えても内部キャッシュの値は保持される",
			buildCache: func() *StaticCache {
				minVal := 10.0
				maxVal := 20.0
				return &StaticCache{RatingBands: []*ratingband.RatingBand{{ID: 1, Label: "A", MinInclusive: &minVal, MaxExclusive: &maxVal, SortOrder: 1}}}
			},
			mutate: func(got []*ratingband.RatingBand) {
				if got[0].MinInclusive != nil {
					*got[0].MinInclusive = 99.9
				}
				if got[0].MaxExclusive != nil {
					*got[0].MaxExclusive = 199.9
				}
			},
			wantLabel:        "A",
			wantMinInclusive: float64Ptr(10.0),
			wantMaxExclusive: float64Ptr(20.0),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Given
			cache := tt.buildCache()
			adapter := NewChartStatsMasterProviderAdapter(cache)

			// When
			got := adapter.RatingBands()
			tt.mutate(got)

			// Then
			assert.Equal(t, tt.wantLabel, cache.RatingBands[0].Label)
			if tt.wantMinInclusive == nil {
				assert.Nil(t, cache.RatingBands[0].MinInclusive)
			} else {
				assert.NotNil(t, cache.RatingBands[0].MinInclusive)
				assert.Equal(t, *tt.wantMinInclusive, *cache.RatingBands[0].MinInclusive)
			}
			if tt.wantMaxExclusive == nil {
				assert.Nil(t, cache.RatingBands[0].MaxExclusive)
			} else {
				assert.NotNil(t, cache.RatingBands[0].MaxExclusive)
				assert.Equal(t, *tt.wantMaxExclusive, *cache.RatingBands[0].MaxExclusive)
			}
		})
	}
}

func float64Ptr(v float64) *float64 {
	return &v
}
