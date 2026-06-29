package api_internal

import (
	"net/http"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

type apiTokenStatusResponse struct {
	HasToken  bool       `json:"has_token"`
	CreatedAt *time.Time `json:"created_at"`
}

// APITokenHandler はAPIトークンに関するHTTPリクエストを処理します。
type APITokenHandler struct {
	usecase usecase.APITokenUsecase
}

// NewAPITokenHandler はAPITokenHandlerを生成します。
func NewAPITokenHandler(usecase usecase.APITokenUsecase) *APITokenHandler {
	return &APITokenHandler{usecase: usecase}
}

// GetStatus は自分のAPIトークンの発行状態を返します。
func (h *APITokenHandler) GetStatus(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	token, err := h.usecase.GetStatus(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	if token == nil {
		return c.JSON(http.StatusOK, apiTokenStatusResponse{
			HasToken:  false,
			CreatedAt: nil,
		})
	}

	createdAt := token.CreatedAt
	return c.JSON(http.StatusOK, apiTokenStatusResponse{
		HasToken:  true,
		CreatedAt: &createdAt,
	})
}

// Generate はAPIトークンを発行し、プレーントークンを返却します。
func (h *APITokenHandler) Generate(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	token, err := h.usecase.Generate(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"token": token,
	})
}

// Delete は自分のAPIトークンを削除します。
func (h *APITokenHandler) Delete(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	if err := h.usecase.Delete(c.Request().Context(), user.ID); err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
