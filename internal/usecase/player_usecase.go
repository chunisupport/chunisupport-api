package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// PlayerUsecase はプレイヤーに関するビジネスロジックを扱うユースケースです。
type PlayerUsecase interface {
	// CreatePlayer は新しいプレイヤーを作成します。
	CreatePlayer(ctx context.Context, userID int, name string) (*entity.Player, error)
}
