package masterdata

import (
	"context"
	"fmt"
	"slices"
	"sync"

	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
)

// Loader は動的・静的マスタを同時に読み込むための境界です。
type Loader interface {
	Load(ctx context.Context) (*Cache, *StaticCache, error)
}

// RuntimeCache は再読み込み可能なマスタキャッシュです。
type RuntimeCache struct {
	mu       sync.RWMutex
	reloadMu sync.Mutex
	dynamic  *Cache
	static   *StaticCache
	loader   Loader
}

// NewRuntimeCache は初期ロードを実行してRuntimeCacheを生成します。
func NewRuntimeCache(ctx context.Context, loader Loader) (*RuntimeCache, error) {
	if loader == nil {
		return nil, fmt.Errorf("%w: loader is nil", repository.ErrRepositoryOperationFailed)
	}

	dynamic, static, err := loader.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to load initial cache: %w", repository.ErrRepositoryOperationFailed, err)
	}
	if dynamic == nil || static == nil {
		return nil, fmt.Errorf("%w: loader returned nil cache", repository.ErrRepositoryOperationFailed)
	}

	return &RuntimeCache{dynamic: dynamic, static: static, loader: loader}, nil
}

// Reload はマスタを再読み込みし、成功時のみスワップします。
func (c *RuntimeCache) Reload(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("%w: runtime cache is nil", repository.ErrRepositoryOperationFailed)
	}

	c.reloadMu.Lock()
	defer c.reloadMu.Unlock()

	if c.loader == nil {
		return fmt.Errorf("%w: loader is nil", repository.ErrRepositoryOperationFailed)
	}

	dynamic, static, err := c.loader.Load(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to reload cache: %w", repository.ErrRepositoryOperationFailed, err)
	}
	if dynamic == nil || static == nil {
		return fmt.Errorf("%w: loader returned nil cache", repository.ErrRepositoryOperationFailed)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.dynamic = dynamic
	c.static = static
	return nil
}

// snapshot は動的マスタの現在値のコピーを返します。
func (c *RuntimeCache) snapshot() *Cache {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.dynamic == nil {
		return nil
	}

	// 浅いコピーを作成（マップ自体は共有されるが、構造体ポインタは新規）
	copied := *c.dynamic
	return &copied
}

// staticSnapshot は静的マスタの現在値の浅いコピーを返します。
func (c *RuntimeCache) staticSnapshot() *StaticCache {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.static == nil {
		return nil
	}

	// スナップショット取得自体の責務は参照の固定化に限定し、
	// 可変コレクションの防衛的コピーは各公開アクセサ側で行う。
	copied := *c.static
	return &copied
}

func (c *RuntimeCache) PlayerDataMasters() *domainmasterdata.PlayerDataMasters {
	dynamic := c.snapshot()
	if dynamic == nil {
		return nil
	}
	return dynamic.PlayerDataMasters()
}

func (c *RuntimeCache) SongMasters() *domainmasterdata.SongMasters {
	dynamic := c.snapshot()
	if dynamic == nil {
		return nil
	}
	return dynamic.SongMasters()
}

func (c *RuntimeCache) GetAccountTypeNameByID(id int) string {
	dynamic := c.snapshot()
	if dynamic == nil {
		return "UNKNOWN"
	}
	return dynamic.GetAccountTypeNameByID(id)
}

func (c *RuntimeCache) GoalMasters() *domainmasterdata.GoalMasters {
	dynamic := c.snapshot()
	if dynamic == nil {
		return nil
	}
	return dynamic.GoalMasters()
}

func (c *RuntimeCache) MasterDataMasters() *domainmasterdata.MasterDataMasters {
	dynamic := c.snapshot()
	if dynamic == nil {
		return nil
	}
	return dynamic.MasterDataMasters()
}

func (c *RuntimeCache) RatingBands() []*ratingband.RatingBand {
	static := c.staticSnapshot()
	if static == nil || len(static.RatingBands) == 0 {
		return []*ratingband.RatingBand{}
	}
	// 呼び出し側からの変更が内部キャッシュへ伝播しないように、各要素をコピーして返す。
	res := make([]*ratingband.RatingBand, len(static.RatingBands))
	for i, b := range static.RatingBands {
		if b == nil {
			continue
		}
		copy := *b
		res[i] = &copy
	}
	return res
}
