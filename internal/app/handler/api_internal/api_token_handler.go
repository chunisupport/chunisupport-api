package api_internal

import (
	"net/http"
	"strconv"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

type apiTokenResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type apiTokenListResponse struct {
	Tokens []apiTokenResponse `json:"tokens"`
	Limit  int                `json:"limit"`
}

type apiTokenGenerateRequest struct {
	Name string `json:"name" validate:"api_token_name"`
}

type apiTokenGenerateResponse struct {
	Token string           `json:"token"`
	Item  apiTokenResponse `json:"item"`
}

// APITokenHandler はAPIトークンに関するHTTPリクエストを処理します。
type APITokenHandler struct {
	usecase usecase.APITokenUsecase
}

// NewAPITokenHandler はAPITokenHandlerを生成します。
func NewAPITokenHandler(usecase usecase.APITokenUsecase) *APITokenHandler {
	return &APITokenHandler{usecase: usecase}
}

// List は自分のAPIトークン一覧を返します。
func (h *APITokenHandler) List(c echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	tokens, err := h.usecase.List(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	items := make([]apiTokenResponse, 0, len(tokens))
	for _, token := range tokens {
		items = append(items, toAPITokenResponse(token))
	}

	return c.JSON(http.StatusOK, apiTokenListResponse{
		Tokens: items,
		Limit:  usecase.APITokenMaxCountPerUser,
	})
}

// Generate はAPIトークンを発行し、プレーントークンを返却します。
func (h *APITokenHandler) Generate(c echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	var req apiTokenGenerateRequest
	if err := apphandler.BindOptionalStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	rawToken, token, err := h.usecase.Generate(c.Request().Context(), user.ID, req.Name)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, apiTokenGenerateResponse{
		Token: rawToken,
		Item:  toAPITokenResponse(token),
	})
}

// Delete は自分の指定APIトークンを削除します。
func (h *APITokenHandler) Delete(c echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	tokenID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || tokenID <= 0 {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	if err := h.usecase.Delete(c.Request().Context(), user.ID, tokenID); err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteAll は自分のAPIトークンをすべて削除します。
func (h *APITokenHandler) DeleteAll(c echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	if err := h.usecase.DeleteAll(c.Request().Context(), user.ID); err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func toAPITokenResponse(token *entity.APIToken) apiTokenResponse {
	return apiTokenResponse{
		ID:        token.ID,
		Name:      token.Name,
		CreatedAt: token.CreatedAt,
	}
}
