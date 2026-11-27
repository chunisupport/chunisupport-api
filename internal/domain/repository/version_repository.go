package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// VersionRepository はバージョン集約の永続化を扱います。
type VersionRepository interface {
	FindAll(ctx context.Context, exec Executor) ([]*entity.Version, error)
	FindByID(ctx context.Context, exec Executor, id int) (*entity.Version, error)
	FindByIDForUpdate(ctx context.Context, exec Executor, id int) (*entity.Version, error)
	FindByName(ctx context.Context, exec Executor, name string) (*entity.Version, error)
	FindLatest(ctx context.Context, exec Executor) (*entity.Version, error)
	ExistsSongInRange(ctx context.Context, exec Executor, from time.Time, to *time.Time) (bool, error)
	Create(ctx context.Context, exec Executor, version *entity.Version) (*entity.Version, error)
	Save(ctx context.Context, exec Executor, version *entity.Version) error
	Delete(ctx context.Context, exec Executor, id int) error
}

// VersionCacheReloader は永続化済みバージョンをキャッシュへ再読込します。
type VersionCacheReloader interface {
	ReloadVersions(ctx context.Context) error
}
