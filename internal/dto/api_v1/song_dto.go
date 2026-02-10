package api_v1

import (
	"encoding/json"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

// V1ChartDTO は外部API v1 用の譜面情報DTOです。
type V1ChartDTO struct {
	Const          chartconstant.ChartConstant `json:"const"`
	IsConstUnknown bool                        `json:"is_const_unknown"`
	Notes          *int                        `json:"notes"`
}

// V1OrderedChartsMap はchartsのキーを特定の順序でJSON出力するためのカスタム型です。
type V1OrderedChartsMap map[string]*V1ChartDTO

// MarshalJSON はJSONマーシャリング時にchartsのキーを
// BASIC→ADVANCED→EXPERT→MASTER→ULTIMAの順序で出力します。
// 譜面が存在しない難易度はnullとして出力されます。
func (o V1OrderedChartsMap) MarshalJSON() ([]byte, error) {
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

// V1SongDTO は外部API v1 用の楽曲情報DTOです。
type V1SongDTO struct {
	DisplayID string             `json:"id"`
	Title     string             `json:"title"`
	Artist    string             `json:"artist"`
	Genre     *string            `json:"genre"`
	BPM       *int               `json:"bpm"`
	Release   *string            `json:"release"`
	Jacket    *string            `json:"jacket"`
	MaxOP     float64            `json:"maxop"`
	Charts    V1OrderedChartsMap `json:"charts"`
}

// V1SongsResponse は外部API v1 用の楽曲一覧レスポンスです。
type V1SongsResponse struct {
	Songs []*V1SongDTO `json:"songs"`
}

// ToV1ChartDTO はChartエンティティから V1ChartDTO へ変換します。
func ToV1ChartDTO(chart *entity.Chart) *V1ChartDTO {
	if chart == nil {
		return nil
	}

	var notesPtr *int
	if chart.Notes != nil {
		notes := int(*chart.Notes)
		notesPtr = &notes
	}

	return &V1ChartDTO{
		Const:          chart.Const,
		IsConstUnknown: chart.IsConstUnknown,
		Notes:          notesPtr,
	}
}

// ToV1SongDTO はSongエンティティから V1SongDTO へ変換します。
// Charts フィールドは空のmapで初期化されます。ハンドラー層で別途設定してください。
func ToV1SongDTO(song *entity.Song, genreNamesByID map[int]string) *V1SongDTO {
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

	return &V1SongDTO{
		DisplayID: song.DisplayID,
		Title:     song.Title,
		Artist:    song.Artist,
		Genre:     genrePtr,
		BPM:       song.BPM,
		Release:   releaseDateStr,
		Jacket:    song.Jacket,
		MaxOP:     calcSongMaxOP(song.Charts),
		Charts:    make(V1OrderedChartsMap),
	}
}

// calcSongMaxOP は楽曲に紐づく全譜面のうち、最も定数が高い譜面で理論値(AJC)を取った際のOPを返します。
func calcSongMaxOP(charts []*entity.Chart) float64 {
	if len(charts) == 0 {
		return 0
	}

	maxChartConst := float64(charts[0].Const)
	for _, chart := range charts[1:] {
		maxChartConst = max(maxChartConst, float64(chart.Const))
	}

	const theoreticalScore = uint32(1010000)
	const allJusticeComboLampID = 3
	return service.CalcSingleOverpower(theoreticalScore, maxChartConst, allJusticeComboLampID)
}
