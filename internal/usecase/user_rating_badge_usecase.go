package usecase

import (
	"context"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// GetPublicOfficialRating はバッジ表示に不要な称号やOVER POWERの計算を避けて公式RATINGだけを取得します。
func (s *userUsecase) GetPublicOfficialRating(ctx context.Context, username string) (*float64, error) {
	player, err := s.GetPublicBadgePlayer(ctx, username)
	if err != nil || player == nil {
		return nil, err
	}
	return &player.OfficialRating, nil
}

// GetPublicBadgePlayer は称号取得や指標の再計算を避け、匿名閲覧できるプレイヤーの保存値だけを取得します。
func (s *userUsecase) GetPublicBadgePlayer(ctx context.Context, username string) (*entity.Player, error) {
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
	return player, err
}
