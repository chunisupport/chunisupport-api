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

	// GetLatestUpdatedAtExcludingWorldsend はWORLD'S END以外の楽曲一覧全体の最終更新日時を返します。
	// includeDeleted=false の場合、削除済み楽曲のチャート更新は除外されますが、
	// 楽曲自体の updated_at（削除操作による変更を含む）は常に集計対象とします。
	GetLatestUpdatedAtExcludingWorldsend(ctx context.Context, exec Executor, includeDeleted bool) (*time.Time, error)

	// FindByDisplayID は指定されたDisplayIDの通常楽曲（WORLD'S END除く）を取得します。
	// 削除済み楽曲も取得します。
	// 各楽曲には関連する譜面情報が含まれます。
	FindByDisplayID(ctx context.Context, exec Executor, displayID string) (*entity.Song, error)

	// FindByDisplayIDs は指定されたDisplayIDのリストに該当する通常楽曲（WORLD'S END除く）を取得します。
	// 存在しないDisplayIDがある場合でもエラーにはせず、存在する楽曲のみを返します。
	// 各楽曲には関連する譜面情報が含まれます。
	FindByDisplayIDs(ctx context.Context, exec Executor, displayIDs []string) ([]*entity.Song, error)

	// Save は楽曲エンティティの現在の状態を永続化します。
	// 対象が存在しない場合は ErrSongNotFound を返します。
	Save(ctx context.Context, exec Executor, song *entity.Song) error

	// UpdateSongs は楽曲および譜面情報を一括更新します。
	// トランザクション管理はUseCase層（TransactionManager経由）で行います。
	// 存在しない楽曲・譜面がある場合はエラーを返します。
	UpdateSongs(ctx context.Context, exec Executor, songs []*entity.Song) error
}
