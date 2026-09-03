package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internalhandler "github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/chunirec"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/reiwa"
	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFirebaseAuthenticator struct{}

func (stubFirebaseAuthenticator) Authenticate(ctx context.Context, idToken string) (*entity.User, error) {
	return authenticateTestUser(idToken), nil
}

func (stubFirebaseAuthenticator) AuthenticateOptional(ctx context.Context, idToken string) (*entity.User, error) {
	return authenticateTestUser(idToken), nil
}

type stubMaintenanceUsecase struct {
	state usecase.MaintenanceState
}

type stubAdminUserStatisticsUsecase struct{}

func (stubAdminUserStatisticsUsecase) Get(context.Context) (usecase.AdminUserStatisticsOutput, error) {
	return usecase.AdminUserStatisticsOutput{}, nil
}

func (s stubMaintenanceUsecase) Current() usecase.MaintenanceState {
	return s.state
}

func (s stubMaintenanceUsecase) Update(context.Context, int, bool, string) (usecase.MaintenanceState, error) {
	return s.state, nil
}

type countingAuthenticator struct {
	authenticateCalls         int
	authenticateOptionalCalls int
}

type roleCountingAuthenticator struct {
	authenticateCalls int
}

func (a *roleCountingAuthenticator) Authenticate(_ context.Context, idToken string) (*entity.User, error) {
	a.authenticateCalls++
	return authenticateTestUser(idToken), nil
}

func (a *roleCountingAuthenticator) AuthenticateOptional(_ context.Context, idToken string) (*entity.User, error) {
	return authenticateTestUser(idToken), nil
}

func (a *countingAuthenticator) Authenticate(_ context.Context, _ string) (*entity.User, error) {
	a.authenticateCalls++
	return nil, usecase.ErrInvalidIDToken
}

func (a *countingAuthenticator) AuthenticateOptional(_ context.Context, _ string) (*entity.User, error) {
	a.authenticateOptionalCalls++
	return nil, usecase.ErrInvalidIDToken
}

func authenticateTestUser(idToken string) *entity.User {
	switch idToken {
	case "editor-token":
		return &entity.User{ID: 1, AccountTypeID: info.AccountTypeEditor}
	case "admin-token":
		return &entity.User{ID: 2, AccountTypeID: info.AccountTypeAdmin}
	default:
		return &entity.User{ID: 3, AccountTypeID: info.AccountTypePlayer}
	}
}

type stubAPITokenUsecase struct{}

func (stubAPITokenUsecase) Generate(ctx context.Context, userID int, name string) (*usecase.GeneratedAPITokenOutput, error) {
	return nil, nil
}

func (stubAPITokenUsecase) List(ctx context.Context, userID int) ([]*usecase.APITokenOutput, error) {
	return nil, nil
}

func (stubAPITokenUsecase) Rename(ctx context.Context, userID int, id string, name string) (*usecase.APITokenOutput, error) {
	return nil, nil
}

func (stubAPITokenUsecase) Validate(ctx context.Context, rawToken string) (*entity.User, *entity.APIToken, error) {
	return authenticateTestUser(rawToken), &entity.APIToken{ID: 1}, nil
}

func (stubAPITokenUsecase) Delete(ctx context.Context, userID int, id string) error {
	return nil
}

func TestRegisterRoutes_楽曲追加削除はEDITORを拒否する(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "通常楽曲追加はEDITOR拒否", method: http.MethodPost, path: "/internal/songs"},
		{name: "通常楽曲削除はEDITOR拒否", method: http.MethodDelete, path: "/internal/songs/abcd1234"},
		{name: "WORLDS END楽曲追加はEDITOR拒否", method: http.MethodPost, path: "/internal/worldsend-songs"},
		{name: "WORLDS END楽曲削除はEDITOR拒否", method: http.MethodDelete, path: "/internal/worldsend-songs/abcd1234"},
		{name: "称号一覧はEDITOR拒否", method: http.MethodGet, path: "/internal/honors"},
		{name: "称号追加はEDITOR拒否", method: http.MethodPost, path: "/internal/honors"},
		{name: "バージョン一覧はEDITOR拒否", method: http.MethodGet, path: "/internal/admin/versions"},
		{name: "バージョン追加はEDITOR拒否", method: http.MethodPost, path: "/internal/admin/versions"},
		{name: "バージョン改名はEDITOR拒否", method: http.MethodPut, path: "/internal/admin/versions/1"},
		{name: "バージョン削除はEDITOR拒否", method: http.MethodDelete, path: "/internal/admin/versions/1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			registerRoutes(e, newAuthorizationTestHandlers(), stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, nil, stubMaintenanceUsecase{}, config.Config{})

			req := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, nil)
			req.Header.Set(echo.HeaderAuthorization, "Bearer editor-token")
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			require.Equal(t, http.StatusForbidden, rec.Code)
			assert.Contains(t, rec.Body.String(), "forbidden")
		})
	}
}

func TestRegisterRoutes_ユーザー集計はADMINだけが取得できる(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "ADMINは取得できる", token: "admin-token", wantStatus: http.StatusOK},
		{name: "EDITORは拒否される", token: "editor-token", wantStatus: http.StatusForbidden},
		{name: "PLAYERは拒否される", token: "player-token", wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			handlers := newAuthorizationTestHandlers()
			handlers.AdminUserStatistics = internalhandler.NewAdminUserStatisticsHandler(stubAdminUserStatisticsUsecase{})
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			registerRoutes(e, handlers, stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, nil, stubMaintenanceUsecase{}, config.Config{})
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/admin/user-stats", nil)
			req.Header.Set(echo.HeaderAuthorization, "Bearer "+tt.token)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestRegisterRoutes_譜面ランキングはADMINだけが取得できる(t *testing.T) {
	roles := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "ADMINは取得できる", token: "admin-token", wantStatus: http.StatusOK},
		{name: "EDITORは拒否される", token: "editor-token", wantStatus: http.StatusForbidden},
		{name: "PLAYERは拒否される", token: "player-token", wantStatus: http.StatusForbidden},
	}
	paths := []struct {
		name string
		path string
	}{
		{name: "通常譜面", path: "/internal/admin/chart-rankings/songs/0000000000000001/charts/MASTER"},
		{name: "WORLD'S END", path: "/internal/admin/chart-rankings/worldsend-songs/0000000000000002"},
	}

	for _, path := range paths {
		for _, role := range roles {
			t.Run(path.name+"/"+role.name, func(t *testing.T) {
				// Given
				handlers := newAuthorizationTestHandlers()
				handlers.AdminChartRanking = internalhandler.NewAdminChartRankingHandler(stubAdminChartRankingUsecase{})
				e := echo.New()
				e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
				registerRoutes(e, handlers, stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, nil, stubMaintenanceUsecase{}, config.Config{})
				req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path.path, nil)
				req.Header.Set(echo.HeaderAuthorization, "Bearer "+role.token)
				rec := httptest.NewRecorder()

				// When
				e.ServeHTTP(rec, req)

				// Then
				assert.Equal(t, role.wantStatus, rec.Code)
			})
		}
	}
}

type stubAdminChartRankingUsecase struct{}

func (stubAdminChartRankingUsecase) GetStandard(context.Context, string, string) (*usecase.AdminChartRankingResult, error) {
	return &usecase.AdminChartRankingResult{}, nil
}

func (stubAdminChartRankingUsecase) GetWorldsend(context.Context, string) (*usecase.AdminChartRankingResult, error) {
	return &usecase.AdminChartRankingResult{}, nil
}

func TestRegisterRoutes_外部楽曲更新はEDITOR以上のAPIトークンを要求する(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "PLAYERは拒否される",
			token:      "player-token",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "EDITORは更新できる",
			token:      "editor-token",
			wantStatus: http.StatusNoContent,
			wantCalled: true,
		},
		{
			name:       "ADMINは更新できる",
			token:      "admin-token",
			wantStatus: http.StatusNoContent,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.Validator = NewCustomValidator()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			called := false
			handlers := newAuthorizationTestHandlers()
			handlers.V1Song = api_v1.NewV1SongHandler(&testutil.MockSongUsecase{
				UpdateSongsFunc: func(ctx context.Context, requests []*usecase.UpdateSongInput) error {
					called = true
					require.Len(t, requests, 1)
					assert.Equal(t, "1234567890abcdef", requests[0].DisplayID)
					return nil
				},
			}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
			registerRoutes(e, handlers, stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, stubAPITokenUsecase{}, stubMaintenanceUsecase{}, config.Config{})

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/v1/songs", bytes.NewBufferString(`[{"id":"1234567890abcdef","title":"テスト楽曲","artist":"テストアーティスト"}]`))
			req.Header.Set(echo.HeaderAuthorization, "Bearer "+tt.token)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			require.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantCalled, called)
		})
	}
}

func TestRegisterRoutes_V1ユーザーレーティングはAPIトークンを要求する(t *testing.T) {
	// Given
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	registerRoutes(e, newAuthorizationTestHandlers(), stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, nil, stubMaintenanceUsecase{}, config.Config{})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/users/testuser/rating", nil)
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":"missing_token"`)
}

func TestRegisterRoutes_外部スコア履歴はAPIトークンを要求する(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "通常譜面", path: "/v1/songs/song-id/score-history/MASTER?username=testuser"},
		{name: "WORLD'S END譜面", path: "/v1/worldsend-songs/song-id/score-history?username=testuser"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			registerRoutes(e, newAuthorizationTestHandlers(), stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, nil, stubMaintenanceUsecase{}, config.Config{})
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			require.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Contains(t, rec.Body.String(), `"code":"missing_token"`)
		})
	}
}

func TestRegisterRoutes_外部公式指標履歴はAPIトークンを要求する(t *testing.T) {
	// Given
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	registerRoutes(e, newAuthorizationTestHandlers(), stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, nil, stubMaintenanceUsecase{}, config.Config{})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/users/testuser/rating-op-history", nil)
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":"missing_token"`)
}

func TestRegisterRoutes_外部譜面定数更新はEDITOR以上のAPIトークンを要求する(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		wantStatus int
		wantCalled bool
	}{
		{name: "PLAYERは拒否される", token: "player-token", wantStatus: http.StatusForbidden},
		{name: "EDITORは更新できる", token: "editor-token", wantStatus: http.StatusOK, wantCalled: true},
		{name: "ADMINは更新できる", token: "admin-token", wantStatus: http.StatusOK, wantCalled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.Validator = NewCustomValidator()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			called := false
			handlers := newAuthorizationTestHandlers()
			handlers.V1Song = api_v1.NewV1SongHandler(&testutil.MockSongUsecase{
				UpdateChartConstantFunc: func(_ context.Context, input usecase.UpdateChartConstantInput) (*entity.Song, error) {
					called = true
					assert.Equal(t, "123", input.OfficialIdx)
					return &entity.Song{OfficialIdx: "123", Charts: []*entity.Chart{}}, nil
				},
			}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
			registerRoutes(e, handlers, stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, stubAPITokenUsecase{}, stubMaintenanceUsecase{}, config.Config{})

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/v1/songs/chart-constant", bytes.NewBufferString(
				`{"official_idx":"123","difficulty":"MAS","const":14.7}`,
			))
			req.Header.Set(echo.HeaderAuthorization, "Bearer "+tt.token)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			require.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantCalled, called)
		})
	}
}

func TestVersionRoute_ADMINのAPIトークンを要求する(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{
			name:       "PLAYERは拒否される",
			token:      "player-token",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "ADMINは取得できる",
			token:      "admin-token",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			e.GET("/version", handleVersion, appmiddleware.APITokenMiddleware(stubAPITokenUsecase{}), appmiddleware.RequireRole(info.AccountTypeAdmin))

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/version", nil)
			req.Header.Set(echo.HeaderAuthorization, "Bearer "+tt.token)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			require.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus != http.StatusOK {
				return
			}

			var response map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, info.Revision, response["commit_hash"])
		})
	}
}

func TestRegisterRoutes_公開GETはread最適化認証を使い書き込みはstrict認証を使う(t *testing.T) {
	// Given
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	strictAuth := &countingAuthenticator{}
	readOptimizedAuth := &countingAuthenticator{}
	registerRoutes(e, newAuthorizationTestHandlers(), strictAuth, readOptimizedAuth, nil, stubMaintenanceUsecase{}, config.Config{})

	// When
	getReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/songs", nil)
	getReq.Header.Set(echo.HeaderAuthorization, "Bearer any-token")
	getRec := httptest.NewRecorder()
	e.ServeHTTP(getRec, getReq)

	postReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/internal/songs", nil)
	postReq.Header.Set(echo.HeaderAuthorization, "Bearer any-token")
	postRec := httptest.NewRecorder()
	e.ServeHTTP(postRec, postReq)

	// Then
	require.Equal(t, http.StatusUnauthorized, getRec.Code)
	require.Equal(t, http.StatusUnauthorized, postRec.Code)
	assert.Equal(t, 1, readOptimizedAuth.authenticateOptionalCalls)
	assert.Equal(t, 0, readOptimizedAuth.authenticateCalls)
	assert.Equal(t, 1, strictAuth.authenticateCalls)
	assert.Equal(t, 0, strictAuth.authenticateOptionalCalls)
}

func TestRegisterRoutes_users公開GETはstrict認証を使う(t *testing.T) {
	// Given
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	strictAuth := &countingAuthenticator{}
	readOptimizedAuth := &countingAuthenticator{}
	registerRoutes(e, newAuthorizationTestHandlers(), strictAuth, readOptimizedAuth, nil, stubMaintenanceUsecase{}, config.Config{})

	// When
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/users/test/profile", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer any-token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Then
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, 1, strictAuth.authenticateOptionalCalls)
	assert.Equal(t, 0, readOptimizedAuth.authenticateOptionalCalls)
}

func TestRegisterRoutes_usersUpdatedAtはread最適化認証を使う(t *testing.T) {
	// Given
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	strictAuth := &countingAuthenticator{}
	readOptimizedAuth := &countingAuthenticator{}
	registerRoutes(e, newAuthorizationTestHandlers(), strictAuth, readOptimizedAuth, nil, stubMaintenanceUsecase{}, config.Config{})

	// When
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/users/test/updated-at", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer any-token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Then
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, 0, strictAuth.authenticateOptionalCalls)
	assert.Equal(t, 1, readOptimizedAuth.authenticateOptionalCalls)
}

func TestRegisterRoutes_メンテナンス中の標準API経路を遮断する(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		token  string
	}{
		{name: "未認証internal API", method: http.MethodGet, path: "/internal/master"},
		{name: "PLAYERのv1 API", method: http.MethodGet, path: "/v1/songs", token: "player-token"},
		{name: "signup", method: http.MethodPost, path: "/internal/auth/signup"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			handlers := newAuthorizationTestHandlers()
			maintenance := stubMaintenanceUsecase{state: usecase.MaintenanceState{
				Enabled:   true,
				Comment:   "データ更新中です",
				UpdatedAt: time.Date(2026, time.July, 26, 3, 30, 0, 0, time.UTC),
			}}
			handlers.SystemMaintenance = internalhandler.NewSystemMaintenanceHandler(maintenance)
			registerRoutes(
				e,
				handlers,
				stubFirebaseAuthenticator{},
				stubFirebaseAuthenticator{},
				stubAPITokenUsecase{},
				maintenance,
				config.Config{},
			)
			req := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, nil)
			if tt.token != "" {
				req.Header.Set(echo.HeaderAuthorization, "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			require.Equal(t, http.StatusServiceUnavailable, rec.Code)
			assert.Equal(t, "60", rec.Header().Get(echo.HeaderRetryAfter))
			assert.Equal(t, "no-store", rec.Header().Get(echo.HeaderCacheControl))
			assert.Contains(t, rec.Body.String(), `"code":"maintenance_mode"`)
		})
	}
}

func TestRegisterRoutes_メンテナンス中も状態確認とADMIN更新を許可する(t *testing.T) {
	// Given
	maintenance := stubMaintenanceUsecase{state: usecase.MaintenanceState{
		Enabled:   true,
		Comment:   "データ更新中です",
		UpdatedAt: time.Date(2026, time.July, 26, 3, 30, 0, 0, time.UTC),
	}}
	handlers := newAuthorizationTestHandlers()
	handlers.SystemMaintenance = internalhandler.NewSystemMaintenanceHandler(maintenance)
	authenticator := new(roleCountingAuthenticator)
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	registerRoutes(
		e,
		handlers,
		authenticator,
		authenticator,
		stubAPITokenUsecase{},
		maintenance,
		config.Config{},
	)

	statusReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/system/status", nil)
	statusRec := httptest.NewRecorder()
	e.ServeHTTP(statusRec, statusReq)
	require.Equal(t, http.StatusOK, statusRec.Code)
	assert.Equal(t, "no-store", statusRec.Header().Get(echo.HeaderCacheControl))
	assert.Contains(t, statusRec.Body.String(), `"status":"maintenance"`)
	assert.Zero(t, authenticator.authenticateCalls)

	updateReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPut,
		"/internal/admin/maintenance",
		bytes.NewBufferString(`{"enabled":false,"comment":""}`),
	)
	updateReq.Header.Set(echo.HeaderAuthorization, "Bearer admin-token")
	updateReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	updateRec := httptest.NewRecorder()

	// When
	e.ServeHTTP(updateRec, updateReq)

	// Then
	require.Equal(t, http.StatusOK, updateRec.Code)
	assert.Equal(t, "no-store", updateRec.Header().Get(echo.HeaderCacheControl))
	assert.Equal(t, 1, authenticator.authenticateCalls)
}

func TestRegisterRoutes_メンテナンス中もEDITORは状態変更できない(t *testing.T) {
	// Given
	maintenance := stubMaintenanceUsecase{state: usecase.MaintenanceState{Enabled: true}}
	handlers := newAuthorizationTestHandlers()
	handlers.SystemMaintenance = internalhandler.NewSystemMaintenanceHandler(maintenance)
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	registerRoutes(
		e,
		handlers,
		stubFirebaseAuthenticator{},
		stubFirebaseAuthenticator{},
		stubAPITokenUsecase{},
		maintenance,
		config.Config{},
	)
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPut,
		"/internal/admin/maintenance",
		bytes.NewBufferString(`{"enabled":false,"comment":""}`),
	)
	req.Header.Set(echo.HeaderAuthorization, "Bearer editor-token")
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	require.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":"forbidden"`)
}

func TestRegisterRoutes_メンテナンス中のcompatは既存503形式を維持する(t *testing.T) {
	paths := []string{
		"/compat/chunirec/2.0/music/showall",
		"/compat/reiwa/1/chunithm_record/original",
		"/compat/reiwa/1/chunithm_versions.json",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			// Given
			maintenance := stubMaintenanceUsecase{state: usecase.MaintenanceState{Enabled: true}}
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			registerRoutes(
				e,
				newAuthorizationTestHandlers(),
				stubFirebaseAuthenticator{},
				stubFirebaseAuthenticator{},
				stubAPITokenUsecase{},
				maintenance,
				config.Config{},
			)
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
			req.Header.Set(echo.HeaderAuthorization, "Bearer player-token")
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			require.Equal(t, http.StatusServiceUnavailable, rec.Code)
			assert.Equal(t, "60", rec.Header().Get(echo.HeaderRetryAfter))
			assert.Equal(t, "no-store", rec.Header().Get(echo.HeaderCacheControl))
			assert.JSONEq(t, `{
				"error": {
					"code": 503,
					"message": "service unavailable.",
					"additional_message": ""
				}
			}`, rec.Body.String())
		})
	}
}

func TestRegisterRoutes_メンテナンス中も未登録パスは404(t *testing.T) {
	paths := []string{
		"/internal/not-registered",
		"/v1/not-registered",
		"/compat/chunirec/2.0/not-registered",
		"/compat/reiwa/1/not-registered",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			// Given
			maintenance := stubMaintenanceUsecase{state: usecase.MaintenanceState{Enabled: true}}
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			registerRoutes(
				e,
				newAuthorizationTestHandlers(),
				stubFirebaseAuthenticator{},
				stubFirebaseAuthenticator{},
				stubAPITokenUsecase{},
				maintenance,
				config.Config{},
			)
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			require.Equal(t, http.StatusNotFound, rec.Code)
			assert.Contains(t, rec.Body.String(), `"code":"not_found"`)
			assert.NotContains(t, rec.Body.String(), "maintenance_mode")
		})
	}
}

func newAuthorizationTestHandlers() *Handlers {
	return &Handlers{
		Login:               new(internalhandler.LoginHandler),
		Signup:              new(internalhandler.SignupHandler),
		Profile:             new(internalhandler.ProfileHandler),
		User:                new(internalhandler.UserHandler),
		AdminUser:           new(internalhandler.AdminUserHandler),
		AdminUserStatistics: new(internalhandler.AdminUserStatisticsHandler),
		AdminChartRanking:   new(internalhandler.AdminChartRankingHandler),
		Song:                new(internalhandler.SongHandler),
		BestSlotStats:       new(internalhandler.BestSlotStatsHandler),
		Honor:               new(internalhandler.HonorHandler),
		Worldsend:           new(internalhandler.WorldsendHandler),
		APIToken:            new(internalhandler.APITokenHandler),
		Me:                  new(internalhandler.MeHandler),
		DataTransfer:        internalhandler.NewDataTransferHandler(),
		MasterData:          new(internalhandler.MasterDataHandler),
		Goal:                new(internalhandler.GoalHandler),
		SystemMaintenance:   internalhandler.NewSystemMaintenanceHandler(stubMaintenanceUsecase{}),
		TemporaryPlayerData: new(internalhandler.TemporaryPlayerDataHandler),
		V1Song:              new(api_v1.V1SongHandler),
		V1Worldsend:         new(api_v1.V1WorldsendHandler),
		V1User:              new(api_v1.V1UserHandler),
		V1Version:           new(api_v1.V1VersionHandler),
		Chunirec:            new(chunirec.ChunirecHandler),
		Reiwa:               new(reiwa.ReiwaHandler),
	}
}
