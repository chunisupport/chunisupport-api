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
	// FindByIDWithHonors はIDでプレイヤーと称号情報をまとめて検索します。対象が存在しない場合は ErrPlayerNotFound を返します。
	FindByIDWithHonors(ctx context.Context, exec Executor, id int) (*PlayerWithHonors, error)
	// FindByUserID はユーザーIDでプレイヤーを検索します。見つからない場合は(nil, nil)を返します。
	FindByUserID(ctx context.Context, exec Executor, userID int) (*entity.Player, error)
	// FindHonorsByPlayerID はプレイヤーIDで称号情報を取得します。スロット順（1,2,3）でソートされます。
	FindHonorsByPlayerID(ctx context.Context, exec Executor, playerID int) ([]*entity.PlayerHonor, error)
	// UpdateCalculatedRatings はプレイヤーの計算されたレーティング情報を更新します。
	UpdateCalculatedRatings(ctx context.Context, exec Executor, playerID int, calculatedRating, bestAverage, newAverage float64) error
	// Save はプレイヤー情報を保存します（ID=0の場合はINSERT、それ以外はUPDATE）。
	// INSERT時は player が user_id や player_name、player_level など必須カラムを保持している前提です。
	// INSERTの場合、playerのIDフィールドが更新されます。
	Save(ctx context.Context, exec Executor, player *entity.Player) error
	// DeleteByUserID はユーザーに紐づくプレイヤーを削除します。関連データはON DELETE CASCADEで削除されます。
	DeleteByUserID(ctx context.Context, exec Executor, userID int) error
}
