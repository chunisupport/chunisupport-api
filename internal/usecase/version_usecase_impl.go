package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

type versionUsecaseImpl struct {
	versionRepo   repository.VersionRepository
	cacheReloader repository.VersionCacheReloader
	tm            TransactionManager
	exec          repository.Executor
	updateMu      sync.Mutex
}

// NewVersionUsecase は管理者向けバージョンユースケースを生成します。
func NewVersionUsecase(versionRepo repository.VersionRepository, cacheReloader repository.VersionCacheReloader, tm TransactionManager, exec repository.Executor) VersionUsecase {
	return &versionUsecaseImpl{versionRepo: versionRepo, cacheReloader: cacheReloader, tm: tm, exec: exec}
}

func (u *versionUsecaseImpl) ListAll(ctx context.Context) ([]*entity.Version, error) {
	return u.versionRepo.FindAll(ctx, u.exec)
}

func (u *versionUsecaseImpl) Create(ctx context.Context, name string, releasedAt time.Time) (*entity.Version, error) {
	version, err := entity.NewVersion(name, releasedAt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidVersionInput, err)
	}

	u.updateMu.Lock()
	defer u.updateMu.Unlock()

	var created *entity.Version
	err = u.tm.Transactional(ctx, func(tx repository.Executor) error {
		versions, err := u.versionRepo.FindAll(ctx, tx)
		if err != nil {
			return err
		}
		for _, existing := range versions {
			if sameDate(existing.ReleasedAt, version.ReleasedAt) {
				return fmt.Errorf("%w: released_at already exists", ErrInvalidVersionInput)
			}
		}
		created, err = u.versionRepo.Create(ctx, tx, version)
		return err
	})
	if err != nil {
		return nil, err
	}
	u.reload(ctx)
	return created, nil
}

func (u *versionUsecaseImpl) Rename(ctx context.Context, id int, newName string) (*entity.Version, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidVersionInput)
	}

	u.updateMu.Lock()
	defer u.updateMu.Unlock()

	var updated *entity.Version
	err := u.tm.Transactional(ctx, func(tx repository.Executor) error {
		current, err := u.versionRepo.FindByIDForUpdate(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := current.Rename(newName); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidVersionInput, err)
		}
		if err := u.versionRepo.Save(ctx, tx, current); err != nil {
			return err
		}
		updated = current
		return nil
	})
	if err != nil {
		return nil, err
	}
	u.reload(ctx)
	return updated, nil
}

func (u *versionUsecaseImpl) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidVersionInput)
	}

	u.updateMu.Lock()
	defer u.updateMu.Unlock()
	versionRangeMutationMu.Lock()
	defer versionRangeMutationMu.Unlock()

	err := u.tm.Transactional(ctx, func(tx repository.Executor) error {
		current, err := u.versionRepo.FindByIDForUpdate(ctx, tx, id)
		if err != nil {
			return err
		}
		latest, err := u.versionRepo.FindLatest(ctx, tx)
		if err != nil {
			return err
		}
		if latest.ID != current.ID {
			return ErrVersionNotLatest
		}
		inUse, err := u.versionRepo.ExistsSongInRange(ctx, tx, current.ReleasedAt, nil)
		if err != nil {
			return err
		}
		if inUse {
			return ErrVersionInUse
		}
		return u.versionRepo.Delete(ctx, tx, id)
	})
	if err != nil {
		return err
	}
	u.reload(ctx)
	return nil
}

// reload はコミット済みのDB更新結果をキャッシュへ反映します。
// 再読込失敗はログで検知し、確定済みの操作をAPI上の失敗として扱わないよう呼び出し元には返しません。
func (u *versionUsecaseImpl) reload(ctx context.Context) {
	reloadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), info.VersionCacheReloadTimeout)
	defer cancel()
	if err := u.cacheReloader.ReloadVersions(reloadCtx); err != nil {
		slog.ErrorContext(reloadCtx, "バージョンキャッシュの再読込に失敗しました", "error", err)
	}
}

func sameDate(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}
