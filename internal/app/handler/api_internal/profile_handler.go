package api_internal

import (
	"log/slog"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/reauthtoken"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
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

	return c.JSON(http.StatusOK, dto_internal.UserDTO{Username: userDTO.Username, AccountType: userDTO.AccountType, IsPrivate: userDTO.IsPrivate, LastScoreUpdate: userDTO.LastScoreUpdate})
}

type updatePrivacyRequest struct {
	IsPrivate bool `json:"is_private"`
}

type updateUsernameRequest struct {
	Username string `json:"username"`
}

// UpdateUsername は再認証済みユーザーのユーザー名変更を処理します。
func (h *ProfileHandler) UpdateUsername(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}
	reauthToken, err := reauthtoken.New(c.Request().Header.Get(reauthTokenHeader))
	if err != nil {
		return apierror.ErrRecentSignInRequired
	}
	req := new(updateUsernameRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	updatedUsername, err := h.userCredentialUsecase.UpdateUsername(c.Request().Context(), user.ID, req.Username, reauthToken)
	if err != nil {
		apiErr := apierror.FromUsecaseError(err)
		logProfileUpdateFailure("Failed to update username", user.ID, apiErr)
		return apiErr
	}
	return c.JSON(http.StatusOK, map[string]string{"username": updatedUsername})
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
		apiErr := apierror.FromUsecaseError(err)
		logProfileUpdateFailure("Failed to update privacy setting", user.ID, apiErr)
		return apiErr
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
		apiErr := apierror.FromUsecaseError(err)
		logProfileUpdateFailure("Failed to delete user", user.ID, apiErr)
		return apiErr
	}

	return c.NoContent(http.StatusNoContent)
}

// logProfileUpdateFailure はプロフィール更新系の失敗をログに出力します。
// 禁止語・使用済み・バリデーションなどの想定内クライアントエラー（4xx系）は
// Error通知の対象にしないためInfoに落とし、想定外のサーバーエラー（5xx系）のみErrorとします。
// 4xx系の詳細は共通エラーハンドラー側でもWarnログとして記録されます。
func logProfileUpdateFailure(message string, userID int, apiErr *apierror.APIError) {
	if apiErr.HTTPStatus >= http.StatusInternalServerError {
		slog.Error(message, "user_id", userID, "code", apiErr.Code, "error", apiErr.Internal)
		return
	}
	slog.Info(message, "user_id", userID, "code", apiErr.Code, "status", apiErr.HTTPStatus)
}
