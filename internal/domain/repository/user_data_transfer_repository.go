package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// UserDataTransferRepository は、ユーザーに属する移行対象データを一つの集約として保存・復元します。
type UserDataTransferRepository interface {
	ExportSnapshot(ctx context.Context, userID int) (*entity.UserDataTransferSnapshot, error)
	FindUnresolvedReferences(ctx context.Context, snapshot *entity.UserDataTransferSnapshot) ([]string, error)
	IsDestinationEmpty(ctx context.Context, userID int) (bool, error)
	ImportSnapshot(ctx context.Context, userID int, snapshot *entity.UserDataTransferSnapshot) (int, error)
}
