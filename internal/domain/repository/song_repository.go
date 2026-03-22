package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// SongRepository は楽曲に関する永続化処理を定義するリポジトリです。
// 個々のリポジトリと同様に、メソッドではExecutorを明示的に受け取り、
// UseCase側からのトランザクション制御を可能にします。
type SongRepository interface {
	// FindAllExcludingWorldsend はWORLD'S END以外の全楽曲を取得します。
	// includeDeleted=false の場合、削除済み楽曲は除外されます。
	// 各楽曲には関連する譜面情報が含まれます。
	FindAllExcludingWorldsend(ctx context.Context, exec Executor, includeDeleted bool) ([]*entity.Song, error)

	// GetLatestUpdatedAtExcludingWorldsend はWORLD'S END以外の楽曲一覧全体の最終更新日時を返します。
	// includeDeleted=false の場合でも、削除済み楽曲のチャート更新は考慮されますが、
	// 楽曲自体の updated_at（削除フラグ変更による更新など）も正しく計算される必要があります。
	GetLatestUpdatedAtExcludingWorldsend(ctx context.Context, exec Executor, includeDeleted bool) (*time.Time, error)

	// FindByDisplayID は指定されたDisplayIDの通常楽曲（WORLD'S END以外）を取得します。
	// 削除済み楽曲も取得対象です。
	// 各楽曲には関連する譜面情報が含まれます。
	FindByDisplayID(ctx context.Context, exec Executor, displayID string) (*entity.Song, error)

	// FindByDisplayIDs は指定されたDisplayIDのリストに該当する通常楽曲（WORLD'S END以外）を取得します。
	// 存在しないDisplayIDがある場合でもエラーにはせず、存在する楽曲のみを返します。
	// 各楽曲には関連する譜面情報が含まれます。
	FindByDisplayIDs(ctx context.Context, exec Executor, displayIDs []string) ([]*entity.Song, error)

	// Save は楽曲エンティティの現在の状態を永続化します。
	// 対象楽曲が存在しない場合は ErrSongNotFound を返します。
	Save(ctx context.Context, exec Executor, song *entity.Song) error

	// UpdateSongs は楽曲および譜面情報を一括更新します。
	// トランザクション境界はUseCase側のTransactionManagerで制御します。
	// 存在しない楽曲・譜面がある場合はエラーを返します。
	UpdateSongs(ctx context.Context, exec Executor, songs []*entity.Song) error
}
