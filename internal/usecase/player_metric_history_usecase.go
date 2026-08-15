package usecase

import (
	"context"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// ErrPlayerMetricHistoryNotFound は公式指標履歴を取得できない場合に返されます。
var ErrPlayerMetricHistoryNotFound = errors.New("player metric history not found")

// PlayerMetricHistoryUsecase はプレイヤー単位の公式RATING・公式OVER POWER履歴を提供します。
type PlayerMetricHistoryUsecase interface {
	Get(ctx context.Context, username string, requester *entity.User) ([]entity.PlayerMetricHistoryEntry, error)
}

type playerMetricHistoryUsecase struct {
	exec           repository.Executor
	userRepo       repository.UserRepository
	historyQuery   repository.PlayerMetricHistoryQueryService
	friendshipRepo repository.FriendshipRepository
}

// NewPlayerMetricHistoryUsecase は公式指標履歴取得ユースケースを生成します。
func NewPlayerMetricHistoryUsecase(
	exec repository.Executor,
	userRepo repository.UserRepository,
	historyQuery repository.PlayerMetricHistoryQueryService,
	friendshipRepo repository.FriendshipRepository,
) PlayerMetricHistoryUsecase {
	return &playerMetricHistoryUsecase{
		exec: exec, userRepo: userRepo, historyQuery: historyQuery, friendshipRepo: friendshipRepo,
	}
}

func (us *playerMetricHistoryUsecase) Get(ctx context.Context, username string, requester *entity.User) ([]entity.PlayerMetricHistoryEntry, error) {
	user, err := us.userRepo.FindByUsername(ctx, us.exec, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	accessible, err := canAccessPrivateUser(ctx, us.exec, us.friendshipRepo, user, requester)
	if err != nil {
		return nil, err
	}
	if !accessible {
		return nil, ErrUserPrivate
	}
	if user.PlayerID == nil {
		return nil, ErrPlayerMetricHistoryNotFound
	}
	entries, err := us.historyQuery.FindTimeline(ctx, *user.PlayerID)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, ErrPlayerMetricHistoryNotFound
	}
	return entries, nil
}
