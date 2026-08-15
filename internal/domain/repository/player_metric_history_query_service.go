package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// PlayerMetricHistoryQueryService はプレイヤー公式指標のタイムラインを読み取ります。
type PlayerMetricHistoryQueryService interface {
	FindTimeline(ctx context.Context, playerID int) ([]entity.PlayerMetricHistoryEntry, error)
}
