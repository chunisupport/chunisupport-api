package api_internal

import (
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/labstack/echo/v5"
)

func (h *CourseHandler) ReplaceTracks(c *echo.Context) error {
	return handler.ReplaceCourseTracks(c, h.usecase)
}
