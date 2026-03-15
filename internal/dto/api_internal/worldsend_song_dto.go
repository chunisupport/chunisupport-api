package api_internal

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto"
)

// WorldsendChartDTO は WORLD'S END 譜面情報を外部に公開するためのDTOです。
type WorldsendChartDTO struct {
	Attribute *string `json:"attribute"`  // WORLD'S END 属性（光、蔵、改、狂、etc.）
	LevelStar *int    `json:"level_star"` // WORLD'S END レベル（1～5）
	Notes     *int    `json:"notes"`      // ノーツ数
}

// WorldsendSongDTO は WORLD'S END 楽曲情報を外部に公開するためのDTOです。
// WORLD'S END はレーティング対象外のため、charts は "WORLDSEND" キーのみを持ち maxop フィールドは存在しません。
type WorldsendSongDTO struct {
	DisplayID   string                        `json:"id"`
	Title       string                        `json:"title"`
	Artist      string                        `json:"artist"`
	Genre       *string                       `json:"genre"`
	BPM         *int                          `json:"bpm"`
	Release     *string                       `json:"release"`
	Jacket      *string                       `json:"jacket"`
	OfficialIdx string                        `json:"official_idx"`
	Charts      map[string]*WorldsendChartDTO `json:"charts"`
}

// WorldsendSongsResponse は WORLD'S END 楽曲一覧のレスポンスを表します。
type WorldsendSongsResponse struct {
	Songs []*WorldsendSongDTO `json:"songs"`
}

// AdminWorldsendSongDTO は管理者向けの WORLD'S END 楽曲情報DTOです。
type AdminWorldsendSongDTO struct {
	*WorldsendSongDTO
	IsDeleted bool `json:"is_deleted"`
}

// AdminWorldsendSongsResponse は管理者向け WORLD'S END 楽曲一覧のレスポンスを表します。
type AdminWorldsendSongsResponse struct {
	Songs []*AdminWorldsendSongDTO `json:"songs"`
}

// UpdateWorldsendChartRequest は WORLD'S END 譜面更新リクエストを表します。
type UpdateWorldsendChartRequest struct {
	Attribute *string `json:"attribute"`
	LevelStar *int    `json:"level_star" validate:"omitempty,min=1,max=5"`
	Notes     *int    `json:"notes" validate:"omitempty,gte=0"`
}

// UpdateWorldsendSongRequest は WORLD'S END 楽曲更新リクエストを表します。
type UpdateWorldsendSongRequest struct {
	DisplayID  string                                  `json:"id" validate:"required,len=16,hexadecimal,lowercase"`
	Title      string                                  `json:"title" validate:"required"`
	Artist     string                                  `json:"artist" validate:"required"`
	Genre      *string                                 `json:"genre"`
	BPM        *int                                    `json:"bpm" validate:"omitempty,gt=0"`
	ReleasedAt *DateOnly                               `json:"released_at"`
	Jacket     *string                                 `json:"jacket"`
	Charts     map[string]*UpdateWorldsendChartRequest `json:"charts" validate:"dive"`
}

// ToWorldsendChartDTO は WorldsendChart エンティティから WorldsendChartDTO へ変換します。
func ToWorldsendChartDTO(chart *entity.WorldsendChart) *WorldsendChartDTO {
	if chart == nil {
		return nil
	}

	return &WorldsendChartDTO{
		Attribute: chart.Attribute,
		LevelStar: dto.ToLevelStarIntPtr(chart.LevelStar),
		Notes:     dto.ToNotesIntPtr(chart.Notes),
	}
}

// ToWorldsendSongDTO は Song エンティティと WorldsendChart エンティティから WorldsendSongDTO へ変換します。
// genreNamesByID を使用してジャンルIDを名称に変換します。
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
		Charts:      charts,
	}
}
