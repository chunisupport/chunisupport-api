package api_internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRecordFilterUsecase struct {
	listCalled     bool
	listUserID     int
	listFilterType string
	createCalled   bool
	createUserID   int
	createInput    *usecase.RecordFilterInput
	updateCalled   bool
	updateUserID   int
	updateID       string
	updateInput    *usecase.RecordFilterInput
	deleteCalled   bool
	deleteUserID   int
	deleteID       string
}

func (m *mockRecordFilterUsecase) List(ctx context.Context, userID int, filterType string) ([]*usecase.RecordFilterOutput, error) {
	m.listCalled = true
	m.listUserID = userID
	m.listFilterType = filterType
	return []*usecase.RecordFilterOutput{
		{
			ID:            "11111111-1111-1111-1111-111111111111",
			Name:          "高難度",
			FilterType:    usecase.RecordFilterTypeStandard,
			SchemaVersion: 3,
			Filter:        []byte(`{"title":"","difficulties":["MASTER"]}`),
			CreatedAt:     time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
			UpdatedAt:     time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
		},
	}, nil
}

func (m *mockRecordFilterUsecase) Create(ctx context.Context, userID int, input *usecase.RecordFilterInput) (*usecase.RecordFilterOutput, error) {
	m.createCalled = true
	m.createUserID = userID
	m.createInput = input
	return &usecase.RecordFilterOutput{
		ID:            "22222222-2222-2222-2222-222222222222",
		Name:          input.Name,
		FilterType:    input.FilterType,
		SchemaVersion: input.SchemaVersion,
		Filter:        input.Filter,
		CreatedAt:     time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (m *mockRecordFilterUsecase) Update(ctx context.Context, userID int, id string, input *usecase.RecordFilterInput) (*usecase.RecordFilterOutput, error) {
	m.updateCalled = true
	m.updateUserID = userID
	m.updateID = id
	m.updateInput = input
	return &usecase.RecordFilterOutput{
		ID:            id,
		Name:          input.Name,
		FilterType:    input.FilterType,
		SchemaVersion: input.SchemaVersion,
		Filter:        input.Filter,
		CreatedAt:     time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (m *mockRecordFilterUsecase) Delete(ctx context.Context, userID int, id string) error {
	m.deleteCalled = true
	m.deleteUserID = userID
	m.deleteID = id
	return nil
}

func TestRecordFilterHandlerList(t *testing.T) {
	e := echo.New()
	uc := &mockRecordFilterUsecase{}
	h := NewRecordFilterHandler(uc)
	req := httptest.NewRequest(http.MethodGet, "/internal/me/record-filters?filter_type=standard", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 10})

	err := h.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, uc.listCalled)
	// 認証ユーザーの ID をそのまま usecase へ渡すことを確かめる。
	assert.Equal(t, 10, uc.listUserID)
	assert.Equal(t, "standard", uc.listFilterType)

	var response map[string][]map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response["filters"], 1)
	assert.Equal(t, "高難度", response["filters"][0]["name"])
	assert.Equal(t, "standard", response["filters"][0]["filter_type"])
	assert.Equal(t, float64(3), response["filters"][0]["schema_version"])
}

func TestRecordFilterHandlerCreateRejectsUnknownTopLevelKey(t *testing.T) {
	e := echo.New()
	uc := &mockRecordFilterUsecase{}
	h := NewRecordFilterHandler(uc)
	body := `{"name":"条件","filter_type":"standard","schema_version":3,"filter":{"title":""},"unknown":1}`
	req := httptest.NewRequest(http.MethodPost, "/internal/me/record-filters", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 10})

	err := h.Create(c)
	require.Error(t, err)
	apiErr := &apierror.APIError{}
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
	assert.False(t, uc.createCalled)
}

func TestRecordFilterHandlerCreateSuccess(t *testing.T) {
	// Given: 認証ユーザーによる作成リクエスト。
	e := echo.New()
	uc := &mockRecordFilterUsecase{}
	h := NewRecordFilterHandler(uc)
	body := `{"name":"高難度FC狙い","filter_type":"standard","schema_version":3,"filter":{"title":"","difficulties":["MASTER","ULTIMA"]}}`
	req := httptest.NewRequest(http.MethodPost, "/internal/me/record-filters", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 42})

	// When: 作成ハンドラーを呼ぶ。
	err := h.Create(c)

	// Then: usecase へ渡る値を具体値で照合し、レスポンス全体を固定値で比較する。
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	require.True(t, uc.createCalled)
	assert.Equal(t, 42, uc.createUserID)
	require.NotNil(t, uc.createInput)
	assert.Equal(t, "高難度FC狙い", uc.createInput.Name)
	assert.Equal(t, "standard", uc.createInput.FilterType)
	assert.Equal(t, 3, uc.createInput.SchemaVersion)
	assert.JSONEq(t, `{"title":"","difficulties":["MASTER","ULTIMA"]}`, string(uc.createInput.Filter))
	assert.JSONEq(t, `{"id":"22222222-2222-2222-2222-222222222222","name":"高難度FC狙い","filter_type":"standard","schema_version":3,"filter":{"title":"","difficulties":["MASTER","ULTIMA"]},"created_at":"2026-06-15T12:00:00Z","updated_at":"2026-06-15T12:00:00Z"}`, rec.Body.String())
}

func TestRecordFilterHandlerUpdateSuccess(t *testing.T) {
	// Given: パス ID と更新内容を指定したリクエスト。
	e := echo.New()
	uc := &mockRecordFilterUsecase{}
	h := NewRecordFilterHandler(uc)
	const filterID = "33333333-3333-3333-3333-333333333333"
	body := `{"name":"WORLD'S ENDまとめ","filter_type":"worldsend","schema_version":5,"filter":{"title":"更新後"}}`
	req := httptest.NewRequest(http.MethodPut, "/internal/me/record-filters/"+filterID, bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/internal/me/record-filters/:id")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: filterID}})
	c.Set("userEntity", &entity.User{ID: 42})

	// When: 更新ハンドラーを呼ぶ。
	err := h.Update(c)

	// Then: パス ID を含む usecase 引数とレスポンス全体を固定値で比較する。
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	require.True(t, uc.updateCalled)
	assert.Equal(t, 42, uc.updateUserID)
	assert.Equal(t, filterID, uc.updateID)
	require.NotNil(t, uc.updateInput)
	assert.Equal(t, "WORLD'S ENDまとめ", uc.updateInput.Name)
	assert.Equal(t, "worldsend", uc.updateInput.FilterType)
	assert.Equal(t, 5, uc.updateInput.SchemaVersion)
	assert.JSONEq(t, `{"title":"更新後"}`, string(uc.updateInput.Filter))
	assert.JSONEq(t, `{"id":"33333333-3333-3333-3333-333333333333","name":"WORLD'S ENDまとめ","filter_type":"worldsend","schema_version":5,"filter":{"title":"更新後"},"created_at":"2026-06-15T12:00:00Z","updated_at":"2026-06-16T12:00:00Z"}`, rec.Body.String())
}

func TestRecordFilterHandlerDeleteSuccess(t *testing.T) {
	// Given: パス ID を指定した削除リクエスト。
	e := echo.New()
	uc := &mockRecordFilterUsecase{}
	h := NewRecordFilterHandler(uc)
	const filterID = "44444444-4444-4444-4444-444444444444"
	req := httptest.NewRequest(http.MethodDelete, "/internal/me/record-filters/"+filterID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/internal/me/record-filters/:id")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: filterID}})
	c.Set("userEntity", &entity.User{ID: 42})

	// When: 削除ハンドラーを呼ぶ。
	err := h.Delete(c)

	// Then: usecase 引数を具体値で照合し、204 と空本文を確かめる。
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, 0, rec.Body.Len())
	require.True(t, uc.deleteCalled)
	assert.Equal(t, 42, uc.deleteUserID)
	assert.Equal(t, filterID, uc.deleteID)
}

func TestRecordFilterHandlerUnauthenticated(t *testing.T) {
	tests := []struct {
		name string
		// Given: 呼び出す操作
		call func(*RecordFilterHandler, *echo.Context) error
	}{
		{name: "一覧", call: (*RecordFilterHandler).List},
		{name: "作成", call: (*RecordFilterHandler).Create},
		{name: "更新", call: (*RecordFilterHandler).Update},
		{name: "削除", call: (*RecordFilterHandler).Delete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 認証コンテキストにユーザーを設定しない。認証以外で拒否されないよう、本文とパスIDはどの操作でも有効な値にする。
			e := echo.New()
			uc := &mockRecordFilterUsecase{}
			h := NewRecordFilterHandler(uc)
			req := httptest.NewRequest(http.MethodPost, "/internal/me/record-filters?filter_type=standard", bytes.NewBufferString(`{"name":"高難度FC狙い","filter_type":"standard","schema_version":3,"filter":{"title":""}}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "id", Value: "33333333-3333-3333-3333-333333333333"}})

			// When
			err := tt.call(h, c)

			// Then: 401 を返し、usecase を呼ばない。
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, apierror.CodeUnauthorized, apiErr.Code)
			assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
			assert.False(t, uc.listCalled || uc.createCalled || uc.updateCalled || uc.deleteCalled, "usecase が呼ばれてはいけない")
		})
	}
}
