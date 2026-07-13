package chunirec

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	internalhandler "github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// ChunirecHandler はchunirec互換APIのハンドラです
type ChunirecHandler struct {
	songUsecase usecase.SongUsecase
	userUsecase usecase.UserUsecase
	masterCache *masterdata.Cache
	location    *time.Location
}

// NewChunirecHandler はChunirecHandlerの新しいインスタンスを返します
func NewChunirecHandler(songUsecase usecase.SongUsecase, userUsecase usecase.UserUsecase, masterCache *masterdata.Cache, location *time.Location) *ChunirecHandler {
	if location == nil {
		location = time.UTC
	}
	return &ChunirecHandler{
		songUsecase: songUsecase,
		userUsecase: userUsecase,
		masterCache: masterCache,
		location:    location,
	}
}

// GetMusicShowAll は全楽曲情報をchunirec互換形式で返します
// GET /compat/chunirec/2.0/music/showall
func (h *ChunirecHandler) GetMusicShowAll(c *echo.Context) error {
	ctx := c.Request().Context()

	// 楽曲を取得 (削除済みを含まない、requesterAccountTypeIDはnil)
	songs, err := h.songUsecase.GetAllSongsExcludingWorldsend(ctx, false, nil)
	if err != nil {
		return err
	}

	// DTOに変換
	masters := h.masterCache.SongMasters()
	response := ToMusicShowAllResponse(songs, masters)

	return c.JSON(http.StatusOK, response)
}

// GetMusicShow は指定されたDisplay IDの楽曲情報をchunirec互換形式で返します
// GET /compat/chunirec/2.0/music/show?id=xxx
func (h *ChunirecHandler) GetMusicShow(c *echo.Context) error {
	ctx := c.Request().Context()

	// クエリパラメータ id を取得
	displayID := c.QueryParam("id")
	if displayID == "" {
		return apierror.ErrValidationFailed
	}
	validDisplayID, apiErr := handler.ValidateDisplayID(displayID)
	if apiErr != nil {
		return apiErr
	}

	// 楽曲を取得
	requesterAccountTypeID := handler.GetRequesterAccountTypeID(c)
	song, err := h.songUsecase.GetSongByDisplayID(ctx, validDisplayID, requesterAccountTypeID)
	if err != nil {
		if errors.Is(err, repository.ErrSongNotFound) {
			return apierror.ErrSongNotFound
		}
		slog.Error("failed to get song", "displayID", displayID, "error", err)
		return apierror.ErrInternalError.WithInternal(err)
	}

	// DTOに変換
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

	result, err := h.userUsecase.GetUserProfileRecordView(ctx, username, requester, false)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound):
			return apierror.ErrUserNotFound
		case errors.Is(err, usecase.ErrUserPrivate):
			return apierror.ErrUserNotFound
		default:
			slog.Error("failed to get user records", "username", username, "error", err)
			return apierror.ErrInternalError.WithInternal(err)
		}
	}

	songs, err := h.songUsecase.GetAllSongsExcludingWorldsend(ctx, false, nil)
	if err != nil {
		slog.Error("failed to get songs for chunirec records", "username", username, "error", err)
		return apierror.ErrInternalError.WithInternal(err)
	}

	var records []*dto.PlayerRecordDTO
	if result != nil && result.Records != nil {
		records = internalhandler.ToPlayerRecordDTOs(result.Records.All)
	}
	response := ToRecordsShowAllResponse(records, h.genresBySongID(songs), h.location)

	return c.JSON(http.StatusOK, response)
}

// GetUserShow は指定されたユーザーのプロフィールをchunirec互換形式で返します
// GET /compat/chunirec/2.0/users/show
func (h *ChunirecHandler) GetUserShow(c *echo.Context) error {
	ctx := c.Request().Context()

	validUsername, apiErr := h.resolveTargetUsername(c)
	if apiErr != nil {
		return apiErr
	}

	// requester はAPIトークン所有者（非公開ユーザーの本人アクセス判定用）
	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}

	// ユーザープロファイルとレコードを取得
	result, err := h.userUsecase.GetUserProfileWithRecords(ctx, validUsername, requester, false)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound):
			return apierror.ErrUserNotFound
		case errors.Is(err, usecase.ErrUserPrivate):
			// セキュリティ: 非公開と未発見を区別しない
			return apierror.ErrUserNotFound
		default:
			slog.Error("failed to get user profile", "username", validUsername, "error", err)
			return apierror.ErrInternalError.WithInternal(err)
		}
	}

	// chunirec互換DTOに変換
	response := ToChunirecUserDTO(internalhandler.ToUserProfileWithRecordsDTO(result), h.masterCache, h.location)

	return c.JSON(http.StatusOK, response)
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

func (h *ChunirecHandler) genresBySongID(songs []*entity.Song) map[string]string {
	genres := make(map[string]string, len(songs))
	masters := h.masterCache.SongMasters()
	for _, song := range songs {
		if song == nil || song.GenreID == nil {
			continue
		}
		if genreName, ok := masters.GenreNamesByID[*song.GenreID]; ok {
			genres[song.DisplayID] = genreName
		}
	}
	return genres
}
