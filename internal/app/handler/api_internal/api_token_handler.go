package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// APITokenHandler はAPIトークンに関するHTTPリクエストを処理します。
type APITokenHandler struct {
	usecase usecase.APITokenUsecase
}

// NewAPITokenHandler はAPITokenHandlerを生成します。
func NewAPITokenHandler(usecase usecase.APITokenUsecase) *APITokenHandler {
	return &APITokenHandler{usecase: usecase}
}

// Generate はAPIトークンを発行し、プレーントークンを返却します。
func (h *APITokenHandler) Generate(c echo.Context) error {
	user, ok := c.Get("userEntity").(*entity.User)
	if !ok || user == nil {
		return apierror.ErrUnauthorized
	}

	token, err := h.usecase.Generate(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"token": token,
	})
}

// Delete は自分のAPIトークンを削除します。
func (h *APITokenHandler) Delete(c echo.Context) error {
	user, ok := c.Get("userEntity").(*entity.User)
	if !ok || user == nil {
		return apierror.ErrUnauthorized
	}

	if err := h.usecase.Delete(c.Request().Context(), user.ID); err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.NoContent(http.StatusNoContent)
}
