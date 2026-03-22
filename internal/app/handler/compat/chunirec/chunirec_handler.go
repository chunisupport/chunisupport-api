package chunirec

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// ChunirecHandler はchunirec互換APIのハンドラです
type ChunirecHandler struct {
	songUsecase usecase.SongUsecase
	userUsecase usecase.UserUsecase
	masterCache *masterdata.Cache
}

// NewChunirecHandler はChunirecHandlerの新しいインスタンスを返します
func NewChunirecHandler(songUsecase usecase.SongUsecase, userUsecase usecase.UserUsecase, masterCache *masterdata.Cache) *ChunirecHandler {
	return &ChunirecHandler{
		songUsecase: songUsecase,
		userUsecase: userUsecase,
		masterCache: masterCache,
	}
}

// GetMusicShowAll は全楽曲情報をchunirec互換形式で返します
// GET /compat/chunirec/2.0/music/showall
func (h *ChunirecHandler) GetMusicShowAll(c echo.Context) error {
	ctx := c.Request().Context()

	songs, err := h.songUsecase.GetAllSongsExcludingWorldsend(ctx, false, nil)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, ToMusicShowAllResponse(songs, h.masterCache.SongMasters()))
}

// GetMusicShow は指定されたDisplay IDの楽曲情報をchunirec互換形式で返します
// GET /compat/chunirec/2.0/music/show?id=xxx
func (h *ChunirecHandler) GetMusicShow(c echo.Context) error {
	ctx := c.Request().Context()
	displayID := c.QueryParam("id")
	if displayID == "" {
		return apierror.ErrValidationFailed
	}

	requesterAccountTypeID := handler.GetRequesterAccountTypeID(c)
	song, err := h.songUsecase.GetSongByDisplayID(ctx, displayID, requesterAccountTypeID)
	if err != nil {
		if errors.Is(err, repository.ErrSongNotFound) {
			return apierror.ErrSongNotFound
		}
		slog.Error("failed to get song", "displayID", displayID, "error", err)
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.JSON(http.StatusOK, ToMusicShowResponse(song, h.masterCache.SongMasters()))
}

// GetUserShow は指定されたユーザーのプロフィールをchunirec互換形式で返します
// GET /compat/chunirec/2.0/users/show
func (h *ChunirecHandler) GetUserShow(c echo.Context) error {
	ctx := c.Request().Context()
	username := c.QueryParam("user_name")

	if username == "" {
		if userEntity, ok := c.Get("userEntity").(*entity.User); ok && userEntity != nil {
			username = userEntity.Username.String()
		} else {
			return apierror.ErrUnauthorized
		}
	}

	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}

	result, err := h.userUsecase.GetUserProfileWithRecords(ctx, username, requester, false)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound):
			return apierror.ErrUserNotFound
		case errors.Is(err, usecase.ErrUserPrivate):
			return apierror.ErrUserNotFound
		case errors.Is(err, usecase.ErrPlayerNotLinked):
			return apierror.ErrUserNotFound
		default:
			slog.Error("failed to get user profile", "username", username, "error", err)
			return apierror.ErrInternalError.WithInternal(err)
		}
	}

	return c.JSON(http.StatusOK, ToChunirecUserDTO(result, h.masterCache))
}
