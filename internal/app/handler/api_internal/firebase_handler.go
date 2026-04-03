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

// FirebaseHandler は Firebase ログイン・連携リクエストを処理します。
type FirebaseHandler struct {
	firebaseLinkUsecase  usecase.FirebaseLinkUsecase
	firebaseLoginUsecase usecase.FirebaseLoginUsecase
	cookieSecure         bool
	cookieSameSite       http.SameSite
}

// NewFirebaseHandler は FirebaseHandler を生成します。
func NewFirebaseHandler(firebaseLinkUsecase usecase.FirebaseLinkUsecase, firebaseLoginUsecase usecase.FirebaseLoginUsecase, cookieSecure bool, cookieSameSite http.SameSite) *FirebaseHandler {
	return &FirebaseHandler{
		firebaseLinkUsecase:  firebaseLinkUsecase,
		firebaseLoginUsecase: firebaseLoginUsecase,
		cookieSecure:         cookieSecure,
		cookieSameSite:       cookieSameSite,
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
