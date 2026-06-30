package api_internal

import (
	"log/slog"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/reauthtoken"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

const reauthTokenHeader = "X-Reauth-Token" // #nosec G101 -- HTTPヘッダー名であり、認証情報（シークレット）ではないため

// ProfileHandler は認証済みユーザーのプロフィール関連リクエストを処理します。
type ProfileHandler struct {
	userCredentialUsecase usecase.UserCredentialUsecase
}

// NewProfileHandler は新しいProfileHandlerを生成します。
func NewProfileHandler(userCredentialUsecase usecase.UserCredentialUsecase) *ProfileHandler {
	return &ProfileHandler{
		userCredentialUsecase: userCredentialUsecase,
	}
}

// Me は認証済みユーザー自身の情報を取得するリクエストを処理します。
func (h *ProfileHandler) Me(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	userDTO, err := h.userCredentialUsecase.GetUser(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, userDTO)
}

type updatePrivacyRequest struct {
	IsPrivate bool `json:"is_private"`
}

// UpdatePrivacy は認証済みユーザーの非公開設定を更新するリクエストを処理します。
func (h *ProfileHandler) UpdatePrivacy(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	req := new(updatePrivacyRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	if err := h.userCredentialUsecase.UpdatePrivacy(c.Request().Context(), user.ID, req.IsPrivate); err != nil {
		slog.Error("Failed to update privacy setting", "user_id", user.ID, "error", err)
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, map[string]any{"is_private": req.IsPrivate})
}

// DeleteAccount は認証済みユーザーの物理削除を行うリクエストを処理します。
func (h *ProfileHandler) DeleteAccount(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	reauthToken, err := reauthtoken.New(c.Request().Header.Get(reauthTokenHeader))
	if err != nil {
		return apierror.ErrRecentSignInRequired
	}

	if err := h.userCredentialUsecase.DeleteOwnAccount(c.Request().Context(), user.ID, reauthToken); err != nil {
		slog.Error("Failed to delete user", "user_id", user.ID, "error", err)
		return apierror.FromUsecaseError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
