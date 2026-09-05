package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// VersionUsecase は管理者向けバージョン操作を提供します。
type VersionUsecase interface {
	ListAll(ctx context.Context) ([]*entity.Version, error)
	Create(ctx context.Context, name string, releasedAt time.Time) (*entity.Version, error)
	Rename(ctx context.Context, id int, newName string) (*entity.Version, error)
	Delete(ctx context.Context, id int) error
}
