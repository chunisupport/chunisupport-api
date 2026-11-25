package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displaylevel"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
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

// ReplaceCourseTracks は内外のAPIでリクエストの検証と更新処理を統一します。
func ReplaceCourseTracks(c *echo.Context, uc usecase.CourseUsecase) error {
	displayID, apiErr := ValidateDisplayID(c.Param("id"))
	if apiErr != nil {
		return apiErr
	}
	var req dto.ReplaceCourseTracksRequest
	if err := BindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if req.Tracks == nil {
		return apierror.ErrValidationFailed.WithInternal(errors.New("tracks配列は必須です"))
	}
	tracks := make([]entity.CourseTrack, 0, len(req.Tracks))
	for _, item := range req.Tracks {
		filter, err := parseCourseTrackFilter(item.Filter)
		if err != nil {
			return apierror.ErrValidationFailed.WithInternal(err)
		}
		track := entity.CourseTrack{
			Number: item.Number, SelectionType: entity.CourseTrackSelectionType(item.SelectionType),
			Filter: filter,
		}
		for _, chart := range item.Charts {
			track.Charts = append(track.Charts, entity.CourseChartRef{SongID: chart.SongID, Difficulty: chart.Difficulty})
		}
		tracks = append(tracks, track)
	}
	if err := uc.ReplaceTracks(c.Request().Context(), displayID, tracks); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func parseCourseTrackFilter(input *dto.CourseTrackFilterDTO) (*entity.CourseTrackFilter, error) {
	if input == nil {
		return nil, nil
	}
	filter := &entity.CourseTrackFilter{Genres: input.Genres}
	if input.MinLevel != nil {
		level, err := displaylevel.Parse(*input.MinLevel)
		if err != nil {
			return nil, fmt.Errorf("min_level: %w", err)
		}
		filter.MinLevel = &level
	}
	if input.MaxLevel != nil {
		level, err := displaylevel.Parse(*input.MaxLevel)
		if err != nil {
			return nil, fmt.Errorf("max_level: %w", err)
		}
		filter.MaxLevel = &level
	}
	return filter, nil
}
