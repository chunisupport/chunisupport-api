package api_internal

import (
	"net/http"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

type signupRequest struct {
	Username string `json:"username" validate:"required,username"`
}

// SignupHandler は Firebase Bearer トークンを用いた初回登録を処理します。
type SignupHandler struct {
	signupUsecase usecase.SignupUsecase
}

// NewSignupHandler は SignupHandler を生成します。
func NewSignupHandler(signupUsecase usecase.SignupUsecase) *SignupHandler {
	return &SignupHandler{signupUsecase: signupUsecase}
}

// Signup は Bearer の Firebase ID トークンでアプリ内ユーザーを作成します。
func (h *SignupHandler) Signup(c echo.Context) error {
	req := new(signupRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	idToken := extractSignupBearerToken(c)
	if idToken == "" {
		return apierror.ErrMissingToken
	}

	user, err := h.signupUsecase.Signup(c.Request().Context(), idToken, req.Username)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusCreated, user)
}

func extractSignupBearerToken(c echo.Context) string {
	authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
	if authHeader == "" {
		return ""
	}

	scheme, token, found := strings.Cut(authHeader, " ")
	if !found || !strings.EqualFold(strings.TrimSpace(scheme), "Bearer") {
		return ""
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}

	return token
}
