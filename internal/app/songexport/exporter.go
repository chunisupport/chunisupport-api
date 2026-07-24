package songexport

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

type songSource interface {
	GetAllSongsExcludingWorldsend(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*entity.Song, error)
	CalcSongMaxOP(song *entity.Song) float64
}

type worldsendSource interface {
	GetAllWorldsendSongs(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*entity.WorldsendSongWithChart, error)
}

// JSONWriter は生成済みJSONをオブジェクトストレージへ保存します。
type JSONWriter interface {
	PutJSON(ctx context.Context, objectKey string, body []byte) error
}

// Result は正常に公開したスナップショットの件数を表します。
type Result struct {
	SongCount          int
	WorldsendSongCount int
}

// Exporter は公開用の楽曲一覧JSONを生成してオブジェクトストレージへ保存します。
type Exporter struct {
	songs                 songSource
	worldsendSongs        worldsendSource
	genreNamesByID        map[int]string
	difficultyNamesByID   map[int]string
	objectStorageJSONSink JSONWriter
}

// NewExporter は楽曲スナップショットのエクスポーターを生成します。
func NewExporter(
	songs songSource,
	worldsendSongs worldsendSource,
	genreNamesByID map[int]string,
	difficultyNamesByID map[int]string,
	objectStorageJSONSink JSONWriter,
) *Exporter {
	return &Exporter{
		songs:                 songs,
		worldsendSongs:        worldsendSongs,
		genreNamesByID:        genreNamesByID,
		difficultyNamesByID:   difficultyNamesByID,
		objectStorageJSONSink: objectStorageJSONSink,
	}
}

// Export は通常楽曲とWORLD'S END楽曲を取得し、両方のJSON生成後に固定キーへ保存します。
// 取得件数が0件の場合は、異常な空スナップショットによる上書きを防ぐため失敗させます。
func (e *Exporter) Export(ctx context.Context) (Result, error) {
	songs, err := e.songs.GetAllSongsExcludingWorldsend(ctx, false, nil)
	if err != nil {
		return Result{}, fmt.Errorf("failed to get songs for snapshot export: %w", err)
	}
	if len(songs) == 0 {
		return Result{}, fmt.Errorf("refusing to export an empty song snapshot")
	}

	worldsendSongs, err := e.worldsendSongs.GetAllWorldsendSongs(ctx, false, nil)
	if err != nil {
		return Result{}, fmt.Errorf("failed to get worldsend songs for snapshot export: %w", err)
	}
	if len(worldsendSongs) == 0 {
		return Result{}, fmt.Errorf("refusing to export an empty worldsend song snapshot")
	}

	songsJSON, err := json.Marshal(api_v1.NewV1SongsResponse(
		songs,
		e.genreNamesByID,
		e.difficultyNamesByID,
		e.songs.CalcSongMaxOP,
	))
	if err != nil {
		return Result{}, fmt.Errorf("failed to marshal song snapshot: %w", err)
	}
	worldsendSongsJSON, err := json.Marshal(api_v1.NewV1WorldsendSongsResponse(worldsendSongs, e.genreNamesByID))
	if err != nil {
		return Result{}, fmt.Errorf("failed to marshal worldsend song snapshot: %w", err)
	}

	if err := e.objectStorageJSONSink.PutJSON(ctx, info.SongSnapshotObjectKey, songsJSON); err != nil {
		return Result{}, fmt.Errorf("failed to upload %s: %w", info.SongSnapshotObjectKey, err)
	}
	if err := e.objectStorageJSONSink.PutJSON(ctx, info.WorldsendSongSnapshotObjectKey, worldsendSongsJSON); err != nil {
		return Result{}, fmt.Errorf("failed to upload %s: %w", info.WorldsendSongSnapshotObjectKey, err)
	}

	return Result{
		SongCount:          len(songs),
		WorldsendSongCount: len(worldsendSongs),
	}, nil
}
