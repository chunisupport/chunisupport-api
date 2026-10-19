package consolidation

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/importer"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/songchart"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindSongsMissingFromSources(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	target, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)"})
	require.NoError(t, err)
	defer target.Close()

	// Given: MySQL 側に既存楽曲があり、データソースには一部だけが含まれる
	_, err = target.DB().ExecContext(ctx, `
		INSERT INTO songs (display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted)
		VALUES
			('official', 'Official', 'Artist', 1, 'OFFICIAL', 0, 0),
			('additional', 'Additional', 'Artist', 1, 'ADDITIONAL', 0, 0),
			('missing', 'Missing', 'Artist', 1, 'MISSING', 0, 0),
			('deleted', 'Deleted', 'Artist', 1, 'DELETED', 0, 1),
			('renumbered', 'Renumbered', 'Artist', 1, 'OLD-IDX', 0, 0)
	`)
	require.NoError(t, err)
	sources := ConsolidationSources{
		// 統合時にジャンル不明などで除外される楽曲もデータソースには存在するため、取り込んだままの ID で判定する
		Official: &importer.OfficialData{{ID: "OFFICIAL"}, {ID: "NEW-IDX"}},
		// additional_songs の ID は取り込み時に前後の空白を除去しないため、判定時に除去する
		AdditionalSongs: &importer.AdditionalSongsData{Songs: []importer.AdditionalSong{{ID: " ADDITIONAL "}}},
	}

	// When
	got, err := findSongsMissingFromSources(ctx, target.DB(), sources)

	// Then: 削除済みを除き、official_idx がデータソースに無い楽曲を削除候補とする
	// official_idx の振り直しで残った旧行も削除が必要な重複のため対象に含める
	require.NoError(t, err)
	assert.Equal(t, []missingSong{
		{DisplayID: "missing", OfficialIdx: "MISSING", Title: "Missing", Artist: "Artist", IsWorldsend: 0},
		{DisplayID: "renumbered", OfficialIdx: "OLD-IDX", Title: "Renumbered", Artist: "Artist", IsWorldsend: 0},
	}, got)
}
