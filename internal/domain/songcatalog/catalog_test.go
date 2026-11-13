package songcatalog

import (
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCatalog() *Catalog {
	updatedAt := time.Date(2026, time.November, 13, 10, 0, 0, 0, time.UTC)
	songs := []*entity.Song{
		{ID: 1, DisplayID: "active", Charts: []*entity.Chart{{ID: 10, SongID: 1, DifficultyID: entity.DifficultyIDMaster}}},
		{ID: 2, DisplayID: "deleted", IsDeleted: true, Charts: []*entity.Chart{{ID: 20, SongID: 2}}},
	}
	worldsendSongs := []*entity.WorldsendSongWithChart{
		{Song: &entity.Song{ID: 3, DisplayID: "we-active", Charts: []*entity.Chart{}}, Chart: &entity.WorldsendChart{ID: 30, SongID: 3}},
		{Song: &entity.Song{ID: 4, DisplayID: "we-deleted", IsDeleted: true, Charts: []*entity.Chart{}}, Chart: &entity.WorldsendChart{ID: 40, SongID: 4}},
	}
	return New(7, songs, worldsendSongs, &updatedAt)
}

func TestCatalog_Songs(t *testing.T) {
	tests := []struct {
		name           string
		includeDeleted bool
		expected       []string
	}{
		{name: "削除済みを除くと有効な楽曲だけを返す", includeDeleted: false, expected: []string{"active"}},
		{name: "削除済みを含めると全楽曲を元の順で返す", includeDeleted: true, expected: []string{"active", "deleted"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			catalog := newTestCatalog()

			// When
			songs := catalog.Songs(tt.includeDeleted)

			// Then
			actual := make([]string, 0, len(songs))
			for _, song := range songs {
				actual = append(actual, song.DisplayID)
			}
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestCatalog_WorldsendSongs(t *testing.T) {
	tests := []struct {
		name           string
		includeDeleted bool
		expected       []string
	}{
		{name: "削除済みを除くと有効な楽曲だけを返す", includeDeleted: false, expected: []string{"we-active"}},
		{name: "削除済みを含めると全楽曲を元の順で返す", includeDeleted: true, expected: []string{"we-active", "we-deleted"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			catalog := newTestCatalog()

			// When
			songs := catalog.WorldsendSongs(tt.includeDeleted)

			// Then
			actual := make([]string, 0, len(songs))
			for _, song := range songs {
				actual = append(actual, song.Song.DisplayID)
			}
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestCatalog_取得結果を変更しても共有データは汚染されない(t *testing.T) {
	// Given
	catalog := newTestCatalog()

	// When
	songs := catalog.Songs(true)
	songs[0].Title = "変更"
	songs[0].Charts[0].IsConstUnknown = true
	songs[0] = nil
	worldsendSongs := catalog.WorldsendSongs(true)
	worldsendSongs[0].Song.Title = "変更"
	worldsendSongs[0].Chart.SongID = 999
	updatedAt := catalog.UpdatedAt()
	*updatedAt = time.Time{}

	// Then
	again := catalog.Songs(true)
	require.NotNil(t, again[0])
	assert.Empty(t, again[0].Title)
	assert.False(t, again[0].Charts[0].IsConstUnknown)
	againWorldsend := catalog.WorldsendSongs(true)
	assert.Empty(t, againWorldsend[0].Song.Title)
	assert.Equal(t, 3, againWorldsend[0].Chart.SongID)
	assert.Equal(t, time.Date(2026, time.November, 13, 10, 0, 0, 0, time.UTC), *catalog.UpdatedAt())
}

func TestNew_構築に渡した値を後から変更しても公開データは変わらない(t *testing.T) {
	// Given
	songs := []*entity.Song{{ID: 1, DisplayID: "song", Charts: []*entity.Chart{}}}
	worldsendSongs := []*entity.WorldsendSongWithChart{{Song: &entity.Song{ID: 2, DisplayID: "we", Charts: []*entity.Chart{}}, Chart: &entity.WorldsendChart{ID: 20}}}
	updatedAt := time.Date(2026, time.November, 13, 10, 0, 0, 0, time.UTC)

	// When
	catalog := New(1, songs, worldsendSongs, &updatedAt)
	songs[0].DisplayID = "changed"
	worldsendSongs[0].Song.DisplayID = "changed"
	updatedAt = time.Time{}

	// Then
	assert.Equal(t, "song", catalog.Songs(false)[0].DisplayID)
	assert.Equal(t, "we", catalog.WorldsendSongs(false)[0].Song.DisplayID)
	assert.Equal(t, time.Date(2026, time.November, 13, 10, 0, 0, 0, time.UTC), *catalog.UpdatedAt())
}

func TestCatalog_RevisionとUpdatedAt(t *testing.T) {
	// Given
	catalog := newTestCatalog()
	empty := New(0, nil, nil, nil)

	// When / Then
	assert.Equal(t, uint64(7), catalog.Revision())
	assert.Nil(t, empty.UpdatedAt())
	assert.Empty(t, empty.Songs(true))
	assert.Empty(t, empty.WorldsendSongs(true))
}
