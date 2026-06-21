package api_internal

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
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

// GetUserProfile はユーザー名とプレイヤーデータのみを返す軽量なハンドラです。
func (h *UserHandler) GetUserProfile(c echo.Context) error {
	username, apiErr := handler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}
	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}

	result, err := h.userUsecase.GetUserProfile(c.Request().Context(), username, requester)
	if err != nil {
		return h.handleUserProfileError(err, username, "user profile")
	}

	return c.JSON(http.StatusOK, result)
}

// GetUserUpdatedAt はユーザー関連データの updated_at のみを返す軽量なハンドラです。
func (h *UserHandler) GetUserUpdatedAt(c echo.Context) error {
	username, apiErr := handler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}
	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}

	result, err := h.userUsecase.GetUserUpdatedAt(c.Request().Context(), username, requester)
	if err != nil {
		return h.handleUserProfileError(err, username, "user updated at")
	}

	return c.JSON(http.StatusOK, result)
}

// GetUserRating はユーザー名をキーにレーティング枠のみを返すハンドラです。
func (h *UserHandler) GetUserRating(c echo.Context) error {
	username, apiErr := handler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}
	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}

	result, err := h.userUsecase.GetUserProfileRatingView(c.Request().Context(), username, requester)
	if err != nil {
		return h.handleUserProfileError(err, username, "user rating")
	}

	return c.JSON(http.StatusOK, toUserRatingDTO(result))
}

// GetUserRecord はユーザー名をキーにレコード枠のみを返すハンドラです。
func (h *UserHandler) GetUserRecord(c echo.Context) error {
	username, apiErr := handler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}
	includeNoPlay, _ := strconv.ParseBool(c.QueryParam("include_noplay"))
	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}

	result, err := h.userUsecase.GetUserProfileRecordView(c.Request().Context(), username, requester, includeNoPlay)
	if err != nil {
		return h.handleUserProfileError(err, username, "user record")
	}

	return c.JSON(http.StatusOK, toUserRecordDTO(result))
}

// GetUserProfileWithRecords はユーザープロファイルとレコードを一括取得するハンドラです。
func (h *UserHandler) GetUserProfileWithRecords(c echo.Context) error {
	username, apiErr := handler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}
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

	includeNoPlay, _ := strconv.ParseBool(c.QueryParam("include_noplay"))
	if view == "record" {
		result, err := h.userUsecase.GetUserProfileRecordView(c.Request().Context(), username, requester, includeNoPlay)
		if err != nil {
			return h.handleUserProfileError(err, username, "user profile record view")
		}
		return c.JSON(http.StatusOK, result)
	}

	result, err := h.userUsecase.GetUserProfileWithRecords(c.Request().Context(), username, requester, includeNoPlay)
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
	default:
		if errors.Is(err, context.Canceled) {
			slog.Warn("failed to get "+contextDescription+" due to context canceled", "username", username, "error", err)
		} else {
			slog.Error("failed to get "+contextDescription, "username", username, "error", err)
		}
		return apierror.ErrInternalError.WithInternal(err)
	}
}

func toUserRatingDTO(result *dto_internal.UserProfileRatingViewDTO) *dto_internal.UserRatingDTO {
	if result == nil {
		return nil
	}

	ratingDTO := &dto_internal.UserRatingDTO{
		Best:          []*dto.PlayerRecordDTO{},
		BestCandidate: []*dto.PlayerRecordDTO{},
		New:           []*dto.PlayerRecordDTO{},
		NewCandidate:  []*dto.PlayerRecordDTO{},
		Meta: &dto_internal.UserRatingMetaDTO{
			UpdatedAt: result.UpdatedAt,
		},
	}
	if result.Player != nil {
		ratingDTO.Rating = result.Player.CalculatedRating
		ratingDTO.BestAverage = result.Player.BestAverageRating
		ratingDTO.NewAverage = result.Player.NewAverageRating
	}

	if result.Records == nil {
		return ratingDTO
	}

	ratingDTO.Best = result.Records.Best
	ratingDTO.BestCandidate = result.Records.BestCandidate
	ratingDTO.New = result.Records.New
	ratingDTO.NewCandidate = result.Records.NewCandidate
	ratingDTO.Meta.UpdatedAt = &result.Records.UpdatedAt

	return ratingDTO
}

func toUserRecordDTO(result *dto_internal.UserProfileRecordViewDTO) *dto_internal.UserRecordDTO {
	if result == nil {
		return nil
	}

	recordDTO := &dto_internal.UserRecordDTO{
		All:       []*dto.PlayerRecordDTO{},
		Worldsend: []*dto.WorldsendRecordDTO{},
		Meta: &dto_internal.UserRecordMetaDTO{
			UpdatedAt: result.UpdatedAt,
		},
	}

	if result.Records == nil {
		return recordDTO
	}

	recordDTO.All = result.Records.All
	recordDTO.Worldsend = result.Records.Worldsend
	recordDTO.Meta.UpdatedAt = &result.Records.UpdatedAt

	return recordDTO
}

// DeleteUser はユーザーを物理削除するハンドラです（ADMIN権限必須）。
func (h *UserHandler) DeleteUser(c echo.Context) error {
	username, apiErr := handler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}
	requester, ok := c.Get("userEntity").(*entity.User)
	if !ok {
		// 認証ミドルウェアが正しく機能していれば、この分岐に入ることはありません。
		// 安全のため、不正なリクエストとして処理します。
		return apierror.ErrUnauthorized
	}

	if err := h.userUsecase.DeleteUser(c.Request().Context(), requester, username); err != nil {
		if !errors.Is(err, usecase.ErrAdminRequired) && !errors.Is(err, usecase.ErrUserNotFound) {
			slog.Error("failed to delete user", "username", username, "error", err)
		}
		return apierror.FromUsecaseError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
