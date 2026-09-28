package usecase

import (
	"context"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// GetPublicOfficialRating はバッジ表示に不要な称号やOVER POWERの計算を避けて公式RATINGだけを取得します。
func (s *userUsecase) GetPublicOfficialRating(ctx context.Context, username string) (*float64, error) {
	user, err := s.getAccessibleUser(ctx, username, nil)
	if err != nil {
		return nil, err
	}
	if !user.HasLinkedPlayer() {
		return nil, nil
	}

	player, err := s.playerRepo.FindByID(ctx, s.db, *user.PlayerID)
	if errors.Is(err, repository.ErrPlayerNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if player == nil {
		return nil, nil
	}
	return &player.OfficialRating, nil
}
