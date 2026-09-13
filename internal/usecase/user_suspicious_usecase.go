package usecase

import (
	"context"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// UserSuspiciousUsecase は管理者による不審アカウントフラグ変更を扱います。
type UserSuspiciousUsecase interface {
	// ChangeSuspicious は指定ユーザーの不審アカウントフラグを変更します。
	ChangeSuspicious(ctx context.Context, requester *entity.User, username string, isSuspicious bool) error
}

type userSuspiciousUsecase struct {
	db       repository.Executor
	tm       TransactionManager
	userRepo repository.UserRepository
}

// NewUserSuspiciousUsecase は不審アカウントフラグ変更ユースケースを生成します。
func NewUserSuspiciousUsecase(db repository.Executor, tm TransactionManager, userRepo repository.UserRepository) UserSuspiciousUsecase {
	return &userSuspiciousUsecase{db: db, tm: tm, userRepo: userRepo}
}

// ChangeSuspicious はADMINが指定ユーザーの不審アカウントフラグを変更します。
// 対象行をロックしてから保存することで、同時更新時に他の変更を上書きしません。
func (u *userSuspiciousUsecase) ChangeSuspicious(ctx context.Context, requester *entity.User, username string, isSuspicious bool) error {
	if requester == nil || !info.HasRole(requester.AccountTypeID, info.AccountTypeAdmin) {
		return ErrAdminRequired
	}

	target, err := u.userRepo.FindByUsername(ctx, u.db, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	return u.tm.Transactional(ctx, func(tx repository.Executor) error {
		lockedTarget, err := u.userRepo.FindByIDForUpdate(ctx, tx, target.ID)
		if err != nil {
			if errors.Is(err, repository.ErrUserNotFound) {
				return ErrUserNotFound
			}
			return err
		}

		if lockedTarget.IsSuspicious == isSuspicious {
			return nil
		}
		lockedTarget.ChangeSuspicious(isSuspicious)
		return u.userRepo.Save(ctx, tx, lockedTarget)
	})
}
