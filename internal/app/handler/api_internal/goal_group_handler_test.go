package api_internal

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockGoalGroupUsecase struct {
	createdName   string
	updateCalled  bool
	updatedID     uint32
	updatedName   string
	deleteCalled  bool
	deletedID     uint32
	reorderCalled bool
	reordered     []uint32
}

func (m *mockGoalGroupUsecase) List(ctx context.Context, userID int) ([]*usecase.GoalGroupOutput, error) {
	return []*usecase.GoalGroupOutput{}, nil
}

func (m *mockGoalGroupUsecase) Create(ctx context.Context, userID int, name string) (*usecase.GoalGroupOutput, error) {
	m.createdName = name
	return &usecase.GoalGroupOutput{ID: 1, Name: name, SortOrder: 1}, nil
}

func (m *mockGoalGroupUsecase) Update(ctx context.Context, userID int, id uint32, name string) (*usecase.GoalGroupOutput, error) {
	m.updateCalled = true
	m.updatedID = id
	m.updatedName = name
	return &usecase.GoalGroupOutput{ID: id, Name: name, SortOrder: 1}, nil
}

func (m *mockGoalGroupUsecase) Delete(ctx context.Context, userID int, id uint32) error {
	m.deleteCalled = true
	m.deletedID = id
	return nil
}

func (m *mockGoalGroupUsecase) Reorder(ctx context.Context, userID int, orderedGroupIDs []uint32) error {
	m.reorderCalled = true
	// slices.Clone は nil と空配列の区別を保ったまま複写する。
	m.reordered = slices.Clone(orderedGroupIDs)
	return nil
}

func TestGoalGroupHandler_Create(t *testing.T) {
	// Given
	e := echo.New()
	e.Validator = &goalTestValidator{validator: validator.New()}
	mock := &mockGoalGroupUsecase{}
	handler := NewGoalGroupHandler(mock)
	request := httptest.NewRequest(http.MethodPost, "/internal/me/goal-groups", bytes.NewBufferString(`{"name":"攻略中"}`))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.Set("userEntity", &entity.User{ID: 1})

	// When
	err := handler.Create(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "攻略中", mock.createdName)
}

func TestGoalGroupHandler_Reorder(t *testing.T) {
	// Given
	e := echo.New()
	mock := &mockGoalGroupUsecase{}
	handler := NewGoalGroupHandler(mock)
	request := httptest.NewRequest(http.MethodPut, "/internal/me/goal-groups/order", bytes.NewBufferString(`{"group_ids":[3,1,2]}`))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.Set("userEntity", &entity.User{ID: 1})

	// When
	err := handler.Reorder(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, []uint32{3, 1, 2}, mock.reordered)
}

func TestGoalGroupHandlerReorderRejectsMissingAndNullGroupIDs(t *testing.T) {
	// Given
	tests := []struct {
		name string
		// Given: リクエストボディ
		body string
	}{
		{
			name: "group_idsを未指定のとき拒否する",
			body: `{ }`,
		},
		{
			name: "group_idsがnullのとき拒否する",
			body: `{"group_ids":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			mock := &mockGoalGroupUsecase{}
			handler := NewGoalGroupHandler(mock)
			request := httptest.NewRequest(http.MethodPut, "/internal/me/goal-groups/order", bytes.NewBufferString(tt.body))
			request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			recorder := httptest.NewRecorder()
			c := e.NewContext(request, recorder)
			c.Set("userEntity", &entity.User{ID: 1})

			// When
			err := handler.Reorder(c)

			// Then: 形式不正として拒否し、usecase を呼ばない。
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
			assert.False(t, mock.reorderCalled)
		})
	}
}

func TestGoalGroupHandlerReorderAcceptsEmptyGroupIDs(t *testing.T) {
	// Given: 空配列は未指定・null とは区別され、usecase の順序検証へ渡す。
	e := echo.New()
	mock := &mockGoalGroupUsecase{}
	handler := NewGoalGroupHandler(mock)
	request := httptest.NewRequest(http.MethodPut, "/internal/me/goal-groups/order", bytes.NewBufferString(`{"group_ids":[]}`))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.Set("userEntity", &entity.User{ID: 1})

	// When
	err := handler.Reorder(c)

	// Then: 受理し、空のまま（nil ではなく）usecase へ渡す。
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, recorder.Code)
	require.True(t, mock.reorderCalled)
	assert.NotNil(t, mock.reordered)
	assert.Empty(t, mock.reordered)
}

func TestGoalGroupHandlerUpdatePathIDBoundaries(t *testing.T) {
	// Given: 有効な更新ボディを用意し、パス ID の解釈だけを変えて検証する。
	const validBody = `{"name":"攻略中"}`
	tests := []struct {
		name string
		// Given: パスパラメータの ID 文字列
		pathID string
		// Then
		wantBadRequest bool
		expectedID     uint32
	}{
		{
			name:           "負の値は拒否しusecaseを呼ばない",
			pathID:         "-1",
			wantBadRequest: true,
		},
		{
			name:           "32ビットの範囲を超える値は拒否しusecaseを呼ばない",
			pathID:         "4294967296",
			wantBadRequest: true,
		},
		{
			name:           "範囲の最大値はそのままusecaseへ渡す",
			pathID:         "4294967295",
			wantBadRequest: false,
			expectedID:     4294967295,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.Validator = &goalTestValidator{validator: validator.New()}
			mock := &mockGoalGroupUsecase{}
			handler := NewGoalGroupHandler(mock)
			request := httptest.NewRequest(http.MethodPut, "/internal/me/goal-groups/"+tt.pathID, bytes.NewBufferString(validBody))
			request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			recorder := httptest.NewRecorder()
			c := e.NewContext(request, recorder)
			c.SetPath("/internal/me/goal-groups/:id")
			c.SetPathValues(echo.PathValues{{Name: "id", Value: tt.pathID}})
			c.Set("userEntity", &entity.User{ID: 1})

			// When
			err := handler.Update(c)

			// Then
			if tt.wantBadRequest {
				var apiErr *apierror.APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
				assert.False(t, mock.updateCalled)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, recorder.Code)
			require.True(t, mock.updateCalled)
			assert.Equal(t, tt.expectedID, mock.updatedID)
			assert.Equal(t, "攻略中", mock.updatedName)
		})
	}
}

func TestGoalGroupHandlerDeleteRejectsInvalidPathID(t *testing.T) {
	// Given: 削除も更新と同じ parseGoalGroupID の解釈を通ることを確かめる。
	e := echo.New()
	mock := &mockGoalGroupUsecase{}
	handler := NewGoalGroupHandler(mock)
	request := httptest.NewRequest(http.MethodDelete, "/internal/me/goal-groups/-1", nil)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetPath("/internal/me/goal-groups/:id")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "-1"}})
	c.Set("userEntity", &entity.User{ID: 1})

	// When
	err := handler.Delete(c)

	// Then: 形式不正として拒否し、usecase を呼ばない。
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
	assert.False(t, mock.deleteCalled)
}
