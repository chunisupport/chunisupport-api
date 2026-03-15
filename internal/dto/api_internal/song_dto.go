package api_internal

import (
	"encoding/json"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

// ChartDTO は譜面情報を外部に公開するためのDTOです。
type ChartDTO struct {
	Const          chartconstant.ChartConstant `json:"const"`
	IsConstUnknown bool                        `json:"is_const_unknown"`
	Notes          *int                        `json:"notes"`
}

// OrderedChartsMap はchartsのキーを特定の順序でJSON出力するためのカスタム型です。
type OrderedChartsMap map[string]*ChartDTO

// MarshalJSON はJSONマーシャリング時にchartsのキーを
// BASIC→ADVANCED→EXPERT→MASTER→ULTIMAの順序で出力します。
// 譜面が存在しない難易度はnullとして出力されます。
func (o OrderedChartsMap) MarshalJSON() ([]byte, error) {
	// 難易度の順序を定義（大文字で統一）
	orderedKeys := []string{"BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA"}

	// 手動でJSONを構築して順序を保証
	var jsonParts []string
	jsonParts = append(jsonParts, "{")

	first := true
	for _, key := range orderedKeys {
		if !first {
			jsonParts = append(jsonParts, ",")
		}
		first = false

		// キーを追加
		jsonParts = append(jsonParts, `"`+key+`":`)

		// 値をマーシャル（存在しない場合はnull）
		if chart, exists := o[key]; exists && chart != nil {
			chartJSON, err := json.Marshal(chart)
			if err != nil {
				return nil, err
			}
			jsonParts = append(jsonParts, string(chartJSON))
		} else {
			jsonParts = append(jsonParts, "null")
		}
	}

	jsonParts = append(jsonParts, "}")

	result := ""
	for _, part := range jsonParts {
		result += part
	}

	return []byte(result), nil
}

// SongDTO は楽曲情報を外部に公開するためのDTOです。
type SongDTO struct {
	DisplayID      string           `json:"id"`
	Title          string           `json:"title"`
	Artist         string           `json:"artist"`
	Genre          *string          `json:"genre"`
	BPM            *int             `json:"bpm"`
	Release        *string          `json:"release"`
	Jacket         *string          `json:"jacket"`
	OfficialIdx    string           `json:"official_idx"`
	MaxOP          float64          `json:"maxop"`
	IsMaxOPUnknown bool             `json:"is_maxop_unknown"`
	Charts         OrderedChartsMap `json:"charts"`
}

// SongsResponse は楽曲一覧のレスポンスを表します。
type SongsResponse struct {
	Songs []*SongDTO `json:"songs"`
}

// AdminSongDTO は管理者向けの楽曲情報DTOです。
type AdminSongDTO struct {
	*SongDTO
	IsDeleted bool `json:"is_deleted"`
}

// AdminSongsResponse は管理者向け楽曲一覧のレスポンスを表します。
type AdminSongsResponse struct {
	Songs []*AdminSongDTO `json:"songs"`
}

// UpdateChartRequest は譜面更新リクエストを表します。
type UpdateChartRequest struct {
	Const          float64 `json:"const" validate:"gte=0"`
	IsConstUnknown bool    `json:"is_const_unknown"`
	Notes          *int    `json:"notes" validate:"omitempty,gte=0"`
}

// UpdateSongRequest は楽曲更新リクエストを表します。
type UpdateSongRequest struct {
	DisplayID  string                         `json:"id" validate:"required,len=16,hexadecimal,lowercase"`
	Title      string                         `json:"title" validate:"required"`
	Artist     string                         `json:"artist" validate:"required"`
	Genre      *string                        `json:"genre"`
	BPM        *int                           `json:"bpm" validate:"omitempty,gt=0"`
	ReleasedAt *DateOnly                      `json:"released_at"`
	Jacket     *string                        `json:"jacket"`
	Charts     map[string]*UpdateChartRequest `json:"charts" validate:"dive"`
}

// ToChartDTO はChartエンティティからChartDTOへ変換します。
func ToChartDTO(chart *entity.Chart) *ChartDTO {
	if chart == nil {
		return nil
	}

	var notesPtr *int
	if chart.Notes != nil {
		notes := int(*chart.Notes)
		notesPtr = &notes
	}

	return &ChartDTO{
		Const:          chart.Const,
		IsConstUnknown: chart.IsConstUnknown,
		Notes:          notesPtr,
	}
}

// ToSongDTO はSongエンティティからSongDTOへ変換します。
// genreNamesByID を使用してジャンルIDを名称に変換します。
// maxOP は呼び出し元で計算済みの値を受け取ります。
// Charts フィールドは空のmapで初期化されます。ハンドラー層で別途設定してください。
func ToSongDTO(song *entity.Song, genreNamesByID map[int]string, maxOP float64) *SongDTO {
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

	return &SongDTO{
		DisplayID:      song.DisplayID,
		Title:          song.Title,
		Artist:         song.Artist,
		Genre:          genrePtr,
		BPM:            song.BPM,
		Release:        releaseDateStr,
		Jacket:         song.Jacket,
		OfficialIdx:    song.OfficialIdx,
		MaxOP:          maxOP,
		IsMaxOPUnknown: song.IsMaxOPUnknown,
		Charts:         make(OrderedChartsMap),
	}
}
