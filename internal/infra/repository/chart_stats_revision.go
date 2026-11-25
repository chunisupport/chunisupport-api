package repository

import (
	"context"
	"fmt"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// TouchChartStatsRevision は譜面統計の変更世代を1つ進めます。
// 統計テーブルを入れ替えるトランザクション内で呼び出してください。
// 同じトランザクションでコミットされるため、統計データの更新と世代の更新が別々に見えることはありません。
func TouchChartStatsRevision(ctx context.Context, exec domainrepo.Executor) error {
	result, err := exec.ExecContext(ctx, `UPDATE chart_stats_revision SET revision = revision + 1 WHERE id = 1`)
	if err != nil {
		return fmt.Errorf("%w: touch chart stats revision: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: touch chart stats revision: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	if affected != 1 {
		return fmt.Errorf("%w: chart stats revision row is missing", domainrepo.ErrRepositoryOperationFailed)
	}
	return nil
}

// findChartStatsRevision は譜面統計の現在の変更世代を返します。
func findChartStatsRevision(ctx context.Context, exec domainrepo.Executor) (uint64, error) {
	var revision uint64
	if err := exec.GetContext(ctx, &revision, `SELECT revision FROM chart_stats_revision WHERE id = 1`); err != nil {
		return 0, fmt.Errorf("%w: find chart stats revision: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return revision, nil
}
