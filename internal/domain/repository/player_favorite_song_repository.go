package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// PlayerFavoriteSongRepository はプレイヤーのお気に入り楽曲に関する永続化を扱うリポジトリです。
type PlayerFavoriteSongRepository interface {
	// CountByPlayerID はプレイヤーのお気に入り楽曲数を返します。
	CountByPlayerID(ctx context.Context, exec Executor, playerID int) (int, error)

	// Exists は指定されたプレイヤーと楽曲のお気に入りが存在するかを返します。
	Exists(ctx context.Context, exec Executor, playerID int, songID int) (bool, error)

	// Save はお気に入り楽曲を保存します。
	// 既に存在する場合は何も変更しません（冪等）。
	Save(ctx context.Context, exec Executor, favorite *entity.PlayerFavoriteSong) error

	// Delete はお気に入り楽曲を削除します。
	// 存在しない場合もエラーにはなりません（冪等）。
	Delete(ctx context.Context, exec Executor, playerID int, songID int) error

	// DeleteBySongID は指定された楽曲のお気に入りをすべて削除します。
	DeleteBySongID(ctx context.Context, exec Executor, songID int) error
}
