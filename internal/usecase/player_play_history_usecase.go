package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// PlayerPlayHistoryOutput は管理画面へ返すプレイ履歴の表示データです。
type PlayerPlayHistoryOutput struct {
	Title      *string
	Difficulty *string
	Score      int
	ClearLamp  *string
	ComboLamp  *string
	FullChain  *string
	PlayedAt   time.Time
}

// PlayerPlayHistoryUsecase は対象ユーザーのプレイ履歴取得を定義します。
type PlayerPlayHistoryUsecase interface {
	// Get は閲覧可能なユーザーの直近プレイ履歴を受信順で返します。
	Get(ctx context.Context, username string, requester *entity.User) ([]PlayerPlayHistoryOutput, error)
}

type playerPlayHistoryUsecase struct {
	db           repository.Executor
	userRepo     repository.UserRepository
	friendships  repository.FriendshipRepository
	queryService repository.PlayerPlayHistoryQueryService
}

// NewPlayerPlayHistoryUsecase はプレイ履歴取得ユースケースを生成します。
func NewPlayerPlayHistoryUsecase(
	db repository.Executor,
	userRepo repository.UserRepository,
	friendships repository.FriendshipRepository,
	queryService repository.PlayerPlayHistoryQueryService,
) PlayerPlayHistoryUsecase {
	return &playerPlayHistoryUsecase{
		db: db, userRepo: userRepo, friendships: friendships, queryService: queryService,
	}
}

// Get は対象ユーザーの公開範囲を既存のプロフィール取得と同じ規則で確認します。
func (u *playerPlayHistoryUsecase) Get(ctx context.Context, username string, requester *entity.User) ([]PlayerPlayHistoryOutput, error) {
	user, err := getAccessibleUser(ctx, u.db, u.userRepo, u.friendships, username, requester)
	if err != nil {
		return nil, err
	}
	if user.PlayerID == nil {
		return []PlayerPlayHistoryOutput{}, nil
	}

	entries, err := u.queryService.FindByPlayerID(ctx, u.db, *user.PlayerID)
	if err != nil {
		return nil, err
	}
	result := make([]PlayerPlayHistoryOutput, 0, len(entries))
	for _, entry := range entries {
		result = append(result, PlayerPlayHistoryOutput{
			Title: entry.Title, Difficulty: entry.Difficulty, Score: entry.Score,
			ClearLamp: entry.ClearLamp, ComboLamp: entry.ComboLamp, FullChain: entry.FullChain,
			PlayedAt: entry.PlayedAt,
		})
	}
	return result, nil
}
