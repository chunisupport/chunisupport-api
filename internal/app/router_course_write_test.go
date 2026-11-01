package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_v1"
	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/apitokenpermission"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type courseWriteUsecaseStub struct {
	usecase.CourseUsecase
	called bool
	err    error
}

func (s *courseWriteUsecaseStub) Create(_ context.Context, input usecase.CreateCourseInput) (*usecase.CourseOutput, error) {
	s.called = true
	return &usecase.CourseOutput{ID: 42, DisplayID: "0123456789abcdef", Idx: input.Idx, Name: input.Name, Class: input.Class}, s.err
}

func (s *courseWriteUsecaseStub) Update(_ context.Context, id string, input usecase.UpdateCourseInput) (*usecase.CourseOutput, error) {
	s.called = true
	return &usecase.CourseOutput{ID: 42, DisplayID: id, Idx: "50020", Name: input.Name, Class: input.Class}, s.err
}

type courseWriteTokenStub struct{ stubAPITokenUsecase }

func (s courseWriteTokenStub) Validate(ctx context.Context, token string) (*entity.User, *entity.APIToken, error) {
	if token == "invalid-token" {
		return nil, nil, usecase.ErrInvalidAPIToken
	}
	if token == "read-admin-token" {
		return authenticateTestUser("admin-token"), &entity.APIToken{ID: 1, Permission: apitokenpermission.Read}, nil
	}
	return s.stubAPITokenUsecase.Validate(ctx, token)
}

func TestRegisterRoutes_外部コース変更の権限(t *testing.T) {
	for _, operation := range []struct {
		name, method, path, body string
		status                   int
	}{
		{"作成", http.MethodPost, "/v1/courses", `{"idx":"50020","name":"コース","class":"1"}`, http.StatusCreated},
		{"編集", http.MethodPut, "/v1/courses/0123456789abcdef", `{"name":"コース","class":"1"}`, http.StatusOK},
	} {
		for _, token := range []string{"", "invalid-token", "player-token", "editor-token", "admin-token", "read-editor-token", "read-admin-token"} {
			t.Run(operation.name+"/"+token, func(t *testing.T) {
				// Given
				uc := &courseWriteUsecaseStub{}
				handlers := newAuthorizationTestHandlers()
				handlers.V1Course = api_v1.NewV1CourseHandler(uc)
				e := echo.New()
				e.Validator = NewCustomValidator()
				e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
				registerRoutes(e, handlers, stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, courseWriteTokenStub{}, stubMaintenanceUsecase{}, config.Config{})
				req := httptest.NewRequest(operation.method, operation.path, strings.NewReader(operation.body))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				if token != "" {
					req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
				}
				rec := httptest.NewRecorder()
				allowed := token == "admin-token" || (operation.method == http.MethodPut && token == "editor-token")
				wantStatus := http.StatusForbidden
				if allowed {
					wantStatus = operation.status
				}
				if token == "" || token == "invalid-token" {
					wantStatus = http.StatusUnauthorized
				}

				// When
				e.ServeHTTP(rec, req)

				// Then
				require.Equal(t, wantStatus, rec.Code, rec.Body.String())
				assert.Equal(t, allowed, uc.called)
				if allowed {
					assert.JSONEq(t, `{"id":"0123456789abcdef","idx":"50020","name":"コース","class":"1"}`, rec.Body.String())
				}
			})
		}
	}
}

func TestRegisterRoutes_外部コース変更の入力とエラー(t *testing.T) {
	for _, tt := range []struct {
		name, method, path, body string
		err                      error
		status                   int
		called                   bool
	}{
		{name: "作成の必須項目不足", method: http.MethodPost, path: "/v1/courses", body: `{"name":"コース","class":"1"}`, status: 422},
		{name: "編集の必須項目不足", method: http.MethodPut, path: "/v1/courses/0123456789abcdef", body: `{"class":"1"}`, status: 422},
		{name: "作成の長すぎる名称", method: http.MethodPost, path: "/v1/courses", body: `{"idx":"50020","name":"` + strings.Repeat("あ", 256) + `","class":"1"}`, status: 422},
		{name: "編集の不正ID", method: http.MethodPut, path: "/v1/courses/42", body: `{"name":"コース","class":"1"}`, status: 422},
		{name: "編集でidx変更を拒否", method: http.MethodPut, path: "/v1/courses/0123456789abcdef", body: `{"idx":"50021","name":"コース","class":"1"}`, status: 400},
		{name: "作成の不正JSON", method: http.MethodPost, path: "/v1/courses", body: `{`, status: 400},
		{name: "公式idx重複", method: http.MethodPost, path: "/v1/courses", body: `{"idx":"50020","name":"コース","class":"1"}`, err: repository.ErrDuplicateOfficialIdx, status: 409, called: true},
		{name: "編集対象なし", method: http.MethodPut, path: "/v1/courses/0123456789abcdef", body: `{"name":"コース","class":"1"}`, err: usecase.ErrCourseNotFound, status: 404, called: true},
		{name: "作成のドメイン入力エラー", method: http.MethodPost, path: "/v1/courses", body: `{"idx":"50020","name":"コース","class":"unknown"}`, err: usecase.ErrInvalidCourseInput, status: 422, called: true},
		{name: "作成のキャンセル原因保持", method: http.MethodPost, path: "/v1/courses", body: `{"idx":"50020","name":"コース","class":"1"}`, err: context.Canceled, status: 499, called: true},
		{name: "編集のキャンセル原因保持", method: http.MethodPut, path: "/v1/courses/0123456789abcdef", body: `{"name":"コース","class":"1"}`, err: context.Canceled, status: 499, called: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			uc := &courseWriteUsecaseStub{err: tt.err}
			handlers := newAuthorizationTestHandlers()
			handlers.V1Course = api_v1.NewV1CourseHandler(uc)
			e := echo.New()
			e.Validator = NewCustomValidator()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			registerRoutes(e, handlers, stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, courseWriteTokenStub{}, stubMaintenanceUsecase{}, config.Config{})
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			req.Header.Set(echo.HeaderAuthorization, "Bearer admin-token")
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			assert.Equal(t, tt.status, rec.Code, rec.Body.String())
			assert.Equal(t, tt.called, uc.called)
		})
	}
}
