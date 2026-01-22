package usecase

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/dto/api_internal"
)

// SongUsecase は楽曲に関するユースケースを提供します。
type SongUsecase interface {
	// GetAllSongsExcludingWorldsend はWORLD'S END以外の全楽曲を取得します。
	// includeDeletedがfalseの場合、削除済み楽曲は除外されます。
	GetAllSongsExcludingWorldsend(ctx context.Context, includeDeleted bool) ([]*repository.SongWithCharts, error)

	// GetSongByDisplayID は指定されたDisplayIDの楽曲を取得します。
	GetSongByDisplayID(ctx context.Context, displayID string) (*repository.SongWithCharts, error)

	// DeleteSong は指定されたDisplayIDの楽曲を論理削除します。
	DeleteSong(ctx context.Context, displayID string) error

	// RestoreSong は指定されたDisplayIDの楽曲を復活させます。
	RestoreSong(ctx context.Context, displayID string) error

	// UpdateSongs は楽曲および譜面情報を一括更新します。
	// マスタデータの検証およびリポジトリへの委譲を行います。
	UpdateSongs(ctx context.Context, requests []*api_internal.UpdateSongRequest) error

	// GetChartStatisticsByChartIDs は指定された譜面IDリストの統計を一括取得します。
	// 譜面IDをキーとするマップで返します（統計が存在しない譜面は空のスライス）。
	GetChartStatisticsByChartIDs(ctx context.Context, chartIDs []int) (map[int][]*entity.ChartStatistics, error)
}
