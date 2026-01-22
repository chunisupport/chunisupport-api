package api_v1

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Qman110101/chunisupport-api/internal/app/apierror"
	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/dto/api_v1"
	"github.com/Qman110101/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// V1UserHandler は外部API v1 のユーザー関連エンドポイントを処理します。
type V1UserHandler struct {
	userUsecase usecase.UserUsecase
}

// NewV1UserHandler は新しい V1UserHandler を生成します。
func NewV1UserHandler(userUsecase usecase.UserUsecase) *V1UserHandler {
	return &V1UserHandler{userUsecase: userUsecase}
}

// GetUser は指定された username のユーザープロファイルを取得します。
func (h *V1UserHandler) GetUser(c echo.Context) error {
	username := c.Param("username")
	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}
	result, err := h.userUsecase.GetUserProfileWithRecords(c.Request().Context(), username, requester)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound):
			return apierror.ErrUserNotFound
		case errors.Is(err, usecase.ErrUserPrivate):
			// セキュリティ: 非公開と未発見を区別しない
			return apierror.ErrUserNotFound
		case errors.Is(err, usecase.ErrPlayerNotLinked):
			// セキュリティ: プレイヤー未紐付も404で隠蔽
			return apierror.ErrUserNotFound
		default:
			slog.Error("failed to get user profile", "username", username, "error", err)
			return apierror.ErrInternalError.WithInternal(err)
		}
	}

	// 既存DTOから V1DTO へ変換
	return c.JSON(http.StatusOK, api_v1.ToV1UserProfileDTO(result))
}
