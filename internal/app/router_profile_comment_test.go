package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	internalhandler "github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/featureflag"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingProfileCommentUsecase struct {
	gets    int
	updates int
}

func (u *recordingProfileCommentUsecase) Get(context.Context, string, *entity.User) (*string, error) {
	u.gets++
	comment := "よろしく"
	return &comment, nil
}

func (u *recordingProfileCommentUsecase) Update(context.Context, int, string) (*string, error) {
	u.updates++
	comment := "よろしく"
	return &comment, nil
}

func TestRegisterRoutes_プロフィールコメントはフラグが有効なときだけ登録する(t *testing.T) {
	tests := []struct {
		name        string
		enabled     bool
		maintenance bool
	}{
		{name: "無効なら未登録として404"},
		{name: "無効かつメンテナンス中も認証より前に404", maintenance: true},
		{name: "有効なら保存と取得を受け付ける", enabled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: ハンドラーは常に渡す。無効時に呼ばれないことを呼び出し回数で確認する。
			commentUsecase := &recordingProfileCommentUsecase{}
			handlers := newAuthorizationTestHandlers()
			handlers.UserProfileComment = internalhandler.NewUserProfileCommentHandler(commentUsecase)
			cfg := config.Config{}
			if tt.enabled {
				cfg.FeatureFlags = featureflag.New(featureflag.ProfileComment)
			}
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			registerRoutes(
				e,
				handlers,
				stubFirebaseAuthenticator{},
				stubFirebaseAuthenticator{},
				nil,
				stubMaintenanceUsecase{state: usecase.MaintenanceState{Enabled: tt.maintenance}},
				cfg,
			)

			putReq := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/internal/me/comment", strings.NewReader(`{"comment":"よろしく"}`))
			putReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			putReq.Header.Set(echo.HeaderAuthorization, "Bearer player-token")
			putRec := httptest.NewRecorder()
			getReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/users/tester/comment", nil)
			getRec := httptest.NewRecorder()

			// When
			e.ServeHTTP(putRec, putReq)
			e.ServeHTTP(getRec, getReq)

			// Then
			if tt.enabled {
				require.Equal(t, http.StatusOK, putRec.Code)
				assert.JSONEq(t, `{"comment":"よろしく"}`, putRec.Body.String())
				require.Equal(t, http.StatusOK, getRec.Code)
				assert.JSONEq(t, `{"comment":"よろしく"}`, getRec.Body.String())
				assert.Equal(t, 1, commentUsecase.updates)
				assert.Equal(t, 1, commentUsecase.gets)
				return
			}
			require.Equal(t, http.StatusNotFound, putRec.Code)
			assert.Contains(t, putRec.Body.String(), `"code":"not_found"`)
			assert.NotContains(t, putRec.Body.String(), "maintenance_mode")
			require.Equal(t, http.StatusNotFound, getRec.Code)
			assert.Contains(t, getRec.Body.String(), `"code":"not_found"`)
			assert.Zero(t, commentUsecase.updates)
			assert.Zero(t, commentUsecase.gets)
		})
	}
}
