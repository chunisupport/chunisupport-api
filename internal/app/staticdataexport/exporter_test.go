package staticdataexport

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/chunirec"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/reiwa"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSongSource struct {
	songs []*entity.Song
	err   error
}

func (s *stubSongSource) GetAllSongsExcludingWorldsend(context.Context, bool, *int) ([]*entity.Song, error) {
	return s.songs, s.err
}

func (s *stubSongSource) CalcSongMaxOP(*entity.Song) float64 {
	return 90
}

type stubWorldsendSource struct {
	songs []*entity.WorldsendSongWithChart
	err   error
}

func (s *stubWorldsendSource) GetAllWorldsendSongs(context.Context, bool, *int) ([]*entity.WorldsendSongWithChart, error) {
	return s.songs, s.err
}

type recordingWriter struct {
	objects   map[string][]byte
	failOnKey string
	putKeys   []string
}

type recordingCachePurger struct {
	objectKeys []string
	err        error
}

func (p *recordingCachePurger) Purge(_ context.Context, objectKeys []string) error {
	p.objectKeys = append([]string(nil), objectKeys...)
	return p.err
}

func (w *recordingWriter) PutJSON(_ context.Context, objectKey string, body []byte) error {
	w.putKeys = append(w.putKeys, objectKey)
	if objectKey == w.failOnKey {
		return errors.New("upload failed")
	}
	if w.objects == nil {
		w.objects = make(map[string][]byte)
	}
	w.objects[objectKey] = append([]byte(nil), body...)
	return nil
}

func buildTestChunirecSnapshot(songs []*entity.Song) (any, int) {
	response := chunirec.ToMusicShowAllResponse(songs, nil)
	return response, len(response)
}

func buildTestReiwaSnapshot(songs []*entity.Song) (any, int) {
	response := reiwa.ToChunithmRecordOriginalResponse(songs, nil)
	return response, len(response)
}

func TestExporterExport_一覧JSONを固定キーへアップロードする(t *testing.T) {
	// Given
	constant, err := chartconstant.NewChartConstant(13.5)
	require.NoError(t, err)
	genreID := 1
	song := &entity.Song{
		DisplayID:   "0123456789abcdef",
		Title:       "通常楽曲",
		Artist:      "アーティスト",
		GenreID:     &genreID,
		OfficialIdx: "1",
		Charts: []*entity.Chart{{
			DifficultyID: 4,
			Const:        constant,
		}},
	}
	worldsendSong := &entity.WorldsendSongWithChart{
		Song: &entity.Song{
			DisplayID:   "fedcba9876543210",
			Title:       "WORLD'S END楽曲",
			Artist:      "アーティスト",
			OfficialIdx: "2",
		},
		Chart: &entity.WorldsendChart{},
	}
	writer := &recordingWriter{}
	cachePurger := &recordingCachePurger{}
	exporter := NewExporter(
		&stubSongSource{songs: []*entity.Song{song}},
		&stubWorldsendSource{songs: []*entity.WorldsendSongWithChart{worldsendSong}},
		map[int]string{1: "POPS & ANIME"},
		map[int]string{1: "BASIC", 2: "ADVANCED", 3: "EXPERT", 4: "MASTER", 5: "ULTIMA"},
		buildTestChunirecSnapshot,
		buildTestReiwaSnapshot,
		writer,
		cachePurger,
	)

	// When
	result, err := exporter.Export(context.Background())

	// Then
	require.NoError(t, err)
	assert.Equal(t, Result{SongCount: 1, WorldsendSongCount: 1, ChunirecSongCount: 1, ReiwaRecordCount: 1}, result)
	require.Contains(t, writer.objects, info.SongSnapshotObjectKey)
	require.Contains(t, writer.objects, info.WorldsendSongSnapshotObjectKey)
	require.Contains(t, writer.objects, info.ChunirecSongSnapshotObjectKey)
	require.Contains(t, writer.objects, info.ReiwaSongSnapshotObjectKey)

	var songsResponse api_v1.V1SongsResponse
	require.NoError(t, json.Unmarshal(writer.objects[info.SongSnapshotObjectKey], &songsResponse))
	require.Len(t, songsResponse.Songs, 1)
	assert.Equal(t, "通常楽曲", songsResponse.Songs[0].Title)
	require.NotNil(t, songsResponse.Songs[0].Charts["MASTER"])

	var worldsendResponse api_v1.V1WorldsendSongsResponse
	require.NoError(t, json.Unmarshal(writer.objects[info.WorldsendSongSnapshotObjectKey], &worldsendResponse))
	require.Len(t, worldsendResponse.Songs, 1)
	assert.Equal(t, "WORLD'S END楽曲", worldsendResponse.Songs[0].Title)

	var chunirecResponse []struct {
		Meta struct {
			Title string `json:"title"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(writer.objects[info.ChunirecSongSnapshotObjectKey], &chunirecResponse))
	require.Len(t, chunirecResponse, 1)
	assert.Equal(t, "通常楽曲", chunirecResponse[0].Meta.Title)

	var reiwaResponse []struct {
		Title string `json:"title"`
		Diff  string `json:"diff"`
	}
	require.NoError(t, json.Unmarshal(writer.objects[info.ReiwaSongSnapshotObjectKey], &reiwaResponse))
	require.Len(t, reiwaResponse, 1)
	assert.Equal(t, "通常楽曲", reiwaResponse[0].Title)
	assert.Equal(t, "MAS", reiwaResponse[0].Diff)
	assert.Equal(t, []string{
		info.SongSnapshotObjectKey,
		info.WorldsendSongSnapshotObjectKey,
		info.ChunirecSongSnapshotObjectKey,
		info.ReiwaSongSnapshotObjectKey,
	}, cachePurger.objectKeys)
}

func TestExporterExport_取得または検証失敗時はアップロードしない(t *testing.T) {
	worldsendSong := &entity.WorldsendSongWithChart{
		Song:  &entity.Song{DisplayID: "fedcba9876543210"},
		Chart: &entity.WorldsendChart{},
	}
	tests := []struct {
		name            string
		songSource      *stubSongSource
		worldsendSource *stubWorldsendSource
	}{
		{
			name:            "通常楽曲の取得失敗",
			songSource:      &stubSongSource{err: errors.New("query failed")},
			worldsendSource: &stubWorldsendSource{songs: []*entity.WorldsendSongWithChart{worldsendSong}},
		},
		{
			name:            "通常楽曲が0件",
			songSource:      &stubSongSource{songs: []*entity.Song{}},
			worldsendSource: &stubWorldsendSource{songs: []*entity.WorldsendSongWithChart{worldsendSong}},
		},
		{
			name:            "WORLD'S END楽曲が0件",
			songSource:      &stubSongSource{songs: []*entity.Song{{DisplayID: "0123456789abcdef"}}},
			worldsendSource: &stubWorldsendSource{songs: []*entity.WorldsendSongWithChart{}},
		},
		{
			name:            "reiwa互換の譜面が0件",
			songSource:      &stubSongSource{songs: []*entity.Song{{DisplayID: "0123456789abcdef"}}},
			worldsendSource: &stubWorldsendSource{songs: []*entity.WorldsendSongWithChart{worldsendSong}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			writer := &recordingWriter{}
			exporter := NewExporter(
				tt.songSource,
				tt.worldsendSource,
				nil,
				nil,
				buildTestChunirecSnapshot,
				buildTestReiwaSnapshot,
				writer,
				&recordingCachePurger{},
			)

			// When
			_, err := exporter.Export(context.Background())

			// Then
			assert.Error(t, err)
			assert.Empty(t, writer.objects)
		})
	}
}

func TestExporterExport_アップロード失敗を返す(t *testing.T) {
	// Given
	writer := &recordingWriter{failOnKey: info.WorldsendSongSnapshotObjectKey}
	cachePurger := &recordingCachePurger{}
	exporter := NewExporter(
		&stubSongSource{songs: []*entity.Song{{
			DisplayID: "0123456789abcdef",
			Charts:    []*entity.Chart{{DifficultyID: 4}},
		}}},
		&stubWorldsendSource{songs: []*entity.WorldsendSongWithChart{{
			Song:  &entity.Song{DisplayID: "fedcba9876543210"},
			Chart: &entity.WorldsendChart{},
		}}},
		nil,
		nil,
		buildTestChunirecSnapshot,
		buildTestReiwaSnapshot,
		writer,
		cachePurger,
	)

	// When
	_, err := exporter.Export(context.Background())

	// Then
	require.Error(t, err)
	assert.Contains(t, err.Error(), info.WorldsendSongSnapshotObjectKey)
	assert.Contains(t, writer.objects, info.SongSnapshotObjectKey)
	assert.Empty(t, cachePurger.objectKeys)
}

func TestExporterExport_互換APIのアップロード失敗を返す(t *testing.T) {
	tests := []struct {
		name      string
		failOnKey string
	}{
		{name: "chunirec互換", failOnKey: info.ChunirecSongSnapshotObjectKey},
		{name: "reiwa互換", failOnKey: info.ReiwaSongSnapshotObjectKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			writer := &recordingWriter{failOnKey: tt.failOnKey}
			cachePurger := &recordingCachePurger{}
			exporter := NewExporter(
				&stubSongSource{songs: []*entity.Song{{
					DisplayID: "0123456789abcdef",
					Charts:    []*entity.Chart{{DifficultyID: 4}},
				}}},
				&stubWorldsendSource{songs: []*entity.WorldsendSongWithChart{{
					Song:  &entity.Song{DisplayID: "fedcba9876543210"},
					Chart: &entity.WorldsendChart{},
				}}},
				nil,
				nil,
				buildTestChunirecSnapshot,
				buildTestReiwaSnapshot,
				writer,
				cachePurger,
			)

			// When
			_, err := exporter.Export(context.Background())

			// Then
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.failOnKey)
			assert.NotContains(t, writer.objects, tt.failOnKey)
			assert.Empty(t, cachePurger.objectKeys)
			if tt.failOnKey == info.ChunirecSongSnapshotObjectKey {
				assert.Equal(t, []string{
					info.SongSnapshotObjectKey,
					info.WorldsendSongSnapshotObjectKey,
					info.ChunirecSongSnapshotObjectKey,
				}, writer.putKeys)
				assert.NotContains(t, writer.objects, info.ReiwaSongSnapshotObjectKey)
			} else {
				assert.Equal(t, []string{
					info.SongSnapshotObjectKey,
					info.WorldsendSongSnapshotObjectKey,
					info.ChunirecSongSnapshotObjectKey,
					info.ReiwaSongSnapshotObjectKey,
				}, writer.putKeys)
				assert.Contains(t, writer.objects, info.ChunirecSongSnapshotObjectKey)
			}
		})
	}
}

func TestExporterExport_キャッシュパージ失敗を返す(t *testing.T) {
	// Given
	cachePurger := &recordingCachePurger{err: errors.New("purge failed")}
	exporter := NewExporter(
		&stubSongSource{songs: []*entity.Song{{
			DisplayID: "0123456789abcdef",
			Charts:    []*entity.Chart{{DifficultyID: 4}},
		}}},
		&stubWorldsendSource{songs: []*entity.WorldsendSongWithChart{{
			Song:  &entity.Song{DisplayID: "fedcba9876543210"},
			Chart: &entity.WorldsendChart{},
		}}},
		nil,
		nil,
		buildTestChunirecSnapshot,
		buildTestReiwaSnapshot,
		&recordingWriter{},
		cachePurger,
	)

	// When
	_, err := exporter.Export(context.Background())

	// Then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to purge static data cache")
	assert.Len(t, cachePurger.objectKeys, 4)
}
