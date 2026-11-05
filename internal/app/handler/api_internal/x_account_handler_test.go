package api_internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubXAccountUsecase struct {
	getOutput  *usecase.XAccountOutput
	syncOutput *usecase.XAccountOutput
	syncErr    error
	syncInput  usecase.XAccountSyncInput
	syncUserID int
}

func (s *stubXAccountUsecase) Get(_ context.Context, _ int) (*usecase.XAccountOutput, error) {
	return s.getOutput, nil
}

func (s *stubXAccountUsecase) Sync(_ context.Context, user *entity.User, input usecase.XAccountSyncInput) (*usecase.XAccountOutput, error) {
	s.syncUserID = user.ID
	s.syncInput = input
	return s.syncOutput, s.syncErr
}

func newXAccountHandlerContext(method string, body string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, "/internal/me/x-account", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 10})
	return c, rec
}

func TestXAccountHandler_Get(t *testing.T) {
	tests := []struct {
		name     string
		output   *usecase.XAccountOutput
		wantBody string
	}{
		{name: "保存済みならIDとユーザー名を返す", output: &usecase.XAccountOutput{ID: "123", Username: "chuni"}, wantBody: `{"x_account":{"id":"123","username":"chuni"}}`},
		{name: "未保存ならnullを返す", output: nil, wantBody: `{"x_account":null}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			h := NewXAccountHandler(&stubXAccountUsecase{getOutput: tt.output})
			c, rec := newXAccountHandlerContext(http.MethodGet, "")

			// When
			err := h.Get(c)

			// Then
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.JSONEq(t, tt.wantBody, rec.Body.String())
		})
	}
}

func TestXAccountHandler_Sync_取得したXアカウントを返す(t *testing.T) {
	// Given
	uc := &stubXAccountUsecase{syncOutput: &usecase.XAccountOutput{ID: "123", Username: "chuni"}}
	h := NewXAccountHandler(uc)
	c, rec := newXAccountHandlerContext(http.MethodPut, `{"access_token":"token","access_token_secret":"secret"}`)

	// When
	err := h.Sync(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"id":"123","username":"chuni"}`, rec.Body.String())
	assert.Equal(t, 10, uc.syncUserID)
	assert.Equal(t, usecase.XAccountSyncInput{AccessToken: "token", AccessTokenSecret: "secret"}, uc.syncInput)
}

func TestXAccountHandler_Sync_エラー応答(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		syncErr  error
		wantCode string
	}{
		{name: "未知のフィールドは不正なリクエスト", body: `{"access_token":"token","access_token_secret":"secret","username":"fake"}`, wantCode: apierror.CodeBadRequest},
		{name: "ユースケースのエラーをAPIエラーへ変換する", body: `{"access_token":"token","access_token_secret":"secret"}`, syncErr: usecase.ErrInvalidXCredentials, wantCode: apierror.CodeInvalidXCredentials},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			h := NewXAccountHandler(&stubXAccountUsecase{syncErr: tt.syncErr})
			c, _ := newXAccountHandlerContext(http.MethodPut, tt.body)

			// When
			err := h.Sync(c)

			// Then
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.wantCode, apiErr.Code)
		})
	}
}
