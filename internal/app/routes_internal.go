package app

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/featureflag"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
)

// internalRouteMiddlewares は内部APIの各グループで共有するルート単位のミドルウェアです。
// レート制限は状態を持つため、同じ制限枠を共有するルートには同じインスタンスを渡します。
type internalRouteMiddlewares struct {
	firebaseAuthStrict                echo.MiddlewareFunc
	firebaseAuthReadOptimized         echo.MiddlewareFunc
	optionalFirebaseAuthReadOptimized echo.MiddlewareFunc
	anonymousRateLimit                echo.MiddlewareFunc
	authenticatedReadRateLimit        echo.MiddlewareFunc
	heavyReadConcurrencyLimit         echo.MiddlewareFunc
	dataTransferRateLimit             echo.MiddlewareFunc
	requireEditor                     echo.MiddlewareFunc
	requireAdmin                      echo.MiddlewareFunc
}

// registerInternalRoutes はフロントエンド向け内部APIのルートを登録します。
func registerInternalRoutes(
	e *echo.Echo,
	handlers *Handlers,
	firebaseAuthenticatorStrict middleware.FirebaseAuthenticator,
	firebaseAuthenticatorReadOptimized middleware.FirebaseAuthenticator,
	maintenanceStateProvider usecase.MaintenanceStateProvider,
	cfg config.Config,
) {
	internal := e.Group("/internal")
	internal.Use(rejectUnmatchedRoute)
	// 一時保存APIの拡張CORSは、メンテナンス503にも適用できるようゲートより先に評価します。
	internal.Use(echoMiddleware.CORSWithConfig(newTemporaryPlayerDataCORSConfig(cfg)))
	internal.Use(middleware.FirebaseMaintenanceMiddleware(maintenanceStateProvider, firebaseAuthenticatorStrict))

	mw := internalRouteMiddlewares{
		firebaseAuthStrict: middleware.FirebaseIDTokenMiddleware(firebaseAuthenticatorStrict),
		// 読み取り専用GETは失効確認（Firebaseへの問い合わせ）を省き、応答遅延とクライアント切断を減らします。
		firebaseAuthReadOptimized:         middleware.FirebaseIDTokenMiddleware(firebaseAuthenticatorReadOptimized),
		optionalFirebaseAuthReadOptimized: middleware.OptionalFirebaseIDTokenMiddleware(firebaseAuthenticatorReadOptimized),
		anonymousRateLimit: middleware.AnonymousIPRateLimitMiddleware(middleware.RateLimitConfig{
			Requests: info.InternalPublicRateLimitRequests,
			Window:   info.InternalPublicRateLimitWindow,
		}),
		// 認証済みユーザーの参照系は全グループで1つの枠を共有し、エンドポイントを変えた回避を防ぎます。
		authenticatedReadRateLimit: middleware.AuthenticatedUserRateLimitMiddleware(middleware.RateLimitConfig{
			Requests: info.InternalAuthenticatedReadRateLimitRequests,
			Window:   info.InternalAuthenticatedReadRateLimitWindow,
		}),
		heavyReadConcurrencyLimit: middleware.ConcurrencyLimitMiddleware(info.HeavyReadConcurrencyPerClient, info.HeavyReadConcurrencyGlobal),
		dataTransferRateLimit: middleware.UserRateLimitMiddleware(middleware.RateLimitConfig{
			Requests: info.DataTransferRateLimitRequests,
			Window:   info.DataTransferRateLimitWindow,
		}),
		requireEditor: middleware.RequireRole(info.AccountTypeEditor),
		requireAdmin:  middleware.RequireRole(info.AccountTypeAdmin),
	}

	internal.GET("/system/status", handlers.SystemMaintenance.Status, echoMiddleware.CORSWithConfig(newExternalCORSConfig(cfg)))

	registerInternalAuthRoutes(internal, handlers, mw)
	registerInternalMeRoutes(internal, handlers, mw, cfg)
	registerInternalFriendRoutes(internal, handlers, mw)
	registerInternalTemporaryPlayerDataRoutes(internal, handlers, mw)
	registerInternalUserRoutes(internal, handlers, mw, cfg)
	registerInternalAdminRoutes(internal, handlers, mw)
	registerInternalSongRoutes(internal, handlers, mw)
	registerInternalCourseRoutes(internal, handlers, mw)
	registerInternalMasterRoutes(internal, handlers, mw)
}

// registerInternalAuthRoutes はログイン・新規登録・APIトークン管理のルートを登録します。
func registerInternalAuthRoutes(internal *echo.Group, handlers *Handlers, mw internalRouteMiddlewares) {
	authGroup := internal.Group("/auth")
	authGroup.POST("/login", handlers.Login.Login, middleware.IPRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.LoginRateLimitRequests,
		Window:   info.LoginRateLimitWindow,
	}))
	authGroup.POST("/signup", handlers.Signup.Signup, middleware.IPRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.RegisterRateLimitRequests,
		Window:   info.RegisterRateLimitWindow,
	}))
	authGroup.GET("/api-tokens", handlers.APIToken.List, mw.firebaseAuthStrict)
	authGroup.POST("/api-tokens", handlers.APIToken.Generate, mw.firebaseAuthStrict)
	authGroup.PATCH("/api-tokens/:id", handlers.APIToken.Rename, mw.firebaseAuthStrict)
	authGroup.DELETE("/api-tokens/:id", handlers.APIToken.Delete, mw.firebaseAuthStrict)
}

// registerInternalMeRoutes はログインユーザー自身のリソースのルートを登録します。
// 同じパスプレフィックスで読み取りと書き込みの認証方式が異なるため、Group を分けて登録します。
func registerInternalMeRoutes(internal *echo.Group, handlers *Handlers, mw internalRouteMiddlewares, cfg config.Config) {
	meReadGroup := internal.Group("/me")
	meReadGroup.Use(mw.firebaseAuthReadOptimized, mw.authenticatedReadRateLimit)
	meReadGroup.GET("", handlers.Profile.Me)
	meReadGroup.GET("/player-data/latest-update", handlers.Me.GetLatestPlayerUpdate)
	meReadGroup.GET("/player-data/updates", handlers.Me.GetRecentPlayerUpdates)
	meReadGroup.GET("/goals", handlers.Goal.List)
	meReadGroup.GET("/goal-groups", handlers.GoalGroup.List)
	meReadGroup.GET("/record-filters", handlers.RecordFilter.List)
	meReadGroup.GET("/x-account", handlers.XAccount.Get)

	meGroup := internal.Group("/me")
	meGroup.Use(mw.firebaseAuthStrict)
	meGroup.POST("/data-transfer/export", handlers.DataTransfer.Export, mw.dataTransferRateLimit)
	meGroup.POST("/data-transfer/validate", handlers.DataTransfer.Validate, mw.dataTransferRateLimit)
	meGroup.POST("/data-transfer/import", handlers.DataTransfer.Import, mw.dataTransferRateLimit)
	meGroup.PUT("/privacy", handlers.Profile.UpdatePrivacy)
	meGroup.PUT("/username", handlers.Profile.UpdateUsername)
	// 無効時は未登録パスと同じ404にするため、ルート自体を登録しません。
	if cfg.FeatureFlags.Enabled(featureflag.ProfileComment) {
		meGroup.PUT("/comment", handlers.UserProfileComment.Update)
	}
	meGroup.PUT("/firebase-account", handlers.Profile.ChangeFirebaseAccount)
	meGroup.DELETE("", handlers.Profile.DeleteAccount)
	meGroup.POST("/register-data", handlers.Me.RegisterData, middleware.UserRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.RegisterDataRateLimitRequests,
		Window:   info.RegisterDataRateLimitWindow,
	}))
	meGroup.DELETE("/player-data", handlers.Me.DeletePlayerData)
	meGroup.POST("/goals", handlers.Goal.Create)
	meGroup.PUT("/goals/order", handlers.Goal.Reorder)
	meGroup.PUT("/goals/:id", handlers.Goal.Update)
	meGroup.DELETE("/goals/:id", handlers.Goal.Delete)
	meGroup.POST("/goal-groups", handlers.GoalGroup.Create)
	meGroup.PUT("/goal-groups/order", handlers.GoalGroup.Reorder)
	meGroup.PUT("/goal-groups/:id", handlers.GoalGroup.Update)
	meGroup.DELETE("/goal-groups/:id", handlers.GoalGroup.Delete)
	meGroup.POST("/record-filters", handlers.RecordFilter.Create)
	meGroup.PUT("/record-filters/:id", handlers.RecordFilter.Update)
	meGroup.DELETE("/record-filters/:id", handlers.RecordFilter.Delete)
	meGroup.PUT("/x-account", handlers.XAccount.Sync, middleware.UserRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.XAccountSyncRateLimitRequests,
		Window:   info.XAccountSyncRateLimitWindow,
	}))
	meGroup.POST("/locked-songs", handlers.PlayerLockedSong.Lock)
	meGroup.POST("/locked-songs/batch", handlers.PlayerLockedSong.Batch)
	meGroup.DELETE("/locked-songs/:id", handlers.PlayerLockedSong.Unlock)
	meGroup.POST("/favorite-songs", handlers.PlayerFavoriteSong.Add)
	meGroup.DELETE("/favorite-songs/:id", handlers.PlayerFavoriteSong.Remove)
}

// registerInternalFriendRoutes はフレンド関係とフレンド間比較のルートを登録します。
func registerInternalFriendRoutes(internal *echo.Group, handlers *Handlers, mw internalRouteMiddlewares) {
	friendshipReadGroup := internal.Group("/friends")
	friendshipReadGroup.Use(mw.firebaseAuthReadOptimized, mw.authenticatedReadRateLimit)
	friendshipReadGroup.GET("", handlers.Friendship.ListFriends)
	friendshipReadGroup.GET("/requests/received", handlers.Friendship.ListReceivedRequests)
	friendshipReadGroup.GET("/requests/sent", handlers.Friendship.ListSentRequests)
	friendshipReadGroup.GET("/slots", handlers.Friendship.GetSlots)

	friendshipGroup := internal.Group("/friends")
	friendshipGroup.Use(mw.firebaseAuthStrict)
	friendshipGroup.POST("/requests", handlers.Friendship.SendRequest)
	friendshipGroup.POST("/requests/:username/accept", handlers.Friendship.AcceptRequest)
	friendshipGroup.POST("/requests/:username/reject", handlers.Friendship.RejectRequest)
	friendshipGroup.DELETE("/requests/:username", handlers.Friendship.CancelRequest)
	friendshipGroup.DELETE("/:username", handlers.Friendship.Remove)

	friendRankingGroup := internal.Group("/friend-rankings")
	friendRankingGroup.Use(mw.firebaseAuthReadOptimized, mw.authenticatedReadRateLimit, mw.heavyReadConcurrencyLimit)
	friendRankingGroup.GET("/songs/:id/charts/:difficulty", handlers.FriendChartRanking.GetStandard)
	friendRankingGroup.GET("/worldsend-songs/:id", handlers.FriendChartRanking.GetWorldsend)

	friendComparisonGroup := internal.Group("/friend-comparisons")
	friendComparisonGroup.Use(mw.firebaseAuthReadOptimized, mw.authenticatedReadRateLimit, mw.heavyReadConcurrencyLimit)
	friendComparisonGroup.GET("/:username/charts/:difficulty", handlers.FriendScoreComparison.Get)
	friendComparisonGroup.GET("/:username/worldsend", handlers.FriendScoreComparison.GetWorldsend)
}

// registerInternalTemporaryPlayerDataRoutes は外部サイトからのプレイデータ一時保存と確定のルートを登録します。
func registerInternalTemporaryPlayerDataRoutes(internal *echo.Group, handlers *Handlers, mw internalRouteMiddlewares) {
	temporaryPlayerDataGroup := internal.Group("/player-data")
	temporaryPlayerDataGroup.OPTIONS("/temp", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	temporaryPlayerDataGroup.POST("/temp", handlers.TemporaryPlayerData.CreateTemporaryData, middleware.IPRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.TempDataRateLimitPerMin,
		Window:   info.TempDataRateLimitWindow,
	}))
	temporaryPlayerDataGroup.POST("/commit", handlers.TemporaryPlayerData.CommitTemporaryData, mw.firebaseAuthStrict, middleware.UserRateLimitMiddleware(middleware.RateLimitConfig{
		Requests: info.RegisterDataRateLimitRequests,
		Window:   info.RegisterDataRateLimitWindow,
	}))
}

// registerInternalUserRoutes はユーザー公開情報と管理者向けユーザー管理のルートを登録します。
func registerInternalUserRoutes(internal *echo.Group, handlers *Handlers, mw internalRouteMiddlewares, cfg config.Config) {
	publicUsersGroup := internal.Group("/users")
	publicUsersGroup.Use(mw.optionalFirebaseAuthReadOptimized, mw.anonymousRateLimit, mw.authenticatedReadRateLimit)
	publicUsersGroup.GET("/:username/updated-at", handlers.User.GetUserUpdatedAt)
	publicUsersGroup.GET("/:username/profile", handlers.User.GetUserProfile)
	if cfg.FeatureFlags.Enabled(featureflag.ProfileComment) {
		publicUsersGroup.GET("/:username/comment", handlers.UserProfileComment.Get)
	}
	publicUsersGroup.GET("/:username/rating", handlers.User.GetUserRating)
	publicUsersGroup.GET("/:username/rating-op-history", handlers.InternalMetricHistory.Get)
	publicUsersGroup.GET("/:username/record/songs/:id", handlers.User.GetUserSongRecord)
	publicUsersGroup.GET("/:username/record/songs/:id/:difficulty/history", handlers.InternalScoreHistory.GetStandard)
	publicUsersGroup.GET("/:username/record/worldsend-songs/:id", handlers.User.GetUserWorldsendSongRecord)
	publicUsersGroup.GET("/:username/record/courses", handlers.Course.GetUserRecords)
	publicUsersGroup.GET("/:username/record/courses/:id", handlers.Course.GetUserRecord)
	publicUsersGroup.GET("/:username/record/courses/:id/history", handlers.InternalScoreHistory.GetCourse)
	publicUsersGroup.GET("/:username/record/worldsend-songs/:id/history", handlers.InternalScoreHistory.GetWorldsend)
	publicUsersGroup.GET("/:username/record", handlers.User.GetUserRecord, mw.heavyReadConcurrencyLimit)
	publicUsersGroup.GET("/:username/locked-songs", handlers.PlayerLockedSong.List)
	publicUsersGroup.GET("/:username/favorite-songs", handlers.PlayerFavoriteSong.List)
	publicUsersGroup.GET("/:username", handlers.User.GetUserProfileWithRecords, mw.heavyReadConcurrencyLimit)

	usersGroup := internal.Group("/users")
	usersGroup.Use(mw.firebaseAuthStrict)
	usersGroup.GET("/", handlers.AdminUser.GetAllUsers, mw.requireAdmin)
	usersGroup.DELETE("/:username", handlers.User.DeleteUser, mw.requireAdmin)
	usersGroup.PATCH("/:username/permission", handlers.UserPermission.UpdatePermission, mw.requireAdmin)
	usersGroup.PATCH("/:username/suspicious", handlers.UserSuspicious.UpdateSuspicious, mw.requireAdmin)
}

// registerInternalAdminRoutes は管理者専用のルートを登録します。
func registerInternalAdminRoutes(internal *echo.Group, handlers *Handlers, mw internalRouteMiddlewares) {
	adminGroup := internal.Group("/admin")
	adminGroup.Use(mw.firebaseAuthStrict, mw.requireAdmin)
	adminGroup.GET("/build-info", handleAdminBuildInfo)
	adminGroup.GET("/user-stats", handlers.AdminUserStatistics.Get)
	adminGroup.GET("/chart-rankings/songs/:id/charts/:difficulty", handlers.AdminChartRanking.GetStandard)
	adminGroup.GET("/chart-rankings/worldsend-songs/:id", handlers.AdminChartRanking.GetWorldsend)
	adminGroup.PUT("/maintenance", handlers.SystemMaintenance.Update)
	adminGroup.GET("/versions", handlers.Version.List)
	adminGroup.POST("/versions", handlers.Version.Create)
	adminGroup.PUT("/versions/:id", handlers.Version.Rename)
	adminGroup.DELETE("/versions/:id", handlers.Version.Delete)
	adminGroup.GET("/song-batch/jobs", handlers.SongBatch.List)
	adminGroup.POST("/song-batch/jobs", handlers.SongBatch.Start)
	adminGroup.GET("/song-batch/jobs/:id", handlers.SongBatch.Get)
	adminGroup.GET("/chart-stats-batch/jobs", handlers.ChartStatsBatch.List)
	adminGroup.POST("/chart-stats-batch/jobs", handlers.ChartStatsBatch.Start)
	adminGroup.GET("/chart-stats-batch/jobs/:id", handlers.ChartStatsBatch.Get)

	honorsGroup := internal.Group("/honors")
	honorsGroup.Use(mw.firebaseAuthStrict, mw.requireAdmin)
	honorsGroup.GET("", handlers.Honor.ListHonors)
	honorsGroup.GET("/:id", handlers.Honor.GetHonor)
	honorsGroup.POST("", handlers.Honor.CreateHonor)
	honorsGroup.PUT("/:id", handlers.Honor.UpdateHonor)
	honorsGroup.DELETE("/:id", handlers.Honor.DeleteHonor)
}

// registerInternalSongRoutes は楽曲・WORLD'S END楽曲・ベスト枠統計のルートを登録します。
func registerInternalSongRoutes(internal *echo.Group, handlers *Handlers, mw internalRouteMiddlewares) {
	publicSongsGroup := internal.Group("/songs")
	publicSongsGroup.Use(mw.optionalFirebaseAuthReadOptimized, mw.anonymousRateLimit, mw.authenticatedReadRateLimit)
	publicSongsGroup.GET("/updated-at", handlers.Song.GetSongsUpdatedAt)
	publicSongsGroup.GET("", handlers.Song.GetSongs)
	publicSongsGroup.GET("/:id", handlers.Song.GetSong)
	publicSongsGroup.GET("/:id/stats/:difficulty", handlers.Song.GetChartStatsByDifficulty)
	publicSongsGroup.GET("/:id/best-slot-stats/:difficulty", handlers.BestSlotStats.GetChartStats)

	bestSlotRankingGroup := internal.Group("/best-slot-rankings")
	bestSlotRankingGroup.Use(mw.optionalFirebaseAuthReadOptimized, mw.anonymousRateLimit, mw.authenticatedReadRateLimit)
	bestSlotRankingGroup.GET("", handlers.BestSlotStats.GetRanking, mw.heavyReadConcurrencyLimit)

	publicWorldsendGroup := internal.Group("/worldsend-songs")
	publicWorldsendGroup.Use(mw.optionalFirebaseAuthReadOptimized, mw.anonymousRateLimit, mw.authenticatedReadRateLimit)
	publicWorldsendGroup.GET("", handlers.Worldsend.GetWorldsendSongs)
	publicWorldsendGroup.GET("/:id", handlers.Worldsend.GetWorldsendSong)

	songsGroup := internal.Group("/songs")
	songsGroup.Use(mw.firebaseAuthStrict)
	songsGroup.POST("", handlers.Song.CreateSong, mw.requireAdmin)
	songsGroup.PUT("", handlers.Song.UpdateSongs, mw.requireEditor)
	songsGroup.DELETE("/:id", handlers.Song.DeleteSong, mw.requireAdmin)
	songsGroup.POST("/:id/restore", handlers.Song.RestoreSong, mw.requireEditor)

	worldsendGroup := internal.Group("/worldsend-songs")
	worldsendGroup.Use(mw.firebaseAuthStrict)
	worldsendGroup.POST("", handlers.Worldsend.CreateWorldsendSong, mw.requireAdmin)
	worldsendGroup.PUT("", handlers.Worldsend.UpdateWorldsendSongs, mw.requireEditor)
	worldsendGroup.DELETE("/:id", handlers.Worldsend.DeleteWorldsendSong, mw.requireAdmin)
	worldsendGroup.POST("/:id/restore", handlers.Worldsend.RestoreWorldsendSong, mw.requireEditor)

	editorSongsGroup := internal.Group("/editor/songs")
	editorSongsGroup.Use(mw.firebaseAuthStrict, mw.requireEditor)
	editorSongsGroup.GET("", handlers.Song.GetEditorSongs)
	editorSongsGroup.GET("/:id", handlers.Song.GetEditorSong)

	editorWorldsendGroup := internal.Group("/editor/worldsend-songs")
	editorWorldsendGroup.Use(mw.firebaseAuthStrict, mw.requireEditor)
	editorWorldsendGroup.GET("", handlers.Worldsend.GetEditorWorldsendSongs)
	editorWorldsendGroup.GET("/:id", handlers.Worldsend.GetEditorWorldsendSong)
}

// registerInternalCourseRoutes はコースのルートを登録します。
func registerInternalCourseRoutes(internal *echo.Group, handlers *Handlers, mw internalRouteMiddlewares) {
	publicCoursesGroup := internal.Group("/courses")
	publicCoursesGroup.Use(mw.optionalFirebaseAuthReadOptimized, mw.anonymousRateLimit, mw.authenticatedReadRateLimit)
	publicCoursesGroup.GET("/updated-at", handlers.Course.GetCoursesUpdatedAt)
	publicCoursesGroup.GET("", handlers.Course.List)
	publicCoursesGroup.GET("/:id", handlers.Course.Get)
	publicCoursesGroup.GET("/:id/stats", handlers.CourseStats.Get)

	coursesGroup := internal.Group("/courses")
	coursesGroup.Use(mw.firebaseAuthStrict)
	coursesGroup.POST("", handlers.Course.Create, mw.requireAdmin)
	coursesGroup.PUT("/:id", handlers.Course.Update, mw.requireEditor)
	coursesGroup.PUT("/:id/tracks", handlers.Course.ReplaceTracks, mw.requireEditor)
	coursesGroup.DELETE("/:id", handlers.Course.Delete, mw.requireAdmin)
	coursesGroup.POST("/:id/restore", handlers.Course.Restore, mw.requireEditor)

	editorCoursesGroup := internal.Group("/editor/courses")
	editorCoursesGroup.Use(mw.firebaseAuthStrict, mw.requireEditor)
	editorCoursesGroup.GET("", handlers.Course.ListEditor)
	editorCoursesGroup.GET("/:id", handlers.Course.GetEditor)
}

// registerInternalMasterRoutes はマスタデータのルートを登録します。
func registerInternalMasterRoutes(internal *echo.Group, handlers *Handlers, mw internalRouteMiddlewares) {
	masterGroup := internal.Group("/master")
	masterGroup.GET("", handlers.MasterData.GetMasterData)
	masterGroup.GET("/permissions", handlers.MasterData.GetPermissions, mw.firebaseAuthStrict, mw.requireAdmin)
	masterGroup.GET("/name-folders", handlers.MasterData.GetNameFolders)
	masterGroup.GET("/genres", handlers.MasterData.GetGenres)
	masterGroup.GET("/versions", handlers.MasterData.GetVersions)
	masterGroup.GET("/honor-types", handlers.MasterData.GetHonorTypes)
}
