package api_internal

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// UserHandler はユーザー関連のHTTPリクエストを処理します。
type UserHandler struct {
	userUsecase usecase.UserUsecase
}

// NewUserHandler は新しいUserHandlerを生成します。
func NewUserHandler(userUsecase usecase.UserUsecase) *UserHandler {
	return &UserHandler{userUsecase: userUsecase}
}

// GetUserProfileWithRecords はユーザープロファイルとレコードを一括取得するハンドラです。
func (h *UserHandler) GetUserProfileWithRecords(c echo.Context) error {
	username := c.Param("username")
	view := c.QueryParam("view")
	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}
	if view == "rating" {
		result, err := h.userUsecase.GetUserProfileRatingView(c.Request().Context(), username, requester)
		if err != nil {
			return h.handleUserProfileError(err, username, "user profile rating view")
		}
		return c.JSON(http.StatusOK, result)
	}

	result, err := h.userUsecase.GetUserProfileWithRecords(c.Request().Context(), username, requester)
	if err != nil {
		return h.handleUserProfileError(err, username, "user profile with records")
	}

	return c.JSON(http.StatusOK, result)
}

func (h *UserHandler) handleUserProfileError(err error, username string, contextDescription string) error {
	switch {
	case errors.Is(err, usecase.ErrUserNotFound):
		return apierror.ErrUserNotFound
	case errors.Is(err, usecase.ErrUserPrivate):
		// セキュリティ: 非公開と未発見を区別しない
		return apierror.ErrUserNotFound
	case errors.Is(err, usecase.ErrPlayerNotLinked):
		// セキュリティ: プレイヤー未紐付も404で隠蔽
		return apierror.ErrUserNotFound
	default:
		if errors.Is(err, context.Canceled) {
			slog.Warn("failed to get "+contextDescription+" due to context canceled", "username", username, "error", err)
		} else {
			slog.Error("failed to get "+contextDescription, "username", username, "error", err)
		}
		return apierror.ErrInternalError.WithInternal(err)
	}
}

// DeleteUser はユーザーを論理削除するハンドラです（ADMIN権限必須）。
func (h *UserHandler) DeleteUser(c echo.Context) error {
	username := c.Param("username")
	requester, ok := c.Get("userEntity").(*entity.User)
	if !ok {
		// 認証ミドルウェアが正しく機能していれば、この分岐に入ることはありません。
		// 安全のため、不正なリクエストとして処理します。
		return apierror.ErrUnauthorized
	}

	if err := h.userUsecase.DeleteUser(c.Request().Context(), requester, username); err != nil {
		switch {
		case errors.Is(err, usecase.ErrAdminRequired):
			return apierror.ErrForbidden
		case errors.Is(err, usecase.ErrUserNotFound):
			return apierror.ErrUserNotFound
		case errors.Is(err, usecase.ErrUserAlreadyDeleted):
			// セキュリティ: 削除済みであることを隠蔽
			return apierror.ErrOperationFailed
		default:
			slog.Error("failed to delete user", "username", username, "error", err)
			return apierror.ErrInternalError.WithInternal(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// RestoreUser はユーザーを復活させるハンドラです（ADMIN権限必須）。
func (h *UserHandler) RestoreUser(c echo.Context) error {
	username := c.Param("username")
	requester, ok := c.Get("userEntity").(*entity.User)
	if !ok {
		// 認証ミドルウェアが正しく機能していれば、この分岐に入ることはありません。
		// 安全のため、不正なリクエストとして処理します。
		return apierror.ErrUnauthorized
	}

	if err := h.userUsecase.RestoreUser(c.Request().Context(), requester, username); err != nil {
		switch {
		case errors.Is(err, usecase.ErrAdminRequired):
			return apierror.ErrForbidden
		case errors.Is(err, usecase.ErrUserNotFound):
			return apierror.ErrUserNotFound
		case errors.Is(err, usecase.ErrUserNotDeleted):
			// セキュリティ: 未削除であることを隠蔽
			return apierror.ErrOperationFailed
		default:
			slog.Error("failed to restore user", "username", username, "error", err)
			return apierror.ErrInternalError.WithInternal(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}
