package app

import (
	"context"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/chunirec"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/reiwa"
	"github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/featureflag"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/datatransfer"
	infradb "github.com/chunisupport/chunisupport-api/internal/infra/db"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/infra/openaimoderation"
	infra "github.com/chunisupport/chunisupport-api/internal/infra/repository"
	infrasongbatch "github.com/chunisupport/chunisupport-api/internal/infra/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/infra/transaction"
	"github.com/chunisupport/chunisupport-api/internal/infra/turnstile"
	"github.com/chunisupport/chunisupport-api/internal/infra/xapi"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/jmoiron/sqlx"
)

// Handlers はすべてのハンドラーを保持するコンテナです
type Handlers struct {
	Login                 *api_internal.LoginHandler
	Signup                *api_internal.SignupHandler
	Profile               *api_internal.ProfileHandler
	User                  *api_internal.UserHandler
	AdminUser             *api_internal.AdminUserHandler
	UserPermission        *api_internal.UserPermissionHandler
	UserSuspicious        *api_internal.UserSuspiciousHandler
	AdminUserStatistics   *api_internal.AdminUserStatisticsHandler
	AdminChartRanking     *api_internal.AdminChartRankingHandler
	Song                  *api_internal.SongHandler
	Honor                 *api_internal.HonorHandler
	Version               *api_internal.VersionHandler
	Worldsend             *api_internal.WorldsendHandler
	APIToken              *api_internal.APITokenHandler
	Me                    *api_internal.MeHandler
	DataTransfer          *api_internal.DataTransferHandler
	MasterData            *api_internal.MasterDataHandler
	Goal                  *api_internal.GoalHandler
	GoalGroup             *api_internal.GoalGroupHandler
	RecordFilter          *api_internal.RecordFilterHandler
	XAccount              *api_internal.XAccountHandler
	UserProfileComment    *api_internal.UserProfileCommentHandler
	TemporaryPlayerData   *api_internal.TemporaryPlayerDataHandler
	PlayerLockedSong      *api_internal.PlayerLockedSongHandler
	PlayerFavoriteSong    *api_internal.PlayerFavoriteSongHandler
	Friendship            *api_internal.FriendshipHandler
	FriendChartRanking    *api_internal.FriendChartRankingHandler
	FriendScoreComparison *api_internal.FriendScoreComparisonHandler
	BestSlotStats         *api_internal.BestSlotStatsHandler
	InternalScoreHistory  *api_internal.ScoreHistoryHandler
	InternalMetricHistory *api_internal.PlayerMetricHistoryHandler
	Course                *api_internal.CourseHandler
	SystemMaintenance     *api_internal.SystemMaintenanceHandler
	SongBatch             *api_internal.SongBatchHandler
	ChartStatsBatch       *api_internal.ChartStatsBatchHandler
	// 外部API v1 用ハンドラ
	V1Song        *api_v1.V1SongHandler
	V1Worldsend   *api_v1.V1WorldsendHandler
	V1User        *api_v1.V1UserHandler
	V1Version     *api_v1.V1VersionHandler
	ScoreHistory  *api_v1.ScoreHistoryHandler
	MetricHistory *api_v1.PlayerMetricHistoryHandler
	V1Course      *api_v1.V1CourseHandler
	// chunirec互換APIハンドラ
	Chunirec *chunirec.ChunirecHandler
	// reiwa互換APIハンドラ
	Reiwa *reiwa.ReiwaHandler
}

// routerDependencies は composition root で構築した依存のうち、ルート登録とサーバー停止処理が必要とするものです。
// 共有依存（認証・メンテナンス状態・楽曲カタログなど）はここで一度だけ生成し、各所へ同じインスタンスを渡します。
type routerDependencies struct {
	handlers *Handlers
	// strict は失効確認を行う認証、readOptimized は読み取り専用GET向けに失効確認を省く認証です。
	firebaseAuthenticatorStrict        middleware.FirebaseAuthenticator
	firebaseAuthenticatorReadOptimized middleware.FirebaseAuthenticator
	apiTokenUsecase                    usecase.APITokenUsecase
	maintenanceStateProvider           usecase.MaintenanceStateProvider
	// batchJobs はサーバー停止時に DB 接続を閉じる前に完了を待つバックグラウンド処理です。
	batchJobs           []backgroundBatchJobs
	songCatalogProvider *infra.SongCatalogProvider
}

// buildRouterDependencies は Repository・Usecase・Handler を構築します。
// DIフレームワークは使わず、依存の生成箇所をこの関数に集約して所有関係を一箇所で追えるようにします。
// ctx は管理画面から起動したバッチの寿命にもなるため、サーバー停止時にキャンセルされるものを渡します。
func buildRouterDependencies(ctx context.Context, db *sqlx.DB, cfg config.Config, masterCache *masterdata.Cache, staticMasterCache *masterdata.StaticCache, firebaseTokenVerifier usecase.TokenVerifier, firebaseUserDeleter usecase.FirebaseUserDeleter, firebaseProviderFinder usecase.FirebaseProviderAccountFinder) (*routerDependencies, error) {
	userRepo := infra.NewUserRepository(db)
	playerRepo := infra.NewPlayerRepository(db)
	playerRecordRepo := infra.NewPlayerRecordRepository(db)
	worldsendRecordRepo := infra.NewWorldsendRecordRepository(db)
	playerDataRepo := infra.NewPlayerDataRepository(db)
	scoreHistoryRepo := infra.NewScoreHistoryRepository(db)
	playerMetricHistoryQuery := infra.NewPlayerMetricHistoryQueryService(db)
	worldsendChartRepo := infra.NewWorldsendChartRepository(db)
	chartStatsRepo := infra.NewChartStatsRepository(db)
	apiTokenRepo := infra.NewAPITokenRepository(db)
	songRepo := infra.NewSongRepository(db)
	goalRepo := infra.NewGoalRepository(db)
	goalGroupRepo := infra.NewGoalGroupRepository()
	recordFilterRepo := infra.NewRecordFilterRepository()
	honorRepo := infra.NewHonorRepository(db)
	versionRepo := infra.NewVersionRepository()
	playerLockedSongRepo := infra.NewPlayerLockedSongRepository()
	playerFavoriteSongRepo := infra.NewPlayerFavoriteSongRepository()
	playerFavoriteSongQueryService := infra.NewPlayerFavoriteSongQueryService()
	playerFavoriteSongLocker := infra.NewPlayerFavoriteSongLocker()
	friendshipRepo := infra.NewFriendshipRepository()
	friendChartRankingQueryService := infra.NewFriendChartRankingQueryService()
	friendScoreComparisonQueryService := infra.NewFriendScoreComparisonQueryService(db)
	adminChartRankingQueryService := infra.NewAdminChartRankingQueryService(db)
	bestSlotRankingQueryService := infra.NewBestSlotRankingQueryService(db)
	overpowerDenominatorProvider := infra.NewOverpowerDenominatorProvider(db)
	songCatalogProvider := infra.NewSongCatalogProvider(db)
	userUpdatedAtQuery := infra.NewUserUpdatedAtQueryService()
	chunirecQuery := infra.NewChunirecQueryService(db)
	adminUserStatisticsQuery := infra.NewAdminUserStatisticsQueryService(db)
	courseRepo := infra.NewCourseRepository(db)
	systemMaintenanceRepo := infra.NewSystemMaintenanceRepository(db)
	tm := transaction.NewTransactionManager(db)
	systemMaintenanceUsecase, err := usecase.NewSystemMaintenanceUsecase(ctx, systemMaintenanceRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize system maintenance state: %w", err)
	}
	recentSignInVerifier := requireRecentSignInVerifier(firebaseTokenVerifier)
	usernamePolicy, err := service.NewForbiddenUsernamePolicy(cfg.UsernamePolicy.Exact, cfg.UsernamePolicy.Contains)
	if err != nil {
		panic(fmt.Sprintf("failed to create username policy: %v", err))
	}
	userCredentialUsecase := usecase.NewUserCredentialUsecaseWithUsernamePolicy(db, tm, userRepo, playerRecordRepo, goalRepo, recentSignInVerifier, firebaseUserDeleter, masterCache, usernamePolicy)
	apiTokenUsecase := usecase.NewAPITokenUsecase(db, tm, apiTokenRepo, userRepo)
	userUsecase := usecase.NewUserUsecaseWithFirebaseDeleterAndOverpowerDenominator(
		db, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo, songRepo, worldsendChartRepo, masterCache, friendshipRepo,
		firebaseUserDeleter, playerLockedSongRepo, overpowerDenominatorProvider, userUpdatedAtQuery, songCatalogProvider, courseRepo, tm, goalRepo,
	)
	chunirecUsecase := usecase.NewChunirecUsecase(db, userRepo, friendshipRepo, chunirecQuery)
	playerDataUsecase := usecase.NewPlayerDataUsecaseWithScoreHistory(tm, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo, honorRepo, playerDataRepo, playerLockedSongRepo, masterCache, scoreHistoryRepo, courseRepo)
	courseUsecase := usecase.NewCourseUsecase(db, tm, courseRepo, userRepo, friendshipRepo)
	scoreHistoryUsecase := usecase.NewScoreHistoryUsecase(db, userRepo, songRepo, worldsendChartRepo, courseRepo, scoreHistoryRepo, friendshipRepo, masterCache)
	playerMetricHistoryUsecase := usecase.NewPlayerMetricHistoryUsecase(db, userRepo, playerMetricHistoryQuery, friendshipRepo)
	temporaryPlayerDataRepo := infra.NewTemporaryPlayerDataRepository(info.TempDataMaxEntriesPerIP, cfg.TempData.MaxTotalMB*1024*1024)
	temporaryPlayerDataUsecase := usecase.NewTemporaryPlayerDataUsecase(db, temporaryPlayerDataRepo, playerDataUsecase, info.TempDataTTL, api_internal.DecodeTemporaryPlayerDataPayload)
	songUsecase := usecase.NewSongUsecaseWithCascadeDelete(songRepo, masterCache, tm, db, overpowerDenominatorProvider, playerFavoriteSongRepo, playerLockedSongRepo, songCatalogProvider)
	honorUsecase := usecase.NewHonorUsecase(honorRepo, masterCache, tm, db)
	versionUsecase := usecase.NewVersionUsecase(versionRepo, masterCache, tm, db)
	chartStatsMasterProvider := masterdata.NewChartStatsMasterProviderAdapter(staticMasterCache)
	chartStatsUsecase := usecase.NewChartStatsUsecase(songRepo, worldsendChartRepo, chartStatsRepo, masterCache, chartStatsMasterProvider, db)
	worldsendUsecase := usecase.NewWorldsendUsecase(worldsendChartRepo, tm, db, songCatalogProvider)
	goalUsecase := usecase.NewGoalUsecase(db, tm, goalRepo, masterCache, goalGroupRepo)
	goalGroupUsecase := usecase.NewGoalGroupUsecase(db, tm, goalGroupRepo, goalRepo)
	recordFilterUsecase := usecase.NewRecordFilterUsecase(db, tm, recordFilterRepo, userRepo)
	playerLockedSongQueryService := infra.NewPlayerLockedSongQueryService()
	playerSongIDResolver := infra.NewPlayerSongIDResolver()
	playerLockedSongUsecase, err := usecase.NewPlayerLockedSongUsecase(db, tm, userRepo, playerRepo, playerRecordRepo, playerDataRepo, songRepo, playerLockedSongRepo, friendshipRepo, playerLockedSongQueryService, playerSongIDResolver)
	if err != nil {
		panic(fmt.Sprintf("failed to create player locked song usecase: %v", err))
	}
	playerFavoriteSongUsecase, err := usecase.NewPlayerFavoriteSongUsecase(db, tm, userRepo, playerRepo, songRepo, playerFavoriteSongRepo, friendshipRepo, playerFavoriteSongQueryService, playerFavoriteSongLocker, playerSongIDResolver)
	if err != nil {
		panic(fmt.Sprintf("failed to create player favorite song usecase: %v", err))
	}
	friendshipUsecase, err := usecase.NewFriendshipUsecase(db, tm, userRepo, friendshipRepo)
	if err != nil {
		panic(fmt.Sprintf("failed to create friendship usecase: %v", err))
	}
	friendChartRankingUsecase := usecase.NewFriendChartRankingUsecase(db, friendChartRankingQueryService)
	friendScoreComparisonUsecase := usecase.NewFriendScoreComparisonUsecase(friendScoreComparisonQueryService)
	adminChartRankingUsecase := usecase.NewAdminChartRankingUsecase(adminChartRankingQueryService)
	bestSlotRankingUsecase := usecase.NewBestSlotRankingUsecase(bestSlotRankingQueryService, chartStatsMasterProvider)
	masterDataUsecase := usecase.NewMasterDataUsecase(masterCache, chartStatsMasterProvider)
	dataTransferCodec, err := datatransfer.NewCodec(cfg.DataTransferHMACSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize data transfer codec: %w", err)
	}
	dataTransferRepo := infra.NewUserDataTransferRepository(db)
	dataTransferUsecase := usecase.NewUserDataTransferUsecase(dataTransferCodec, dataTransferRepo, goalUsecase)
	songBatchJobUsecase := usecase.NewSongBatchJobUsecase(
		ctx,
		infradb.NewAdvisoryLockProvider(db),
		infra.NewSongBatchJobRepository(db),
		infrasongbatch.NewSongBatchUsecase(db, cfg.SongBatch.WikiBaseURL),
		overpowerDenominatorProvider,
		songCatalogProvider,
	)
	chartStatsBatchJobUsecase := usecase.NewChartStatsBatchJobUsecase(
		ctx,
		infradb.NewAdvisoryLockProvider(db),
		infra.NewChartStatsBatchJobRepository(db),
		usecase.NewChartStatsBatchUsecase(infra.NewChartStatsBatchRepository(db)),
	)

	// 許可ホストはフロントの CORS オリジンだけを使う。
	// 外部サイト向けに追加する new.chunithm-net.com はログイン画面のホストではない。
	turnstileVerifier := turnstile.NewVerifier(cfg.Turnstile.SecretKey, cfg.CORS.AllowOrigins)
	firebaseAuthUsecaseStrict := usecase.NewFirebaseAuthUsecase(db, userRepo, firebaseTokenVerifier)
	firebaseAuthUsecaseReadOptimized := usecase.NewFirebaseAuthUsecase(db, userRepo, usecase.NewReadOptimizedTokenVerifier(firebaseTokenVerifier))
	loginUsecase := usecase.NewLoginUsecase(firebaseAuthUsecaseStrict, turnstileVerifier, masterCache, systemMaintenanceUsecase)
	signupUsecase := usecase.NewSignupUsecase(tm, userRepo, firebaseTokenVerifier, turnstileVerifier, masterCache, usernamePolicy)
	adminUserStatisticsUsecase := usecase.NewAdminUserStatisticsUsecase(adminUserStatisticsQuery)
	userSuspiciousUsecase := usecase.NewUserSuspiciousUsecase(db, tm, userRepo)
	xAccountUsecase := usecase.NewXAccountUsecase(db, infra.NewXAccountRepository(), firebaseProviderFinder, xapi.NewUserClient(cfg.XAPI.ConsumerKey, cfg.XAPI.ConsumerSecret))
	// 無効な環境では Moderation クライアントを作らない。キー未設定でも起動でき、APIも呼ばれない。
	var userProfileCommentHandler *api_internal.UserProfileCommentHandler
	if cfg.FeatureFlags.Enabled(featureflag.ProfileComment) {
		userProfileCommentUsecase := usecase.NewUserProfileCommentUsecase(db, infra.NewUserProfileRepository(), userRepo, friendshipRepo, openaimoderation.NewCommentModerator(cfg.OpenAI.APIKey))
		userProfileCommentHandler = api_internal.NewUserProfileCommentHandler(userProfileCommentUsecase)
	}
	handlers := &Handlers{
		Login:                 api_internal.NewLoginHandler(loginUsecase),
		Signup:                api_internal.NewSignupHandler(signupUsecase),
		Profile:               api_internal.NewProfileHandler(userCredentialUsecase),
		User:                  api_internal.NewUserHandler(userUsecase),
		AdminUser:             api_internal.NewAdminUserHandler(userUsecase),
		UserPermission:        api_internal.NewUserPermissionHandler(usecase.NewUserPermissionUsecase(db, tm, userRepo)),
		UserSuspicious:        api_internal.NewUserSuspiciousHandler(userSuspiciousUsecase),
		AdminUserStatistics:   api_internal.NewAdminUserStatisticsHandler(adminUserStatisticsUsecase),
		AdminChartRanking:     api_internal.NewAdminChartRankingHandler(adminChartRankingUsecase),
		Song:                  api_internal.NewSongHandler(songUsecase, chartStatsUsecase, masterCache, staticMasterCache),
		Honor:                 api_internal.NewHonorHandler(honorUsecase),
		Version:               api_internal.NewVersionHandler(versionUsecase),
		Worldsend:             api_internal.NewWorldsendHandler(worldsendUsecase, masterCache),
		APIToken:              api_internal.NewAPITokenHandler(apiTokenUsecase),
		Me:                    api_internal.NewMeHandler(playerDataUsecase),
		DataTransfer:          api_internal.NewDataTransferHandler(dataTransferUsecase),
		MasterData:            api_internal.NewMasterDataHandler(masterDataUsecase),
		Goal:                  api_internal.NewGoalHandler(goalUsecase),
		GoalGroup:             api_internal.NewGoalGroupHandler(goalGroupUsecase),
		RecordFilter:          api_internal.NewRecordFilterHandler(recordFilterUsecase),
		XAccount:              api_internal.NewXAccountHandler(xAccountUsecase),
		UserProfileComment:    userProfileCommentHandler,
		TemporaryPlayerData:   api_internal.NewTemporaryPlayerDataHandler(temporaryPlayerDataUsecase),
		PlayerLockedSong:      api_internal.NewPlayerLockedSongHandler(playerLockedSongUsecase),
		PlayerFavoriteSong:    api_internal.NewPlayerFavoriteSongHandler(playerFavoriteSongUsecase),
		Friendship:            api_internal.NewFriendshipHandler(friendshipUsecase),
		FriendChartRanking:    api_internal.NewFriendChartRankingHandler(friendChartRankingUsecase),
		FriendScoreComparison: api_internal.NewFriendScoreComparisonHandler(friendScoreComparisonUsecase),
		BestSlotStats:         api_internal.NewBestSlotStatsHandler(bestSlotRankingUsecase, chartStatsUsecase, chartStatsMasterProvider),
		InternalScoreHistory:  api_internal.NewScoreHistoryHandler(scoreHistoryUsecase),
		InternalMetricHistory: api_internal.NewPlayerMetricHistoryHandler(playerMetricHistoryUsecase),
		Course:                api_internal.NewCourseHandler(courseUsecase),
		SystemMaintenance:     api_internal.NewSystemMaintenanceHandler(systemMaintenanceUsecase),
		SongBatch:             api_internal.NewSongBatchHandler(songBatchJobUsecase),
		ChartStatsBatch:       api_internal.NewChartStatsBatchHandler(chartStatsBatchJobUsecase),
		V1Song:                api_v1.NewV1SongHandler(songUsecase, chartStatsUsecase, masterCache, staticMasterCache),
		V1Worldsend:           api_v1.NewV1WorldsendHandler(worldsendUsecase, masterCache),
		V1User:                api_v1.NewV1UserHandler(userUsecase),
		V1Version:             api_v1.NewV1VersionHandler(masterDataUsecase),
		ScoreHistory:          api_v1.NewScoreHistoryHandler(scoreHistoryUsecase),
		MetricHistory:         api_v1.NewPlayerMetricHistoryHandler(playerMetricHistoryUsecase),
		V1Course:              api_v1.NewV1CourseHandler(courseUsecase),
		Chunirec:              chunirec.NewChunirecHandler(songUsecase, chunirecUsecase, masterCache, cfg.Location),
		Reiwa:                 reiwa.NewReiwaHandler(songUsecase, worldsendUsecase, masterDataUsecase, masterCache),
	}

	return &routerDependencies{
		handlers:                           handlers,
		firebaseAuthenticatorStrict:        firebaseAuthUsecaseStrict,
		firebaseAuthenticatorReadOptimized: firebaseAuthUsecaseReadOptimized,
		apiTokenUsecase:                    apiTokenUsecase,
		maintenanceStateProvider:           systemMaintenanceUsecase,
		batchJobs:                          []backgroundBatchJobs{songBatchJobUsecase, chartStatsBatchJobUsecase},
		songCatalogProvider:                songCatalogProvider,
	}, nil
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
