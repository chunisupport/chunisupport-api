package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

// SongUsecase は楽曲に関するユースケースを提供します。
type SongUsecase interface {
	// GetAllSongsExcludingWorldsend はWORLD'S END以外の全楽曲を取得します。
	// includeDeletedがfalseの場合、削除済み楽曲は除外されます。
	GetAllSongsExcludingWorldsend(ctx context.Context, includeDeleted bool) ([]*entity.Song, error)

	// GetSongByDisplayID は指定されたDisplayIDの楽曲を取得します。
	GetSongByDisplayID(ctx context.Context, displayID string) (*entity.Song, error)

	// DeleteSong は指定されたDisplayIDの楽曲を論理削除します。
	DeleteSong(ctx context.Context, displayID string) error

	// RestoreSong は指定されたDisplayIDの楽曲を復活させます。
	RestoreSong(ctx context.Context, displayID string) error

	// UpdateSongs は楽曲および譜面情報を一括更新します。
	// マスタデータの検証およびリポジトリへの委譲を行います。
	UpdateSongs(ctx context.Context, requests []*api_internal.UpdateSongRequest) error
}
