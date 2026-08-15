package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
)

// PlayerFavoriteSongUsecase はプレイヤーお気に入り楽曲のユースケースを定義します。
type PlayerFavoriteSongUsecase interface {
	// List は指定されたユーザーのお気に入り楽曲一覧を取得します。
	List(ctx context.Context, username string, requester *entity.User) ([]*PlayerFavoriteSongOutput, error)

	// Add は指定された楽曲をお気に入りに登録します。
	Add(ctx context.Context, userID int, displayID displayid.DisplayID) error

	// Remove は指定された楽曲をお気に入りから解除します。
	Remove(ctx context.Context, userID int, displayID displayid.DisplayID) error
}

// PlayerFavoriteSongOutput はお気に入り楽曲一覧の出力です。
type PlayerFavoriteSongOutput struct {
	DisplayID   string
	Title       string
	Jacket      *string
	FavoritedAt time.Time
}

// PlayerFavoriteSongReadModel はお気に入り楽曲一覧の読み取りモデルです。
type PlayerFavoriteSongReadModel struct {
	DisplayID   string
	Title       string
	Jacket      *string
	FavoritedAt time.Time
}

// PlayerFavoriteSongQueryService はお気に入り楽曲一覧を楽曲情報付きで取得するポートです。
type PlayerFavoriteSongQueryService interface {
	ListWithSongDetailsByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*PlayerFavoriteSongReadModel, error)
}

// PlayerFavoriteSongLocker はお気に入り登録時のプレイヤー行ロックを行うポートです。
type PlayerFavoriteSongLocker interface {
	LockPlayer(ctx context.Context, exec repository.Executor, playerID int) error
}
