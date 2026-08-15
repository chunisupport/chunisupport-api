package repository

import (
	"context"
	"time"
)

// AdminUserStatistics は管理者画面に表示するユーザー集計値です。
type AdminUserStatistics struct {
	TotalUsers                 int
	UsersWithPlayerData        int
	ActivePlayerDataLast30Days int
}

// AdminUserStatisticsQueryService は管理者向けユーザー集計の読み取りを提供します。
type AdminUserStatisticsQueryService interface {
	Get(ctx context.Context, activeSince time.Time) (AdminUserStatistics, error)
}
