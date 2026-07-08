package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

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
		return false, nil
	}
	return friendshipRepo.ExistsMutualAccepted(ctx, exec, requester.ID, target.ID)
}
