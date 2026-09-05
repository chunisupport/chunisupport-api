package api_internal

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type versionUsecaseMock struct {
	versions   []*entity.Version
	createErr  error
	renameErr  error
	deleteErr  error
	createCall bool
	renameCall bool
	deleteCall bool
}

func (m *versionUsecaseMock) ListAll(context.Context) ([]*entity.Version, error) {
	return m.versions, nil
}
func (m *versionUsecaseMock) Create(_ context.Context, name string, releasedAt time.Time) (*entity.Version, error) {
	m.createCall = true
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &entity.Version{ID: 1, Name: name, ReleasedAt: releasedAt}, nil
}
func (m *versionUsecaseMock) Rename(_ context.Context, id int, name string) (*entity.Version, error) {
	m.renameCall = true
	if m.renameErr != nil {
		return nil, m.renameErr
	}
	return &entity.Version{ID: id, Name: name, ReleasedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, nil
}
func (m *versionUsecaseMock) Delete(context.Context, int) error {
	m.deleteCall = true
	return m.deleteErr
}

func TestVersionHandler_List(t *testing.T) {
	uc := &versionUsecaseMock{versions: []*entity.Version{{ID: 1, Name: "CHUNITHM VERSE", ReleasedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}}}
	handler := NewVersionHandler(uc)
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/internal/admin/versions", nil), rec)

	err := handler.List(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `[{"id":1,"name":"CHUNITHM VERSE","released_at":"2025-01-01"}]`, rec.Body.String())
}

func TestVersionHandler_Create(t *testing.T) {
	uc := &versionUsecaseMock{}
	handler := NewVersionHandler(uc)
	e := echo.New()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/admin/versions", bytes.NewBufferString(`{"name":"CHUNITHM VERSE","released_at":"2025-01-01"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c := e.NewContext(req, rec)

	err := handler.Create(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.True(t, uc.createCall)
}

func TestVersionHandler_Create_不正JSONは400(t *testing.T) {
	uc := &versionUsecaseMock{}
	handler := NewVersionHandler(uc)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/admin/versions", bytes.NewBufferString(`{"name":`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	err := handler.Create(e.NewContext(req, httptest.NewRecorder()))

	apiErr := requireVersionAPIError(t, err)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	assert.False(t, uc.createCall)
}

func TestVersionHandler_Create_不正日付は422(t *testing.T) {
	uc := &versionUsecaseMock{}
	handler := NewVersionHandler(uc)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/admin/versions", bytes.NewBufferString(`{"name":"CHUNITHM VERSE","released_at":"2025-02-30"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	err := handler.Create(e.NewContext(req, httptest.NewRecorder()))

	apiErr := requireVersionAPIError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	assert.Equal(t, apierror.CodeInvalidVersionInput, apiErr.Code)
	assert.False(t, uc.createCall)
}

func TestVersionHandler_Create_名前重複は409(t *testing.T) {
	uc := &versionUsecaseMock{createErr: repository.ErrVersionConflict}
	handler := NewVersionHandler(uc)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/admin/versions", bytes.NewBufferString(`{"name":"CHUNITHM VERSE","released_at":"2025-01-01"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	err := handler.Create(e.NewContext(req, httptest.NewRecorder()))

	apiErr := requireVersionAPIError(t, err)
	assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
	assert.Equal(t, apierror.CodeVersionNameConflict, apiErr.Code)
}

func TestVersionHandler_Rename_不正IDは400(t *testing.T) {
	uc := &versionUsecaseMock{}
	handler := NewVersionHandler(uc)
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodPut, "/internal/admin/versions/invalid", bytes.NewBufferString(`{"name":"CHUNITHM VERSE"}`)), httptest.NewRecorder())
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "invalid"}})

	err := handler.Rename(c)

	apiErr := requireVersionAPIError(t, err)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	assert.False(t, uc.renameCall)
}

func TestVersionHandler_Rename_releasedAtは拒否する(t *testing.T) {
	for _, releasedAt := range []string{`"2025-01-01"`, `null`} {
		t.Run(releasedAt, func(t *testing.T) {
			uc := &versionUsecaseMock{}
			handler := NewVersionHandler(uc)
			e := echo.New()
			body := `{"name":"CHUNITHM VERSE","released_at":` + releasedAt + `}`
			req := httptest.NewRequest(http.MethodPut, "/internal/admin/versions/1", bytes.NewBufferString(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			c := e.NewContext(req, httptest.NewRecorder())
			c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})

			err := handler.Rename(c)

			apiErr := requireVersionAPIError(t, err)
			assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
			assert.False(t, uc.renameCall)
		})
	}
}

func TestVersionHandler_Delete_競合エラーを対応付ける(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{name: "最新版ではない", err: usecase.ErrVersionNotLatest, code: apierror.CodeVersionNotLatest},
		{name: "曲が存在する", err: usecase.ErrVersionInUse, code: apierror.CodeVersionInUse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &versionUsecaseMock{deleteErr: tt.err}
			handler := NewVersionHandler(uc)
			e := echo.New()
			c := e.NewContext(httptest.NewRequest(http.MethodDelete, "/internal/admin/versions/1", nil), httptest.NewRecorder())
			c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})

			err := handler.Delete(c)

			apiErr := requireVersionAPIError(t, err)
			assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
			assert.Equal(t, tt.code, apiErr.Code)
		})
	}
}

func TestVersionHandler_Delete_正常系(t *testing.T) {
	uc := &versionUsecaseMock{}
	handler := NewVersionHandler(uc)
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodDelete, "/internal/admin/versions/1", nil), rec)
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})

	err := handler.Delete(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.True(t, uc.deleteCall)
}

func requireVersionAPIError(t *testing.T, err error) *apierror.APIError {
	t.Helper()
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.True(t, errors.As(err, &apiErr))
	return apiErr
}
