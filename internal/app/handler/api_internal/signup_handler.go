package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/httpheader"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

type signupRequest struct {
	Username string `json:"username" validate:"username"`
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

	idToken := httpheader.ExtractBearerToken(c.Request().Header)
	if idToken == "" {
		return apierror.ErrMissingToken
	}

	user, err := h.signupUsecase.Signup(c.Request().Context(), idToken, req.Username)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusCreated, user)
}
