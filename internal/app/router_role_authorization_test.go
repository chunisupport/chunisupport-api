package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/featureflag"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/apitokenpermission"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// docs/API.mdのエンドポイント一覧に基づく独立した期待表です。
// 空欄はEDITOR/ADMIN制限の対象外であり、匿名アクセス可能という意味ではありません。
// 制限のないルートも列挙し、新規ルートの認可方針が未確認のまま追加されることを防ぎます。
func routeRolePolicies() map[string]string {
	return map[string]string{
		"OPTIONS /healthz":                   "",
		"GET /healthz":                       "",
		"GET /":                              "",
		"GET /version":                       "apiAdminRead",
		"GET /badges/users/:username/rating": "",
		"GET /badges/users/:username/rating-calculated":                      "",
		"GET /badges/users/:username/overpower":                              "",
		"GET /badges/users/:username/overpower-percent":                      "",
		"GET /internal/system/status":                                        "",
		"POST /internal/auth/login":                                          "",
		"POST /internal/auth/signup":                                         "",
		"GET /internal/auth/api-tokens":                                      "",
		"POST /internal/auth/api-tokens":                                     "",
		"PATCH /internal/auth/api-tokens/:id":                                "",
		"DELETE /internal/auth/api-tokens/:id":                               "",
		"GET /internal/me":                                                   "",
		"GET /internal/me/player-data/latest-update":                         "",
		"GET /internal/me/player-data/updates":                               "",
		"GET /internal/me/goals":                                             "",
		"GET /internal/me/goal-groups":                                       "",
		"GET /internal/me/record-filters":                                    "",
		"GET /internal/me/x-account":                                         "",
		"POST /internal/me/data-transfer/export":                             "",
		"POST /internal/me/data-transfer/validate":                           "",
		"POST /internal/me/data-transfer/import":                             "",
		"PUT /internal/me/privacy":                                           "",
		"PUT /internal/me/username":                                          "",
		"PUT /internal/me/comment":                                           "",
		"PUT /internal/me/firebase-account":                                  "",
		"DELETE /internal/me":                                                "",
		"POST /internal/me/register-data":                                    "",
		"DELETE /internal/me/player-data":                                    "",
		"POST /internal/me/goals":                                            "",
		"PUT /internal/me/goals/order":                                       "",
		"PUT /internal/me/goals/:id":                                         "",
		"DELETE /internal/me/goals/:id":                                      "",
		"POST /internal/me/goal-groups":                                      "",
		"PUT /internal/me/goal-groups/order":                                 "",
		"PUT /internal/me/goal-groups/:id":                                   "",
		"DELETE /internal/me/goal-groups/:id":                                "",
		"POST /internal/me/record-filters":                                   "",
		"PUT /internal/me/record-filters/:id":                                "",
		"DELETE /internal/me/record-filters/:id":                             "",
		"PUT /internal/me/x-account":                                         "",
		"POST /internal/me/locked-songs":                                     "",
		"POST /internal/me/locked-songs/batch":                               "",
		"DELETE /internal/me/locked-songs/:id":                               "",
		"POST /internal/me/favorite-songs":                                   "",
		"DELETE /internal/me/favorite-songs/:id":                             "",
		"GET /internal/friends":                                              "",
		"GET /internal/friends/requests/received":                            "",
		"GET /internal/friends/requests/sent":                                "",
		"GET /internal/friends/slots":                                        "",
		"POST /internal/friends/requests":                                    "",
		"POST /internal/friends/requests/:username/accept":                   "",
		"POST /internal/friends/requests/:username/reject":                   "",
		"DELETE /internal/friends/requests/:username":                        "",
		"DELETE /internal/friends/:username":                                 "",
		"GET /internal/friend-rankings/songs/:id/charts/:difficulty":         "",
		"GET /internal/friend-rankings/worldsend-songs/:id":                  "",
		"GET /internal/friend-comparisons/:username/charts/:difficulty":      "",
		"GET /internal/friend-comparisons/:username/worldsend":               "",
		"OPTIONS /internal/player-data/temp":                                 "",
		"POST /internal/player-data/temp":                                    "",
		"POST /internal/player-data/commit":                                  "",
		"GET /internal/users/:username/updated-at":                           "",
		"GET /internal/users/:username/profile":                              "",
		"GET /internal/users/:username/comment":                              "",
		"GET /internal/users/:username/rating":                               "",
		"GET /internal/users/:username/rating-op-history":                    "",
		"GET /internal/users/:username/record/songs/:id":                     "",
		"GET /internal/users/:username/record/songs/:id/:difficulty/history": "",
		"GET /internal/users/:username/record/worldsend-songs/:id":           "",
		"GET /internal/users/:username/record/courses":                       "",
		"GET /internal/users/:username/record/courses/:id":                   "",
		"GET /internal/users/:username/record/worldsend-songs/:id/history":   "",
		"GET /internal/users/:username/record":                               "",
		"GET /internal/users/:username/locked-songs":                         "",
		"GET /internal/users/:username/favorite-songs":                       "",
		"GET /internal/users/:username":                                      "",
		"GET /internal/users/":                                               "firebaseAdmin",
		"DELETE /internal/users/:username":                                   "firebaseAdmin",
		"PATCH /internal/users/:username/permission":                         "firebaseAdmin",
		"PATCH /internal/users/:username/suspicious":                         "firebaseAdmin",
		"GET /internal/admin/build-info":                                     "firebaseAdmin",
		"GET /internal/admin/user-stats":                                     "firebaseAdmin",
		"GET /internal/admin/chart-rankings/songs/:id/charts/:difficulty":    "firebaseAdmin",
		"GET /internal/admin/chart-rankings/worldsend-songs/:id":             "firebaseAdmin",
		"PUT /internal/admin/maintenance":                                    "firebaseAdmin",
		"GET /internal/admin/versions":                                       "firebaseAdmin",
		"POST /internal/admin/versions":                                      "firebaseAdmin",
		"PUT /internal/admin/versions/:id":                                   "firebaseAdmin",
		"DELETE /internal/admin/versions/:id":                                "firebaseAdmin",
		"GET /internal/admin/song-batch/jobs":                                "firebaseAdmin",
		"POST /internal/admin/song-batch/jobs":                               "firebaseAdmin",
		"GET /internal/admin/song-batch/jobs/:id":                            "firebaseAdmin",
		"GET /internal/admin/chart-stats-batch/jobs":                         "firebaseAdmin",
		"POST /internal/admin/chart-stats-batch/jobs":                        "firebaseAdmin",
		"GET /internal/admin/chart-stats-batch/jobs/:id":                     "firebaseAdmin",
		"GET /internal/honors":                                               "firebaseAdmin",
		"GET /internal/honors/:id":                                           "firebaseAdmin",
		"POST /internal/honors":                                              "firebaseAdmin",
		"PUT /internal/honors/:id":                                           "firebaseAdmin",
		"DELETE /internal/honors/:id":                                        "firebaseAdmin",
		"GET /internal/songs/updated-at":                                     "",
		"GET /internal/songs":                                                "",
		"GET /internal/songs/:id":                                            "",
		"GET /internal/songs/:id/stats/:difficulty":                          "",
		"GET /internal/songs/:id/best-slot-stats/:difficulty":                "",
		"GET /internal/best-slot-rankings":                                   "",
		"GET /internal/worldsend-songs":                                      "",
		"GET /internal/worldsend-songs/:id":                                  "",
		"POST /internal/songs":                                               "firebaseAdmin",
		"PUT /internal/songs":                                                "firebaseEditor",
		"DELETE /internal/songs/:id":                                         "firebaseAdmin",
		"POST /internal/songs/:id/restore":                                   "firebaseEditor",
		"POST /internal/worldsend-songs":                                     "firebaseAdmin",
		"PUT /internal/worldsend-songs":                                      "firebaseEditor",
		"DELETE /internal/worldsend-songs/:id":                               "firebaseAdmin",
		"POST /internal/worldsend-songs/:id/restore":                         "firebaseEditor",
		"GET /internal/editor/songs":                                         "firebaseEditor",
		"GET /internal/editor/songs/:id":                                     "firebaseEditor",
		"GET /internal/editor/worldsend-songs":                               "firebaseEditor",
		"GET /internal/editor/worldsend-songs/:id":                           "firebaseEditor",
		"GET /internal/courses/updated-at":                                   "",
		"GET /internal/courses":                                              "",
		"GET /internal/courses/:id":                                          "",
		"POST /internal/courses":                                             "firebaseAdmin",
		"PUT /internal/courses/:id":                                          "firebaseEditor",
		"DELETE /internal/courses/:id":                                       "firebaseAdmin",
		"POST /internal/courses/:id/restore":                                 "firebaseEditor",
		"GET /internal/editor/courses":                                       "firebaseEditor",
		"GET /internal/editor/courses/:id":                                   "firebaseEditor",
		"GET /internal/master":                                               "",
		"GET /internal/master/permissions":                                   "firebaseAdmin",
		"GET /internal/master/name-folders":                                  "",
		"GET /internal/master/genres":                                        "",
		"GET /internal/master/versions":                                      "",
		"GET /internal/master/honor-types":                                   "",
		"GET /v1/songs/:id/score-history/:difficulty":                        "",
		"GET /v1/worldsend-songs/:id/score-history":                          "",
		"GET /v1/users/:username/rating-op-history":                          "",
		"GET /v1/songs":                                                      "",
		"PUT /v1/songs":                                                      "apiEditorWrite",
		"PATCH /v1/songs/chart-constant":                                     "apiEditorWrite",
		"GET /v1/songs/:id":                                                  "",
		"GET /v1/songs/:id/stats/:difficulty":                                "",
		"GET /v1/worldsend-songs":                                            "",
		"GET /v1/worldsend-songs/:id":                                        "",
		"GET /v1/users/:username":                                            "",
		"GET /v1/users/:username/rating":                                     "",
		"GET /v1/users/:username/records/courses":                            "",
		"GET /v1/courses":                                                    "",
		"GET /v1/courses/:id":                                                "",
		"POST /v1/courses":                                                   "apiAdminWrite",
		"PUT /v1/courses/:id":                                                "apiEditorWrite",
		"GET /v1/master/versions":                                            "",
		"GET /compat/chunirec/2.0/music/showall":                             "",
		"GET /compat/chunirec/2.0/music/show":                                "",
		"GET /compat/chunirec/2.0/records/showall":                           "",
		"GET /compat/chunirec/2.0/records/profile":                           "",
		"GET /compat/chunirec/2.0/users/show":                                "",
		"GET /compat/reiwa/1/chunithm_record/original":                       "",
		"GET /compat/reiwa/1/chunirec_all.json":                              "",
		"GET /compat/reiwa/1/chunithm_versions.json":                         "",
	}
}

// authorizationHandlerSpyRouterはEchoの実ルーターに委譲し、末端Handlerの到達だけを記録します。
// 認証・認可を含む本番のミドルウェア列は変更せず、Handler個別の入力検証やDB操作と切り離します。
type authorizationHandlerSpyRouter struct {
	echo.Router
	reached []string
}

func (r *authorizationHandlerSpyRouter) Add(route echo.Route) (echo.RouteInfo, error) {
	if route.Method != echo.RouteNotFound {
		route.Handler = func(c *echo.Context) error {
			r.reached = append(r.reached, route.Method+" "+route.Path)
			return c.NoContent(http.StatusNoContent)
		}
	}
	return r.Router.Add(route)
}

type routeFirebaseAuthenticator struct{}

func (routeFirebaseAuthenticator) Authenticate(_ context.Context, token string) (*entity.User, error) {
	value, ok := strings.CutPrefix(token, "firebase:")
	if !ok {
		return nil, usecase.ErrInvalidIDToken
	}
	return routeAuthorizationUser(value), nil
}

func (a routeFirebaseAuthenticator) AuthenticateOptional(ctx context.Context, token string) (*entity.User, error) {
	return a.Authenticate(ctx, token)
}

type routeAPITokenUsecase struct{ stubAPITokenUsecase }

func (routeAPITokenUsecase) Validate(_ context.Context, token string) (*entity.User, *entity.APIToken, error) {
	value, ok := strings.CutPrefix(token, "api:")
	if !ok {
		return nil, nil, usecase.ErrInvalidAPIToken
	}
	permission := apitokenpermission.ReadWrite
	if readValue, read := strings.CutPrefix(value, "read-"); read {
		value = readValue
		permission = apitokenpermission.Read
	}
	return routeAuthorizationUser(value), &entity.APIToken{ID: 1, Permission: permission}, nil
}

func routeAuthorizationUser(token string) *entity.User {
	if token == "extdev-token" {
		return &entity.User{ID: 5, AccountTypeID: info.AccountTypeExtDev}
	}
	return authenticateTestUser(token)
}

func newRouteAuthorizationSpy(t *testing.T) (*echo.Echo, *authorizationHandlerSpyRouter) {
	t.Helper()
	router := &authorizationHandlerSpyRouter{Router: echo.NewRouter(echo.RouterConfig{})}
	e := echo.NewWithConfig(echo.Config{Router: router})
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	// 条件付きルートも棚卸しの対象にするため、プロフィールコメントを有効にします。
	cfg := config.Config{FeatureFlags: featureflag.New(featureflag.ProfileComment)}
	auth := routeFirebaseAuthenticator{}
	tokens := routeAPITokenUsecase{}
	maintenance := stubMaintenanceUsecase{}
	registerSystemRoutes(e, auth, tokens, maintenance, cfg)
	registerRoutes(e, newAuthorizationTestHandlers(), auth, auth, tokens, maintenance, cfg)
	return e, router
}

func TestRegisterRoutes_全登録ルートに認可方針が定義されている(t *testing.T) {
	// Given
	e, _ := newRouteAuthorizationSpy(t)
	var expected, actual []string
	for key := range routeRolePolicies() {
		expected = append(expected, key)
	}

	// When
	for _, route := range e.Router().Routes() {
		// Echoがグループ用に自動登録する404経路はAPIエンドポイントではありません。
		if route.Method != echo.RouteNotFound {
			actual = append(actual, route.Method+" "+route.Path)
		}
	}

	// Then: 未分類の新規ルートと、表に残った未登録ルートの両方を検出します。
	assert.ElementsMatch(t, expected, actual)
}

func TestRegisterRoutes_EDITOR_ADMIN制限ルートの認可(t *testing.T) {
	type requestCase struct {
		name   string
		token  string
		status int
	}
	// 期待値はロール判定関数から計算せず、認証方式・境界・書き込み権限ごとに明示します。
	requests := map[string][]requestCase{
		"firebaseAdmin": {
			{"未認証", "", http.StatusUnauthorized},
			{"PLAYER", "firebase:player-token", http.StatusForbidden},
			{"EXTDEV", "firebase:extdev-token", http.StatusForbidden},
			{"EDITOR", "firebase:editor-token", http.StatusForbidden},
			{"ADMIN", "firebase:admin-token", http.StatusNoContent},
			{"APIトークンは代用不可", "api:admin-token", http.StatusUnauthorized},
		},
		"firebaseEditor": {
			{"未認証", "", http.StatusUnauthorized},
			{"PLAYER", "firebase:player-token", http.StatusForbidden},
			{"EXTDEV", "firebase:extdev-token", http.StatusForbidden},
			{"EDITOR", "firebase:editor-token", http.StatusNoContent},
			{"ADMIN", "firebase:admin-token", http.StatusNoContent},
			{"APIトークンは代用不可", "api:admin-token", http.StatusUnauthorized},
		},
		"apiAdminWrite": {
			{"未認証", "", http.StatusUnauthorized},
			{"PLAYER", "api:player-token", http.StatusForbidden},
			{"EXTDEV", "api:extdev-token", http.StatusForbidden},
			{"EDITOR", "api:editor-token", http.StatusForbidden},
			{"ADMIN", "api:admin-token", http.StatusNoContent},
			{"readのADMIN", "api:read-admin-token", http.StatusForbidden},
			{"Firebaseトークンは代用不可", "firebase:admin-token", http.StatusUnauthorized},
		},
		"apiEditorWrite": {
			{"未認証", "", http.StatusUnauthorized},
			{"PLAYER", "api:player-token", http.StatusForbidden},
			{"EXTDEV", "api:extdev-token", http.StatusForbidden},
			{"EDITOR", "api:editor-token", http.StatusNoContent},
			{"ADMIN", "api:admin-token", http.StatusNoContent},
			{"readのEDITOR", "api:read-editor-token", http.StatusForbidden},
			{"readのADMIN", "api:read-admin-token", http.StatusForbidden},
			{"Firebaseトークンは代用不可", "firebase:admin-token", http.StatusUnauthorized},
		},
		"apiAdminRead": {
			{"未認証", "", http.StatusUnauthorized},
			{"PLAYER", "api:player-token", http.StatusForbidden},
			{"EXTDEV", "api:extdev-token", http.StatusForbidden},
			{"EDITOR", "api:editor-token", http.StatusForbidden},
			{"ADMIN", "api:admin-token", http.StatusNoContent},
			{"readのADMIN", "api:read-admin-token", http.StatusNoContent},
			{"Firebaseトークンは代用不可", "firebase:admin-token", http.StatusUnauthorized},
		},
	}
	for key, policy := range routeRolePolicies() {
		if policy == "" {
			continue
		}
		t.Run(key, func(t *testing.T) {
			// Given
			cases, ok := requests[policy]
			require.True(t, ok, "未定義の認可方針: %s", policy)
			e, router := newRouteAuthorizationSpy(t)
			method, path, _ := strings.Cut(key, " ")
			path = strings.NewReplacer(":id", "0123456789abcdef", ":username", "targetuser", ":difficulty", "MASTER").Replace(path)
			for _, tt := range cases {
				t.Run(tt.name, func(t *testing.T) {
					router.reached = nil
					req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
					if tt.token != "" {
						req.Header.Set(echo.HeaderAuthorization, "Bearer "+tt.token)
					}
					rec := httptest.NewRecorder()

					// When
					e.ServeHTTP(rec, req)

					// Then: ステータスに加え、拒否時にはHandlerへ到達していないことを検証します。
					assert.Equal(t, tt.status, rec.Code, rec.Body.String())
					if tt.status == http.StatusNoContent {
						assert.Equal(t, []string{key}, router.reached)
					} else {
						assert.Empty(t, router.reached)
					}
				})
			}
		})
	}
}
