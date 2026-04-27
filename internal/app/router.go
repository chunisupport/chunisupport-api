package app

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/chunirec"
	"github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	vo_username "github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
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
	v := validator.New()
	if err := v.RegisterValidation("username", validateUsername); err != nil {
		panic(err)
	}
	return &CustomValidator{Validator: v}
}

func validateUsername(fl validator.FieldLevel) bool {
	_, err := vo_username.NewUserName(fl.Field().String())
	return err == nil
}

// Validate は与えられた構造体を検証します。
func (cv *CustomValidator) Validate(i any) error {
	if err := cv.Validator.Struct(i); err != nil {
		// 詳細なエラーはログに出力し、クライアントには汎用的なエラーコードを返す
		slog.Warn("Validation error", "error", err.Error())
		var validationErrors validator.ValidationErrors
		if ok := errors.As(err, &validationErrors); ok {
			return apierror.ErrValidationFailed.WithInternal(apierror.ValidationErrors(validationErrors))
		}
		return apierror.ErrValidationFailed.WithInternal(err)
	}
	return nil
}

// Handlers はすべてのハンドラーを保持するコンテナです
type Handlers struct {
	Signup              *api_internal.SignupHandler
	Profile             *api_internal.ProfileHandler
	User                *api_internal.UserHandler
	AdminUser           *api_internal.AdminUserHandler
	Song                *api_internal.SongHandler
	Worldsend           *api_internal.WorldsendHandler
	APIToken            *api_internal.APITokenHandler
	Me                  *api_internal.MeHandler
	MasterData          *api_internal.MasterDataHandler
	Goal                *api_internal.GoalHandler
	TemporaryPlayerData *api_internal.TemporaryPlayerDataHandler
	// 外部API v1 用ハンドラ
	V1Song      *api_v1.V1SongHandler
	V1Worldsend *api_v1.V1WorldsendHandler
	V1User      *api_v1.V1UserHandler
	V1Version   *api_v1.V1VersionHandler
	// chunirec互換APIハンドラ
	Chunirec *chunirec.ChunirecHandler
}

// NewRouter はルートが設定された新しいEchoインスタンスを作成します
// echoLogWriterはnilの場合があります（ログ設定失敗時）
func NewRouter(db *sqlx.DB, staticDB *sqlx.DB, cfg config.Config, masterCache *masterdata.Cache, staticMasterCache *masterdata.StaticCache, firebaseTokenVerifier usecase.TokenVerifier, firebaseUserDeleter usecase.FirebaseUserDeleter, echoLogWriter io.Writer) *echo.Echo {
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
	e.Use(echoMiddleware.CORSWithConfig(newDefaultCORSConfig(cfg)))

	// DI - Services
	userRepo := infra.NewUserRepository(db)
	playerRepo := infra.NewPlayerRepository(db)
	playerRecordRepo := infra.NewPlayerRecordRepository(db)
	worldsendRecordRepo := infra.NewWorldsendRecordRepository(db)
	playerDataRepo := infra.NewPlayerDataRepository(db)
	worldsendChartRepo := infra.NewWorldsendChartRepository(db)
	chartStatsRepo := infra.NewChartStatsRepository(staticDB)
	apiTokenRepo := infra.NewAPITokenRepository(db)
	songRepo := infra.NewSongRepository(db)
	goalRepo := infra.NewGoalRepository(db)
	honorRepo := infra.NewHonorRepository(db)
	tm := transaction.NewTransactionManager(db)
	recentSignInVerifier := requireRecentSignInVerifier(firebaseTokenVerifier)
	userCredentialUsecase := usecase.NewUserCredentialUsecaseWithFirebaseServices(db, tm, userRepo, playerRecordRepo, recentSignInVerifier, firebaseUserDeleter, masterCache)
	apiTokenUsecase := usecase.NewAPITokenService(db, apiTokenRepo, userRepo)
	userUsecase := usecase.NewUserServiceWithFirebaseDeleter(db, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo, songRepo, worldsendChartRepo, masterCache, firebaseUserDeleter)
	playerDataUsecase := usecase.NewPlayerDataService(tm, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo, honorRepo, playerDataRepo, masterCache)
	temporaryPlayerDataRepo := infra.NewTemporaryPlayerDataRepository(info.TempDataMaxEntriesPerIP, cfg.TempData.MaxTotalMB*1024*1024)
	temporaryPlayerDataUsecase := usecase.NewTemporaryPlayerDataUsecase(db, temporaryPlayerDataRepo, playerDataUsecase, info.TempDataTTL)
	songUsecase := usecase.NewSongService(songRepo, masterCache, tm, db)
	chartStatsMasterProvider := masterdata.NewChartStatsMasterProviderAdapter(staticMasterCache)
	chartStatsUsecase := usecase.NewChartStatsUsecase(songRepo, worldsendChartRepo, chartStatsRepo, masterCache, chartStatsMasterProvider, db, staticDB)
	worldsendUsecase := usecase.NewWorldsendUsecase(worldsendChartRepo, tm, db)
	goalUsecase := usecase.NewGoalUsecase(db, tm, goalRepo, masterCache)
	masterDataUsecase := usecase.NewMasterDataUsecase(masterCache, chartStatsMasterProvider)

	// DI - Handlers
	firebaseAuthUsecase := usecase.NewFirebaseAuthUsecase(db, userRepo, firebaseTokenVerifier)
	signupUsecase := usecase.NewSignupUsecase(tm, userRepo, firebaseTokenVerifier, masterCache)
	handlers := &Handlers{
		Signup:              api_internal.NewSignupHandler(signupUsecase),
		Profile:             api_internal.NewProfileHandler(userCredentialUsecase),
		User:                api_internal.NewUserHandler(userUsecase),
		AdminUser:           api_internal.NewAdminUserHandler(userUsecase),
		Song:                api_internal.NewSongHandler(songUsecase, chartStatsUsecase, masterCache, staticMasterCache),
		Worldsend:           api_internal.NewWorldsendHandler(worldsendUsecase, masterCache),
		APIToken:            api_internal.NewAPITokenHandler(apiTokenUsecase),
		Me:                  api_internal.NewMeHandler(playerDataUsecase),
		MasterData:          api_internal.NewMasterDataHandler(masterDataUsecase),
		Goal:                api_internal.NewGoalHandler(goalUsecase),
		TemporaryPlayerData: api_internal.NewTemporaryPlayerDataHandler(temporaryPlayerDataUsecase),
		// 外部API v1 用ハンドラ
		V1Song:      api_v1.NewV1SongHandler(songUsecase, chartStatsUsecase, masterCache, staticMasterCache),
		V1Worldsend: api_v1.NewV1WorldsendHandler(worldsendUsecase, masterCache),
		V1User:      api_v1.NewV1UserHandler(userUsecase),
		V1Version:   api_v1.NewV1VersionHandler(masterDataUsecase),
		// chunirec互換APIハンドラ
		Chunirec: chunirec.NewChunirecHandler(songUsecase, userUsecase, masterCache),
	}

	// ルートの設定
	rootCORS := echoMiddleware.CORSWithConfig(newExternalCORSConfig(cfg))
	e.OPTIONS("/", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	}, rootCORS)
	// TODO: 外部向けhealthエンドポイントを作る
	e.GET("/", func(c echo.Context) error {
		// 将来的に変更の可能性あり
		return c.JSON(http.StatusOK, map[string]string{
			"app_name": "chunisupport-api",
		})
	}, rootCORS)
	e.GET("/health", handleHealth(db), middleware.APITokenMiddleware(apiTokenUsecase), middleware.RequireRole(info.AccountTypeAdmin))

	// ルートの登録
	registerRoutes(e, handlers, firebaseAuthUsecase, apiTokenUsecase, cfg)

	return e
}

func requireRecentSignInVerifier(firebaseTokenVerifier usecase.TokenVerifier) usecase.RecentSignInVerifier {
	if firebaseTokenVerifier == nil {
		return nil
	}

	recentSignInVerifier, ok := firebaseTokenVerifier.(usecase.RecentSignInVerifier)
	if !ok {
		panic(fmt.Sprintf("firebase token verifier must implement recent sign-in verifier: %T", firebaseTokenVerifier))
	}

	return recentSignInVerifier
}

// registerRoutes はすべてのルートを登録します
func registerRoutes(e *echo.Echo, handlers *Handlers, firebaseAuthenticator middleware.FirebaseAuthenticator, apiTokenUsecase usecase.APITokenUsecase, cfg config.Config) {
	// api.chunisupport.net/internal
	internal := e.Group("/internal")

	// Firebase認証ミドルウェア
	firebaseAuth := middleware.FirebaseIDTokenMiddleware(firebaseAuthenticator)
	optionalFirebaseAuth := middleware.OptionalFirebaseIDTokenMiddleware(firebaseAuthenticator)
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
		// Firebase経由の初回登録: 1分間に5回まで
		authGroup.POST("/signup", handlers.Signup.Signup, middleware.IPRateLimitMiddleware(middleware.RateLimitConfig{
			Requests: info.RegisterRateLimitRequests,
			Window:   info.RegisterRateLimitWindow,
		}))
		authGroup.GET("/api-tokens", handlers.APIToken.GetStatus, firebaseAuth)
		authGroup.POST("/api-tokens", handlers.APIToken.Generate, firebaseAuth)
		authGroup.DELETE("/api-tokens", handlers.APIToken.Delete, firebaseAuth)
	}

	// api.chunisupport.net/internal/me
	meGroup := internal.Group("/me")
	meGroup.Use(firebaseAuth)
	{
		meGroup.GET("", handlers.Profile.Me)
		meGroup.PUT("/privacy", handlers.Profile.UpdatePrivacy)
		meGroup.DELETE("", handlers.Profile.DeleteAccount)
		meGroup.POST("/register-data", handlers.Me.RegisterData, middleware.UserRateLimitMiddleware(middleware.RateLimitConfig{
			Requests: info.RegisterDataRateLimitRequests,
			Window:   info.RegisterDataRateLimitWindow,
		}))
		meGroup.DELETE("/player-data", handlers.Me.DeletePlayerData)
		meGroup.GET("/goals", handlers.Goal.List)
		meGroup.POST("/goals", handlers.Goal.Create)
		meGroup.PUT("/goals/:id", handlers.Goal.Update)
		meGroup.DELETE("/goals/:id", handlers.Goal.Delete)
	}

	temporaryPlayerDataGroup := internal.Group("/player-data")
	tempDataCORS := echoMiddleware.CORSWithConfig(newExternalCORSConfig(cfg))
	temporaryPlayerDataGroup.OPTIONS("/temp", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	}, tempDataCORS)
	temporaryPlayerDataGroup.POST("/temp", handlers.TemporaryPlayerData.CreateTemporaryData, tempDataCORS, middleware.IPRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.TempDataRateLimitPerMin,
		Window:   info.TempDataRateLimitWindow,
	}))
	temporaryPlayerDataGroup.POST("/commit", handlers.TemporaryPlayerData.CommitTemporaryData, firebaseAuth, middleware.UserRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.RegisterDataRateLimitRequests,
		Window:   info.RegisterDataRateLimitWindow,
	}))

	// api.chunisupport.net/internal/users
	publicUsersGroup := internal.Group("/users")
	publicUsersGroup.Use(optionalFirebaseAuth, anonymousRateLimit)
	{
		publicUsersGroup.GET("/:username/profile", handlers.User.GetUserProfile)
		publicUsersGroup.GET("/:username/updated-at", handlers.User.GetUserUpdatedAt)
		publicUsersGroup.GET("/:username/rating", handlers.User.GetUserRating)
		publicUsersGroup.GET("/:username/record", handlers.User.GetUserRecord)
		publicUsersGroup.GET("/:username", handlers.User.GetUserProfileWithRecords)
	}

	usersGroup := internal.Group("/users")
	usersGroup.Use(firebaseAuth)
	{
		usersGroup.GET("/", handlers.AdminUser.GetAllUsers, requireAdmin)
		usersGroup.DELETE("/:username", handlers.User.DeleteUser, requireAdmin)
	}

	// api.chunisupport.net/internal/songs
	publicSongsGroup := internal.Group("/songs")
	publicSongsGroup.Use(optionalFirebaseAuth, anonymousRateLimit)
	{
		publicSongsGroup.GET("/updated-at", handlers.Song.GetSongsUpdatedAt)
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
	songsGroup.Use(firebaseAuth)
	{
		songsGroup.POST("", handlers.Song.CreateSong, requireAdmin)
		songsGroup.PUT("", handlers.Song.UpdateSongs, requireEditor)
		songsGroup.DELETE("/:displayid", handlers.Song.DeleteSong, requireAdmin)
		songsGroup.POST("/:displayid/restore", handlers.Song.RestoreSong, requireEditor)

		// WORLD'S END 楽曲エンドポイント
		worldsendGroup := songsGroup.Group("/worldsend")
		{
			worldsendGroup.POST("", handlers.Worldsend.CreateWorldsendSong, requireAdmin)
			worldsendGroup.PUT("", handlers.Worldsend.UpdateWorldsendSongs, requireEditor)
			worldsendGroup.DELETE("/:displayid", handlers.Worldsend.DeleteWorldsendSong, requireAdmin)
			worldsendGroup.POST("/:displayid/restore", handlers.Worldsend.RestoreWorldsendSong, requireEditor)
		}
	}

	editorSongsGroup := internal.Group("/editor/songs")
	editorSongsGroup.Use(firebaseAuth, requireEditor)
	{
		editorSongsGroup.GET("", handlers.Song.GetEditorSongs)
		editorSongsGroup.GET("/:displayid", handlers.Song.GetEditorSong)
		editorSongsGroup.GET("/worldsend", handlers.Worldsend.GetEditorWorldsendSongs)
		editorSongsGroup.GET("/worldsend/:displayid", handlers.Worldsend.GetEditorWorldsendSong)
	}

	// api.chunisupport.net/internal/master
	masterGroup := internal.Group("/master")
	masterGroup.Use(firebaseAuth)
	{
		masterGroup.GET("", handlers.MasterData.GetMasterData)
	}

	{
		masterGroup.GET("/versions", handlers.MasterData.GetVersions)
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
		apiV1.GET("/master/versions", handlers.V1Version.GetVersions)
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

func newDefaultCORSConfig(cfg config.Config) echoMiddleware.CORSConfig {
	return newCORSConfig(cfg.CORS.AllowOrigins, cfg, func(c echo.Context) bool {
		return isExternalCORSPath(c.Request().URL.Path)
	})
}

func newExternalCORSConfig(cfg config.Config) echoMiddleware.CORSConfig {
	allowOrigins := slices.Clone(cfg.CORS.AllowOrigins)
	if !slices.Contains(allowOrigins, info.ExternalCORSAllowOrigin) {
		allowOrigins = append(allowOrigins, info.ExternalCORSAllowOrigin)
	}

	return newCORSConfig(allowOrigins, cfg, nil)
}

func isExternalCORSPath(path string) bool {
	// TODO: 前者は外部向けhealthエンドポイントに変えるつもり
	return path == "/" || path == "/internal/player-data/temp"
}

func newCORSConfig(allowOrigins []string, cfg config.Config, skipper echoMiddleware.Skipper) echoMiddleware.CORSConfig {
	return echoMiddleware.CORSConfig{
		Skipper:          skipper,
		AllowOrigins:     allowOrigins,
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
			echo.HeaderContentEncoding,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			"X-Reauth-Token",
		},
		ExposeHeaders: []string{
			echo.HeaderContentLength,
		},
		MaxAge: cfg.CORS.MaxAge,
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
