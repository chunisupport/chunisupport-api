package handler

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

// ToUpdateSongInputs はAPI更新リクエストをユースケース入力へ変換します。
func ToUpdateSongInputs(requests []*api_internal.UpdateSongRequest) []*usecase.UpdateSongInput {
	result := make([]*usecase.UpdateSongInput, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			result = append(result, nil)
			continue
		}
		charts := make(map[string]*usecase.UpdateChartInput, len(request.Charts))
		for difficulty, chart := range request.Charts {
			if chart == nil {
				charts[difficulty] = nil
				continue
			}
			charts[difficulty] = &usecase.UpdateChartInput{
				Const:          chart.Const,
				IsConstUnknown: chart.IsConstUnknown,
				Notes:          chart.Notes,
				NotesDesigner:  chart.NotesDesigner,
			}
		}
		var releasedAt *time.Time
		if request.ReleasedAt != nil {
			value := request.ReleasedAt.Time
			releasedAt = &value
		}
		result = append(result, &usecase.UpdateSongInput{
			DisplayID: request.DisplayID, Title: request.Title, Reading: request.Reading,
			Artist: request.Artist, Genre: request.Genre, BPM: request.BPM, ReleasedAt: releasedAt,
			Jacket: request.Jacket, IsNew: request.IsNew, Charts: charts,
		})
	}
	return result
}
