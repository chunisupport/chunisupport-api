package api_internal

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto"
)

// WorldsendChartDTO は WORLD'S END 譜面情報を外部に公開するためのDTOです。
type WorldsendChartDTO struct {
	WeKanji *string `json:"we_kanji"` // WORLD'S END カテゴリ漢字（光、蔵、改、狂、etc.）
	WeStar  *int    `json:"we_star"`  // WORLD'S END 星の数（1～5）
	Notes   *int    `json:"notes"`    // ノーツ数
}

// WorldsendSongDTO は WORLD'S END 楽曲情報を外部に公開するためのDTOです。
// 通常楽曲の SongDTO と同じ形状を持ちますが、charts は "WORLDSEND" キーのみです。
type WorldsendSongDTO struct {
	DisplayID   string                        `json:"id"`
	Title       string                        `json:"title"`
	Artist      string                        `json:"artist"`
	Genre       *string                       `json:"genre"`
	BPM         *int                          `json:"bpm"`
	Release     *string                       `json:"release"`
	Jacket      *string                       `json:"jacket"`
	OfficialIdx string                        `json:"official_idx"`
	MaxOP       *float64                      `json:"maxop"`
	Charts      map[string]*WorldsendChartDTO `json:"charts"`
}

// WorldsendSongsResponse は WORLD'S END 楽曲一覧のレスポンスを表します。
type WorldsendSongsResponse struct {
	Songs []*WorldsendSongDTO `json:"songs"`
}

// ToWorldsendChartDTO は WorldsendChart エンティティから WorldsendChartDTO へ変換します。
func ToWorldsendChartDTO(chart *entity.WorldsendChart) *WorldsendChartDTO {
	if chart == nil {
		return nil
	}

	return &WorldsendChartDTO{
		WeKanji: chart.WeKanji,
		WeStar:  chart.WeStar,
		Notes:   dto.ToNotesIntPtr(chart.Notes),
	}
}

// ToWorldsendSongDTO は Song エンティティと WorldsendChart エンティティから WorldsendSongDTO へ変換します。
// genreNamesByID を使用してジャンルIDを名称に変換します。
// maxOP は WORLD'S END ではレーティング対象外のため常に null です。
func ToWorldsendSongDTO(song *entity.Song, chart *entity.WorldsendChart, genreNamesByID map[int]string) *WorldsendSongDTO {
	if song == nil {
		return nil
	}

	var genrePtr *string
	if song.GenreID != nil {
		if name, ok := genreNamesByID[*song.GenreID]; ok {
			genrePtr = &name
		}
	}

	var releaseDateStr *string
	if song.ReleasedAt != nil {
		formatted := song.ReleasedAt.Format("2006-01-02")
		releaseDateStr = &formatted
	}

	charts := make(map[string]*WorldsendChartDTO)
	charts["WORLDSEND"] = ToWorldsendChartDTO(chart)

	return &WorldsendSongDTO{
		DisplayID:   song.DisplayID,
		Title:       song.Title,
		Artist:      song.Artist,
		Genre:       genrePtr,
		BPM:         song.BPM,
		Release:     releaseDateStr,
		Jacket:      song.Jacket,
		OfficialIdx: song.OfficialIdx,
		MaxOP:       nil, // WORLD'S END はレーティング対象外
		Charts:      charts,
	}
}
