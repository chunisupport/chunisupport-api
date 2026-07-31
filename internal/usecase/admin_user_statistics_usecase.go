package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// AdminUserStatisticsOutput は管理者画面に返すユーザー集計値です。
type AdminUserStatisticsOutput struct {
	TotalUsers                 int
	UsersWithPlayerData        int
	ActivePlayerDataLast30Days int
}

// AdminUserStatisticsUsecase は管理者向けユーザー集計を取得します。
type AdminUserStatisticsUsecase interface {
	Get(ctx context.Context) (AdminUserStatisticsOutput, error)
}

type adminUserStatisticsUsecase struct {
	queryService repository.AdminUserStatisticsQueryService
	now          func() time.Time
}

// NewAdminUserStatisticsUsecase は管理者向けユーザー集計ユースケースを生成します。
func NewAdminUserStatisticsUsecase(queryService repository.AdminUserStatisticsQueryService) AdminUserStatisticsUsecase {
	return newAdminUserStatisticsUsecase(queryService, time.Now)
}

func newAdminUserStatisticsUsecase(queryService repository.AdminUserStatisticsQueryService, now func() time.Time) AdminUserStatisticsUsecase {
	return &adminUserStatisticsUsecase{queryService: queryService, now: now}
}

// Get は現在時刻から直近30日間を対象としてユーザー集計値を取得します。
func (u *adminUserStatisticsUsecase) Get(ctx context.Context) (AdminUserStatisticsOutput, error) {
	activeSince := u.now().Add(-info.AdminUserStatisticsActivePeriod)
	statistics, err := u.queryService.Get(ctx, activeSince)
	if err != nil {
		slog.Error("failed to fetch admin user statistics", "error", err)
		return AdminUserStatisticsOutput{}, err
	}

	return AdminUserStatisticsOutput{
		TotalUsers:                 statistics.TotalUsers,
		UsersWithPlayerData:        statistics.UsersWithPlayerData,
		ActivePlayerDataLast30Days: statistics.ActivePlayerDataLast30Days,
	}, nil
}
