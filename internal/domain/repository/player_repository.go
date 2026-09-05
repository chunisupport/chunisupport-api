package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

type PlayerWithHonors struct {
	Player *entity.Player
	Honors []*entity.PlayerHonor
}

// PlayerRepository はプレイヤーに関する永続化を扱うリポジトリです。
type PlayerRepository interface {
	// FindByID はIDでプレイヤーを検索します。対象が存在しない場合は ErrPlayerNotFound を返します。
	FindByID(ctx context.Context, exec Executor, id int) (*entity.Player, error)
	// FindByIDForUpdate は既存集約全体をロックし、同一トランザクション内の変更とSaveに使用します。
	FindByIDForUpdate(ctx context.Context, exec Executor, id int) (*entity.Player, error)
	// FindByIDWithHonors はIDでプレイヤーと称号情報をまとめて検索します。対象が存在しない場合は ErrPlayerNotFound を返します。
	FindByIDWithHonors(ctx context.Context, exec Executor, id int) (*PlayerWithHonors, error)
	// FindByUserID はユーザーIDでプレイヤーを検索します。見つからない場合は(nil, nil)を返します。
	FindByUserID(ctx context.Context, exec Executor, userID int) (*entity.Player, error)
	// FindByUserIDForUpdate はユーザーIDでプレイヤーを検索し、更新用に行ロックします。
	FindByUserIDForUpdate(ctx context.Context, exec Executor, userID int) (*entity.Player, error)
	// FindHonorsByPlayerID はプレイヤーIDで称号情報を取得します。スロット順（1,2,3）でソートされます。
	FindHonorsByPlayerID(ctx context.Context, exec Executor, playerID int) ([]*entity.PlayerHonor, error)
	// Save はプロフィール・公式指標・計算指標・取得日時をまとめて保存します。
	// 既存集約は更新用検索で取得し、関連レコードの読み書きからSaveまで同じトランザクションでロックを保持します。
	// 新規集約は所有ユーザーのロック、またはuser_idの一意制約で重複を防ぎます。
	// ID・UserID・CreatedAtは作成後不変、OverpowerPercentは永続化しない派生値です。
	// INSERT時は player が user_id や player_name、player_level など必須カラムを保持している前提です。
	// INSERTの場合、playerのIDフィールドが更新されます。
	Save(ctx context.Context, exec Executor, player *entity.Player) error
	// DeleteByUserID はユーザーに紐づくプレイヤーを削除します。関連データはON DELETE CASCADEで削除されます。
	DeleteByUserID(ctx context.Context, exec Executor, userID int) error
}
