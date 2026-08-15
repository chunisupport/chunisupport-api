package api_internal

import (
	"net/http"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

type apiTokenNameRequest struct {
	Name string `json:"name"`
}

type apiTokenResponse struct {
	ID          uint64     `json:"id"`
	Name        string     `json:"name"`
	TokenPrefix *string    `json:"token_prefix"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type generatedAPITokenResponse struct {
	*apiTokenResponse
	Token string `json:"token"`
}

type apiTokensResponse struct {
	Tokens []*apiTokenResponse `json:"tokens"`
}

// APITokenHandler はAPIトークンに関するHTTPリクエストを処理します。
type APITokenHandler struct {
	usecase usecase.APITokenUsecase
}

// NewAPITokenHandler はAPITokenHandlerを生成します。
func NewAPITokenHandler(usecase usecase.APITokenUsecase) *APITokenHandler {
	return &APITokenHandler{usecase: usecase}
}

// List は自分が所有するAPIトークンの管理情報を返します。
func (h *APITokenHandler) List(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	tokens, err := h.usecase.List(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	items := make([]*apiTokenResponse, 0, len(tokens))
	for _, token := range tokens {
		items = append(items, toAPITokenResponse(token))
	}
	return c.JSON(http.StatusOK, &apiTokensResponse{Tokens: items})
}

// Generate は名前付きAPIトークンを発行し、平文を一度だけ返します。
func (h *APITokenHandler) Generate(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	var request apiTokenNameRequest
	if err := apphandler.BindStrictJSON(c, &request); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	generated, err := h.usecase.Generate(c.Request().Context(), user.ID, request.Name)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusCreated, &generatedAPITokenResponse{
		apiTokenResponse: toAPITokenResponse(generated.Metadata),
		Token:            generated.Token,
	})
}

// Rename は自分が所有するAPIトークンの名前を変更します。
func (h *APITokenHandler) Rename(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	var request apiTokenNameRequest
	if err := apphandler.BindStrictJSON(c, &request); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	token, err := h.usecase.Rename(c.Request().Context(), user.ID, c.Param("id"), request.Name)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toAPITokenResponse(token))
}

// Delete は自分が所有するAPIトークンをID指定で削除します。
func (h *APITokenHandler) Delete(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	if err := h.usecase.Delete(c.Request().Context(), user.ID, c.Param("id")); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func toAPITokenResponse(token *usecase.APITokenOutput) *apiTokenResponse {
	return &apiTokenResponse{
		ID:          token.ID,
		Name:        token.Name,
		TokenPrefix: token.TokenPrefix,
		LastUsedAt:  token.LastUsedAt,
		CreatedAt:   token.CreatedAt,
	}
}
