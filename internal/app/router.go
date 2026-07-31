package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/chunirec"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/reiwa"
	"github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	vo_username "github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	infra "github.com/chunisupport/chunisupport-api/internal/infra/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/transaction"
	"github.com/chunisupport/chunisupport-api/internal/infra/turnstile"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
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
	Login                *api_internal.LoginHandler
	Signup               *api_internal.SignupHandler
	Profile              *api_internal.ProfileHandler
	User                 *api_internal.UserHandler
	AdminUser            *api_internal.AdminUserHandler
	AdminUserStatistics  *api_internal.AdminUserStatisticsHandler
	Song                 *api_internal.SongHandler
	Honor                *api_internal.HonorHandler
	Worldsend            *api_internal.WorldsendHandler
	APIToken             *api_internal.APITokenHandler
	Me                   *api_internal.MeHandler
	MasterData           *api_internal.MasterDataHandler
	Goal                 *api_internal.GoalHandler
	GoalGroup            *api_internal.GoalGroupHandler
	RecordFilter         *api_internal.RecordFilterHandler
	TemporaryPlayerData  *api_internal.TemporaryPlayerDataHandler
	PlayerLockedSong     *api_internal.PlayerLockedSongHandler
	PlayerFavoriteSong   *api_internal.PlayerFavoriteSongHandler
	Friendship           *api_internal.FriendshipHandler
	FriendChartRanking   *api_internal.FriendChartRankingHandler
	BestSlotStats        *api_internal.BestSlotStatsHandler
	InternalScoreHistory *api_internal.ScoreHistoryHandler
	Course               *api_internal.CourseHandler
	SystemMaintenance    *api_internal.SystemMaintenanceHandler
	// 外部API v1 用ハンドラ
	V1Song       *api_v1.V1SongHandler
	V1Worldsend  *api_v1.V1WorldsendHandler
	V1User       *api_v1.V1UserHandler
	V1Version    *api_v1.V1VersionHandler
	ScoreHistory *api_v1.ScoreHistoryHandler
	V1Course     *api_v1.V1CourseHandler
	// chunirec互換APIハンドラ
	Chunirec *chunirec.ChunirecHandler
	// reiwa互換APIハンドラ
	Reiwa *reiwa.ReiwaHandler
}

// NewRouter はルートが設定された新しいEchoインスタンスを作成します
// echoLogWriterがnilの場合は、テストなどの直接構築時にアクセスログミドルウェアを無効化します。
func NewRouter(ctx context.Context, db *sqlx.DB, cfg config.Config, masterCache *masterdata.Cache, staticMasterCache *masterdata.StaticCache, firebaseTokenVerifier usecase.TokenVerifier, firebaseUserDeleter usecase.FirebaseUserDeleter, echoLogWriter io.Writer) (*echo.Echo, error) {
	e := echo.New()
	e.Validator = NewCustomValidator()
	e.JSONSerializer = NewTimezoneJSONSerializer(cfg.Location)

	// カスタムエラーハンドラーの設定
	e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler

	// ミドルウェアの設定
	// Echoのロガーを設定
	if echoLogWriter != nil {
		e.Logger = slog.New(slog.NewTextHandler(echoLogWriter, nil))
		e.Use(echoMiddleware.RequestLogger())
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
	scoreHistoryRepo := infra.NewScoreHistoryRepository(db)
	worldsendChartRepo := infra.NewWorldsendChartRepository(db)
	chartStatsRepo := infra.NewChartStatsRepository(db)
	apiTokenRepo := infra.NewAPITokenRepository(db)
	songRepo := infra.NewSongRepository(db)
	goalRepo := infra.NewGoalRepository(db)
	goalGroupRepo := infra.NewGoalGroupRepository()
	recordFilterRepo := infra.NewRecordFilterRepository(db)
	honorRepo := infra.NewHonorRepository(db)
	playerLockedSongRepo := infra.NewPlayerLockedSongRepository()
	playerFavoriteSongRepo := infra.NewPlayerFavoriteSongRepository()
	playerFavoriteSongQueryService := infra.NewPlayerFavoriteSongQueryService()
	playerFavoriteSongLocker := infra.NewPlayerFavoriteSongLocker()
	friendshipRepo := infra.NewFriendshipRepository()
	friendChartRankingQueryService := infra.NewFriendChartRankingQueryService()
	bestSlotRankingQueryService := infra.NewBestSlotRankingQueryService(db)
	overpowerDenominatorProvider := infra.NewOverpowerDenominatorProvider(db)
	userUpdatedAtQuery := infra.NewUserUpdatedAtQueryService()
	adminUserStatisticsQuery := infra.NewAdminUserStatisticsQueryService(db)
	courseRepo := infra.NewCourseRepository(db)
	systemMaintenanceRepo := infra.NewSystemMaintenanceRepository(db)
	tm := transaction.NewTransactionManager(db)
	systemMaintenanceUsecase, err := usecase.NewSystemMaintenanceUsecase(ctx, systemMaintenanceRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize system maintenance state: %w", err)
	}
	recentSignInVerifier := requireRecentSignInVerifier(firebaseTokenVerifier)
	userCredentialUsecase := usecase.NewUserCredentialUsecaseWithFirebaseServices(db, tm, userRepo, playerRecordRepo, recentSignInVerifier, firebaseUserDeleter, masterCache)
	apiTokenUsecase := usecase.NewAPITokenUsecase(db, tm, apiTokenRepo, userRepo)
	userUsecase := usecase.NewUserUsecaseWithFirebaseDeleterAndOverpowerDenominator(db, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo, songRepo, worldsendChartRepo, masterCache, firebaseUserDeleter, playerLockedSongRepo, overpowerDenominatorProvider, userUpdatedAtQuery)
	if configurable, ok := userUsecase.(interface {
		SetFriendshipRepository(repository.FriendshipRepository)
	}); ok {
		configurable.SetFriendshipRepository(friendshipRepo)
	}
	if configurable, ok := userUsecase.(interface {
		SetCourseRepository(repository.CourseRepository)
	}); ok {
		configurable.SetCourseRepository(courseRepo)
	}
	playerDataUsecase := usecase.NewPlayerDataUsecaseWithScoreHistory(tm, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo, honorRepo, playerDataRepo, playerLockedSongRepo, masterCache, scoreHistoryRepo, courseRepo)
	courseUsecase := usecase.NewCourseUsecase(db, courseRepo, userRepo, friendshipRepo)
	scoreHistoryUsecase := usecase.NewScoreHistoryUsecase(db, userRepo, songRepo, worldsendChartRepo, scoreHistoryRepo, masterCache)
	if configurable, ok := scoreHistoryUsecase.(interface {
		SetFriendshipRepository(repository.FriendshipRepository)
	}); ok {
		configurable.SetFriendshipRepository(friendshipRepo)
	}
	temporaryPlayerDataRepo := infra.NewTemporaryPlayerDataRepository(info.TempDataMaxEntriesPerIP, cfg.TempData.MaxTotalMB*1024*1024)
	temporaryPlayerDataUsecase := usecase.NewTemporaryPlayerDataUsecase(db, temporaryPlayerDataRepo, playerDataUsecase, info.TempDataTTL)
	songUsecase := usecase.NewSongUsecaseWithCascadeDelete(songRepo, masterCache, tm, db, overpowerDenominatorProvider, playerFavoriteSongRepo, playerLockedSongRepo)
	honorUsecase := usecase.NewHonorUsecase(honorRepo, masterCache, tm, db)
	chartStatsMasterProvider := masterdata.NewChartStatsMasterProviderAdapter(staticMasterCache)
	chartStatsUsecase := usecase.NewChartStatsUsecase(songRepo, worldsendChartRepo, chartStatsRepo, masterCache, chartStatsMasterProvider, db)
	worldsendUsecase := usecase.NewWorldsendUsecase(worldsendChartRepo, tm, db)
	goalUsecase := usecase.NewGoalUsecase(db, tm, goalRepo, masterCache, goalGroupRepo)
	goalGroupUsecase := usecase.NewGoalGroupUsecase(db, tm, goalGroupRepo, goalRepo)
	recordFilterUsecase := usecase.NewRecordFilterUsecase(recordFilterRepo)
	playerLockedSongQueryService := infra.NewPlayerLockedSongQueryService()
	playerSongIDResolver := infra.NewPlayerSongIDResolver()
	playerLockedSongUsecase, err := usecase.NewPlayerLockedSongUsecase(db, tm, userRepo, playerRepo, playerRecordRepo, playerDataRepo, songRepo, playerLockedSongRepo, playerLockedSongQueryService, playerSongIDResolver)
	if err != nil {
		panic(fmt.Sprintf("failed to create player locked song usecase: %v", err))
	}
	if configurable, ok := playerLockedSongUsecase.(interface {
		SetFriendshipRepository(repository.FriendshipRepository)
	}); ok {
		configurable.SetFriendshipRepository(friendshipRepo)
	}
	playerFavoriteSongUsecase, err := usecase.NewPlayerFavoriteSongUsecase(db, tm, userRepo, playerRepo, songRepo, playerFavoriteSongRepo, playerFavoriteSongQueryService, playerFavoriteSongLocker, playerSongIDResolver)
	if err != nil {
		panic(fmt.Sprintf("failed to create player favorite song usecase: %v", err))
	}
	if configurable, ok := playerFavoriteSongUsecase.(interface {
		SetFriendshipRepository(repository.FriendshipRepository)
	}); ok {
		configurable.SetFriendshipRepository(friendshipRepo)
	}
	friendshipUsecase, err := usecase.NewFriendshipUsecase(db, tm, userRepo, friendshipRepo)
	if err != nil {
		panic(fmt.Sprintf("failed to create friendship usecase: %v", err))
	}
	friendChartRankingUsecase := usecase.NewFriendChartRankingUsecase(db, friendChartRankingQueryService)
	bestSlotRankingUsecase := usecase.NewBestSlotRankingUsecase(bestSlotRankingQueryService, chartStatsMasterProvider)
	masterDataUsecase := usecase.NewMasterDataUsecase(masterCache, chartStatsMasterProvider)

	// DI - Handlers
	turnstileVerifier := turnstile.NewVerifier(cfg.Turnstile.SecretKey)
	firebaseAuthUsecaseStrict := usecase.NewFirebaseAuthUsecase(db, userRepo, firebaseTokenVerifier)
	firebaseAuthUsecaseReadOptimized := usecase.NewFirebaseAuthUsecase(db, userRepo, usecase.NewReadOptimizedTokenVerifier(firebaseTokenVerifier))
	loginUsecase := usecase.NewLoginUsecase(firebaseAuthUsecaseStrict, turnstileVerifier, masterCache, systemMaintenanceUsecase)
	usernamePolicy, err := service.NewForbiddenUsernamePolicy(cfg.UsernamePolicy.Exact, cfg.UsernamePolicy.Contains)
	if err != nil {
		panic(fmt.Sprintf("failed to create username policy: %v", err))
	}
	signupUsecase := usecase.NewSignupUsecase(tm, userRepo, firebaseTokenVerifier, turnstileVerifier, masterCache, usernamePolicy)
	adminUserStatisticsUsecase := usecase.NewAdminUserStatisticsUsecase(adminUserStatisticsQuery)
	handlers := &Handlers{
		Login:                api_internal.NewLoginHandler(loginUsecase),
		Signup:               api_internal.NewSignupHandler(signupUsecase),
		Profile:              api_internal.NewProfileHandler(userCredentialUsecase),
		User:                 api_internal.NewUserHandler(userUsecase),
		AdminUser:            api_internal.NewAdminUserHandler(userUsecase),
		AdminUserStatistics:  api_internal.NewAdminUserStatisticsHandler(adminUserStatisticsUsecase),
		Song:                 api_internal.NewSongHandler(songUsecase, chartStatsUsecase, masterCache, staticMasterCache),
		Honor:                api_internal.NewHonorHandler(honorUsecase),
		Worldsend:            api_internal.NewWorldsendHandler(worldsendUsecase, masterCache),
		APIToken:             api_internal.NewAPITokenHandler(apiTokenUsecase),
		Me:                   api_internal.NewMeHandler(playerDataUsecase),
		MasterData:           api_internal.NewMasterDataHandler(masterDataUsecase),
		Goal:                 api_internal.NewGoalHandler(goalUsecase),
		GoalGroup:            api_internal.NewGoalGroupHandler(goalGroupUsecase),
		RecordFilter:         api_internal.NewRecordFilterHandler(recordFilterUsecase),
		TemporaryPlayerData:  api_internal.NewTemporaryPlayerDataHandler(temporaryPlayerDataUsecase),
		PlayerLockedSong:     api_internal.NewPlayerLockedSongHandler(playerLockedSongUsecase),
		PlayerFavoriteSong:   api_internal.NewPlayerFavoriteSongHandler(playerFavoriteSongUsecase),
		Friendship:           api_internal.NewFriendshipHandler(friendshipUsecase),
		FriendChartRanking:   api_internal.NewFriendChartRankingHandler(friendChartRankingUsecase),
		BestSlotStats:        api_internal.NewBestSlotStatsHandler(bestSlotRankingUsecase, chartStatsUsecase, chartStatsMasterProvider),
		InternalScoreHistory: api_internal.NewScoreHistoryHandler(scoreHistoryUsecase),
		Course:               api_internal.NewCourseHandler(courseUsecase),
		SystemMaintenance:    api_internal.NewSystemMaintenanceHandler(systemMaintenanceUsecase),
		// 外部API v1 用ハンドラ
		V1Song:       api_v1.NewV1SongHandler(songUsecase, chartStatsUsecase, masterCache, staticMasterCache),
		V1Worldsend:  api_v1.NewV1WorldsendHandler(worldsendUsecase, masterCache),
		V1User:       api_v1.NewV1UserHandler(userUsecase),
		V1Version:    api_v1.NewV1VersionHandler(masterDataUsecase),
		ScoreHistory: api_v1.NewScoreHistoryHandler(scoreHistoryUsecase),
		V1Course:     api_v1.NewV1CourseHandler(courseUsecase),
		// chunirec互換APIハンドラ
		Chunirec: chunirec.NewChunirecHandler(songUsecase, userUsecase, masterCache, cfg.Location),
		// reiwa互換APIハンドラ
		Reiwa: reiwa.NewReiwaHandler(songUsecase, masterCache),
	}

	// ルートの設定
	healthzCORS := echoMiddleware.CORSWithConfig(newExternalCORSConfig(cfg))
	firebaseMaintenance := middleware.FirebaseMaintenanceMiddleware(systemMaintenanceUsecase, firebaseAuthUsecaseStrict)
	apiTokenMaintenance := middleware.APITokenMaintenanceMiddleware(systemMaintenanceUsecase, apiTokenUsecase)
	e.OPTIONS("/healthz", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	}, healthzCORS)
	e.GET("/healthz", handleExternalHealth, healthzCORS)
	e.GET("/", handleRoot, firebaseMaintenance)
	e.GET("/version", handleVersion, apiTokenMaintenance, middleware.APITokenMiddleware(apiTokenUsecase), middleware.RequireRole(info.AccountTypeAdmin))

	// ルートの登録
	registerRoutes(e, handlers, firebaseAuthUsecaseStrict, firebaseAuthUsecaseReadOptimized, apiTokenUsecase, systemMaintenanceUsecase, cfg)

	return e, nil
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
func registerRoutes(
	e *echo.Echo,
	handlers *Handlers,
	firebaseAuthenticatorStrict middleware.FirebaseAuthenticator,
	firebaseAuthenticatorReadOptimized middleware.FirebaseAuthenticator,
	apiTokenUsecase usecase.APITokenUsecase,
	maintenanceStateProvider usecase.MaintenanceStateProvider,
	cfg config.Config,
) {
	// api.chunisupport.net/internal
	internal := e.Group("/internal")
	// 一時保存APIの拡張CORSは、メンテナンス503にも適用できるようゲートより先に評価します。
	internal.Use(echoMiddleware.CORSWithConfig(newTemporaryPlayerDataCORSConfig(cfg)))
	internal.Use(middleware.FirebaseMaintenanceMiddleware(maintenanceStateProvider, firebaseAuthenticatorStrict))

	// Firebase認証ミドルウェア
	firebaseAuthStrict := middleware.FirebaseIDTokenMiddleware(firebaseAuthenticatorStrict)
	optionalFirebaseAuthStrict := middleware.OptionalFirebaseIDTokenMiddleware(firebaseAuthenticatorStrict)
	optionalFirebaseAuthReadOptimized := middleware.OptionalFirebaseIDTokenMiddleware(firebaseAuthenticatorReadOptimized)
	anonymousRateLimit := middleware.AnonymousIPRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.InternalPublicRateLimitRequests,
		Window:   info.InternalPublicRateLimitWindow,
	})

	// EDITOR以上の権限を要求するミドルウェア
	requireEditor := middleware.RequireRole(info.AccountTypeEditor)

	// ADMIN以上の権限を要求するミドルウェア
	requireAdmin := middleware.RequireRole(info.AccountTypeAdmin)

	internal.GET("/system/status", handlers.SystemMaintenance.Status)

	// api.chunisupport.net/internal/auth
	authGroup := internal.Group("/auth")
	{
		authGroup.POST("/login", handlers.Login.Login, middleware.IPRateLimitMiddleware(middleware.RateLimitConfig{
			Requests: info.LoginRateLimitRequests,
			Window:   info.LoginRateLimitWindow,
		}))
		// Firebase経由の初回登録: 1分間に5回まで
		authGroup.POST("/signup", handlers.Signup.Signup, middleware.IPRateLimitMiddleware(middleware.RateLimitConfig{
			Requests: info.RegisterRateLimitRequests,
			Window:   info.RegisterRateLimitWindow,
		}))
		authGroup.GET("/api-tokens", handlers.APIToken.List, firebaseAuthStrict)
		authGroup.POST("/api-tokens", handlers.APIToken.Generate, firebaseAuthStrict)
		authGroup.PATCH("/api-tokens/:id", handlers.APIToken.Rename, firebaseAuthStrict)
		authGroup.DELETE("/api-tokens/:id", handlers.APIToken.Delete, firebaseAuthStrict)
	}

	// api.chunisupport.net/internal/me
	meGroup := internal.Group("/me")
	meGroup.Use(firebaseAuthStrict)
	{
		meGroup.GET("", handlers.Profile.Me)
		meGroup.PUT("/privacy", handlers.Profile.UpdatePrivacy)
		meGroup.DELETE("", handlers.Profile.DeleteAccount)
		meGroup.POST("/register-data", handlers.Me.RegisterData, middleware.UserRateLimitMiddleware(middleware.RateLimitConfig{
			Requests: info.RegisterDataRateLimitRequests,
			Window:   info.RegisterDataRateLimitWindow,
		}))
		meGroup.GET("/player-data/latest-update", handlers.Me.GetLatestPlayerUpdate)
		meGroup.DELETE("/player-data", handlers.Me.DeletePlayerData)
		meGroup.GET("/goals", handlers.Goal.List)
		meGroup.POST("/goals", handlers.Goal.Create)
		meGroup.PUT("/goals/order", handlers.Goal.Reorder)
		meGroup.PUT("/goals/:id", handlers.Goal.Update)
		meGroup.DELETE("/goals/:id", handlers.Goal.Delete)
		meGroup.GET("/goal-groups", handlers.GoalGroup.List)
		meGroup.POST("/goal-groups", handlers.GoalGroup.Create)
		meGroup.PUT("/goal-groups/order", handlers.GoalGroup.Reorder)
		meGroup.PUT("/goal-groups/:id", handlers.GoalGroup.Update)
		meGroup.DELETE("/goal-groups/:id", handlers.GoalGroup.Delete)
		meGroup.GET("/record-filters", handlers.RecordFilter.List)
		meGroup.POST("/record-filters", handlers.RecordFilter.Create)
		meGroup.PUT("/record-filters/:id", handlers.RecordFilter.Update)
		meGroup.DELETE("/record-filters/:id", handlers.RecordFilter.Delete)
		meGroup.POST("/locked-songs", handlers.PlayerLockedSong.Lock)
		meGroup.POST("/locked-songs/batch", handlers.PlayerLockedSong.Batch)
		meGroup.DELETE("/locked-songs/:displayid", handlers.PlayerLockedSong.Unlock)
		meGroup.POST("/favorite-songs", handlers.PlayerFavoriteSong.Add)
		meGroup.DELETE("/favorite-songs/:displayid", handlers.PlayerFavoriteSong.Remove)
	}

	friendshipGroup := internal.Group("/friends")
	friendshipGroup.Use(firebaseAuthStrict)
	{
		friendshipGroup.GET("", handlers.Friendship.ListFriends)
		friendshipGroup.POST("/requests", handlers.Friendship.SendRequest)
		friendshipGroup.GET("/requests/received", handlers.Friendship.ListReceivedRequests)
		friendshipGroup.GET("/requests/sent", handlers.Friendship.ListSentRequests)
		friendshipGroup.POST("/requests/:user_id/accept", handlers.Friendship.AcceptRequest)
		friendshipGroup.POST("/requests/:user_id/reject", handlers.Friendship.RejectRequest)
		friendshipGroup.DELETE("/requests/:user_id", handlers.Friendship.CancelRequest)
		friendshipGroup.DELETE("/:user_id", handlers.Friendship.Remove)
	}

	friendRankingGroup := internal.Group("/friend-rankings")
	friendRankingGroup.Use(firebaseAuthStrict)
	{
		friendRankingGroup.GET("/songs/:displayid/charts/:difficulty", handlers.FriendChartRanking.GetStandard)
		friendRankingGroup.GET("/worldsend-songs/:displayid", handlers.FriendChartRanking.GetWorldsend)
	}

	temporaryPlayerDataGroup := internal.Group("/player-data")
	temporaryPlayerDataGroup.OPTIONS("/temp", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	temporaryPlayerDataGroup.POST("/temp", handlers.TemporaryPlayerData.CreateTemporaryData, middleware.IPRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.TempDataRateLimitPerMin,
		Window:   info.TempDataRateLimitWindow,
	}))
	temporaryPlayerDataGroup.POST("/commit", handlers.TemporaryPlayerData.CommitTemporaryData, firebaseAuthStrict, middleware.UserRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.RegisterDataRateLimitRequests,
		Window:   info.RegisterDataRateLimitWindow,
	}))

	// api.chunisupport.net/internal/users
	userUpdatedAtGroup := internal.Group("/users")
	userUpdatedAtGroup.Use(optionalFirebaseAuthReadOptimized, anonymousRateLimit)
	userUpdatedAtGroup.GET("/:username/updated-at", handlers.User.GetUserUpdatedAt)

	publicUsersGroup := internal.Group("/users")
	publicUsersGroup.Use(optionalFirebaseAuthStrict, anonymousRateLimit)
	{
		publicUsersGroup.GET("/:username/profile", handlers.User.GetUserProfile)
		publicUsersGroup.GET("/:username/rating", handlers.User.GetUserRating)
		publicUsersGroup.GET("/:username/record/songs/:displayid", handlers.User.GetUserSongRecord)
		publicUsersGroup.GET("/:username/record/songs/:displayid/:difficulty/history", handlers.InternalScoreHistory.GetStandard)
		publicUsersGroup.GET("/:username/record/worldsend-songs/:displayid", handlers.User.GetUserWorldsendSongRecord)
		publicUsersGroup.GET("/:username/record/courses", handlers.Course.GetUserRecords)
		publicUsersGroup.GET("/:username/record/courses/:displayid", handlers.Course.GetUserRecord)
		publicUsersGroup.GET("/:username/record/worldsend-songs/:displayid/history", handlers.InternalScoreHistory.GetWorldsend)
		publicUsersGroup.GET("/:username/record", handlers.User.GetUserRecord)
		publicUsersGroup.GET("/:username/locked-songs", handlers.PlayerLockedSong.List)
		publicUsersGroup.GET("/:username/favorite-songs", handlers.PlayerFavoriteSong.List)
		publicUsersGroup.GET("/:username", handlers.User.GetUserProfileWithRecords)
	}

	usersGroup := internal.Group("/users")
	usersGroup.Use(firebaseAuthStrict)
	{
		usersGroup.GET("/", handlers.AdminUser.GetAllUsers, requireAdmin)
		usersGroup.DELETE("/:username", handlers.User.DeleteUser, requireAdmin)
	}

	adminGroup := internal.Group("/admin")
	adminGroup.Use(firebaseAuthStrict, requireAdmin)
	{
		adminGroup.GET("/build-info", handleAdminBuildInfo)
		adminGroup.GET("/user-stats", handlers.AdminUserStatistics.Get)
		adminGroup.PUT("/maintenance", handlers.SystemMaintenance.Update)
	}

	// api.chunisupport.net/internal/honors
	honorsGroup := internal.Group("/honors")
	honorsGroup.Use(firebaseAuthStrict, requireAdmin)
	{
		honorsGroup.GET("", handlers.Honor.ListHonors)
		honorsGroup.GET("/:id", handlers.Honor.GetHonor)
		honorsGroup.POST("", handlers.Honor.CreateHonor)
		honorsGroup.PUT("/:id", handlers.Honor.UpdateHonor)
		honorsGroup.DELETE("/:id", handlers.Honor.DeleteHonor)
	}

	// api.chunisupport.net/internal/songs
	publicSongsGroup := internal.Group("/songs")
	publicSongsGroup.Use(optionalFirebaseAuthReadOptimized, anonymousRateLimit)
	{
		publicSongsGroup.GET("/updated-at", handlers.Song.GetSongsUpdatedAt)
		publicSongsGroup.GET("", handlers.Song.GetSongs)
		publicSongsGroup.GET("/:displayid", handlers.Song.GetSong)
		publicSongsGroup.GET("/:displayid/stats/:difficulty", handlers.Song.GetChartStatsByDifficulty)
		publicSongsGroup.GET("/:displayid/best-slot-stats/:difficulty", handlers.BestSlotStats.GetChartStats)
	}

	bestSlotRankingGroup := internal.Group("/best-slot-rankings")
	bestSlotRankingGroup.Use(optionalFirebaseAuthReadOptimized, anonymousRateLimit)
	{
		bestSlotRankingGroup.GET("", handlers.BestSlotStats.GetRanking)
	}

	// api.chunisupport.net/internal/worldsend-songs
	publicWorldsendGroup := internal.Group("/worldsend-songs")
	publicWorldsendGroup.Use(optionalFirebaseAuthReadOptimized, anonymousRateLimit)
	{
		publicWorldsendGroup.GET("", handlers.Worldsend.GetWorldsendSongs)
		publicWorldsendGroup.GET("/:displayid", handlers.Worldsend.GetWorldsendSong)
	}

	songsGroup := internal.Group("/songs")
	songsGroup.Use(firebaseAuthStrict)
	{
		songsGroup.POST("", handlers.Song.CreateSong, requireAdmin)
		songsGroup.PUT("", handlers.Song.UpdateSongs, requireEditor)
		songsGroup.DELETE("/:displayid", handlers.Song.DeleteSong, requireAdmin)
		songsGroup.POST("/:displayid/restore", handlers.Song.RestoreSong, requireEditor)
	}

	worldsendGroup := internal.Group("/worldsend-songs")
	worldsendGroup.Use(firebaseAuthStrict)
	{
		worldsendGroup.POST("", handlers.Worldsend.CreateWorldsendSong, requireAdmin)
		worldsendGroup.PUT("", handlers.Worldsend.UpdateWorldsendSongs, requireEditor)
		worldsendGroup.DELETE("/:displayid", handlers.Worldsend.DeleteWorldsendSong, requireAdmin)
		worldsendGroup.POST("/:displayid/restore", handlers.Worldsend.RestoreWorldsendSong, requireEditor)
	}

	editorSongsGroup := internal.Group("/editor/songs")
	editorSongsGroup.Use(firebaseAuthStrict, requireEditor)
	{
		editorSongsGroup.GET("", handlers.Song.GetEditorSongs)
		editorSongsGroup.GET("/:displayid", handlers.Song.GetEditorSong)
	}

	editorWorldsendGroup := internal.Group("/editor/worldsend-songs")
	editorWorldsendGroup.Use(firebaseAuthStrict, requireEditor)
	{
		editorWorldsendGroup.GET("", handlers.Worldsend.GetEditorWorldsendSongs)
		editorWorldsendGroup.GET("/:displayid", handlers.Worldsend.GetEditorWorldsendSong)
	}

	publicCoursesGroup := internal.Group("/courses")
	publicCoursesGroup.Use(optionalFirebaseAuthReadOptimized, anonymousRateLimit)
	{
		publicCoursesGroup.GET("/updated-at", handlers.Course.GetCoursesUpdatedAt)
		publicCoursesGroup.GET("", handlers.Course.List)
		publicCoursesGroup.GET("/:displayid", handlers.Course.Get)
	}
	coursesGroup := internal.Group("/courses")
	coursesGroup.Use(firebaseAuthStrict)
	{
		coursesGroup.POST("", handlers.Course.Create, requireAdmin)
		coursesGroup.PUT("/:displayid", handlers.Course.Update, requireEditor)
		coursesGroup.DELETE("/:displayid", handlers.Course.Delete, requireAdmin)
		coursesGroup.POST("/:displayid/restore", handlers.Course.Restore, requireEditor)
	}
	editorCoursesGroup := internal.Group("/editor/courses")
	editorCoursesGroup.Use(firebaseAuthStrict, requireEditor)
	{
		editorCoursesGroup.GET("", handlers.Course.ListEditor)
		editorCoursesGroup.GET("/:displayid", handlers.Course.GetEditor)
	}

	// api.chunisupport.net/internal/master
	masterGroup := internal.Group("/master")
	{
		masterGroup.GET("", handlers.MasterData.GetMasterData)
	}

	{
		masterGroup.GET("/versions", handlers.MasterData.GetVersions)
		masterGroup.GET("/honor-types", handlers.MasterData.GetHonorTypes)
	}

	// 外部APIルートの登録
	// api.chunisupport.net/v1
	scoreHistoryV1 := e.Group("/v1")
	scoreHistoryV1.Use(middleware.APITokenMaintenanceMiddleware(maintenanceStateProvider, apiTokenUsecase))
	scoreHistoryV1.Use(middleware.OptionalAPITokenMiddleware(apiTokenUsecase))
	scoreHistoryV1.Use(middleware.OptionalAPIRateLimitMiddleware(
		info.APIRateLimitRequests,
		info.APIRateLimitEditorRequests,
		info.APIRateLimitAdminRequests,
		info.APIRateLimitWindow,
	))
	scoreHistoryV1.GET("/songs/:displayid/score-history/:difficulty", handlers.ScoreHistory.GetStandard)
	scoreHistoryV1.GET("/worldsend-songs/:displayid/score-history", handlers.ScoreHistory.GetWorldsend)

	apiV1 := e.Group("/v1")
	apiV1.Use(middleware.APITokenMaintenanceMiddleware(maintenanceStateProvider, apiTokenUsecase))
	apiV1.Use(middleware.APITokenMiddleware(apiTokenUsecase))
	// レートリミット: ADMINは15分150,000回、EDITOR/EXTDEVは15分3,000回、その他は15分150回
	apiV1.Use(middleware.APIRateLimitMiddleware(
		info.APIRateLimitRequests,
		info.APIRateLimitEditorRequests,
		info.APIRateLimitAdminRequests,
		info.APIRateLimitWindow,
	))
	{
		apiV1.GET("/songs", handlers.V1Song.GetSongs)
		apiV1.PUT("/songs", handlers.V1Song.UpdateSongs, requireEditor)
		apiV1.PATCH("/songs/chart-constant", handlers.V1Song.UpdateChartConstant, requireEditor)
		apiV1.GET("/songs/:displayid", handlers.V1Song.GetSong)
		apiV1.GET("/songs/:displayid/stats/:difficulty", handlers.V1Song.GetChartStatsByDifficulty)
		apiV1.GET("/worldsend-songs", handlers.V1Worldsend.GetWorldsendSongs)
		apiV1.GET("/worldsend-songs/:displayid", handlers.V1Worldsend.GetWorldsendSong)
		apiV1.GET("/users/:username", handlers.V1User.GetUser)
		apiV1.GET("/users/:username/records/courses", handlers.V1Course.GetUserRecords)
		apiV1.GET("/courses", handlers.V1Course.List)
		apiV1.GET("/courses/:displayid", handlers.V1Course.Get)
		apiV1.GET("/master/versions", handlers.V1Version.GetVersions)
	}

	// chunirec互換APIルートの登録
	// api.chunisupport.net/compat/chunirec/2.0
	chunirecGroup := e.Group("/compat/chunirec/2.0")
	// chunirec専用エラーハンドリング（最初に適用）
	chunirecGroup.Use(chunirec.ChunirecErrorHandlerMiddleware())
	chunirecGroup.Use(middleware.APITokenMaintenanceMiddleware(maintenanceStateProvider, apiTokenUsecase))
	chunirecGroup.Use(middleware.APITokenMiddleware(apiTokenUsecase))
	// レートリミットはv1と同じ設定を適用
	chunirecGroup.Use(middleware.APIRateLimitMiddleware(
		info.APIRateLimitRequests,
		info.APIRateLimitEditorRequests,
		info.APIRateLimitAdminRequests,
		info.APIRateLimitWindow,
	))
	{
		chunirecGroup.GET("/music/showall", handlers.Chunirec.GetMusicShowAll)
		chunirecGroup.GET("/music/show", handlers.Chunirec.GetMusicShow)
		chunirecGroup.GET("/records/showall", handlers.Chunirec.GetRecordsShowAll)
		chunirecGroup.GET("/users/show", handlers.Chunirec.GetUserShow)
	}

	// reiwa互換APIルートの登録
	reiwaGroup := e.Group("/compat/reiwa/1")
	reiwaGroup.Use(reiwa.ReiwaErrorHandlerMiddleware())
	reiwaGroup.Use(middleware.APITokenMaintenanceMiddleware(maintenanceStateProvider, apiTokenUsecase))
	reiwaGroup.Use(middleware.APITokenMiddleware(apiTokenUsecase))
	reiwaGroup.Use(middleware.APIRateLimitMiddleware(
		info.APIRateLimitRequests,
		info.APIRateLimitEditorRequests,
		info.APIRateLimitAdminRequests,
		info.APIRateLimitWindow,
	))
	{
		reiwaGroup.GET("/chunithm_record/original", handlers.Reiwa.GetChunithmRecordOriginal)
	}
}

func newDefaultCORSConfig(cfg config.Config) echoMiddleware.CORSConfig {
	return newCORSConfig(cfg.CORS.AllowOrigins, cfg, func(c *echo.Context) bool {
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

func newTemporaryPlayerDataCORSConfig(cfg config.Config) echoMiddleware.CORSConfig {
	corsConfig := newExternalCORSConfig(cfg)
	corsConfig.Skipper = func(c *echo.Context) bool {
		return c.Request().URL.Path != "/internal/player-data/temp"
	}
	return corsConfig
}

func isExternalCORSPath(path string) bool {
	return path == "/healthz" || path == "/internal/player-data/temp"
}

func newCORSConfig(allowOrigins []string, cfg config.Config, skipper echoMiddleware.Skipper) echoMiddleware.CORSConfig {
	corsConfig := echoMiddleware.CORSConfig{
		Skipper:          skipper,
		AllowOrigins:     allowOrigins,
		AllowCredentials: cfg.CORS.AllowCredentials,
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
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
			echo.HeaderRetryAfter,
		},
		MaxAge: cfg.CORS.MaxAge,
	}

	if hasWildcardCORSOrigin(allowOrigins) {
		corsConfig.UnsafeAllowOriginFunc = newCORSAllowOriginFunc(allowOrigins, cfg.CORS.AllowCredentials)
	}

	return corsConfig
}

// hasWildcardCORSOrigin は Echo v5 の AllowOrigins だけでは扱えないパターン指定があるかを判定します。
func hasWildcardCORSOrigin(allowOrigins []string) bool {
	return slices.ContainsFunc(allowOrigins, func(origin string) bool {
		return origin != "*" && strings.ContainsAny(origin, "*?")
	})
}

// newCORSAllowOriginFunc は設定ファイルの allow_origins 形式を維持したまま、Echo v4 互換のワイルドカード判定を行います。
// UnsafeAllowOriginFunc を設定すると Echo v5 は AllowOrigins を無視するため、完全一致オリジンもこの関数内で判定します。
func newCORSAllowOriginFunc(allowOrigins []string, allowCredentials bool) func(c *echo.Context, origin string) (string, bool, error) {
	type originPattern struct {
		matcher *regexp.Regexp
	}

	var exactOrigins []string
	var patterns []originPattern
	allowAll := false

	for _, allowedOrigin := range allowOrigins {
		if allowedOrigin == "*" {
			if allowCredentials {
				panic("* as allowed origin and AllowCredentials=true is insecure and not allowed")
			}
			allowAll = true
			continue
		}

		if strings.ContainsAny(allowedOrigin, "*?") {
			patterns = append(patterns, originPattern{
				matcher: mustCompileCORSOriginPattern(allowedOrigin),
			})
			continue
		}

		if _, err := parseCORSOrigin(allowedOrigin); err != nil {
			panic(fmt.Sprintf("invalid CORS allow origin %q: %v", allowedOrigin, err))
		}
		exactOrigins = append(exactOrigins, allowedOrigin)
	}

	return func(c *echo.Context, origin string) (string, bool, error) {
		if allowAll {
			return "*", true, nil
		}

		for _, allowedOrigin := range exactOrigins {
			if strings.EqualFold(allowedOrigin, origin) {
				return allowedOrigin, true, nil
			}
		}

		for _, pattern := range patterns {
			if pattern.matcher.MatchString(origin) {
				return origin, true, nil
			}
		}

		return "", false, nil
	}
}

// mustCompileCORSOriginPattern は CORS オリジンの `*` と `?` を正規表現へ変換します。
// 起動時設定の誤りはリクエスト時まで遅延させず、Echo の CORS 設定と同じく起動時に panic させます。
func mustCompileCORSOriginPattern(pattern string) *regexp.Regexp {
	if _, err := parseCORSOrigin(pattern); err != nil {
		panic(fmt.Sprintf("invalid CORS allow origin pattern %q: %v", pattern, err))
	}

	quotedPattern := regexp.QuoteMeta(pattern)
	quotedPattern = strings.ReplaceAll(quotedPattern, `\*`, `.*`)
	quotedPattern = strings.ReplaceAll(quotedPattern, `\?`, `.`)

	matcher, err := regexp.Compile(`(?i)^` + quotedPattern + `$`)
	if err != nil {
		panic(fmt.Sprintf("invalid CORS allow origin pattern %q: %v", pattern, err))
	}

	return matcher
}

// parseCORSOrigin は CORS の Origin として扱える scheme://host[:port] だけを許可します。
func parseCORSOrigin(origin string) (*url.URL, error) {
	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return nil, err
	}
	if parsedOrigin.Scheme == "" || parsedOrigin.Host == "" {
		return nil, fmt.Errorf("scheme or host is missing")
	}
	if parsedOrigin.Path != "" || parsedOrigin.RawQuery != "" || parsedOrigin.Fragment != "" {
		return nil, fmt.Errorf("path, query, and fragment are not allowed")
	}

	return parsedOrigin, nil
}

// handleExternalHealth は外部監視向けの軽量な死活チェック結果を返します。
// 依存サービスの状態を返さないことで、外部公開しても内部構成を推測されにくくします。
func handleExternalHealth(c *echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func handleRoot(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"app_name":   info.Name,
		"build_date": info.BuildDate,
	})
}

// handleAdminBuildInfo は管理者画面向けにAPIのビルド情報を返します。
func handleAdminBuildInfo(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"app_name":    info.Name,
		"build_date":  info.BuildDate,
		"commit_hash": info.Revision,
		"go_version":  runtime.Version(),
	})
}

// handleVersion はADMIN向けにAPIのバージョン識別子を返します。
func handleVersion(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"app_name":    info.Name,
		"build_date":  info.BuildDate,
		"commit_hash": info.Revision,
		"go_version":  runtime.Version(),
	})
}
