package api_internal

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
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
func (h *UserHandler) GetUserProfile(c *echo.Context) error {
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

	return c.JSON(http.StatusOK, toUserProfileDTO(result))
}

// GetUserUpdatedAt はユーザー関連データの updated_at のみを返す軽量なハンドラです。
func (h *UserHandler) GetUserUpdatedAt(c *echo.Context) error {
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

	return c.JSON(http.StatusOK, &dto_internal.UserUpdatedAtDTO{UpdatedAt: result.UpdatedAt})
}

// GetUserRating はユーザー名をキーにレーティング枠のみを返すハンドラです。
func (h *UserHandler) GetUserRating(c *echo.Context) error {
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
func (h *UserHandler) GetUserRecord(c *echo.Context) error {
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

// GetUserSongRecord は通常楽曲1曲分のユーザーレコードを返します。
func (h *UserHandler) GetUserSongRecord(c *echo.Context) error {
	username, apiErr := handler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}
	displayID, apiErr := handler.ValidateDisplayID(c.Param("displayid"))
	if apiErr != nil {
		return apiErr
	}
	includeNoPlay, _ := strconv.ParseBool(c.QueryParam("include_noplay"))
	difficulty := strings.ToUpper(c.QueryParam("difficulty"))
	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}

	result, err := h.userUsecase.GetUserSongRecord(c.Request().Context(), username, requester, displayID, includeNoPlay, difficulty)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toUserSongRecordDTO(result))
}

// GetUserWorldsendSongRecord は WORLD'S END 楽曲1曲分のユーザーレコードを返します。
func (h *UserHandler) GetUserWorldsendSongRecord(c *echo.Context) error {
	username, apiErr := handler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}
	displayID, apiErr := handler.ValidateDisplayID(c.Param("displayid"))
	if apiErr != nil {
		return apiErr
	}
	includeNoPlay, _ := strconv.ParseBool(c.QueryParam("include_noplay"))
	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}

	result, err := h.userUsecase.GetUserWorldsendSongRecord(c.Request().Context(), username, requester, displayID, includeNoPlay)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toUserWorldsendSongRecordDTO(result))
}

// GetUserProfileWithRecords はユーザープロファイルとレコードを一括取得するハンドラです。
func (h *UserHandler) GetUserProfileWithRecords(c *echo.Context) error {
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
		return c.JSON(http.StatusOK, toUserProfileRatingViewDTO(result))
	}

	includeNoPlay, _ := strconv.ParseBool(c.QueryParam("include_noplay"))
	if view == "record" {
		result, err := h.userUsecase.GetUserProfileRecordView(c.Request().Context(), username, requester, includeNoPlay)
		if err != nil {
			return h.handleUserProfileError(err, username, "user profile record view")
		}
		return c.JSON(http.StatusOK, toUserProfileRecordViewDTO(result))
	}

	result, err := h.userUsecase.GetUserProfileWithRecords(c.Request().Context(), username, requester, includeNoPlay)
	if err != nil {
		return h.handleUserProfileError(err, username, "user profile with records")
	}

	return c.JSON(http.StatusOK, ToUserProfileWithRecordsDTO(result))
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

func toUserRatingDTO(result *usecase.UserProfileRatingViewOutput) *dto_internal.UserRatingDTO {
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
	if result.Player != nil && result.Player.Player != nil {
		ratingDTO.Rating = result.Player.Player.CalculatedRating
		ratingDTO.BestAverage = result.Player.Player.BestAverageRating
		ratingDTO.NewAverage = result.Player.Player.NewAverageRating
	}

	if result.Records == nil {
		return ratingDTO
	}

	ratingDTO.Best = ToPlayerRecordDTOs(result.Records.Best)
	ratingDTO.BestCandidate = ToPlayerRecordDTOs(result.Records.BestCandidate)
	ratingDTO.New = ToPlayerRecordDTOs(result.Records.New)
	ratingDTO.NewCandidate = ToPlayerRecordDTOs(result.Records.NewCandidate)
	ratingDTO.Meta.UpdatedAt = &result.Records.UpdatedAt

	return ratingDTO
}

func toUserRecordDTO(result *usecase.UserProfileRecordViewOutput) *dto_internal.UserRecordDTO {
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

	recordDTO.All = ToPlayerRecordDTOs(result.Records.All)
	recordDTO.Worldsend = toWorldsendRecordDTOs(result.Records.Worldsend)
	recordDTO.Courses = toCourseRecordDTOs(result.Records.Courses)
	recordDTO.Meta.UpdatedAt = &result.Records.UpdatedAt

	return recordDTO
}

func ToPlayerRecordDTOs(values []*usecase.PlayerRecordOutput) []*dto.PlayerRecordDTO {
	result := make([]*dto.PlayerRecordDTO, 0, len(values))
	for _, value := range values {
		if value == nil {
			result = append(result, nil)
		} else {
			result = append(result, dto.ToPlayerRecordDTO(value.PlayerRecord))
		}
	}
	return result
}
func toWorldsendRecordDTOs(values []*usecase.WorldsendRecordOutput) []*dto.WorldsendRecordDTO {
	result := make([]*dto.WorldsendRecordDTO, 0, len(values))
	for _, value := range values {
		if value == nil {
			result = append(result, nil)
		} else {
			result = append(result, dto.ToWorldsendRecordDTO(value.PlayerWorldsendRecord))
		}
	}
	return result
}
func toUserPlayerDTO(value *usecase.UserPlayerOutput) *dto.PlayerDTO {
	if value == nil || value.Player == nil {
		return nil
	}
	result := dto.ToPlayerDTO(value.Player)
	result.Honors = make([]*dto.HonorDTO, 0, len(value.Honors))
	for _, honor := range value.Honors {
		if honor == nil {
			continue
		}
		imageURL := ""
		if honor.ImageURL != nil {
			imageURL = *honor.ImageURL
		}
		result.Honors = append(result.Honors, &dto.HonorDTO{Slot: honor.Slot, Name: honor.Name, TypeName: honor.TypeName, ImageURL: imageURL})
	}
	return result
}
func toUserProfileDTO(value *usecase.UserProfileOutput) *dto_internal.UserProfileDTO {
	if value == nil {
		return nil
	}
	return &dto_internal.UserProfileDTO{Username: value.Username, Player: toUserPlayerDTO(value.Player)}
}
func ToUserProfileWithRecordsDTO(value *usecase.UserProfileWithRecordsOutput) *dto_internal.UserProfileWithRecordsDTO {
	if value == nil {
		return nil
	}
	result := &dto_internal.UserProfileWithRecordsDTO{UserID: value.UserID, Username: value.Username, Player: toUserPlayerDTO(value.Player), UpdatedAt: value.UpdatedAt}
	if value.Records != nil {
		result.Records = &dto.UserRecordResponseDTO{UpdatedAt: value.Records.UpdatedAt, Best: ToPlayerRecordDTOs(value.Records.Best), BestCandidate: ToPlayerRecordDTOs(value.Records.BestCandidate), New: ToPlayerRecordDTOs(value.Records.New), NewCandidate: ToPlayerRecordDTOs(value.Records.NewCandidate), All: ToPlayerRecordDTOs(value.Records.All), WorldsEnd: toWorldsendRecordDTOs(value.Records.WorldsEnd), Courses: toCourseRecordDTOs(value.Records.Courses)}
	}
	return result
}
func toUserProfileRatingViewDTO(value *usecase.UserProfileRatingViewOutput) *dto_internal.UserProfileRatingViewDTO {
	if value == nil {
		return nil
	}
	result := &dto_internal.UserProfileRatingViewDTO{Username: value.Username, Player: toUserPlayerDTO(value.Player), UpdatedAt: value.UpdatedAt}
	if value.Records != nil {
		result.Records = &dto_internal.UserRatingRecordResponseDTO{UpdatedAt: value.Records.UpdatedAt, Best: ToPlayerRecordDTOs(value.Records.Best), BestCandidate: ToPlayerRecordDTOs(value.Records.BestCandidate), New: ToPlayerRecordDTOs(value.Records.New), NewCandidate: ToPlayerRecordDTOs(value.Records.NewCandidate)}
	}
	return result
}
func toUserProfileRecordViewDTO(value *usecase.UserProfileRecordViewOutput) *dto_internal.UserProfileRecordViewDTO {
	if value == nil {
		return nil
	}
	result := &dto_internal.UserProfileRecordViewDTO{Username: value.Username, Player: toUserPlayerDTO(value.Player), UpdatedAt: value.UpdatedAt}
	if value.Records != nil {
		result.Records = &dto_internal.UserRecordViewResponseDTO{UpdatedAt: value.Records.UpdatedAt, All: ToPlayerRecordDTOs(value.Records.All), Worldsend: toWorldsendRecordDTOs(value.Records.Worldsend), Courses: toCourseRecordDTOs(value.Records.Courses)}
	}
	return result
}
func toUserSongRecordDTO(value *usecase.UserSongRecordOutput) *dto_internal.UserSongRecordDTO {
	if value == nil {
		return nil
	}
	return &dto_internal.UserSongRecordDTO{Standard: ToPlayerRecordDTOs(value.Standard), Meta: &dto_internal.UserSongRecordMetaDTO{UpdatedAt: value.UpdatedAt}}
}
func toUserWorldsendSongRecordDTO(value *usecase.UserWorldsendSongRecordOutput) *dto_internal.UserWorldsendSongRecordDTO {
	if value == nil {
		return nil
	}
	var record *dto.WorldsendRecordDTO
	if value.Worldsend != nil {
		record = dto.ToWorldsendRecordDTO(value.Worldsend.PlayerWorldsendRecord)
	}
	return &dto_internal.UserWorldsendSongRecordDTO{Worldsend: record, Meta: &dto_internal.UserSongRecordMetaDTO{UpdatedAt: value.UpdatedAt}}
}

// DeleteUser はユーザーを物理削除するハンドラです（ADMIN権限必須）。
func (h *UserHandler) DeleteUser(c *echo.Context) error {
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
