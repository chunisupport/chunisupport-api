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

type userPermissionUsecaseStub struct {
	changePermission func(context.Context, *entity.User, string, string) error
}

func (s *userPermissionUsecaseStub) ChangePermission(ctx context.Context, requester *entity.User, username string, permission string) error {
	return s.changePermission(ctx, requester, username, permission)
}

func TestUserPermissionHandler_UpdatePermission(t *testing.T) {
	t.Run("ADMINの権限変更をユースケースへ渡し204を返す", func(t *testing.T) {
		// Given
		requester := &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin}
		called := false
		handler := api_internal.NewUserPermissionHandler(&userPermissionUsecaseStub{
			changePermission: func(_ context.Context, actualRequester *entity.User, username string, permission string) error {
				called = true
				assert.Same(t, requester, actualRequester)
				assert.Equal(t, "targetuser", username)
				assert.Equal(t, "EDITOR", permission)
				return nil
			},
		})
		e := newTestEcho()
		req := httptest.NewRequest(http.MethodPatch, "/internal/users/targetuser/permission", bytes.NewBufferString(`{"permission":"EDITOR"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "targetuser"}})
		c.Set("userEntity", requester)

		// When
		err := handler.UpdatePermission(c)

		// Then
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("ADMINが自分を降格しようとすると403を返す", func(t *testing.T) {
		// Given
		handler := api_internal.NewUserPermissionHandler(&userPermissionUsecaseStub{
			changePermission: func(context.Context, *entity.User, string, string) error {
				return entity.ErrCannotDemoteOwnAdmin
			},
		})
		e := newTestEcho()
		req := httptest.NewRequest(http.MethodPatch, "/internal/users/adminuser/permission", bytes.NewBufferString(`{"permission":"EDITOR"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "adminuser"}})
		c.Set("userEntity", &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin})

		// When
		err := handler.UpdatePermission(c)

		// Then
		var apiErr *apierror.APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusForbidden, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeForbidden, apiErr.Code)
	})

	t.Run("未知の権限は400を返す", func(t *testing.T) {
		// Given
		handler := api_internal.NewUserPermissionHandler(&userPermissionUsecaseStub{
			changePermission: func(context.Context, *entity.User, string, string) error {
				return entity.ErrInvalidAccountType
			},
		})
		e := newTestEcho()
		req := httptest.NewRequest(http.MethodPatch, "/internal/users/targetuser/permission", bytes.NewBufferString(`{"permission":"INVALID"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "targetuser"}})
		c.Set("userEntity", &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin})

		// When
		err := handler.UpdatePermission(c)

		// Then
		var apiErr *apierror.APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
	})

	t.Run("不正JSONは400を返す", func(t *testing.T) {
		// Given
		called := false
		handler := api_internal.NewUserPermissionHandler(&userPermissionUsecaseStub{
			changePermission: func(context.Context, *entity.User, string, string) error {
				called = true
				return errors.New("must not be called")
			},
		})
		e := newTestEcho()
		req := httptest.NewRequest(http.MethodPatch, "/internal/users/targetuser/permission", bytes.NewBufferString(`{"permission":"EDITOR","unknown":true}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "targetuser"}})
		c.Set("userEntity", &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin})

		// When
		err := handler.UpdatePermission(c)

		// Then
		var apiErr *apierror.APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
		assert.False(t, called)
	})
}
func TestUserPermissionHandler_UpdatePermission_保存競合(t *testing.T) {
	handler := api_internal.NewUserPermissionHandler(&userPermissionUsecaseStub{
		changePermission: func(context.Context, *entity.User, string, string) error {
			return repository.ErrUserConflict
		},
	})
	e := newTestEcho()
	req := httptest.NewRequest(http.MethodPatch, "/internal/users/targetuser/permission", bytes.NewBufferString(`{"permission":"EDITOR"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c := e.NewContext(req, httptest.NewRecorder())
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "targetuser"}})
	c.Set("userEntity", &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin})
	err := handler.UpdatePermission(c)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
	assert.Equal(t, apierror.CodeConflict, apiErr.Code)
}
