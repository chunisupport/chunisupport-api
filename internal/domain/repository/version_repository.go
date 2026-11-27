package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
)

// VersionRepository はバージョン集約の永続化を扱います。
type VersionRepository interface {
	FindAll(ctx context.Context, exec Executor) ([]*entity.Version, error)
	FindByDisplayID(ctx context.Context, exec Executor, displayID displayid.DisplayID) (*entity.Version, error)
	FindByDisplayIDForUpdate(ctx context.Context, exec Executor, displayID displayid.DisplayID) (*entity.Version, error)
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
