package entity

import (
	vo "github.com/chunisupport/chunisupport-api/internal/domain/songbatch/valueobject"
)

// WorldsEndChart はWORLD'S END譜面を表すドメインエンティティ
type WorldsEndChart struct {
	songID  int
	weStar  vo.WeStar
	weKanji vo.WeKanji
	notes   *int
}

// NewWorldsEndChart は新しいWorldsEndChartエンティティを生成します
func NewWorldsEndChart(songID int, weStar vo.WeStar, weKanji vo.WeKanji) *WorldsEndChart {
	return &WorldsEndChart{
		songID:  songID,
		weStar:  weStar,
		weKanji: weKanji,
	}
}

// SongID は楽曲IDを返します
func (c *WorldsEndChart) SongID() int { return c.songID }

// WeStar は星数を返します
func (c *WorldsEndChart) WeStar() vo.WeStar { return c.weStar }

// WeKanji はカテゴリ漢字を返します
func (c *WorldsEndChart) WeKanji() vo.WeKanji { return c.weKanji }

// Notes はノーツ数を返します
func (c *WorldsEndChart) Notes() *int { return c.notes }

// SetNotes はノーツ数を設定します
func (c *WorldsEndChart) SetNotes(notes int) {
	c.notes = &notes
}
