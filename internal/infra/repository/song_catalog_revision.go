package repository

import (
	"context"
	"fmt"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// TouchSongCatalogRevision は楽曲カタログの変更世代を1つ進めます。
// songs / charts / worldsend_charts を変更するトランザクション内で、変更後に呼び出してください。
// 同じトランザクションでコミットされるため、楽曲データの変更と世代の更新が別々に見えることはありません。
// 行ロックで楽曲データの書き込みが直列化されますが、編集・バッチとも頻度が低いため許容します。
func TouchSongCatalogRevision(ctx context.Context, exec domainrepo.Executor) error {
	result, err := exec.ExecContext(ctx, `UPDATE song_catalog_revision SET revision = revision + 1 WHERE id = 1`)
	if err != nil {
		return fmt.Errorf("%w: touch song catalog revision: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: touch song catalog revision: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	if affected != 1 {
		return fmt.Errorf("%w: song catalog revision row is missing", domainrepo.ErrRepositoryOperationFailed)
	}
	return nil
}

// findSongCatalogRevision は楽曲カタログの現在の変更世代を返します。
func findSongCatalogRevision(ctx context.Context, exec domainrepo.Executor) (uint64, error) {
	var revision uint64
	if err := exec.GetContext(ctx, &revision, `SELECT revision FROM song_catalog_revision WHERE id = 1`); err != nil {
		return 0, fmt.Errorf("%w: find song catalog revision: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return revision, nil
}
