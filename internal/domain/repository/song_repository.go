package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// SongRepository は楽曲に関する永続化を扱うリポジトリです。
// 他のリポジトリと同様に、全メソッドでExecutorを受け取り、
// UseCase層からのトランザクション制御を可能にします。
type SongRepository interface {
	// FindAllExcludingWorldsend はWORLD'S END以外の全楽曲を取得します。
	// includeDeletedがfalseの場合、削除済み楽曲は除外されます。
	// 各楽曲には関連する譜面情報が含まれます。
	FindAllExcludingWorldsend(ctx context.Context, exec Executor, includeDeleted bool) ([]*entity.Song, error)

	// FindByDisplayID は指定されたDisplayIDの通常楽曲（WORLD'S END除く）を取得します。
	// 削除済み楽曲も取得します。
	// 各楽曲には関連する譜面情報が含まれます。
	FindByDisplayID(ctx context.Context, exec Executor, displayID string) (*entity.Song, error)

	// FindByOfficialIdx は指定された公式IDの通常楽曲を譜面情報付きで取得します。
	// 削除済み楽曲も取得します。
	FindByOfficialIdx(ctx context.Context, exec Executor, officialIdx string) (*entity.Song, error)

	// FindByDisplayIDs は指定されたDisplayIDのリストに該当する通常楽曲（WORLD'S END除く）を取得します。
	// 存在しないDisplayIDがある場合でもエラーにはせず、存在する楽曲のみを返します。
	// 各楽曲には関連する譜面情報が含まれます。
	FindByDisplayIDs(ctx context.Context, exec Executor, displayIDs []string) ([]*entity.Song, error)

	// FindLatestUpdatedAt は songs, charts, worldsend_charts の updated_at の最大値を返します。
	// 対象データが存在しない場合は nil を返します。
	FindLatestUpdatedAt(ctx context.Context, exec Executor) (*time.Time, error)

	// Save は楽曲集約（楽曲本体と既存譜面）の現在の状態を永続化します。
	// 譜面の追加・削除は行いません。
	// 対象が存在しない場合は ErrSongNotFound を返します。
	Save(ctx context.Context, exec Executor, song *entity.Song) error

	// UpdateSongs は楽曲および譜面情報を一括更新します。
	// トランザクション管理はUseCase層（TransactionManager経由）で行います。
	// 存在しない楽曲・譜面がある場合はエラーを返します。
	UpdateSongs(ctx context.Context, exec Executor, songs []*entity.Song) error

	// Create は新規楽曲を songs および charts テーブルに追加します。
	// display_id 重複時は ErrDuplicateDisplayID を、official_idx 重複時は ErrDuplicateOfficialIdx を返します。
	Create(ctx context.Context, exec Executor, song *entity.Song) (*entity.Song, error)
}
