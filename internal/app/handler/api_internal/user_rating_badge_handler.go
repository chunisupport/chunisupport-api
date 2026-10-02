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
		switch {
		case *rating >= 17.00:
			badge.Color = info.RatingBadgeRainbowExColor
		case *rating >= 16.00:
			badge.Color = info.RatingBadgeRainbowColor
		case *rating >= 15.25:
			badge.Color = info.RatingBadgePlatinumColor
		case *rating >= 14.50:
			badge.Color = info.RatingBadgeGoldColor
		case *rating >= 13.25:
			badge.Color = info.RatingBadgeSilverColor
		case *rating >= 12.00:
			badge.Color = info.RatingBadgeBronzeColor
		case *rating >= 10.00:
			badge.Color = info.RatingBadgePurpleColor
		case *rating >= 7.00:
			badge.Color = info.RatingBadgeRedColor
		case *rating >= 4.00:
			badge.Color = info.RatingBadgeOrangeColor
		default:
			badge.Color = info.RatingBadgeGreenColor
		}
	}
	return c.JSON(http.StatusOK, badge)
}
