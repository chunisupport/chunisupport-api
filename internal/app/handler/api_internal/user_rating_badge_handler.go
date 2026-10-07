package api_internal

import (
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
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

	return c.JSON(http.StatusOK, newBadgeResponse(info.RatingBadgeLabel, rating, 2, "", ratingBadgeColor(rating)))
}

// GetCalculatedRatingBadge は保存済みの計算RATINGを小数4桁で公開します。
func (h *UserHandler) GetCalculatedRatingBadge(c *echo.Context) error {
	player, err := h.getPublicBadgePlayer(c)
	if err != nil {
		return err
	}
	var rating *float64
	if player != nil {
		rating = player.CalculatedRating
	}
	return c.JSON(http.StatusOK, newBadgeResponse(info.RatingBadgeLabel, rating, 4, "", ratingBadgeColor(rating)))
}

// GetOfficialOverpowerBadge は公式OVER POWERの絶対値をポゼッションの色で公開します。
func (h *UserHandler) GetOfficialOverpowerBadge(c *echo.Context) error {
	return h.getOfficialOverpowerBadge(c, false)
}

// GetOfficialOverpowerPercentBadge は公式OVER POWERの割合をポゼッションの色で公開します。
func (h *UserHandler) GetOfficialOverpowerPercentBadge(c *echo.Context) error {
	return h.getOfficialOverpowerBadge(c, true)
}

func (h *UserHandler) getOfficialOverpowerBadge(c *echo.Context, percent bool) error {
	player, err := h.getPublicBadgePlayer(c)
	if err != nil {
		return err
	}
	var value *float64
	color := info.RatingBadgeNoDataColor
	suffix := ""
	if percent {
		suffix = "%"
	}
	if player != nil {
		value = &player.OfficialOverpower
		if percent {
			value = player.OfficialOverpowerPercent
		}
		color = info.OverpowerBadgePossessionColors[player.PossessionID]
	}
	return c.JSON(http.StatusOK, newBadgeResponse(info.OverpowerBadgeLabel, value, 2, suffix, color))
}

func (h *UserHandler) getPublicBadgePlayer(c *echo.Context) (*entity.Player, error) {
	username, apiErr := handler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return nil, apiErr
	}
	player, err := h.userUsecase.GetPublicBadgePlayer(c.Request().Context(), username)
	if err != nil {
		return nil, h.handleUserProfileError(err, username, "player badge")
	}
	return player, nil
}

func newBadgeResponse(label string, value *float64, precision int, suffix, color string) ratingBadgeResponse {
	badge := ratingBadgeResponse{
		SchemaVersion: info.RatingBadgeSchemaVersion,
		Label:         label,
		Message:       info.RatingBadgeNoDataMessage,
		Color:         info.RatingBadgeNoDataColor,
	}
	if value != nil {
		badge.Message = strconv.FormatFloat(*value, 'f', precision, 64) + suffix
		badge.Color = color
	}
	return badge
}

func ratingBadgeColor(rating *float64) string {
	if rating == nil {
		return info.RatingBadgeNoDataColor
	}
	switch {
	case *rating >= 17.00:
		return info.RatingBadgeRainbowExColor
	case *rating >= 16.00:
		return info.RatingBadgeRainbowColor
	case *rating >= 15.25:
		return info.RatingBadgePlatinumColor
	case *rating >= 14.50:
		return info.RatingBadgeGoldColor
	case *rating >= 13.25:
		return info.RatingBadgeSilverColor
	case *rating >= 12.00:
		return info.RatingBadgeBronzeColor
	case *rating >= 10.00:
		return info.RatingBadgePurpleColor
	case *rating >= 7.00:
		return info.RatingBadgeRedColor
	case *rating >= 4.00:
		return info.RatingBadgeOrangeColor
	default:
		return info.RatingBadgeGreenColor
	}
}
