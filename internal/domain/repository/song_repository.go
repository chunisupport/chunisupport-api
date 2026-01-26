package repository

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
)

// SongWithCharts は楽曲とその譜面情報を保持する構造体です。
type SongWithCharts struct {
	Song   *entity.Song
	Charts []*entity.Chart
}

// SongRepository は楽曲に関する永続化を扱うリポジトリです。
// 他のリポジトリと同様に、全メソッドでExecutorを受け取り、
// UseCase層からのトランザクション制御を可能にします。
type SongRepository interface {
	// FindAllExcludingWorldsend はWORLD'S END以外の全楽曲を取得します。
	// includeDeletedがfalseの場合、削除済み楽曲は除外されます。
	// 各楽曲には関連する譜面情報が含まれます。
	FindAllExcludingWorldsend(ctx context.Context, exec Executor, includeDeleted bool) ([]*SongWithCharts, error)

	// FindByDisplayID は指定されたDisplayIDの楽曲を取得します。
	// 削除済み楽曲も取得します。
	FindByDisplayID(ctx context.Context, exec Executor, displayID string) (*SongWithCharts, error)

	// FindByDisplayIDs は指定されたDisplayIDのリストに該当する楽曲を取得します。
	// 存在しないDisplayIDがある場合でもエラーにはせず、存在する楽曲のみを返します。
	FindByDisplayIDs(ctx context.Context, exec Executor, displayIDs []string) ([]*entity.Song, error)

	// DeleteSong は指定されたDisplayIDの楽曲を論理削除します。
	DeleteSong(ctx context.Context, exec Executor, displayID string) error

	// RestoreSong は指定されたDisplayIDの楽曲を復活させます。
	RestoreSong(ctx context.Context, exec Executor, displayID string) error

	// UpdateSongs は楽曲および譜面情報を一括更新します。
	// トランザクション管理はUseCase層（TransactionManager経由）で行います。
	// 存在しない楽曲・譜面がある場合はエラーを返します。
	UpdateSongs(ctx context.Context, exec Executor, songsWithCharts []*SongWithCharts) error
}
