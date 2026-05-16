package usecase

import (
	"context"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/dto"
)

// playerUsecase は PlayerUsecase の実装です。
type playerUsecase struct {
	db         repository.Executor
	playerRepo repository.PlayerRepository
}

// NewPlayerUsecase は新しいPlayerUsecaseを生成します。
func NewPlayerUsecase(db repository.Executor, playerRepo repository.PlayerRepository) PlayerUsecase {
	return &playerUsecase{
		db:         db,
		playerRepo: playerRepo,
	}
}

// CreatePlayer は新しいプレイヤーを作成し、永続化後のDTOを返します。
func (us *playerUsecase) CreatePlayer(ctx context.Context, userID int, name string) (*dto.PlayerDTO, error) {
	// 値オブジェクトを生成
	playerNameVO, err := playername.NewPlayerName(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidPlayerName, err)
	}

	// 新しいプレイヤーエンティティを作成
	player := entity.NewPlayer(userID, playerNameVO)

	// プレイヤーを永続化
	if err := us.playerRepo.Save(ctx, us.db, player); err != nil {
		return nil, err
	}

	return dto.ToPlayerDTO(player), nil
}
