package repository

import (
	"context"
	"testing"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// クライアント切断をエラーハンドラーが 499 として判定できるよう、
// リポジトリ操作エラーが原因エラー (context.Canceled) を保持していることを検証します。
func TestRepositoryOperationError_原因エラーを保持する(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	tests := []struct {
		name string
		// When: キャンセル済みコンテキストで操作する
		run func(ctx context.Context) error
	}{
		{
			name: "ベスト枠ランキングの対象人数取得",
			run: func(ctx context.Context) error {
				_, err := findEligiblePlayerCount(ctx, db, 1)
				return err
			},
		},
		{
			name: "ベスト枠ランキングの統計行取得",
			run: func(ctx context.Context) error {
				_, err := findBestSlotRankingRows(ctx, db, 1)
				return err
			},
		},
		{
			name: "ベスト枠ランキングの譜面取得",
			run: func(ctx context.Context) error {
				_, err := findBestSlotRankingCharts(ctx, db)
				return err
			},
		},
		{
			name: "譜面統計エクスポートのトランザクション開始",
			run: func(ctx context.Context) error {
				_, err := NewChartStatsExportQueryService(db).Get(ctx)
				return err
			},
		},
		{
			name: "譜面統計エクスポートの通常譜面取得",
			run: func(ctx context.Context) error {
				_, err := NewChartStatsExportQueryService(db).getCharts(ctx, db)
				return err
			},
		},
		{
			name: "譜面統計エクスポートのWORLD'S END譜面取得",
			run: func(ctx context.Context) error {
				_, err := NewChartStatsExportQueryService(db).getWorldsendCharts(ctx, db)
				return err
			},
		},
		{
			name: "管理者向け譜面ランキング",
			run: func(ctx context.Context) error {
				return wrapAdminChartRankingQueryError("op", ctx.Err())
			},
		},
		{
			name: "フレンドスコア比較",
			run: func(ctx context.Context) error {
				return wrapFriendScoreComparisonQueryError("op", ctx.Err())
			},
		},
		{
			name: "フレンド譜面ランキング",
			run: func(ctx context.Context) error {
				return wrapFriendChartRankingQueryError("op", ctx.Err())
			},
		},
		{
			name: "お気に入り楽曲",
			run: func(ctx context.Context) error {
				return wrapPlayerFavoriteSongRepositoryError("op", ctx.Err())
			},
		},
		{
			name: "フレンド関係",
			run: func(ctx context.Context) error {
				return wrapFriendshipRepositoryError("op", ctx.Err())
			},
		},
		{
			name: "ロック楽曲",
			run: func(ctx context.Context) error {
				return wrapPlayerLockedSongRepositoryError("op", ctx.Err())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			// When
			err := tt.run(ctx)

			// Then
			assert.ErrorIs(t, err, domainrepo.ErrRepositoryOperationFailed)
			assert.ErrorIs(t, err, context.Canceled)
		})
	}
}
