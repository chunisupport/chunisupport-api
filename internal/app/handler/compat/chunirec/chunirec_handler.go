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
	"github.com/labstack/echo/v5"
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

// GetUserShow は指定されたユーザーのプロフィールをchunirec互換形式で返します
// GET /compat/chunirec/2.0/users/show
func (h *ChunirecHandler) GetUserShow(c *echo.Context) error {
	ctx := c.Request().Context()

	// クエリパラメータ user_name を取得
	username := c.QueryParam("user_name")

	// user_name が指定されていない場合、APIトークン所有者のユーザー名を使用
	if username == "" {
		if userEntity, ok := c.Get("userEntity").(*entity.User); ok && userEntity != nil {
			username = userEntity.Username.String()
		} else {
			// APIトークン認証必須のエンドポイントなので、ここには到達しないはず
			return apierror.ErrUnauthorized
		}
	}
	validUsername, apiErr := handler.ValidateUsername(username)
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
			slog.Error("failed to get user profile", "username", username, "error", err)
			return apierror.ErrInternalError.WithInternal(err)
		}
	}

	// chunirec互換DTOに変換
	response := ToChunirecUserDTO(result, h.masterCache)

	return c.JSON(http.StatusOK, response)
}
