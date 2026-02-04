package repository

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
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

	// DeleteSong は指定された DisplayID の WORLD'S END 楽曲を論理削除します。
	DeleteSong(ctx context.Context, exec Executor, displayID string) error

	// RestoreSong は指定された DisplayID の WORLD'S END 楽曲を復活させます。
	RestoreSong(ctx context.Context, exec Executor, displayID string) error

	// UpdateSongs は WORLD'S END 楽曲および譜面情報を一括更新します。
	UpdateSongs(ctx context.Context, exec Executor, songs []*entity.Song, charts []*entity.WorldsendChart) error
}
