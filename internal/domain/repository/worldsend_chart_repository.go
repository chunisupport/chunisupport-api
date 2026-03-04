package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// WorldsendSongWithChart は WORLD'S END 楽曲とその譜面情報を保持する構造体です。
// WORLD'S END は1曲1譜面が保証されています。
type WorldsendSongWithChart struct {
	Song  *entity.Song
	Chart *entity.WorldsendChart
}

// WorldsendChartRepository は WORLD'S END 譜面に関する永続化を扱うリポジトリです。
type WorldsendChartRepository interface {
	// FindAll は全 WORLD'S END 楽曲を譜面情報付きで取得します。
	// includeDeleted が false の場合、削除済み楽曲は除外されます。
	FindAll(ctx context.Context, exec Executor, includeDeleted bool) ([]*WorldsendSongWithChart, error)

	// FindByDisplayID は指定された DisplayID の WORLD'S END 楽曲を取得します。
	// 削除済み楽曲も取得します。
	FindByDisplayID(ctx context.Context, exec Executor, displayID string) (*WorldsendSongWithChart, error)

	// SaveSong は WORLD'S END 楽曲エンティティの現在の状態を永続化します。
	// 対象が存在しない場合は ErrSongNotFound を返します。
	SaveSong(ctx context.Context, exec Executor, song *entity.Song) error

	// UpdateSongs は WORLD'S END 楽曲および譜面情報を一括更新します。
	UpdateSongs(ctx context.Context, exec Executor, songs []*entity.Song, charts []*entity.WorldsendChart) error
}
