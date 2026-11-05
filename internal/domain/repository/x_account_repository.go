package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// XAccountRepository はユーザーが連携している X アカウント情報の永続化を扱います。
type XAccountRepository interface {
	// FindByUserID は保存済みの X アカウント情報を返します。
	// 一度も取得していないことは通常の状態のため、存在しない場合はエラーではなく nil を返します。
	FindByUserID(ctx context.Context, exec Executor, userID int) (*entity.XAccount, error)
	// Save はユーザーごとに1件の X アカウント情報を作成または更新します。
	// 公開設定は維持し、X の数値ユーザーIDが変わった場合だけ非公開に戻します。
	Save(ctx context.Context, exec Executor, account *entity.XAccount) error
}
