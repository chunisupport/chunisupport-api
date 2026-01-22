package usecase

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/dto"
)

// PlayerUsecase はプレイヤーに関するビジネスロジックを扱うユースケースです。
type PlayerUsecase interface {
	// CreatePlayer は新しいプレイヤーを作成します。
	CreatePlayer(ctx context.Context, name string) (*dto.PlayerDTO, error)
	// GetPlayerByID はIDでプレイヤーを取得します。
	GetPlayerByID(ctx context.Context, id int) (*dto.PlayerDTO, error)
}
