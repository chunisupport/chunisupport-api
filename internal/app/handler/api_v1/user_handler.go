package api_v1

import (
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
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
	includeNoPlay, _ := strconv.ParseBool(c.QueryParam("include_noplay"))
	result, err := h.userUsecase.GetUserProfileWithRecords(c.Request().Context(), username, requester, includeNoPlay)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	// 既存DTOから V1DTO へ変換
	return c.JSON(http.StatusOK, api_v1.ToV1UserProfileDTO(result))
}
