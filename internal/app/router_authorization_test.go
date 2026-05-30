package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	internalhandler "github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/chunirec"
	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
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

type countingAuthenticator struct {
	authenticateCalls         int
	authenticateOptionalCalls int
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

func TestRegisterRoutes_楽曲追加削除はEDITORを拒否する(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "通常楽曲追加はEDITOR拒否", method: http.MethodPost, path: "/internal/songs"},
		{name: "通常楽曲削除はEDITOR拒否", method: http.MethodDelete, path: "/internal/songs/abcd1234"},
		{name: "WORLDS END楽曲追加はEDITOR拒否", method: http.MethodPost, path: "/internal/songs/worldsend"},
		{name: "WORLDS END楽曲削除はEDITOR拒否", method: http.MethodDelete, path: "/internal/songs/worldsend/abcd1234"},
		{name: "称号一覧はEDITOR拒否", method: http.MethodGet, path: "/internal/honors"},
		{name: "称号追加はEDITOR拒否", method: http.MethodPost, path: "/internal/honors"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			registerRoutes(e, newAuthorizationTestHandlers(), stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, nil, config.Config{})

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

func TestRegisterRoutes_公開GETはread最適化認証を使い書き込みはstrict認証を使う(t *testing.T) {
	// Given
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	strictAuth := &countingAuthenticator{}
	readOptimizedAuth := &countingAuthenticator{}
	registerRoutes(e, newAuthorizationTestHandlers(), strictAuth, readOptimizedAuth, nil, config.Config{})

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
	registerRoutes(e, newAuthorizationTestHandlers(), strictAuth, readOptimizedAuth, nil, config.Config{})

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

func newAuthorizationTestHandlers() *Handlers {
	return &Handlers{
		Login:               new(internalhandler.LoginHandler),
		Signup:              new(internalhandler.SignupHandler),
		Profile:             new(internalhandler.ProfileHandler),
		User:                new(internalhandler.UserHandler),
		AdminUser:           new(internalhandler.AdminUserHandler),
		Song:                new(internalhandler.SongHandler),
		Honor:               new(internalhandler.HonorHandler),
		Worldsend:           new(internalhandler.WorldsendHandler),
		APIToken:            new(internalhandler.APITokenHandler),
		Me:                  new(internalhandler.MeHandler),
		MasterData:          new(internalhandler.MasterDataHandler),
		Goal:                new(internalhandler.GoalHandler),
		TemporaryPlayerData: new(internalhandler.TemporaryPlayerDataHandler),
		V1Song:              new(api_v1.V1SongHandler),
		V1Worldsend:         new(api_v1.V1WorldsendHandler),
		V1User:              new(api_v1.V1UserHandler),
		V1Version:           new(api_v1.V1VersionHandler),
		Chunirec:            new(chunirec.ChunirecHandler),
	}
}
