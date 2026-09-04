package staticdataexport

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

// CachePurger はCDN固有の型をアプリケーション層へ漏らさず、公開完了後の無効化だけを要求します。
type CachePurger interface {
	Purge(ctx context.Context, objectKeys []string) error
}

// CompatibilitySnapshotBuilder は通常楽曲から互換API用レスポンスと件数を構築します。
type CompatibilitySnapshotBuilder func(songs []*entity.Song) (response any, count int)

// VersionSnapshotBuilder は公開対象のバージョンから互換API用レスポンスと件数を構築します。
type VersionSnapshotBuilder func(ctx context.Context) (response any, count int)

// Result は正常に公開したスナップショットの件数を表します。
type Result struct {
	SongCount          int
	WorldsendSongCount int
	ChunirecSongCount  int
	ReiwaRecordCount   int
	ReiwaVersionCount  int
}

// Exporter は公開用の静的JSONを生成してオブジェクトストレージへ保存します。
type Exporter struct {
	songs                 songSource
	worldsendSongs        worldsendSource
	genreNamesByID        map[int]string
	difficultyNamesByID   map[int]string
	buildChunirecSnapshot CompatibilitySnapshotBuilder
	buildReiwaSnapshot    CompatibilitySnapshotBuilder
	buildVersionSnapshot  VersionSnapshotBuilder
	objectStorageJSONSink JSONWriter
	cachePurger           CachePurger
}

// NewExporter は静的データのエクスポーターを生成します。
func NewExporter(
	songs songSource,
	worldsendSongs worldsendSource,
	genreNamesByID map[int]string,
	difficultyNamesByID map[int]string,
	buildChunirecSnapshot CompatibilitySnapshotBuilder,
	buildReiwaSnapshot CompatibilitySnapshotBuilder,
	buildVersionSnapshot VersionSnapshotBuilder,
	objectStorageJSONSink JSONWriter,
	cachePurger CachePurger,
) *Exporter {
	return &Exporter{
		songs:                 songs,
		worldsendSongs:        worldsendSongs,
		genreNamesByID:        genreNamesByID,
		difficultyNamesByID:   difficultyNamesByID,
		buildChunirecSnapshot: buildChunirecSnapshot,
		buildReiwaSnapshot:    buildReiwaSnapshot,
		buildVersionSnapshot:  buildVersionSnapshot,
		objectStorageJSONSink: objectStorageJSONSink,
		cachePurger:           cachePurger,
	}
}

// Export は通常楽曲とWORLD'S END楽曲を取得し、5種類のJSON生成後に固定キーへ保存します。
// 通常楽曲、WORLD'S END楽曲、reiwa互換譜面、またはバージョンが0件の場合は、異常な空スナップショットによる上書きを防ぐため失敗させます。
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
	chunirecSongs, chunirecSongCount := e.buildChunirecSnapshot(songs)
	chunirecSongsJSON, err := json.Marshal(chunirecSongs)
	if err != nil {
		return Result{}, fmt.Errorf("failed to marshal chunirec song snapshot: %w", err)
	}
	reiwaRecords, reiwaRecordCount := e.buildReiwaSnapshot(songs)
	if reiwaRecordCount == 0 {
		return Result{}, fmt.Errorf("refusing to export an empty reiwa song snapshot")
	}
	reiwaRecordsJSON, err := json.Marshal(reiwaRecords)
	if err != nil {
		return Result{}, fmt.Errorf("failed to marshal reiwa song snapshot: %w", err)
	}
	versions, versionCount := e.buildVersionSnapshot(ctx)
	if versionCount == 0 {
		return Result{}, fmt.Errorf("refusing to export an empty reiwa version snapshot")
	}
	versionsJSON, err := json.Marshal(versions)
	if err != nil {
		return Result{}, fmt.Errorf("failed to marshal reiwa version snapshot: %w", err)
	}

	if err := e.objectStorageJSONSink.PutJSON(ctx, info.SongSnapshotObjectKey, songsJSON); err != nil {
		return Result{}, fmt.Errorf("failed to upload %s: %w", info.SongSnapshotObjectKey, err)
	}
	if err := e.objectStorageJSONSink.PutJSON(ctx, info.WorldsendSongSnapshotObjectKey, worldsendSongsJSON); err != nil {
		return Result{}, fmt.Errorf("failed to upload %s: %w", info.WorldsendSongSnapshotObjectKey, err)
	}
	if err := e.objectStorageJSONSink.PutJSON(ctx, info.ChunirecSongSnapshotObjectKey, chunirecSongsJSON); err != nil {
		return Result{}, fmt.Errorf("failed to upload %s: %w", info.ChunirecSongSnapshotObjectKey, err)
	}
	if err := e.objectStorageJSONSink.PutJSON(ctx, info.ReiwaSongSnapshotObjectKey, reiwaRecordsJSON); err != nil {
		return Result{}, fmt.Errorf("failed to upload %s: %w", info.ReiwaSongSnapshotObjectKey, err)
	}
	if err := e.objectStorageJSONSink.PutJSON(ctx, info.ReiwaVersionSnapshotObjectKey, versionsJSON); err != nil {
		return Result{}, fmt.Errorf("failed to upload %s: %w", info.ReiwaVersionSnapshotObjectKey, err)
	}

	objectKeys := []string{
		info.SongSnapshotObjectKey,
		info.WorldsendSongSnapshotObjectKey,
		info.ChunirecSongSnapshotObjectKey,
		info.ReiwaSongSnapshotObjectKey,
		info.ReiwaVersionSnapshotObjectKey,
	}
	if err := e.cachePurger.Purge(ctx, objectKeys); err != nil {
		return Result{}, fmt.Errorf("failed to purge static data cache: %w", err)
	}

	return Result{
		SongCount:          len(songs),
		WorldsendSongCount: len(worldsendSongs),
		ChunirecSongCount:  chunirecSongCount,
		ReiwaRecordCount:   reiwaRecordCount,
		ReiwaVersionCount:  versionCount,
	}, nil
}
