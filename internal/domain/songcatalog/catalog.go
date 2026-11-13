// Package songcatalog は通常楽曲・WORLD'S END楽曲と譜面を、同じ世代のスナップショットとして扱います。
package songcatalog

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// Catalog はDBの同一時点から読み取った楽曲・譜面と、その世代の更新時刻をまとめた不変のスナップショットです。
// プロセス内で共有するため、構築時と取得時の両方で複製し、利用側の変更が他のリクエストへ伝わらないようにします。
// 楽曲と更新時刻を1つの値として差し替えることで、更新時刻だけが新しい世代へ先行することを防ぎます。
type Catalog struct {
	revision       uint64
	songs          []*entity.Song
	worldsendSongs []*entity.WorldsendSongWithChart
	updatedAt      *time.Time
}

// New は世代番号・通常楽曲・WORLD'S END楽曲・更新時刻からスナップショットを生成します。
// 引数は複製して保持するため、呼び出し側が後から変更しても公開データは変わりません。
func New(revision uint64, songs []*entity.Song, worldsendSongs []*entity.WorldsendSongWithChart, updatedAt *time.Time) *Catalog {
	return &Catalog{
		revision:       revision,
		songs:          cloneSongs(songs, true),
		worldsendSongs: cloneWorldsendSongs(worldsendSongs, true),
		updatedAt:      cloneTime(updatedAt),
	}
}

// Revision はスナップショットを構築したときのDB上の変更世代を返します。
func (c *Catalog) Revision() uint64 {
	return c.revision
}

// UpdatedAt はこの世代の楽曲・譜面の更新時刻の最大値を返します。対象データがない場合は nil です。
func (c *Catalog) UpdatedAt() *time.Time {
	return cloneTime(c.updatedAt)
}

// Songs は通常楽曲を譜面付きで複製して返します。並び順は楽曲IDの昇順です。
// includeDeleted が false の場合、削除済み楽曲は除外します。
func (c *Catalog) Songs(includeDeleted bool) []*entity.Song {
	return cloneSongs(c.songs, includeDeleted)
}

// WorldsendSongs は WORLD'S END 楽曲を譜面付きで複製して返します。並び順は楽曲IDの昇順です。
// includeDeleted が false の場合、削除済み楽曲は除外します。
func (c *Catalog) WorldsendSongs(includeDeleted bool) []*entity.WorldsendSongWithChart {
	return cloneWorldsendSongs(c.worldsendSongs, includeDeleted)
}

func cloneSongs(songs []*entity.Song, includeDeleted bool) []*entity.Song {
	result := make([]*entity.Song, 0, len(songs))
	for _, song := range songs {
		if !includeDeleted && !song.IsActive() {
			continue
		}
		result = append(result, song.Clone())
	}
	return result
}

func cloneWorldsendSongs(songs []*entity.WorldsendSongWithChart, includeDeleted bool) []*entity.WorldsendSongWithChart {
	result := make([]*entity.WorldsendSongWithChart, 0, len(songs))
	for _, song := range songs {
		if !includeDeleted && !song.Song.IsActive() {
			continue
		}
		result = append(result, song.Clone())
	}
	return result
}

func cloneTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	return new(*t)
}
