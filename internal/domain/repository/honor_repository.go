package repository

import (
	"context"
)

// HonorAssignment はプレイヤーに称号を割り当てるための構造体です。
type HonorAssignment struct {
	PlayerID int
	HonorID  int
	Slot     int // 1=上段, 2=中段, 3=下段
}

// HonorRepository は称号に関する永続化を扱うリポジトリです。
type HonorRepository interface {
	// EnsureHonor は称号を登録または既存のIDを取得します。
	// 称号が存在しなければ登録され、存在すれば既存のIDが返されます。
	// imageURL が指定されている場合は更新します。
	EnsureHonor(ctx context.Context, exec Executor, title string, honorTypeID int, imageURL *string) (int, error)

	// DeletePlayerHonors はプレイヤーの称号割り当てを全て削除します。
	DeletePlayerHonors(ctx context.Context, exec Executor, playerID int) error

	// BulkAssignHonors はプレイヤーに称号を一括で割り当てます。
	BulkAssignHonors(ctx context.Context, exec Executor, assignments []HonorAssignment) error
}
