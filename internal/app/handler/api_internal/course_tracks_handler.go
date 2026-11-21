package api_internal

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displaylevel"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/labstack/echo/v5"
)

func (h *CourseHandler) ReplaceTracks(c *echo.Context) error {
	displayID, apiErr := handler.ValidateDisplayID(c.Param("id"))
	if apiErr != nil {
		return apiErr
	}
	var req internaldto.ReplaceCourseTracksRequest
	if err := handler.BindStrictJSON(c, &req); err != nil {
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
	if err := h.usecase.ReplaceTracks(c.Request().Context(), displayID, tracks); err != nil {
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
