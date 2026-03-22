package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// WorldsendSongWithChart は WORLD'S END 楽曲とその譜面情報を束ねる構造です。
// WORLD'S END は1楽曲1譜面のため、この形で表現します。
type WorldsendSongWithChart struct {
	Song  *entity.Song
	Chart *entity.WorldsendChart
}

// WorldsendUpdate は WORLD'S END 楽曲と譜面の更新情報を表現します。
// Chart が nil の場合は楽曲情報のみ更新します。
type WorldsendUpdate struct {
	Song  *entity.Song
	Chart *entity.WorldsendChart
}

// WorldsendChartRepository は WORLD'S END 譜面に関する永続化処理を定義するリポジトリです。
type WorldsendChartRepository interface {
	// FindAll は全 WORLD'S END 楽曲を譜面情報付きで取得します。
	// includeDeleted が false の場合、削除済み楽曲は除外されます。
	FindAll(ctx context.Context, exec Executor, includeDeleted bool) ([]*WorldsendSongWithChart, error)

	// GetLatestUpdatedAt は WORLD'S END 楽曲一覧全体の最終更新日時を返します。
	GetLatestUpdatedAt(ctx context.Context, exec Executor, includeDeleted bool) (*time.Time, error)

	// FindByDisplayID は指定された DisplayID の WORLD'S END 楽曲を取得します。
	// 削除済み楽曲も取得対象です。
	FindByDisplayID(ctx context.Context, exec Executor, displayID string) (*WorldsendSongWithChart, error)

	// SaveSong は WORLD'S END 楽曲エンティティの現在の状態を永続化します。
	// 対象楽曲が存在しない場合は ErrSongNotFound を返します。
	SaveSong(ctx context.Context, exec Executor, song *entity.Song) error

	// UpdateSongs は WORLD'S END 楽曲および譜面情報を一括更新します。
	UpdateSongs(ctx context.Context, exec Executor, updates []*WorldsendUpdate) error
}
