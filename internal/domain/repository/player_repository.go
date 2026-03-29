package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// PlayerHonor はプレイヤーの称号情報を表す構造体です。
type PlayerHonor struct {
	Slot     int     // 称号スロット: 1=上段, 2=中段, 3=下段
	Name     string  // 称号名
	TypeName string  // 称号タイプ名 (normal, copper, silver, gold, platina, rainbow, etc.)
	ImageURL *string // 称号画像URL
}

type PlayerWithHonors struct {
	Player *entity.Player
	Honors []*PlayerHonor
}

// PlayerRepository はプレイヤーに関する永続化を扱うリポジトリです。
type PlayerRepository interface {
	// FindByID はIDでプレイヤーを検索します。
	FindByID(ctx context.Context, exec Executor, id int) (*entity.Player, error)
	// FindByIDWithHonors はIDでプレイヤーと称号情報をまとめて検索します。
	FindByIDWithHonors(ctx context.Context, exec Executor, id int) (*PlayerWithHonors, error)
	// FindByUserID はユーザーIDでプレイヤーを検索します。見つからない場合は(nil, nil)を返します。
	FindByUserID(ctx context.Context, exec Executor, userID int) (*entity.Player, error)
	// FindHonorsByPlayerID はプレイヤーIDで称号情報を取得します。スロット順（1,2,3）でソートされます。
	FindHonorsByPlayerID(ctx context.Context, exec Executor, playerID int) ([]*PlayerHonor, error)
	// UpdateCalculatedRatings はプレイヤーの計算されたレーティング情報を更新します。
	UpdateCalculatedRatings(ctx context.Context, exec Executor, playerID int, calculatedRating, bestAverage, newAverage float64) error
	// Save はプレイヤー情報を保存します（ID=0の場合はINSERT、それ以外はUPDATE）。
	// INSERT時は player が user_id や player_name、player_level など必須カラムを保持している前提です。
	// INSERTの場合、playerのIDフィールドが更新されます。
	Save(ctx context.Context, exec Executor, player *entity.Player) error
	// DeleteByUserID はユーザーに紐づくプレイヤーを削除します。関連データはON DELETE CASCADEで削除されます。
	DeleteByUserID(ctx context.Context, exec Executor, userID int) error
}
