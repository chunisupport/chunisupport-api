package chunirec

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Qman110101/chunisupport-api/internal/app/apierror"
	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
	"github.com/Qman110101/chunisupport-api/internal/usecase"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

// ChunirecHandler はchunirec互換APIのハンドラです
type ChunirecHandler struct {
	songUsecase usecase.SongUsecase
	userUsecase usecase.UserUsecase
	userRepo    repository.UserRepository
	db          *sqlx.DB
	masterCache *masterdata.Cache
}

// NewChunirecHandler はChunirecHandlerの新しいインスタンスを返します
func NewChunirecHandler(songUsecase usecase.SongUsecase, userUsecase usecase.UserUsecase, userRepo repository.UserRepository, db *sqlx.DB, masterCache *masterdata.Cache) *ChunirecHandler {
	return &ChunirecHandler{
		songUsecase: songUsecase,
		userUsecase: userUsecase,
		userRepo:    userRepo,
		db:          db,
		masterCache: masterCache,
	}
}

// GetMusicShowAll は全楽曲情報をchunirec互換形式で返します
// GET /compat/chunirec/v2.0/music/showall
func (h *ChunirecHandler) GetMusicShowAll(c echo.Context) error {
	ctx := c.Request().Context()

	// 楽曲を取得 (削除済みを含まない)
	songs, err := h.songUsecase.GetAllSongsExcludingWorldsend(ctx, false)
	if err != nil {
		return err
	}

	// DTOに変換
	masters := h.masterCache.SongMasters()
	response := ToMusicShowAllResponse(songs, masters)

	return c.JSON(http.StatusOK, response)
}

// GetUserShow は指定されたユーザーのプロフィールをchunirec互換形式で返します
// GET /compat/chunirec/v2.0/users/show
func (h *ChunirecHandler) GetUserShow(c echo.Context) error {
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

	// requester はAPIトークン所有者（非公開ユーザーの本人アクセス判定用）
	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}

	// ユーザープロファイルとレコードを取得
	result, err := h.userUsecase.GetUserProfileWithRecords(ctx, username, requester)
	if err != nil {
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
			slog.Error("failed to get user profile", "username", username, "error", err)
			return apierror.ErrInternalError.WithInternal(err)
		}
	}

	// UserIDを取得するため、UserRepositoryから対象ユーザーのエンティティを取得
	// TODO: UserProfileWithRecordsDTOにUserIDフィールドを追加してリファクタリング
	user, err := h.userRepo.FindByUsername(ctx, h.db, username)
	if err != nil {
		slog.Error("failed to get user entity for UserID", "username", username, "error", err)
		return apierror.ErrInternalError.WithInternal(err)
	}
	if user == nil {
		return apierror.ErrUserNotFound
	}

	// chunirec互換DTOに変換
	response := ToChunirecUserDTO(result, user.ID, h.masterCache)

	return c.JSON(http.StatusOK, response)
}
