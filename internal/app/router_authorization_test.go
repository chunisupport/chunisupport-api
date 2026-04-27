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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			registerRoutes(e, newAuthorizationTestHandlers(), stubFirebaseAuthenticator{}, nil, config.Config{})

			req := httptest.NewRequest(tt.method, tt.path, nil)
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

func newAuthorizationTestHandlers() *Handlers {
	return &Handlers{
		Signup:              new(internalhandler.SignupHandler),
		Profile:             new(internalhandler.ProfileHandler),
		User:                new(internalhandler.UserHandler),
		AdminUser:           new(internalhandler.AdminUserHandler),
		Song:                new(internalhandler.SongHandler),
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
