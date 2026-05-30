package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// HonorAssignment はプレイヤーに称号を割り当てるための構造体です。
type HonorAssignment struct {
	PlayerID int
	HonorID  int
	Slot     int // 1=上段, 2=中段, 3=下段
}

// HonorRepository は称号に関する永続化を扱うリポジトリです。
type HonorRepository interface {
	// FindAll は称号をID昇順で全件取得します。
	FindAll(ctx context.Context, exec Executor) ([]*entity.Honor, error)

	// FindByID は指定IDの称号を取得します。
	FindByID(ctx context.Context, exec Executor, id int) (*entity.Honor, error)

	// Create は称号を新規登録します。
	Create(ctx context.Context, exec Executor, honor *entity.Honor) (*entity.Honor, error)

	// Save は称号を更新します。
	Save(ctx context.Context, exec Executor, honor *entity.Honor) error

	// Delete は称号を物理削除します。
	Delete(ctx context.Context, exec Executor, id int) error

	// EnsureHonor は称号を登録または既存のIDを取得します。
	// 称号が存在しなければ登録され、存在すれば既存のIDが返されます。
	// imageURL が指定されている場合は更新します。
	EnsureHonor(ctx context.Context, exec Executor, title string, honorTypeID int, imageURL *string) (int, error)

	// DeletePlayerHonors はプレイヤーの称号割り当てを全て削除します。
	DeletePlayerHonors(ctx context.Context, exec Executor, playerID int) error

	// BulkAssignHonors はプレイヤーに称号を一括で割り当てます。
	BulkAssignHonors(ctx context.Context, exec Executor, assignments []HonorAssignment) error
}
