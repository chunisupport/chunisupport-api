package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/httpheader"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

type loginRequest struct {
	TurnstileToken string `json:"turnstile_token" validate:"required"`
}

// LoginHandler はTurnstile検証付きのログインを処理します。
type LoginHandler struct {
	loginUsecase usecase.LoginUsecase
}

// NewLoginHandler はLoginHandlerを生成します。
func NewLoginHandler(loginUsecase usecase.LoginUsecase) *LoginHandler {
	return &LoginHandler{loginUsecase: loginUsecase}
}

// Login はBearerのFirebase IDトークンとTurnstileトークンでログインを検証します。
func (h *LoginHandler) Login(c *echo.Context) error {
	req := new(loginRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	idToken := httpheader.ExtractBearerToken(c.Request().Header)
	if idToken == "" {
		return apierror.ErrMissingToken
	}

	user, err := h.loginUsecase.Login(c.Request().Context(), idToken, req.TurnstileToken, c.RealIP())
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, user)
}
