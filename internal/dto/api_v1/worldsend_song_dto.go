package api_v1

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto"
)

// V1WorldsendChartDTO は外部API v1 用の WORLD'S END 譜面情報DTOです。
type V1WorldsendChartDTO struct {
	Attribute *string `json:"attribute"`  // WORLD'S END 属性（光、蔵、改、狂、etc.）
	LevelStar *int    `json:"level_star"` // WORLD'S END レベル（1～5）
	Notes     *int    `json:"notes"`      // ノーツ数
}

// V1WorldsendSongDTO は外部API v1 用の WORLD'S END 楽曲情報DTOです。
// WORLD'S END はレーティング対象外のため、charts は "WORLDSEND" キーのみを持ち maxop フィールドは存在しません。
type V1WorldsendSongDTO struct {
	DisplayID   string                          `json:"id"`
	Title       string                          `json:"title"`
	Artist      string                          `json:"artist"`
	Genre       *string                         `json:"genre"`
	BPM         *int                            `json:"bpm"`
	Release     *string                         `json:"release"`
	Jacket      *string                         `json:"jacket"`
	OfficialIdx string                          `json:"official_idx"`
	Charts      map[string]*V1WorldsendChartDTO `json:"charts"`
}

// V1WorldsendSongsResponse は外部API v1 用の WORLD'S END 楽曲一覧レスポンスです。
type V1WorldsendSongsResponse struct {
	Songs []*V1WorldsendSongDTO `json:"songs"`
}

// ToV1WorldsendChartDTO は WorldsendChart エンティティから V1WorldsendChartDTO へ変換します。
func ToV1WorldsendChartDTO(chart *entity.WorldsendChart) *V1WorldsendChartDTO {
	if chart == nil {
		return nil
	}

	return &V1WorldsendChartDTO{
		Attribute: chart.Attribute,
		LevelStar: chart.LevelStar,
		Notes:     dto.ToNotesIntPtr(chart.Notes),
	}
}

// ToV1WorldsendSongDTO は Song エンティティと WorldsendChart エンティティから V1WorldsendSongDTO へ変換します。
// genreNamesByID を使用してジャンルIDを名称に変換します。
func ToV1WorldsendSongDTO(song *entity.Song, chart *entity.WorldsendChart, genreNamesByID map[int]string) *V1WorldsendSongDTO {
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

	charts := make(map[string]*V1WorldsendChartDTO)
	charts["WORLDSEND"] = ToV1WorldsendChartDTO(chart)

	return &V1WorldsendSongDTO{
		DisplayID:   song.DisplayID,
		Title:       song.Title,
		Artist:      song.Artist,
		Genre:       genrePtr,
		BPM:         song.BPM,
		Release:     releaseDateStr,
		Jacket:      song.Jacket,
		OfficialIdx: song.OfficialIdx,
		Charts:      charts,
	}
}
