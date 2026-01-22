package api_internal

import (
	"errors"
	"net/http"

	"github.com/Qman110101/chunisupport-api/internal/app/apierror"
	"github.com/Qman110101/chunisupport-api/internal/auth"
	dto_internal "github.com/Qman110101/chunisupport-api/internal/dto/api_internal"
	"github.com/Qman110101/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// SessionHandler はセッション管理関連のHTTPリクエストを処理します。
type SessionHandler struct {
	sessionUsecase usecase.SessionUsecase
}

// NewSessionHandler は新しいSessionHandlerを生成します。
func NewSessionHandler(sessionUsecase usecase.SessionUsecase) *SessionHandler {
	return &SessionHandler{
		sessionUsecase: sessionUsecase,
	}
}

// GetSessionCount はユーザーの有効なセッション数を取得します。
//
// @Summary セッション数取得
// @Description ログイン中のユーザーの有効なセッション数を取得します
// @Tags Session
// @Accept json
// @Produce json
// @Success 200 {object} dto_internal.SessionCountDTO
// @Failure 401 {object} apierror.APIError
// @Failure 500 {object} apierror.APIError
// @Router /internal/me/sessions [get]
func (h *SessionHandler) GetSessionCount(c echo.Context) error {
	claims, ok := c.Get("user").(*auth.Claims)
	if !ok || claims == nil {
		return apierror.ErrUnauthorized.WithInternal(errors.New("JWT claims not found in context"))
	}

	count, err := h.sessionUsecase.GetSessionCount(c.Request().Context(), claims.UserID)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.JSON(http.StatusOK, &dto_internal.SessionCountDTO{
		Count: count,
	})
}

// LogoutOtherSessions は現在のセッション以外をすべてログアウトします。
//
// @Summary 他の端末からログアウト
// @Description 現在のセッション以外のすべてのセッションを削除します
// @Tags Session
// @Accept json
// @Produce json
// @Success 204
// @Failure 401 {object} apierror.APIError
// @Failure 500 {object} apierror.APIError
// @Router /internal/me/sessions [delete]
func (h *SessionHandler) LogoutOtherSessions(c echo.Context) error {
	claims, ok := c.Get("user").(*auth.Claims)
	if !ok || claims == nil {
		return apierror.ErrUnauthorized.WithInternal(errors.New("JWT claims not found in context"))
	}

	err := h.sessionUsecase.LogoutOtherSessions(c.Request().Context(), claims.UserID, claims.SessionID)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.NoContent(http.StatusNoContent)
}
