package songexport

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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
}

func (w *recordingWriter) PutJSON(_ context.Context, objectKey string, body []byte) error {
	if objectKey == w.failOnKey {
		return errors.New("upload failed")
	}
	if w.objects == nil {
		w.objects = make(map[string][]byte)
	}
	w.objects[objectKey] = append([]byte(nil), body...)
	return nil
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
	exporter := NewExporter(
		&stubSongSource{songs: []*entity.Song{song}},
		&stubWorldsendSource{songs: []*entity.WorldsendSongWithChart{worldsendSong}},
		map[int]string{1: "POPS & ANIME"},
		map[int]string{1: "BASIC", 2: "ADVANCED", 3: "EXPERT", 4: "MASTER", 5: "ULTIMA"},
		writer,
	)

	// When
	result, err := exporter.Export(context.Background())

	// Then
	require.NoError(t, err)
	assert.Equal(t, Result{SongCount: 1, WorldsendSongCount: 1}, result)
	require.Contains(t, writer.objects, info.SongSnapshotObjectKey)
	require.Contains(t, writer.objects, info.WorldsendSongSnapshotObjectKey)

	var songsResponse api_v1.V1SongsResponse
	require.NoError(t, json.Unmarshal(writer.objects[info.SongSnapshotObjectKey], &songsResponse))
	require.Len(t, songsResponse.Songs, 1)
	assert.Equal(t, "通常楽曲", songsResponse.Songs[0].Title)
	require.NotNil(t, songsResponse.Songs[0].Charts["MASTER"])

	var worldsendResponse api_v1.V1WorldsendSongsResponse
	require.NoError(t, json.Unmarshal(writer.objects[info.WorldsendSongSnapshotObjectKey], &worldsendResponse))
	require.Len(t, worldsendResponse.Songs, 1)
	assert.Equal(t, "WORLD'S END楽曲", worldsendResponse.Songs[0].Title)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			writer := &recordingWriter{}
			exporter := NewExporter(tt.songSource, tt.worldsendSource, nil, nil, writer)

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
	exporter := NewExporter(
		&stubSongSource{songs: []*entity.Song{{DisplayID: "0123456789abcdef"}}},
		&stubWorldsendSource{songs: []*entity.WorldsendSongWithChart{{
			Song:  &entity.Song{DisplayID: "fedcba9876543210"},
			Chart: &entity.WorldsendChart{},
		}}},
		nil,
		nil,
		writer,
	)

	// When
	_, err := exporter.Export(context.Background())

	// Then
	require.Error(t, err)
	assert.Contains(t, err.Error(), info.WorldsendSongSnapshotObjectKey)
	assert.Contains(t, writer.objects, info.SongSnapshotObjectKey)
}
