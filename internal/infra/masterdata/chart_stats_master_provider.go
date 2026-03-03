package masterdata

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
)

// ChartStatsMasterProviderAdapter はStaticCacheを譜面統計用マスタプロバイダとして公開するアダプタです。
type ChartStatsMasterProviderAdapter struct {
	cache *StaticCache
}

// NewChartStatsMasterProviderAdapter はStaticCacheを repository.ChartStatsMasterProvider に適合させます。
func NewChartStatsMasterProviderAdapter(cache *StaticCache) repository.ChartStatsMasterProvider {
	return &ChartStatsMasterProviderAdapter{cache: cache}
}

// RatingBands は譜面統計で参照するレーティング帯一覧を返します。
func (a *ChartStatsMasterProviderAdapter) RatingBands() []*ratingband.RatingBand {
	if a == nil || a.cache == nil {
		return []*ratingband.RatingBand{}
	}
	return a.cache.RatingBands
}
