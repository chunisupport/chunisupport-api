package api_internal

import (
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
)

type ratingBadgeResponse struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
}

// GetOfficialRatingBadge は認証情報を外部サービスへ渡さずに表示できるよう公開プロフィールの公式RATINGを返します。
func (h *UserHandler) GetOfficialRatingBadge(c *echo.Context) error {
	username, apiErr := handler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}

	// バッジは第三者サービスから認証なしで取得されるため、常に匿名閲覧の権限で判定します。
	rating, err := h.userUsecase.GetPublicOfficialRating(c.Request().Context(), username)
	if err != nil {
		return h.handleUserProfileError(err, username, "official rating badge")
	}

	badge := ratingBadgeResponse{
		SchemaVersion: info.RatingBadgeSchemaVersion,
		Label:         info.RatingBadgeLabel,
		Message:       info.RatingBadgeNoDataMessage,
		Color:         info.RatingBadgeNoDataColor,
	}
	if rating != nil {
		badge.Message = strconv.FormatFloat(*rating, 'f', 2, 64)
		badge.Color = info.RatingBadgeColor
	}
	return c.JSON(http.StatusOK, badge)
}
