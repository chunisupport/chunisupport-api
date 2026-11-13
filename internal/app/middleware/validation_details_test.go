package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomHTTPErrorHandler_入力エラーの詳細だけを公開する(t *testing.T) {
	validation := validator.New()
	type input struct {
		Required string `validate:"required"`
		Short    string `validate:"min=5"`
		Long     string `validate:"max=3"`
		Email    string `validate:"email"`
	}
	err := validation.Struct(input{Short: "x", Long: "long", Email: "not-an-email"})
	var fields validator.ValidationErrors
	require.ErrorAs(t, err, &fields)
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"検証の詳細", fmt.Errorf("外側: %w", apierror.ErrValidationFailed.WithInternal(apierror.ValidationErrors(fields))), 422, `{"error":{"status":422,"code":"validation_failed","message":"入力値の形式に誤りがあります。","details":[{"field":"required","message":"必須項目です。"},{"field":"short","message":"5文字以上で入力してください。"},{"field":"long","message":"3文字以下で入力してください。"},{"field":"email","message":"入力値の形式が不正です。"}]}}`},
		{"詳細なし", apierror.ErrValidationFailed, 422, `{"error":{"status":422,"code":"validation_failed","message":"入力値の形式を確認してください。"}}`},
		{"内部原因を公開しない", apierror.ErrValidationFailed.WithInternal(errors.New("秘密のDB情報")), 422, `{"error":{"status":422,"code":"validation_failed","message":"入力値の形式を確認してください。"}}`},
		{"検証以外には詳細を付けない", apierror.ErrInternalError.WithInternal(apierror.ValidationErrors(fields)), 500, `{"error":{"status":500,"code":"internal_error"}}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
			CustomHTTPErrorHandler(c, tt.err)
			assert.Equal(t, tt.status, rec.Code)
			assert.JSONEq(t, tt.want, rec.Body.String())
		})
	}
}
