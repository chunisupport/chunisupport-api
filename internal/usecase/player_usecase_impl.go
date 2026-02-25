package usecase

import (
	"context"
	"errors"
	"log/slog"

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

// NewPlayerService は新しいPlayerUsecaseを生成します。
func NewPlayerService(db repository.Executor, playerRepo repository.PlayerRepository) PlayerUsecase {
	return &playerUsecase{
		db:         db,
		playerRepo: playerRepo,
	}
}

// CreatePlayer は新しいプレイヤーを作成し、永続化後のDTOを返します。
func (us *playerUsecase) CreatePlayer(ctx context.Context, name string) (*dto.PlayerDTO, error) {
	// 値オブジェクトを生成
	playerNameVO, err := playername.NewPlayerName(name)
	if err != nil {
		return nil, err
	}

	// 新しいプレイヤーエンティティを作成
	player := &entity.Player{
		Name: playerNameVO,
	}

	// プレイヤーを永続化
	if err := us.playerRepo.Save(ctx, us.db, player); err != nil {
		return nil, err
	}

	return dto.ToPlayerDTO(player), nil
}

// GetPlayerByID はIDでプレイヤーを取得し、DTOに変換して返します。
func (us *playerUsecase) GetPlayerByID(ctx context.Context, id int) (*dto.PlayerDTO, error) {
	player, err := us.playerRepo.FindByID(ctx, us.db, id)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Warn("failed to find player by ID due to context canceled", "player_id", id, "error", err)
		} else {
			slog.Error("failed to find player by ID", "player_id", id, "error", err)
		}
		return nil, err
	}

	playerDTO := dto.ToPlayerDTO(player)

	// 称号情報を取得してDTOに設定
	honors, err := us.playerRepo.FindHonorsByPlayerID(ctx, us.db, id)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Warn("failed to find honors by player ID due to context canceled", "player_id", id, "error", err)
		} else {
			slog.Error("failed to find honors by player ID", "player_id", id, "error", err)
		}
		return nil, err
	}

	for _, h := range honors {
		playerDTO.Honors = append(playerDTO.Honors, &dto.HonorDTO{
			Slot:     h.Slot,
			Name:     h.Name,
			TypeName: h.TypeName,
			ImageURL: h.ImageURL,
		})
	}

	return playerDTO, nil
}
