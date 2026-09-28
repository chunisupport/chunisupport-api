package usecase

import (
	"context"
	"errors"
	"slices"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// lockRequesterAndTarget はリクエスト元と対象のユーザー行をロックして取得します。
// 管理操作を実行時点のリクエスト元の権限で判定するため、トランザクション内で呼び出します。
// 同時に互いを操作するリクエストでデッドロックしないよう、常にIDの昇順でロックします。
func lockRequesterAndTarget(ctx context.Context, tx repository.Executor, userRepo repository.UserRepository, requesterID int, targetID int) (*entity.User, *entity.User, error) {
	locked := make(map[int]*entity.User, 2)
	ids := []int{min(requesterID, targetID), max(requesterID, targetID)}
	for _, id := range slices.Compact(ids) {
		user, err := userRepo.FindByIDForUpdate(ctx, tx, id)
		if errors.Is(err, repository.ErrUserNotFound) {
			// 認可確認後にリクエスト元が退会した場合は権限がないものとして扱います。
			if id == requesterID {
				return nil, nil, ErrAdminRequired
			}
			return nil, nil, ErrUserNotFound
		}
		if err != nil {
			return nil, nil, err
		}
		locked[id] = user
	}
	return locked[requesterID], locked[targetID], nil
}
