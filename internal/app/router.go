package app

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/chunirec"
	"github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	infra "github.com/chunisupport/chunisupport-api/internal/infra/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/transaction"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

// CustomValidator はリクエストの検証を行うための構造体です。
type CustomValidator struct {
	Validator *validator.Validate
}

// NewCustomValidator は新しいCustomValidatorを生成します。
func NewCustomValidator() *CustomValidator {
	return &CustomValidator{Validator: validator.New()}
}

// Validate は与えられた構造体を検証します。
func (cv *CustomValidator) Validate(i any) error {
	if err := cv.Validator.Struct(i); err != nil {
		// 詳細なエラーはログに出力し、クライアントには汎用的なエラーコードを返す
		slog.Warn("Validation error", "error", err.Error())
		return apierror.ErrValidationFailed.WithInternal(err)
	}
	return nil
}

// Handlers はすべてのハンドラーを保持するコンテナです
type Handlers struct {
	Auth       *api_internal.AuthHandler
	User       *api_internal.UserHandler
	AdminUser  *api_internal.AdminUserHandler
	Song       *api_internal.SongHandler
	Worldsend  *api_internal.WorldsendHandler
	APIToken   *api_internal.APITokenHandler
	Me         *api_internal.MeHandler
	MasterData *api_internal.MasterDataHandler
	Session    *api_internal.SessionHandler
	// 外部API v1 用ハンドラ
	V1Song      *api_v1.V1SongHandler
	V1Worldsend *api_v1.V1WorldsendHandler
	V1User      *api_v1.V1UserHandler
	// chunirec互換APIハンドラ
	Chunirec *chunirec.ChunirecHandler
}

// NewRouter はルートが設定された新しいEchoインスタンスを作成します
// echoLogWriterはnilの場合があります（ログ設定失敗時）
func NewRouter(db *sqlx.DB, staticDB *sqlx.DB, cfg config.Config, masterCache *masterdata.Cache, staticMasterCache *masterdata.StaticCache, echoLogWriter io.Writer) *echo.Echo {
	e := echo.New()
	e.Validator = NewCustomValidator()

	// カスタムエラーハンドラーの設定
	e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler

	// ミドルウェアの設定
	// Echoのロガーを設定
	if echoLogWriter != nil {
		e.Use(echoMiddleware.LoggerWithConfig(echoMiddleware.LoggerConfig{
			Output: echoLogWriter,
		}))
	}

	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.BodyLimit(info.RequestBodyLimit))

	// CORS設定を適用
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins:     cfg.CORS.AllowOrigins,
		AllowCredentials: cfg.CORS.AllowCredentials,
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
		},
		ExposeHeaders: []string{
			echo.HeaderContentLength,
		},
		MaxAge: cfg.CORS.MaxAge,
	}))

	// DI - Services
	userRepo := infra.NewUserRepository(db)
	playerRepo := infra.NewPlayerRepository(db)
	playerRecordRepo := infra.NewPlayerRecordRepository(db)
	worldsendRecordRepo := infra.NewWorldsendRecordRepository(db)
	playerDataRepo := infra.NewPlayerDataRepository(db)
	worldsendChartRepo := infra.NewWorldsendChartRepository(db)
	chartStatsRepo := infra.NewChartStatsRepository(staticDB)
	sessionRepo := infra.NewSessionRepository(db)
	apiTokenRepo := infra.NewAPITokenRepository(db)
	recoveryCodeRepo := infra.NewRecoveryCodeRepository(db)
	songRepo := infra.NewSongRepository(db)
	honorRepo := infra.NewHonorRepository(db)
	tm := transaction.NewTransactionManager(db)
	authUsecase := usecase.NewAuthService(db, tm, userRepo, sessionRepo, recoveryCodeRepo, playerRecordRepo, cfg.JWTSecret, cfg.Auth.JWTExpirationHour, cfg.Auth.SessionExpirationHour, cfg.PwPepper, masterCache)
	apiTokenUsecase := usecase.NewAPITokenService(db, apiTokenRepo, userRepo)
	playerUsecase := usecase.NewPlayerService(db, playerRepo)
	userUsecase := usecase.NewUserService(db, userRepo, playerRecordRepo, playerUsecase)
	// userUsecase に worldsendRecordRepo を設定（通常レコードとの依存関係を避けるため後から設定）
	if uu, ok := userUsecase.(interface {
		SetWorldsendRecordRepository(repository.WorldsendRecordRepository)
	}); ok {
		uu.SetWorldsendRecordRepository(worldsendRecordRepo)
	}
	playerDataUsecase := usecase.NewPlayerDataService(tm, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo, honorRepo, playerDataRepo, masterCache)
	songUsecase := usecase.NewSongService(songRepo, masterCache, tm, db)
	chartStatsUsecase := usecase.NewChartStatsUsecase(songRepo, worldsendChartRepo, chartStatsRepo, masterCache, staticMasterCache, db, staticDB)
	worldsendUsecase := usecase.NewWorldsendUsecase(worldsendChartRepo, tm, db)
	sessionUsecase := usecase.NewSessionUsecase(sessionRepo, db)

	// DI - Handlers
	sameSite := parseSameSite(cfg.Auth.CookieSameSite)
	handlers := &Handlers{
		Auth:       api_internal.NewAuthHandler(authUsecase, cfg.Auth.CookieSecure, sameSite, masterCache),
		User:       api_internal.NewUserHandler(userUsecase),
		AdminUser:  api_internal.NewAdminUserHandler(userUsecase),
		Song:       api_internal.NewSongHandler(songUsecase, chartStatsUsecase, masterCache, staticMasterCache),
		Worldsend:  api_internal.NewWorldsendHandler(worldsendUsecase),
		APIToken:   api_internal.NewAPITokenHandler(apiTokenUsecase),
		Me:         api_internal.NewMeHandler(playerDataUsecase),
		MasterData: api_internal.NewMasterDataHandler(masterCache, staticMasterCache),
		Session:    api_internal.NewSessionHandler(sessionUsecase),
		// 外部API v1 用ハンドラ
		V1Song:      api_v1.NewV1SongHandler(songUsecase, chartStatsUsecase, masterCache, staticMasterCache),
		V1Worldsend: api_v1.NewV1WorldsendHandler(worldsendUsecase),
		V1User:      api_v1.NewV1UserHandler(userUsecase),
		// chunirec互換APIハンドラ
		Chunirec: chunirec.NewChunirecHandler(songUsecase, userUsecase, userRepo, db, masterCache),
	}

	// ルートの設定
	e.GET("/", func(c echo.Context) error {
		// 将来的に変更の可能性あり
		return c.JSON(http.StatusOK, map[string]string{
			"app_name": "chunisupport-api",
		})
	})
	e.GET("/health", handleHealth(db), middleware.APITokenMiddleware(apiTokenUsecase), middleware.RequireRole(info.AccountTypeAdmin))

	// ルートの登録
	registerRoutes(e, handlers, authUsecase, apiTokenUsecase, cfg.JWTSecret)

	return e
}

// registerRoutes はすべてのルートを登録します
func registerRoutes(e *echo.Echo, handlers *Handlers, authUsecase usecase.AuthUsecase, apiTokenUsecase usecase.APITokenUsecase, secret string) {
	// api.chunisupport.net/internal
	internal := e.Group("/internal")

	// JWT認証ミドルウェア
	jwtAuth := middleware.JWTMiddleware(secret, authUsecase)
	optionalJWTAuth := middleware.OptionalJWTMiddleware(secret, authUsecase)
	anonymousRateLimit := middleware.AnonymousIPRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.InternalPublicRateLimitRequests,
		Window:   info.InternalPublicRateLimitWindow,
	})

	// EDITOR以上の権限を要求するミドルウェア
	requireEditor := middleware.RequireRole(info.AccountTypeEditor)

	// ADMIN以上の権限を要求するミドルウェア
	requireAdmin := middleware.RequireRole(info.AccountTypeAdmin)

	// api.chunisupport.net/internal/auth
	authGroup := internal.Group("/auth")
	{
		// ログイン: 1分間に10回まで
		authGroup.POST("/login", handlers.Auth.Login, middleware.IPRateLimitMiddleware(middleware.RateLimitConfig{
			Requests: info.LoginRateLimitRequests,
			Window:   info.LoginRateLimitWindow,
		}))
		authGroup.POST("/logout", handlers.Auth.Logout, jwtAuth)
		// 登録: 1分間に5回まで
		authGroup.POST("/register", handlers.Auth.Register, middleware.IPRateLimitMiddleware(middleware.RateLimitConfig{
			Requests: info.RegisterRateLimitRequests,
			Window:   info.RegisterRateLimitWindow,
		}))
		authGroup.POST("/recovery-codes", handlers.Auth.RecoverPassword, middleware.IPRateLimitMiddleware(middleware.RateLimitConfig{
			Requests: info.RecoveryCodeRateLimitRequests,
			Window:   info.RecoveryCodeRateLimitWindow,
		}))
		authGroup.POST("/api-tokens", handlers.APIToken.Generate, jwtAuth)
		authGroup.DELETE("/api-tokens", handlers.APIToken.Delete, jwtAuth)
	}

	// api.chunisupport.net/internal/me
	meGroup := internal.Group("/me")
	meGroup.Use(jwtAuth)
	{
		meGroup.GET("", handlers.Auth.Me)
		meGroup.PUT("/privacy", handlers.Auth.UpdatePrivacy)
		meGroup.PUT("/password", handlers.Auth.ChangePassword)
		meGroup.POST("/recovery-codes", handlers.Auth.IssueRecoveryCodes)
		meGroup.DELETE("", handlers.Auth.DeleteAccount)
		meGroup.POST("/register-data", handlers.Me.RegisterData, middleware.UserRateLimitMiddleware(middleware.RateLimitConfig{
			Requests: info.RegisterDataRateLimitRequests,
			Window:   info.RegisterDataRateLimitWindow,
		}))
		meGroup.DELETE("/player-data", handlers.Me.DeletePlayerData)
		// セッション管理
		meGroup.GET("/sessions", handlers.Session.GetSessionCount)
		meGroup.DELETE("/sessions", handlers.Session.LogoutOtherSessions)
	}

	// api.chunisupport.net/internal/users
	publicUsersGroup := internal.Group("/users")
	publicUsersGroup.Use(optionalJWTAuth, anonymousRateLimit)
	{
		publicUsersGroup.GET("/:username", handlers.User.GetUserProfileWithRecords)
	}

	usersGroup := internal.Group("/users")
	usersGroup.Use(jwtAuth)
	{
		usersGroup.GET("/", handlers.AdminUser.GetAllUsers, requireAdmin)
		usersGroup.DELETE("/:username", handlers.User.DeleteUser, requireAdmin)
		usersGroup.POST("/:username/restore", handlers.User.RestoreUser, requireAdmin)
	}

	// api.chunisupport.net/internal/songs
	publicSongsGroup := internal.Group("/songs")
	publicSongsGroup.Use(optionalJWTAuth, anonymousRateLimit)
	{
		publicSongsGroup.GET("", handlers.Song.GetSongs)
		publicSongsGroup.GET("/:displayid", handlers.Song.GetSong)
		publicSongsGroup.GET("/:displayid/stats/:difficulty", handlers.Song.GetChartStatsByDifficulty)

		// WORLD'S END 楽曲エンドポイント
		publicWorldsendGroup := publicSongsGroup.Group("/worldsend")
		{
			publicWorldsendGroup.GET("", handlers.Worldsend.GetWorldsendSongs)
			publicWorldsendGroup.GET("/:displayid", handlers.Worldsend.GetWorldsendSong)
		}
	}

	songsGroup := internal.Group("/songs")
	songsGroup.Use(jwtAuth)
	{
		songsGroup.PUT("", handlers.Song.UpdateSongs, requireEditor)
		songsGroup.DELETE("/:displayid", handlers.Song.DeleteSong, requireEditor)
		songsGroup.POST("/:displayid/restore", handlers.Song.RestoreSong, requireEditor)

		// WORLD'S END 楽曲エンドポイント
		worldsendGroup := songsGroup.Group("/worldsend")
		{
			worldsendGroup.DELETE("/:displayid", handlers.Worldsend.DeleteWorldsendSong, requireEditor)
			worldsendGroup.POST("/:displayid/restore", handlers.Worldsend.RestoreWorldsendSong, requireEditor)
		}
	}

	// api.chunisupport.net/internal/master
	masterGroup := internal.Group("/master")
	masterGroup.Use(jwtAuth)
	{
		masterGroup.GET("", handlers.MasterData.GetMasterData)
	}

	// 外部APIルートの登録
	// api.chunisupport.net/v1
	apiV1 := e.Group("/v1")
	apiV1.Use(middleware.APITokenMiddleware(apiTokenUsecase))
	// レートリミット: ADMINは15分150,000回、その他は15分150回
	apiV1.Use(middleware.APIRateLimitMiddleware(
		info.APIRateLimitRequests,
		info.APIRateLimitAdminRequests,
		info.APIRateLimitWindow,
	))
	{
		apiV1.GET("/songs", handlers.V1Song.GetSongs)
		apiV1.GET("/songs/:displayid", handlers.V1Song.GetSong)
		apiV1.GET("/songs/:displayid/stats/:difficulty", handlers.V1Song.GetChartStatsByDifficulty)
		apiV1.GET("/songs/worldsend", handlers.V1Worldsend.GetWorldsendSongs)
		apiV1.GET("/songs/worldsend/:displayid", handlers.V1Worldsend.GetWorldsendSong)
		apiV1.GET("/users/:username", handlers.V1User.GetUser)
	}

	// chunirec互換APIルートの登録
	// api.chunisupport.net/compat/chunirec/2.0
	chunirecGroup := e.Group("/compat/chunirec/2.0")
	// chunirec専用エラーハンドリング（最初に適用）
	chunirecGroup.Use(chunirec.ChunirecErrorHandlerMiddleware())
	chunirecGroup.Use(middleware.APITokenMiddleware(apiTokenUsecase))
	// レートリミットはv1と同じ設定を適用
	chunirecGroup.Use(middleware.APIRateLimitMiddleware(
		info.APIRateLimitRequests,
		info.APIRateLimitAdminRequests,
		info.APIRateLimitWindow,
	))
	{
		chunirecGroup.GET("/music/showall", handlers.Chunirec.GetMusicShowAll)
		chunirecGroup.GET("/music/show", handlers.Chunirec.GetMusicShow)
		chunirecGroup.GET("/users/show", handlers.Chunirec.GetUserShow)
	}
}

// handleHealth はヘルスチェックエンドポイントのハンドラを返します
// セキュリティを考慮し、内部情報（バージョン、サービス名など）は一切返しません
func handleHealth(db *sqlx.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		// データベース接続状態をチェック
		if err := db.Ping(); err != nil {
			slog.Error("Database health check failed: " + err.Error())
			return c.NoContent(http.StatusServiceUnavailable)
		}

		return c.NoContent(http.StatusOK)
	}
}

// parseSameSite は文字列をhttp.SameSite型に変換します
func parseSameSite(value string) http.SameSite {
	switch value {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	case "lax":
		fallthrough
	default:
		return http.SameSiteLaxMode
	}
}

// echoLogWriter はEchoログ出力用のWriterで、ファイルハンドルのライフサイクル管理が可能
type echoLogWriter struct {
	writer io.Writer
	file   *os.File
}

// Write はio.Writerインターフェースを実装
func (w *echoLogWriter) Write(p []byte) (n int, err error) {
	return w.writer.Write(p)
}

// Close はファイルハンドルをクローズ
func (w *echoLogWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// SetupEchoLogger はEchoのロガーを設定します。
// 戻り値のio.WriteCloserは呼び出し元でClose()を呼ぶ必要があります。
func SetupEchoLogger(cfg config.Config) (io.WriteCloser, error) {
	// ログディレクトリが存在しない場合は作成
	if err := os.MkdirAll(cfg.LogPaths.Echo, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// 現在時刻からファイル名を生成
	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Join(cfg.LogPaths.Echo, fmt.Sprintf("%s.log", timestamp))

	// ファイルを開く（存在しない場合は作成、存在する場合は追記）
	// #nosec G304 -- LogPaths.Echo comes from trusted configuration
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// 標準出力とファイルの両方にログを出力
	return &echoLogWriter{
		writer: io.MultiWriter(os.Stdout, file),
		file:   file,
	}, nil
}
