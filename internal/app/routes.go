package app

import (
	"net/http"
	"runtime"

	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/chunirec"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/reiwa"
	"github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
)

// registerSystemRoutes は依存サービスの構築と分離し、公開範囲と認可を含む本番の登録経路をテストでも共有します。
func registerSystemRoutes(
	e *echo.Echo,
	firebaseAuthenticator middleware.FirebaseAuthenticator,
	apiTokenUsecase usecase.APITokenUsecase,
	maintenanceStateProvider usecase.MaintenanceStateProvider,
	cfg config.Config,
) {
	healthzCORS := echoMiddleware.CORSWithConfig(newExternalCORSConfig(cfg))
	firebaseMaintenance := middleware.FirebaseMaintenanceMiddleware(maintenanceStateProvider, firebaseAuthenticator)
	apiTokenMaintenance := middleware.APITokenMaintenanceMiddleware(maintenanceStateProvider, apiTokenUsecase)
	e.OPTIONS("/healthz", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	}, healthzCORS)
	e.GET("/healthz", handleExternalHealth, healthzCORS)
	e.GET("/", handleRoot, firebaseMaintenance)
	e.GET("/version", handleVersion, apiTokenMaintenance, middleware.APITokenMiddleware(apiTokenUsecase), middleware.RequireRole(info.AccountTypeAdmin))
}

// registerRoutes はすべてのAPI群のルートを登録します。
// API群ごとに認証方式・メンテナンスゲート・レート制限が異なるため、グループ単位の関数へ委譲します。
func registerRoutes(
	e *echo.Echo,
	handlers *Handlers,
	firebaseAuthenticatorStrict middleware.FirebaseAuthenticator,
	firebaseAuthenticatorReadOptimized middleware.FirebaseAuthenticator,
	apiTokenUsecase usecase.APITokenUsecase,
	maintenanceStateProvider usecase.MaintenanceStateProvider,
	cfg config.Config,
) {
	registerInternalRoutes(e, handlers, firebaseAuthenticatorStrict, firebaseAuthenticatorReadOptimized, maintenanceStateProvider, cfg)
	registerBadgeRoutes(e, handlers, firebaseAuthenticatorStrict, maintenanceStateProvider)
	registerV1Routes(e, handlers, apiTokenUsecase, maintenanceStateProvider)
	registerChunirecRoutes(e, handlers, apiTokenUsecase, maintenanceStateProvider)
	registerReiwaRoutes(e, handlers, apiTokenUsecase, maintenanceStateProvider)
}

// registerBadgeRoutes は外部サイトへ埋め込まれるバッジ画像のルートを登録します。
func registerBadgeRoutes(
	e *echo.Echo,
	handlers *Handlers,
	firebaseAuthenticatorStrict middleware.FirebaseAuthenticator,
	maintenanceStateProvider usecase.MaintenanceStateProvider,
) {
	badgeGroup := e.Group("/badges")
	badgeGroup.Use(rejectUnmatchedRoute)
	badgeGroup.Use(middleware.FirebaseMaintenanceMiddleware(maintenanceStateProvider, firebaseAuthenticatorStrict))
	badgeGroup.Use(middleware.IPRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.RatingBadgeIPRateLimitRequests,
		Window:   info.InternalPublicRateLimitWindow,
	}))
	badgeGroup.Use(middleware.IPAndUsernameRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.InternalPublicRateLimitRequests,
		Window:   info.InternalPublicRateLimitWindow,
	}))
	badgeGroup.GET("/users/:username/rating", handlers.User.GetOfficialRatingBadge)
	badgeGroup.GET("/users/:username/rating-calculated", handlers.User.GetCalculatedRatingBadge)
	badgeGroup.GET("/users/:username/overpower", handlers.User.GetOfficialOverpowerBadge)
	badgeGroup.GET("/users/:username/overpower-percent", handlers.User.GetOfficialOverpowerPercentBadge)
}

// useAPITokenGroupMiddlewares はAPIトークン認証を使うグループへ、メンテナンスゲート・認証・レート制限をこの順で適用します。
// レート制限の状態はグループごとに独立させるため、呼び出しのたびに新しいミドルウェアを生成します。
func useAPITokenGroupMiddlewares(group *echo.Group, apiTokenUsecase usecase.APITokenUsecase, maintenanceStateProvider usecase.MaintenanceStateProvider) {
	group.Use(middleware.APITokenMaintenanceMiddleware(maintenanceStateProvider, apiTokenUsecase))
	group.Use(middleware.APITokenMiddleware(apiTokenUsecase))
	group.Use(middleware.APIRateLimitMiddleware(
		info.APIRateLimitRequests,
		info.APIRateLimitEditorRequests,
		info.APIRateLimitAdminRequests,
		info.APIRateLimitWindow,
	))
}

// registerV1Routes は外部API v1 のルートを登録します。
func registerV1Routes(
	e *echo.Echo,
	handlers *Handlers,
	apiTokenUsecase usecase.APITokenUsecase,
	maintenanceStateProvider usecase.MaintenanceStateProvider,
) {
	requireEditor := middleware.RequireRole(info.AccountTypeEditor)
	requireAdmin := middleware.RequireRole(info.AccountTypeAdmin)
	requireAPITokenWrite := middleware.RequireAPITokenWrite()

	apiV1 := e.Group("/v1")
	apiV1.Use(rejectUnmatchedRoute)
	useAPITokenGroupMiddlewares(apiV1, apiTokenUsecase, maintenanceStateProvider)
	apiV1.GET("/songs/:id/score-history/:difficulty", handlers.ScoreHistory.GetStandard)
	apiV1.GET("/worldsend-songs/:id/score-history", handlers.ScoreHistory.GetWorldsend)
	apiV1.GET("/users/:username/rating-op-history", handlers.MetricHistory.Get)
	apiV1.GET("/songs", handlers.V1Song.GetSongs)
	apiV1.PUT("/songs", handlers.V1Song.UpdateSongs, requireAPITokenWrite, requireEditor)
	apiV1.PATCH("/songs/chart-constant", handlers.V1Song.UpdateChartConstant, requireAPITokenWrite, requireEditor)
	apiV1.GET("/songs/:id", handlers.V1Song.GetSong)
	apiV1.GET("/songs/:id/stats/:difficulty", handlers.V1Song.GetChartStatsByDifficulty)
	apiV1.GET("/worldsend-songs", handlers.V1Worldsend.GetWorldsendSongs)
	apiV1.GET("/worldsend-songs/:id", handlers.V1Worldsend.GetWorldsendSong)
	apiV1.GET("/users/:username", handlers.V1User.GetUser)
	apiV1.GET("/users/:username/rating", handlers.V1User.GetUserRating)
	apiV1.GET("/users/:username/records/courses", handlers.V1Course.GetUserRecords)
	apiV1.GET("/courses", handlers.V1Course.List)
	apiV1.GET("/courses/:id", handlers.V1Course.Get)
	apiV1.GET("/courses/:id/score-history", handlers.ScoreHistory.GetCourse)
	apiV1.GET("/courses/:id/stats", handlers.V1CourseStats.Get)
	apiV1.POST("/courses", handlers.V1Course.Create, requireAPITokenWrite, requireAdmin)
	apiV1.PUT("/courses/:id", handlers.V1Course.Update, requireAPITokenWrite, requireEditor)
	apiV1.GET("/master/versions", handlers.V1Version.GetVersions)
}

// registerChunirecRoutes は chunirec 互換APIのルートを登録します。
func registerChunirecRoutes(
	e *echo.Echo,
	handlers *Handlers,
	apiTokenUsecase usecase.APITokenUsecase,
	maintenanceStateProvider usecase.MaintenanceStateProvider,
) {
	chunirecGroup := e.Group("/compat/chunirec/2.0")
	// 後続ミドルウェアのエラーも互換形式で返すため、最初に適用します。
	chunirecGroup.Use(chunirec.ChunirecErrorHandlerMiddleware())
	chunirecGroup.Use(rejectUnmatchedRoute)
	useAPITokenGroupMiddlewares(chunirecGroup, apiTokenUsecase, maintenanceStateProvider)
	chunirecGroup.GET("/music/showall", handlers.Chunirec.GetMusicShowAll)
	chunirecGroup.GET("/music/show", handlers.Chunirec.GetMusicShow)
	chunirecGroup.GET("/records/showall", handlers.Chunirec.GetRecordsShowAll)
	chunirecGroup.GET("/records/profile", handlers.Chunirec.GetRecordsProfile)
	chunirecGroup.GET("/users/show", handlers.Chunirec.GetUserShow)
}

// registerReiwaRoutes は reiwa 互換APIのルートを登録します。
func registerReiwaRoutes(
	e *echo.Echo,
	handlers *Handlers,
	apiTokenUsecase usecase.APITokenUsecase,
	maintenanceStateProvider usecase.MaintenanceStateProvider,
) {
	reiwaGroup := e.Group("/compat/reiwa/1")
	// 後続ミドルウェアのエラーも互換形式で返すため、最初に適用します。
	reiwaGroup.Use(reiwa.ReiwaErrorHandlerMiddleware())
	reiwaGroup.Use(rejectUnmatchedRoute)
	useAPITokenGroupMiddlewares(reiwaGroup, apiTokenUsecase, maintenanceStateProvider)
	reiwaGroup.GET("/chunithm_record/original", handlers.Reiwa.GetChunithmRecordOriginal)
	reiwaGroup.GET("/chunirec_all.json", handlers.Reiwa.GetChunirecAll)
	reiwaGroup.GET("/chunithm_versions.json", handlers.Reiwa.GetChunithmVersions)
}

// rejectUnmatchedRoute はグループの未登録パス（登録済みパスへのメソッド不一致を含む）を、認証・メンテナンス制限より前に404として返します。
// Echoはグループミドルウェアを実行するため未登録パスにもグループのルートを割り当てるので、
// 後続の認証やメンテナンス判定で401・503になることを防ぎます。
// 互換APIでは互換エラー形式で返すため、各グループのエラーハンドラーより後に適用します。
func rejectUnmatchedRoute(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if c.RouteInfo().Method == echo.RouteNotFound {
			return echo.ErrNotFound
		}
		return next(c)
	}
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
