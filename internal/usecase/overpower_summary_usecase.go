package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dtoapiinternal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

// OverpowerSummaryUsecase は本人向け OVER POWER 集計を返します。
type OverpowerSummaryUsecase interface {
	Get(ctx context.Context, user *entity.User) (*dtoapiinternal.OverpowerSummaryResponse, error)
}
