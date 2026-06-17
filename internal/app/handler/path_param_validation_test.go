package handler

import (
	"net/http"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/stretchr/testify/assert"
)

func TestValidateDisplayID(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		wantErrCode  string
		wantHTTPCode int
	}{
		{
			name:  "16桁の小文字16進数は有効",
			value: "1234567890abcdef",
		},
		{
			name:         "短いIDはvalidation_failed",
			value:        "short",
			wantErrCode:  apierror.CodeValidationFailed,
			wantHTTPCode: http.StatusUnprocessableEntity,
		},
		{
			name:         "大文字を含むIDはvalidation_failed",
			value:        "1234567890ABCDEF",
			wantErrCode:  apierror.CodeValidationFailed,
			wantHTTPCode: http.StatusUnprocessableEntity,
		},
		{
			name:         "16進数以外を含むIDはvalidation_failed",
			value:        "1234567890abcdeg",
			wantErrCode:  apierror.CodeValidationFailed,
			wantHTTPCode: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, apiErr := ValidateDisplayID(tt.value)

			if tt.wantErrCode == "" {
				assert.Nil(t, apiErr)
				assert.Equal(t, tt.value, value)
				return
			}

			if assert.NotNil(t, apiErr) {
				assert.Equal(t, tt.wantErrCode, apiErr.Code)
				assert.Equal(t, tt.wantHTTPCode, apiErr.HTTPStatus)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		wantErrCode string
	}{
		{
			name:  "小文字英数字5文字以上は有効",
			value: "testuser1",
		},
		{
			name:        "空文字はusername_empty",
			value:       "",
			wantErrCode: apierror.CodeUsernameEmpty,
		},
		{
			name:        "短すぎるユーザー名はusername_too_short",
			value:       "abc",
			wantErrCode: apierror.CodeUsernameTooShort,
		},
		{
			name:        "大文字を含むユーザー名はusername_invalid_char",
			value:       "TestUser",
			wantErrCode: apierror.CodeUsernameInvalidChar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, apiErr := ValidateUsername(tt.value)

			if tt.wantErrCode == "" {
				assert.Nil(t, apiErr)
				assert.Equal(t, tt.value, value)
				return
			}

			if assert.NotNil(t, apiErr) {
				assert.Equal(t, tt.wantErrCode, apiErr.Code)
				assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
			}
		})
	}
}
