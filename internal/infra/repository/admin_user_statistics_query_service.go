package repository

import (
	"context"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
)

type adminUserStatisticsQueryService struct {
	db *sqlx.DB
}

// NewAdminUserStatisticsQueryService は管理者向けユーザー集計の読み取りサービスを生成します。
func NewAdminUserStatisticsQueryService(db *sqlx.DB) domainrepo.AdminUserStatisticsQueryService {
	return &adminUserStatisticsQueryService{db: db}
}

// Get はユーザー数、プレイヤーデータ連携数、期間内のデータ取得数を1回のDBアクセスで集計します。
func (s *adminUserStatisticsQueryService) Get(ctx context.Context, activeSince time.Time) (domainrepo.AdminUserStatistics, error) {
	const query = `
		SELECT
			(SELECT COUNT(*) FROM users) AS total_users,
			(SELECT COUNT(*) FROM users WHERE player_id IS NOT NULL) AS users_with_player_data,
			(SELECT COUNT(*) FROM players WHERE data_collected_at >= ?) AS active_player_data_last_30_days
	`

	var row struct {
		TotalUsers                 int `db:"total_users"`
		UsersWithPlayerData        int `db:"users_with_player_data"`
		ActivePlayerDataLast30Days int `db:"active_player_data_last_30_days"`
	}
	if err := s.db.GetContext(ctx, &row, query, activeSince); err != nil {
		return domainrepo.AdminUserStatistics{}, err
	}

	return domainrepo.AdminUserStatistics{
		TotalUsers:                 row.TotalUsers,
		UsersWithPlayerData:        row.UsersWithPlayerData,
		ActivePlayerDataLast30Days: row.ActivePlayerDataLast30Days,
	}, nil
}
