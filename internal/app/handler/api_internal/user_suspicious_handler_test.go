package api_internal_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type userSuspiciousUsecaseStub struct {
	changeSuspicious func(context.Context, *entity.User, string, bool) error
}

func (s *userSuspiciousUsecaseStub) ChangeSuspicious(ctx context.Context, requester *entity.User, username string, isSuspicious bool) error {
	return s.changeSuspicious(ctx, requester, username, isSuspicious)
}

func TestUserSuspiciousHandler_UpdateSuspicious(t *testing.T) {
	t.Run("ADMINの不審フラグ変更をユースケースへ渡し204を返す", func(t *testing.T) {
		// Given
		requester := &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin}
		called := false
		handler := api_internal.NewUserSuspiciousHandler(&userSuspiciousUsecaseStub{
			changeSuspicious: func(_ context.Context, actualRequester *entity.User, username string, isSuspicious bool) error {
				called = true
				assert.Same(t, requester, actualRequester)
				assert.Equal(t, "targetuser", username)
				assert.True(t, isSuspicious)
				return nil
			},
		})
		e := newTestEcho()
		req := httptest.NewRequest(http.MethodPatch, "/internal/users/targetuser/suspicious", bytes.NewBufferString(`{"is_suspicious":true}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "targetuser"}})
		c.Set("userEntity", requester)

		// When
		err := handler.UpdateSuspicious(c)

		// Then
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("保存競合は409を返す", func(t *testing.T) {
		// Given
		handler := api_internal.NewUserSuspiciousHandler(&userSuspiciousUsecaseStub{
			changeSuspicious: func(context.Context, *entity.User, string, bool) error {
				return repository.ErrUserConflict
			},
		})
		e := newTestEcho()
		req := httptest.NewRequest(http.MethodPatch, "/internal/users/targetuser/suspicious", bytes.NewBufferString(`{"is_suspicious":false}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "targetuser"}})
		c.Set("userEntity", &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin})

		// When
		err := handler.UpdateSuspicious(c)

		// Then
		var apiErr *apierror.APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeConflict, apiErr.Code)
	})

	t.Run("不正JSONは400を返す", func(t *testing.T) {
		// Given
		called := false
		handler := api_internal.NewUserSuspiciousHandler(&userSuspiciousUsecaseStub{
			changeSuspicious: func(context.Context, *entity.User, string, bool) error {
				called = true
				return errors.New("must not be called")
			},
		})
		e := newTestEcho()
		req := httptest.NewRequest(http.MethodPatch, "/internal/users/targetuser/suspicious", bytes.NewBufferString(`{"is_suspicious":true,"unknown":true}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "targetuser"}})
		c.Set("userEntity", &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin})

		// When
		err := handler.UpdateSuspicious(c)

		// Then
		var apiErr *apierror.APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
		assert.False(t, called)
	})

	t.Run("is_suspiciousが欠落している場合は400を返す", func(t *testing.T) {
		// Given
		called := false
		handler := api_internal.NewUserSuspiciousHandler(&userSuspiciousUsecaseStub{
			changeSuspicious: func(context.Context, *entity.User, string, bool) error {
				called = true
				return nil
			},
		})
		e := newTestEcho()
		req := httptest.NewRequest(http.MethodPatch, "/internal/users/targetuser/suspicious", bytes.NewBufferString(`{}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "targetuser"}})
		c.Set("userEntity", &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin})

		// When
		err := handler.UpdateSuspicious(c)

		// Then
		var apiErr *apierror.APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
		assert.False(t, called)
	})
}
