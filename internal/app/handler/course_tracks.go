package handler

import (
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

func ToCourseTracksDTO(tracks []usecase.CourseTrackOutput) []dto.CourseTrackDTO {
	if tracks == nil {
		return nil
	}
	result := make([]dto.CourseTrackDTO, 0, len(tracks))
	for _, track := range tracks {
		item := dto.CourseTrackDTO{Number: track.Number, SelectionType: string(track.SelectionType)}
		if track.Filter != nil {
			item.Filter = &dto.CourseTrackFilterDTO{Genres: track.Filter.Genres}
			if track.Filter.MinLevel != nil {
				item.Filter.MinLevel = new(track.Filter.MinLevel.String())
			}
			if track.Filter.MaxLevel != nil {
				item.Filter.MaxLevel = new(track.Filter.MaxLevel.String())
			}
		}
		for _, chart := range track.Charts {
			item.Charts = append(item.Charts, dto.CourseChartRefDTO{SongID: chart.SongID, Difficulty: chart.Difficulty})
		}
		if track.Candidates != nil {
			item.Candidates = make([]dto.CourseChartCandidateDTO, 0, len(track.Candidates))
			for _, candidate := range track.Candidates {
				item.Candidates = append(item.Candidates, dto.CourseChartCandidateDTO{
					CourseChartRefDTO: dto.CourseChartRefDTO{SongID: candidate.SongID, Difficulty: candidate.Difficulty},
					Title:             candidate.Title, Const: candidate.Const, IsConstUnknown: candidate.IsConstUnknown, Jacket: candidate.Jacket,
					Attribute: candidate.Attribute, LevelStar: candidate.LevelStar, Notes: candidate.Notes,
				})
			}
		}
		result = append(result, item)
	}
	return result
}
