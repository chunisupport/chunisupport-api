package chunirec

import (
	"math"

	domainmasterdata "github.com/Qman110101/chunisupport-api/internal/domain/masterdata"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
)

// MusicShowAllResponse は全楽曲情報のレスポンスを表します
type MusicShowAllResponse []*MusicItemDTO

// MusicItemDTO は個々の楽曲情報を表します
type MusicItemDTO struct {
	Meta MusicMetaDTO `json:"meta"`
	Data MusicDataDTO `json:"data"`
}

// MusicMetaDTO は楽曲のメタデータを表します
type MusicMetaDTO struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Genre   *string  `json:"genre"`
	Artist  string   `json:"artist"`
	Release *string  `json:"release"`
	BPM     *float64 `json:"bpm"`
}

// MusicDataDTO は楽曲の譜面データを表します
type MusicDataDTO struct {
	BAS *ChartDataDTO `json:"BAS,omitempty"`
	ADV *ChartDataDTO `json:"ADV,omitempty"`
	EXP *ChartDataDTO `json:"EXP,omitempty"`
	MAS *ChartDataDTO `json:"MAS,omitempty"`
	ULT *ChartDataDTO `json:"ULT,omitempty"`
}

// ChartDataDTO は個々の譜面情報を表します
type ChartDataDTO struct {
	Level          float64 `json:"level"`
	Const          float64 `json:"const"`
	MaxCombo       *int    `json:"maxcombo"`
	IsConstUnknown bool    `json:"is_const_unknown"`
}

// ToMusicShowAllResponse はドメインエンティティのリストをDTOに変換します
func ToMusicShowAllResponse(songs []*repository.SongWithCharts, masters *domainmasterdata.SongMasters) MusicShowAllResponse {
	response := make(MusicShowAllResponse, 0, len(songs))
	var genres map[int]string
	if masters != nil {
		genres = masters.GenreNamesByID
	}

	for _, s := range songs {
		item := &MusicItemDTO{
			Meta: MusicMetaDTO{
				ID:      s.Song.DisplayID,
				Title:   s.Song.Title,
				Artist:  s.Song.Artist,
				Release: nil,
				BPM:     nil,
			},
			Data: MusicDataDTO{},
		}

		// Nullable fields handling
		if s.Song.GenreID != nil {
			if genreName, ok := genres[*s.Song.GenreID]; ok {
				item.Meta.Genre = &genreName
			}
		}
		if s.Song.ReleasedAt != nil {
			dateStr := s.Song.ReleasedAt.Format("2006-01-02")
			item.Meta.Release = &dateStr
		}
		if s.Song.BPM != nil {
			bpmVal := float64(*s.Song.BPM)
			item.Meta.BPM = &bpmVal
		}

		// Charts handling
		for _, c := range s.Charts {
			chartDTO := &ChartDataDTO{
				Const:          float64(c.Const),
				IsConstUnknown: c.IsConstUnknown,
				Level:          calculateLevel(float64(c.Const)),
			}

			if c.Notes != nil {
				maxCombo := int(*c.Notes)
				chartDTO.MaxCombo = &maxCombo
			}

			switch c.DifficultyID {
			case 1: // Basic
				item.Data.BAS = chartDTO
			case 2: // Advanced
				item.Data.ADV = chartDTO
			case 3: // Expert
				item.Data.EXP = chartDTO
			case 4: // Master
				item.Data.MAS = chartDTO
			case 5: // Ultima
				item.Data.ULT = chartDTO
			}
		}

		response = append(response, item)
	}

	return response
}

// calculateLevel は定数から表記レベルを計算します
// .0 ~ .4 -> .0
// .5 ~ .9 -> .5
func calculateLevel(constant float64) float64 {
	intPart := math.Floor(constant)
	fracPart := constant - intPart

	if fracPart >= 0.5 {
		return intPart + 0.5
	}
	return intPart
}
