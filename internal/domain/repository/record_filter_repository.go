package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// RecordFilterRepository はユーザーが保存する譜面フィルタの永続化を扱います。
type RecordFilterRepository interface {
	ListByUserID(ctx context.Context, exec Executor, userID int) ([]*entity.RecordFilter, error)
	// FindByIDAndUserID は対象が存在しない場合に ErrRecordFilterNotFound を返します。
	FindByIDAndUserID(ctx context.Context, exec Executor, id []byte, userID int) (*entity.RecordFilter, error)
	// Create は新しいフィルタを保存します。
	Create(ctx context.Context, exec Executor, filter *entity.RecordFilter) error
	// Update は既存のフィルタを更新します。
	// 取得後に削除されたフィルタを作り直さないよう、対象が存在しない場合は ErrRecordFilterNotFound を返します。
	Update(ctx context.Context, exec Executor, filter *entity.RecordFilter) error
	// DeleteByIDAndUserID は対象が存在しない場合に ErrRecordFilterNotFound を返します。
	DeleteByIDAndUserID(ctx context.Context, exec Executor, id []byte, userID int) error
	CountByUserID(ctx context.Context, exec Executor, userID int) (int, error)
}
