package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

type firebaseRequest struct {
	IDToken string `json:"id_token" validate:"required"`
}

type firebaseRegisterRequest struct {
	IDToken  string `json:"id_token" validate:"required"`
	Username string `json:"username" validate:"required,username"`
}

// FirebaseHandler は Firebase ログイン・連携・登録リクエストを処理します。
type FirebaseHandler struct {
	firebaseLinkUsecase     usecase.FirebaseLinkUsecase
	firebaseLoginUsecase    usecase.FirebaseLoginUsecase
	firebaseRegisterUsecase usecase.FirebaseRegisterUsecase
	cookieSecure            bool
	cookieSameSite          http.SameSite
}

// NewFirebaseHandler は FirebaseHandler を生成します。
func NewFirebaseHandler(firebaseLinkUsecase usecase.FirebaseLinkUsecase, firebaseLoginUsecase usecase.FirebaseLoginUsecase, firebaseRegisterUsecase usecase.FirebaseRegisterUsecase, cookieSecure bool, cookieSameSite http.SameSite) *FirebaseHandler {
	return &FirebaseHandler{
		firebaseLinkUsecase:     firebaseLinkUsecase,
		firebaseLoginUsecase:    firebaseLoginUsecase,
		firebaseRegisterUsecase: firebaseRegisterUsecase,
		cookieSecure:            cookieSecure,
		cookieSameSite:          cookieSameSite,
	}
}

// Link はログイン済みユーザーに Firebase UID を紐付けます。
func (h *FirebaseHandler) Link(c echo.Context) error {
	req := new(firebaseRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	if err := h.firebaseLinkUsecase.LinkFirebaseUID(c.Request().Context(), user.ID, req.IDToken); err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// Login は Firebase ID トークンでログインし、認証 Cookie を発行します。
func (h *FirebaseHandler) Login(c echo.Context) error {
	req := new(firebaseRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	token, err := h.firebaseLoginUsecase.LoginWithFirebase(c.Request().Context(), req.IDToken)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	c.SetCookie(newAuthCookie(h.cookieSecure, h.cookieSameSite, token, 0))

	return c.NoContent(http.StatusNoContent)
}

// Register は Firebase ID トークンとユーザー名で新規ユーザーを登録し、認証 Cookie を発行します。
func (h *FirebaseHandler) Register(c echo.Context) error {
	req := new(firebaseRegisterRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	token, err := h.firebaseRegisterUsecase.RegisterWithFirebase(c.Request().Context(), req.IDToken, req.Username)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	c.SetCookie(newAuthCookie(h.cookieSecure, h.cookieSameSite, token, 0))

	return c.NoContent(http.StatusCreated)
}
