package usecase

import (
	"context"
	"errors"
	"slices"

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
// リクエスト元と対象の行をロックし、実行時点でもリクエスト元がADMINであることを確認します。
// ADMINは自分を降格できないため、変更後もリクエスト元はADMINのまま残ります。
// ADMIN同士が同時に互いを降格しても一方は認可に失敗するので、ADMINが0人になりません。
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
		lockedRequester, lockedTarget, err := u.lockRequesterAndTarget(ctx, tx, requester.ID, target.ID)
		if err != nil {
			return err
		}
		if !info.HasRole(lockedRequester.AccountTypeID, info.AccountTypeAdmin) {
			return ErrAdminRequired
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

// lockRequesterAndTarget はリクエスト元と対象のユーザー行をロックして取得します。
// 同時に互いを変更するリクエストでデッドロックしないよう、常にIDの昇順でロックします。
func (u *userPermissionUsecase) lockRequesterAndTarget(ctx context.Context, tx repository.Executor, requesterID int, targetID int) (*entity.User, *entity.User, error) {
	locked := make(map[int]*entity.User, 2)
	ids := []int{min(requesterID, targetID), max(requesterID, targetID)}
	for _, id := range slices.Compact(ids) {
		user, err := u.userRepo.FindByIDForUpdate(ctx, tx, id)
		if errors.Is(err, repository.ErrUserNotFound) {
			// 認可確認後にリクエスト元が退会した場合は権限がないものとして扱います。
			if id == requesterID {
				return nil, nil, ErrAdminRequired
			}
			return nil, nil, ErrUserNotFound
		}
		if err != nil {
			return nil, nil, err
		}
		locked[id] = user
	}
	return locked[requesterID], locked[targetID], nil
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
