package usecase

import (
	"context"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// UserPermissionUsecase は管理者によるユーザー権限変更を扱います。
type UserPermissionUsecase interface {
	// ChangePermission は指定ユーザーの権限を変更します。
	ChangePermission(ctx context.Context, requester *entity.User, username string, permission string) error
}

type userPermissionUsecase struct {
	db       repository.Executor
	tm       TransactionManager
	userRepo repository.UserRepository
}

// NewUserPermissionUsecase はユーザー権限変更ユースケースを生成します。
func NewUserPermissionUsecase(db repository.Executor, tm TransactionManager, userRepo repository.UserRepository) UserPermissionUsecase {
	return &userPermissionUsecase{
		db:       db,
		tm:       tm,
		userRepo: userRepo,
	}
}

// ChangePermission はADMINが指定ユーザーの権限を変更します。
// 対象行をロックしてから集約の規則を適用するため、同時更新で本人降格の制約が破られません。
func (u *userPermissionUsecase) ChangePermission(ctx context.Context, requester *entity.User, username string, permission string) error {
	if requester == nil || !info.HasRole(requester.AccountTypeID, info.AccountTypeAdmin) {
		return ErrAdminRequired
	}

	accountTypeID, err := accountTypeIDFromPermission(permission)
	if err != nil {
		return err
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

		if lockedTarget.AccountTypeID == accountTypeID {
			return nil
		}
		if err := lockedTarget.ChangeAccountType(requester.ID, accountTypeID); err != nil {
			return err
		}
		return u.userRepo.Save(ctx, tx, lockedTarget)
	})
}

func accountTypeIDFromPermission(permission string) (int, error) {
	switch permission {
	case "PLAYER":
		return info.AccountTypePlayer, nil
	case "EDITOR":
		return info.AccountTypeEditor, nil
	case "ADMIN":
		return info.AccountTypeAdmin, nil
	case "EXTDEV":
		return info.AccountTypeExtDev, nil
	default:
		return 0, entity.ErrInvalidAccountType
	}
}
