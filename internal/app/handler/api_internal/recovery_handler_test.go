package api_internal_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRecoveryHandler_IssueRecoveryCodes(t *testing.T) {
	e := newTestEcho()
	h, recoveryMock := newRecoveryHandlerWithMock()

	t.Run("正常系: リカバリーコード再発行", func(t *testing.T) {
		// Given
		user := &entity.User{ID: 30}
		recoveryMock.On("IssueRecoveryCodes", mock.Anything, 30).Return([]string{"AAAA-BBBB-CCCC", "DDDD-EEEE-FFFF"}, nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/internal/me/recovery-codes/issue", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		// When
		err := h.IssueRecoveryCodes(c)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "AAAA-BBBB-CCCC")
		recoveryMock.AssertExpectations(t)
	})
}

func TestRecoveryHandler_RecoverPassword(t *testing.T) {
	e := newTestEcho()
	h, recoveryMock := newRecoveryHandlerWithMock()

	t.Run("正常系: リカバリーコードでパスワード再設定", func(t *testing.T) {
		// Given
		recoveryMock.On("RecoverWithRecoveryCode", mock.Anything, "ABCD-EFGH-IJKL", "newPassword123").Return(nil).Once()
		body := `{"recovery_code":"ABCD-EFGH-IJKL","new_password":"newPassword123"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/recovery-codes", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// When
		err := h.RecoverPassword(c)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		recoveryMock.AssertExpectations(t)
	})

	t.Run("異常系: リカバリーコード形式不正", func(t *testing.T) {
		// Given
		body := `{"recovery_code":"INVALID","new_password":"newPassword123"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/recovery-codes", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// When
		err := h.RecoverPassword(c)

		// Then
		assert.ErrorIs(t, err, apierror.ErrBadRequest)
	})

	t.Run("異常系: 新しいパスワードが短い場合はバリデーションエラー", func(t *testing.T) {
		// Given
		body := `{"recovery_code":"ABCD-EFGH-IJKL","new_password":"short"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/recovery-codes", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// When
		err := h.RecoverPassword(c)

		// Then
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	})
}
