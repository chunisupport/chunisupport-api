package dto

import (
	"encoding/json"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

type CourseTrackDTO struct {
	Number        int                       `json:"track_no"`
	SelectionType string                    `json:"selection_type"`
	Filter        *CourseTrackFilterDTO     `json:"filter,omitempty"`
	Charts        []CourseChartRefDTO       `json:"charts,omitempty"`
	Candidates    []CourseChartCandidateDTO `json:"candidates,omitzero"`
}

type CourseTrackFilterDTO struct {
	MinLevel *string  `json:"min_level,omitempty"`
	MaxLevel *string  `json:"max_level,omitempty"`
	Genres   []string `json:"genres,omitempty"`
}

type CourseChartRefDTO struct {
	SongID     string `json:"id"`
	Difficulty string `json:"difficulty"`
}

type CourseChartCandidateDTO struct {
	CourseChartRefDTO
	Title          string  `json:"title"`
	Const          float64 `json:"const"`
	IsConstUnknown bool    `json:"is_const_unknown"`
	Jacket         *string `json:"jacket"`
	Attribute      *string `json:"attribute"`
	LevelStar      *int    `json:"level_star"`
	Notes          *int    `json:"notes"`
}

// MarshalJSON はWORLD'S END候補だけ通常譜面の定数項目を除き、専用メタデータをnullを含めて出力します。
// 難易度は既存の公開JSONキーをそのまま判定に使い、DTOにAPI内部用の種別フィールドを持ち込まないようにします。
func (c CourseChartCandidateDTO) MarshalJSON() ([]byte, error) {
	if c.Difficulty == entity.CourseTrackDifficultyWorldsend {
		return json.Marshal(struct {
			CourseChartRefDTO
			Title     string  `json:"title"`
			Jacket    *string `json:"jacket"`
			Attribute *string `json:"attribute"`
			LevelStar *int    `json:"level_star"`
			Notes     *int    `json:"notes"`
		}{
			CourseChartRefDTO: c.CourseChartRefDTO,
			Title:             c.Title,
			Jacket:            c.Jacket,
			Attribute:         c.Attribute,
			LevelStar:         c.LevelStar,
			Notes:             c.Notes,
		})
	}

	return json.Marshal(struct {
		CourseChartRefDTO
		Title          string  `json:"title"`
		Const          float64 `json:"const"`
		IsConstUnknown bool    `json:"is_const_unknown"`
		Jacket         *string `json:"jacket"`
	}{
		CourseChartRefDTO: c.CourseChartRefDTO,
		Title:             c.Title,
		Const:             c.Const,
		IsConstUnknown:    c.IsConstUnknown,
		Jacket:            c.Jacket,
	})
}
