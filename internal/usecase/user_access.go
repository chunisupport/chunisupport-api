package usecase

import (
	"context"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// errNilFriendshipRepository はフレンド判定に必要なリポジトリが設定されていない場合に返します。
var errNilFriendshipRepository = errors.New("friendship repository is nil")

// canAccessPrivateUser は requester が target のデータを閲覧できるか判定します。
// 非公開ユーザーは本人と相互フレンドだけが閲覧できます。
// フレンド判定リポジトリがない場合に閲覧不可として扱うと、非公開ユーザーのフレンド閲覧が
// 404 へ静かに退行するため、設定漏れとしてエラーを返します。
func canAccessPrivateUser(ctx context.Context, exec repository.Executor, friendshipRepo repository.FriendshipRepository, target *entity.User, requester *entity.User) (bool, error) {
	if target == nil {
		return false, nil
	}
	if !target.IsPrivate {
		return true, nil
	}
	if requester == nil {
		return false, nil
	}
	if requester.ID == target.ID {
		return true, nil
	}
	if friendshipRepo == nil {
		return false, errNilFriendshipRepository
	}
	return friendshipRepo.ExistsMutualAccepted(ctx, exec, requester.ID, target.ID)
}
