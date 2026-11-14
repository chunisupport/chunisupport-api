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

type stubUserProfileCommentUsecase struct {
	output          *string
	err             error
	calls           int
	userID          int
	input, username string
	requester       *entity.User
}

func (s *stubUserProfileCommentUsecase) Get(_ context.Context, name string, requester *entity.User) (*string, error) {
	s.calls++
	s.username = name
	s.requester = requester
	return s.output, s.err
}
func (s *stubUserProfileCommentUsecase) Update(_ context.Context, id int, input string) (*string, error) {
	s.calls++
	s.userID = id
	s.input = input
	return s.output, s.err
}

func TestUserProfileCommentHandler_Update(t *testing.T) {
	saved := "よろしく"
	tests := []struct {
		name, body         string
		unauth             bool
		output             *string
		err                error
		wantCode, wantBody string
		called             bool
		// wantInput は成功時にusecaseへ転送されるべき入力コメントの原文。前後の空白除去はusecaseの責務のため原文で検証する。
		wantInput string
	}{
		{name: "保存", body: `{"comment":" よろしく "}`, output: &saved, wantBody: `{"comment":"よろしく"}`, called: true, wantInput: " よろしく "},
		{name: "削除", body: `{"comment":""}`, wantBody: `{"comment":null}`, called: true, wantInput: ""},
		{name: "未知フィールド", body: `{"comment":"a","extra":1}`, wantCode: apierror.CodeBadRequest},
		{name: "型不正", body: `{"comment":5}`, wantCode: apierror.CodeBadRequest},
		{name: "フィールド欠落", body: `{}`, wantCode: apierror.CodeBadRequest},
		{name: "nullは拒否", body: `{"comment":null}`, wantCode: apierror.CodeBadRequest},
		{name: "不正JSON", body: `{`, wantCode: apierror.CodeBadRequest},
		{name: "未認証", body: `{"comment":"a"}`, unauth: true, wantCode: apierror.CodeUnauthorized},
		{name: "文字数エラー", body: `{"comment":"a"}`, err: usecase.ErrCommentTooLong, wantCode: apierror.CodeCommentTooLong, called: true},
		{name: "文字種エラー", body: `{"comment":"a"}`, err: usecase.ErrCommentInvalidChar, wantCode: apierror.CodeCommentInvalidChar, called: true},
		{name: "キャンセル", body: `{"comment":"a"}`, err: context.Canceled, wantCode: apierror.CodeInternalError, called: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			uc := &stubUserProfileCommentUsecase{output: tt.output, err: tt.err}
			h := NewUserProfileCommentHandler(uc)
			e := echo.New()
			req := httptest.NewRequest(http.MethodPut, "/internal/me/comment", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if !tt.unauth {
				c.Set("userEntity", &entity.User{ID: 7})
			}
			// When
			err := h.Update(c)
			// Then
			if tt.wantCode != "" {
				var apiErr *apierror.APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.wantCode, apiErr.Code)
				if tt.err != nil {
					assert.ErrorIs(t, err, tt.err)
				}
				if tt.err == usecase.ErrCommentTooLong || tt.err == usecase.ErrCommentInvalidChar {
					assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
				assert.JSONEq(t, tt.wantBody, rec.Body.String())
			}
			if tt.called {
				assert.Equal(t, 1, uc.calls)
				assert.Equal(t, 7, uc.userID)
				if tt.wantCode == "" {
					assert.Equal(t, tt.wantInput, uc.input)
				}
			} else {
				assert.Zero(t, uc.calls)
			}
		})
	}
}

func TestUserProfileCommentHandler_Get(t *testing.T) {
	saved := "よろしく"
	tests := []struct {
		name, username     string
		requester          *entity.User
		output             *string
		err                error
		wantCode, wantBody string
	}{
		{name: "匿名で取得", username: "tester", output: &saved, wantBody: `{"comment":"よろしく"}`},
		{name: "本人で未設定", username: "tester", requester: &entity.User{ID: 7}, wantBody: `{"comment":null}`},
		{name: "存在しない", username: "tester", err: usecase.ErrUserNotFound, wantCode: apierror.CodeUserNotFound},
		{name: "非公開も404", username: "tester", err: usecase.ErrUserPrivate, wantCode: apierror.CodeUserNotFound},
		{name: "不正なユーザー名", username: "x", wantCode: apierror.CodeUsernameTooShort},
		{name: "キャンセル", username: "tester", err: context.Canceled, wantCode: apierror.CodeInternalError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			uc := &stubUserProfileCommentUsecase{output: tt.output, err: tt.err}
			h := NewUserProfileCommentHandler(uc)
			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
			c.SetPathValues(echo.PathValues{{Name: "username", Value: tt.username}})
			if tt.requester != nil {
				c.Set("userEntity", tt.requester)
			}
			// When
			err := h.Get(c)
			// Then
			if tt.wantCode != "" {
				var apiErr *apierror.APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.wantCode, apiErr.Code)
				if tt.err == context.Canceled {
					assert.ErrorIs(t, err, context.Canceled)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
				assert.JSONEq(t, tt.wantBody, rec.Body.String())
			}
			if tt.username == "x" {
				assert.Zero(t, uc.calls)
			} else {
				assert.Equal(t, tt.username, uc.username)
				assert.Same(t, tt.requester, uc.requester)
			}
		})
	}
}
