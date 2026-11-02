package chunirec

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// ChunirecHandler はchunirec互換APIのハンドラです
type ChunirecHandler struct {
	songUsecase     usecase.SongUsecase
	chunirecUsecase usecase.ChunirecUsecase
	masterCache     *masterdata.Cache
	location        *time.Location
}

// NewChunirecHandler はChunirecHandlerの新しいインスタンスを返します
func NewChunirecHandler(songUsecase usecase.SongUsecase, chunirecUsecase usecase.ChunirecUsecase, masterCache *masterdata.Cache, location *time.Location) *ChunirecHandler {
	if location == nil {
		location = time.UTC
	}
	return &ChunirecHandler{
		songUsecase:     songUsecase,
		chunirecUsecase: chunirecUsecase,
		masterCache:     masterCache,
		location:        location,
	}
}

// GetMusicShowAll は全楽曲情報をchunirec互換形式で返します
// GET /compat/chunirec/2.0/music/showall
func (h *ChunirecHandler) GetMusicShowAll(c *echo.Context) error {
	ctx := c.Request().Context()

	songs, err := h.songUsecase.GetAllSongsExcludingWorldsend(ctx, false, nil)
	if err != nil {
		return err
	}

	masters := h.masterCache.SongMasters()
	response := ToMusicShowAllResponse(songs, masters)

	return c.JSON(http.StatusOK, response)
}

// GetMusicShow は指定されたDisplay IDの楽曲情報をchunirec互換形式で返します
// GET /compat/chunirec/2.0/music/show?id=xxx
func (h *ChunirecHandler) GetMusicShow(c *echo.Context) error {
	ctx := c.Request().Context()

	displayID := c.QueryParam("id")
	if displayID == "" {
		return apierror.ErrValidationFailed
	}
	validDisplayID, apiErr := handler.ValidateDisplayID(displayID)
	if apiErr != nil {
		return apiErr
	}

	requesterAccountTypeID := handler.GetRequesterAccountTypeID(c)
	song, err := h.songUsecase.GetSongByDisplayID(ctx, validDisplayID, requesterAccountTypeID)
	if err != nil {
		if errors.Is(err, repository.ErrSongNotFound) {
			return apierror.ErrSongNotFound
		}
		if !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "failed to get song", "displayID", displayID, "error", err)
		}
		return apierror.ErrInternalError.WithInternal(err)
	}

	masters := h.masterCache.SongMasters()
	response := ToMusicShowResponse(song, masters)

	return c.JSON(http.StatusOK, response)
}

// GetRecordsShowAll は指定ユーザーの通常譜面全レコードをchunirec互換形式で返します。
// GET /compat/chunirec/2.0/records/showall
func (h *ChunirecHandler) GetRecordsShowAll(c *echo.Context) error {
	ctx := c.Request().Context()

	username, apiErr := h.resolveTargetUsername(c)
	if apiErr != nil {
		return apiErr
	}

	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}

	result, err := h.chunirecUsecase.GetRecords(ctx, username, requester)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound):
			return apierror.ErrUserNotFound
		case errors.Is(err, usecase.ErrUserPrivate):
			return apierror.ErrUserNotFound
		default:
			if !errors.Is(err, context.Canceled) {
				slog.ErrorContext(ctx, "failed to get user records", "username", username, "error", err)
			}
			return apierror.ErrInternalError.WithInternal(err)
		}
	}

	response := ToChunirecRecordsResponse(result, h.location)

	return c.JSON(http.StatusOK, response)
}

// Deprecated: 既存利用者の文字列型レーティングを維持します。新規利用にはGetRecordsProfileを使用してください。
func (h *ChunirecHandler) GetUserShow(c *echo.Context) error {
	profile, err := h.getProfile(c)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, ToChunirecProfileDTO(profile, h.masterCache, h.location))
}

func (h *ChunirecHandler) GetRecordsProfile(c *echo.Context) error {
	profile, err := h.getProfile(c)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, ToChunirecRecordsProfileDTO(profile, h.masterCache, h.location))
}

func (h *ChunirecHandler) getProfile(c *echo.Context) (*repository.ChunirecProfile, error) {
	ctx := c.Request().Context()

	validUsername, apiErr := h.resolveTargetUsername(c)
	if apiErr != nil {
		return nil, apiErr
	}

	// requester はAPIトークン所有者（非公開ユーザーの本人アクセス判定用）
	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}

	result, err := h.chunirecUsecase.GetProfile(ctx, validUsername, requester)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound):
			return nil, apierror.ErrUserNotFound
		case errors.Is(err, usecase.ErrUserPrivate):
			// セキュリティ: 非公開と未発見を区別しない
			return nil, apierror.ErrUserNotFound
		default:
			if !errors.Is(err, context.Canceled) {
				slog.ErrorContext(ctx, "failed to get user profile", "username", validUsername, "error", err)
			}
			return nil, apierror.ErrInternalError.WithInternal(err)
		}
	}

	return result, nil
}

func (h *ChunirecHandler) resolveTargetUsername(c *echo.Context) (string, *apierror.APIError) {
	username := c.QueryParam("user_name")
	if username == "" {
		if userEntity, ok := c.Get("userEntity").(*entity.User); ok && userEntity != nil {
			username = userEntity.Username.String()
		} else {
			return "", apierror.ErrUnauthorized
		}
	}

	return handler.ValidateUsername(username)
}
