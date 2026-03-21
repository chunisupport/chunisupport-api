package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

type SongListResult struct {
	Songs     []*entity.Song
	UpdatedAt *time.Time
}

// SongUsecase は楽曲に関するユースケースを定義します。
type SongUsecase interface {
	// GetAllSongsExcludingWorldsend はWORLD'S END以外の全楽曲を取得します。
	// includeDeleted が true かつ requesterAccountTypeID が EDITOR 権限を持たない場合、
	// 削除済み楽曲は含められません。
	GetAllSongsExcludingWorldsend(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) (*SongListResult, error)

	// GetSongsLastUpdatedAt はWORLD'S END以外の楽曲一覧全体の最終更新日時を取得します。
	// 互換性のため残しています。
	GetSongsLastUpdatedAt(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) (*time.Time, error)

	// GetSongByDisplayID は指定されたDisplayIDの楽曲を取得します。
	// requesterAccountTypeIDがnilまたはEDITOR権限を持たない場合、
	// 削除済み楽曲はErrSongNotFoundを返します。
	GetSongByDisplayID(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.Song, error)

	// DeleteSong は指定されたDisplayIDの楽曲を論理削除します。
	DeleteSong(ctx context.Context, displayID string) error

	// RestoreSong は指定されたDisplayIDの楽曲を復活させます。
	RestoreSong(ctx context.Context, displayID string) error

	// UpdateSongs は楽曲および譜面情報を一括更新します。
	// マスターデータの読み込みおよびリポジトリへの変換を行います。
	UpdateSongs(ctx context.Context, requests []*api_internal.UpdateSongRequest) error

	// CalcSongMaxOP は楽曲の譜面から逆算した最大OPを計算します。
	CalcSongMaxOP(song *entity.Song) float64
}
