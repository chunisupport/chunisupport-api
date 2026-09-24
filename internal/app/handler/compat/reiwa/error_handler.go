package reiwa

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
)

func ReiwaErrorHandlerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			err := next(c)
			if err != nil {
				handleReiwaError(err, c)
				return nil
			}
			return nil
		}
	}
}

func handleReiwaError(err error, c *echo.Context) {
	var httpStatus int
	var additionalMessage string

	if response, _ := echo.UnwrapResponse(c.Response()); response != nil && response.Committed {
		return
	}

	// クライアント切断はサーバー障害ではないため、互換形式のエラー応答は返さずに記録だけ残します。
	if errors.Is(err, context.Canceled) {
		slog.Info("Reiwa HTTP request canceled by client",
			"method", c.Request().Method,
			"path", c.Request().URL.Path,
			"remote_addr", c.RealIP(),
			"status", info.StatusClientClosedRequest,
			"error", sanitizeLogValue(err.Error()),
		)
		if err := c.NoContent(info.StatusClientClosedRequest); err != nil {
			slog.Debug("Failed to send reiwa client closed response", "error", err)
		}
		return
	}

	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) {
		httpStatus = apiErr.HTTPStatus
		additionalMessage = ""
	} else if he, ok := err.(*echo.HTTPError); ok {
		httpStatus = he.Code
		additionalMessage = ""
	} else {
		httpStatus = http.StatusServiceUnavailable
		additionalMessage = ""
	}

	logReiwaError(httpStatus, err, c)

	var errorResponse ReiwaErrorResponse
	switch httpStatus {
	case http.StatusBadRequest:
		errorResponse = NewBadRequestError(additionalMessage)
	case http.StatusNotFound:
		errorResponse = NewNotFoundError(additionalMessage)
	case http.StatusMethodNotAllowed:
		errorResponse = NewMethodNotAllowedError(additionalMessage)
	case http.StatusTooManyRequests:
		errorResponse = NewTooManyRequestsError(additionalMessage)
	case http.StatusServiceUnavailable:
		errorResponse = NewServiceUnavailableError(additionalMessage)
	default:
		httpStatus = http.StatusServiceUnavailable
		errorResponse = NewServiceUnavailableError(additionalMessage)
	}

	if err := c.JSON(httpStatus, errorResponse); err != nil {
		slog.Error("Failed to send reiwa error response", "error", err)
	}
}

func logReiwaError(status int, err error, c *echo.Context) {
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) && apiErr.Code == apierror.CodeMaintenanceMode {
		return
	}

	errorMessage := sanitizeLogValue(err.Error())
	logger := slog.With("method", c.Request().Method, "path", c.Request().URL.Path, "remote_addr", c.RealIP())
	if status >= 500 {
		logger.Error("Reiwa HTTP error",
			"status", status,
			"error", errorMessage,
		)
		return
	}
	logger.Warn("Reiwa HTTP client error",
		"status", status,
		"error", errorMessage,
	)
}

func sanitizeLogValue(value string) string {
	replacer := strings.NewReplacer("\n", " ", "\r", " ")
	return replacer.Replace(value)
}
