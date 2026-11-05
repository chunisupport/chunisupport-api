package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
)

type xAccountRepository struct{}

// NewXAccountRepository は新しい XAccountRepository を生成します。
func NewXAccountRepository() repository.XAccountRepository {
	return &xAccountRepository{}
}

func (r *xAccountRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.XAccount, error) {
	var model models.XAccountModel
	query := `SELECT user_id, x_user_id, x_username FROM user_x_accounts WHERE user_id = ?`
	if err := exec.GetContext(ctx, &model, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: find x account: %w", repository.ErrRepositoryOperationFailed, err)
	}
	account, err := model.ToEntity()
	if err != nil {
		return nil, fmt.Errorf("%w: restore x account: %w", repository.ErrRepositoryOperationFailed, err)
	}
	return account, nil
}

// Save は user_id を主キーとして行を作成または更新します。
// 公開設定（is_public）は利用者が選んだ値のため、X から取得し直しても上書きしません。
// ただし公開の同意は連携していた X アカウントに対するものなので、x_user_id が変わった場合は非公開に戻します。
// MySQL は SET 句を左から評価するため、x_user_id を書き換える前に is_public を判定します。
// AUTO_INCREMENT 列を持たないため、ON DUPLICATE KEY UPDATE で採番を浪費することはありません。
func (r *xAccountRepository) Save(ctx context.Context, exec repository.Executor, account *entity.XAccount) error {
	query := `
INSERT INTO user_x_accounts (user_id, x_user_id, x_username, updated_at)
VALUES (?, ?, ?, CURRENT_TIMESTAMP)
ON DUPLICATE KEY UPDATE
	is_public = IF(x_user_id = VALUES(x_user_id), is_public, FALSE),
	x_user_id = VALUES(x_user_id),
	x_username = VALUES(x_username),
	updated_at = CURRENT_TIMESTAMP
`
	if _, err := exec.ExecContext(ctx, query, account.UserID(), account.XUserID(), account.Username()); err != nil {
		return fmt.Errorf("%w: save x account: %w", repository.ErrRepositoryOperationFailed, err)
	}
	return nil
}
