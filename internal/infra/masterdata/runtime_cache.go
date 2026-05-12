package masterdata

import (
	"context"
	"fmt"
	"sync"

	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
)

// Loader は動的・静的マスタを同時に読み込むための境界です。
type Loader interface {
	Load(ctx context.Context) (*Cache, *StaticCache, error)
}

// RuntimeCache は再読み込み可能なマスタキャッシュです。
type RuntimeCache struct {
	mu      sync.RWMutex
	dynamic *Cache
	static  *StaticCache
	loader  Loader
}

// NewRuntimeCache は初期ロードを実行してRuntimeCacheを生成します。
func NewRuntimeCache(ctx context.Context, loader Loader) (*RuntimeCache, error) {
	if loader == nil {
		return nil, fmt.Errorf("loader is nil")
	}

	dynamic, static, err := loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	return &RuntimeCache{dynamic: dynamic, static: static, loader: loader}, nil
}

// Reload はマスタを再読み込みし、成功時のみスワップします。
func (c *RuntimeCache) Reload(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("runtime cache is nil")
	}

	dynamic, static, err := c.loader.Load(ctx)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.dynamic = dynamic
	c.static = static
	return nil
}

// Snapshot は動的マスタの現在値を返します。
func (c *RuntimeCache) Snapshot() *Cache {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.dynamic
}

// StaticSnapshot は静的マスタの現在値を返します。
func (c *RuntimeCache) StaticSnapshot() *StaticCache {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.static
}

func (c *RuntimeCache) PlayerDataMasters() *domainmasterdata.PlayerDataMasters {
	dynamic := c.Snapshot()
	if dynamic == nil {
		return nil
	}
	return dynamic.PlayerDataMasters()
}

func (c *RuntimeCache) SongMasters() *domainmasterdata.SongMasters {
	dynamic := c.Snapshot()
	if dynamic == nil {
		return nil
	}
	return dynamic.SongMasters()
}

func (c *RuntimeCache) GetAccountTypeNameByID(id int) string {
	dynamic := c.Snapshot()
	if dynamic == nil {
		return "UNKNOWN"
	}
	return dynamic.GetAccountTypeNameByID(id)
}

func (c *RuntimeCache) GoalMasters() *domainmasterdata.GoalMasters {
	dynamic := c.Snapshot()
	if dynamic == nil {
		return nil
	}
	return dynamic.GoalMasters()
}

func (c *RuntimeCache) MasterDataMasters() *domainmasterdata.MasterDataMasters {
	dynamic := c.Snapshot()
	if dynamic == nil {
		return nil
	}
	return dynamic.MasterDataMasters()
}

func (c *RuntimeCache) RatingBands() []*ratingband.RatingBand {
	static := c.StaticSnapshot()
	if static == nil {
		return nil
	}
	return static.RatingBands
}
