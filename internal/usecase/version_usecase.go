package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
)

// VersionUsecase は管理者向けバージョン操作を提供します。
type VersionUsecase interface {
	ListAll(ctx context.Context) ([]*entity.Version, error)
	Create(ctx context.Context, name, shortName string, releasedAt time.Time) (*entity.Version, error)
	Rename(ctx context.Context, id displayid.DisplayID, newName, newShortName string) (*entity.Version, error)
	Delete(ctx context.Context, id displayid.DisplayID) error
}
