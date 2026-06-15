package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// RecordFilterRepository はユーザーが保存する譜面フィルタの永続化を扱います。
type RecordFilterRepository interface {
	ListByUserID(ctx context.Context, userID int) ([]*entity.RecordFilter, error)
	// FindByIDAndUserID は対象が存在しない場合に ErrRecordFilterNotFound を返します。
	FindByIDAndUserID(ctx context.Context, id []byte, userID int) (*entity.RecordFilter, error)
	Save(ctx context.Context, filter *entity.RecordFilter) error
	// DeleteByIDAndUserID は対象が存在しない場合に ErrRecordFilterNotFound を返します。
	DeleteByIDAndUserID(ctx context.Context, id []byte, userID int) error
	CountByUserID(ctx context.Context, userID int) (int, error)
}
