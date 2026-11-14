package usecase

import (
	"context"
	"log/slog"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// refreshSongCatalog は楽曲データの保存成功後に、共有の楽曲カタログを同期的に再構築します。
// 応答を返す前に差し替えることで、編集直後の参照（未プレイ補完や updated-at）へ保存結果を反映します。
//
// 保存はコミット済みのため、再構築に失敗しても呼び出し元へはエラーを返しません。
// 失敗時は公開中の世代を維持してログに残し、定期的な世代確認で再試行します。
// 編集・検証・更新競合の判定はキャッシュではなくDBを正とするため、古い世代が残っても保存結果は損なわれません。
// リクエストが切断されても再構築を完了させるため、キャンセルを引き継がず上限時間だけを持つコンテキストを使います。
func refreshSongCatalog(ctx context.Context, catalog repository.SongCatalogProvider) {
	if catalog == nil {
		return
	}
	refreshCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), info.SongCatalogRefreshTimeout)
	defer cancel()
	if err := catalog.Refresh(refreshCtx); err != nil {
		slog.ErrorContext(ctx, "failed to refresh song catalog after save", "error", err)
	}
}
