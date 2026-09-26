package models

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/entity"
	"github.com/chunisupport/chunisupport-api/internal/utils"
)

// SongModelForUpsert はUPSERT操作用の楽曲モデルです
type SongModelForUpsert struct {
	DisplayID   string  `db:"display_id"`
	Title       string  `db:"title"`
	Reading     *string `db:"reading"`
	Artist      string  `db:"artist"`
	GenreID     int     `db:"genre_id"`
	OfficialIdx string  `db:"official_idx"`
	Jacket      any     `db:"jacket"`
	BPM         *int    `db:"bpm"`
	ReleasedAt  *string `db:"released_at"`
	IsWorldsEnd int     `db:"is_worldsend"`
	IsNew       int     `db:"is_new"`
}

// FromSongEntityForUpsert はSongエンティティからUPSERT用モデルを生成します
func FromSongEntityForUpsert(s *entity.Song) *SongModelForUpsert {
	return &SongModelForUpsert{
		DisplayID:   s.DisplayID().String(),
		Title:       s.Title(),
		Artist:      s.Artist(),
		GenreID:     s.GenreID(),
		OfficialIdx: s.OfficialIdx().String(),
		Jacket:      s.Jacket().NullableString(),
		BPM:         s.BPM(),
		ReleasedAt:  s.ReleasedAt().StringPtr(),
		IsWorldsEnd: utils.BoolToInt(s.IsWorldsEnd()),
	}
}
