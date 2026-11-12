package api_internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type goalTestValidator struct {
	validator *validator.Validate
}

func (tv *goalTestValidator) Validate(i any) error {
	return tv.validator.Struct(i)
}

type mockGoalUsecase struct {
	createCalled     bool
	updateCalled     bool
	updatedID        uint32
	deleteCalled     bool
	deletedID        uint32
	reorderCalled    bool
	reorderedIDs     []uint32
	reorderedGroupID *uint32
	createErr        error
	updateErr        error
}

func (m *mockGoalUsecase) List(ctx context.Context, userID int) ([]*usecase.GoalOutput, error) {
	return nil, nil
}

func (m *mockGoalUsecase) Create(ctx context.Context, userID int, input *usecase.GoalInput) (*usecase.GoalOutput, error) {
	m.createCalled = true
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &usecase.GoalOutput{ID: 1, Title: input.Title, AchievementType: input.AchievementType, AchievementParams: map[string]any{}, Attributes: map[string]any{}}, nil
}

func (m *mockGoalUsecase) Update(ctx context.Context, userID int, id uint32, input *usecase.GoalInput) (*usecase.GoalOutput, error) {
	m.updateCalled = true
	m.updatedID = id
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return &usecase.GoalOutput{ID: id, Title: input.Title, AchievementType: input.AchievementType, AchievementParams: map[string]any{}, Attributes: map[string]any{}}, nil
}

func (m *mockGoalUsecase) Delete(ctx context.Context, userID int, id uint32) error {
	m.deleteCalled = true
	m.deletedID = id
	return nil
}

func (m *mockGoalUsecase) Reorder(ctx context.Context, userID int, groupID *uint32, orderedGoalIDs []uint32) error {
	m.reorderCalled = true
	m.reorderedGroupID = groupID
	// slices.Clone は nil と空配列の区別を保ったまま複写する。
	m.reorderedIDs = slices.Clone(orderedGoalIDs)
	return nil
}

func TestGoalHandlerCreateRejectsMissingContentType(t *testing.T) {
	e := echo.New()
	e.Validator = &goalTestValidator{validator: validator.New()}
	uc := &mockGoalUsecase{}
	h := NewGoalHandler(uc)

	body := `{"title":"t","achievement_type":"score_count","achievement_params":{"score":1000000,"count":1},"attributes":{}}`
	req := httptest.NewRequest(http.MethodPost, "/internal/me/goals", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 1})

	err := h.Create(c)
	if err == nil {
		require.Fail(t, "expected error")
	}
	apiErr := &apierror.APIError{}
	if !errors.As(err, &apiErr) {
		require.Failf(t, "前提条件失敗", "err type = %T, want *apierror.APIError", err)
	}
	if apiErr.Code != apierror.CodeBadRequest {
		require.Failf(t, "前提条件失敗", "api error code = %s, want %s", apiErr.Code, apierror.CodeBadRequest)
	}
	if uc.createCalled {
		require.Fail(t, "usecase Create should not be called")
	}
}

func TestGoalHandlerCreateRejectsNonJSONContentType(t *testing.T) {
	e := echo.New()
	e.Validator = &goalTestValidator{validator: validator.New()}
	uc := &mockGoalUsecase{}
	h := NewGoalHandler(uc)

	body := `{"title":"t","achievement_type":"score_count","achievement_params":{"score":1000000,"count":1},"attributes":{}}`
	req := httptest.NewRequest(http.MethodPost, "/internal/me/goals", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, "text/plain")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 1})

	err := h.Create(c)
	if err == nil {
		require.Fail(t, "expected error")
	}
	apiErr := &apierror.APIError{}
	if !errors.As(err, &apiErr) {
		require.Failf(t, "前提条件失敗", "err type = %T, want *apierror.APIError", err)
	}
	if apiErr.Code != apierror.CodeBadRequest {
		require.Failf(t, "前提条件失敗", "api error code = %s, want %s", apiErr.Code, apierror.CodeBadRequest)
	}
	if uc.createCalled {
		require.Fail(t, "usecase Create should not be called")
	}
}

func TestGoalHandlerCreateRejectsUnknownTopLevelKey(t *testing.T) {
	e := echo.New()
	e.Validator = &goalTestValidator{validator: validator.New()}
	uc := &mockGoalUsecase{}
	h := NewGoalHandler(uc)

	body := `{"title":"t","achievement_type":"score_count","achievement_params":{"score":1000000,"count":1},"attributes":{},"unknown":1}`
	req := httptest.NewRequest(http.MethodPost, "/internal/me/goals", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 1})

	err := h.Create(c)
	if err == nil {
		require.Fail(t, "expected error")
	}
	apiErr := &apierror.APIError{}
	if !errors.As(err, &apiErr) {
		require.Failf(t, "前提条件失敗", "err type = %T, want *apierror.APIError", err)
	}
	if apiErr.Code != apierror.CodeBadRequest {
		require.Failf(t, "前提条件失敗", "api error code = %s, want %s", apiErr.Code, apierror.CodeBadRequest)
	}
	if uc.createCalled {
		require.Fail(t, "usecase Create should not be called")
	}
}

func TestGoalHandlerUpdateRejectsUnknownTopLevelKey(t *testing.T) {
	e := echo.New()
	e.Validator = &goalTestValidator{validator: validator.New()}
	uc := &mockGoalUsecase{}
	h := NewGoalHandler(uc)

	body := `{"title":"t","achievement_type":"score_count","achievement_params":{"score":1000000,"count":1},"attributes":{},"unknown":1}`
	req := httptest.NewRequest(http.MethodPut, "/internal/me/goals/1", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/internal/me/goals/:id")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
	c.Set("userEntity", &entity.User{ID: 1})

	err := h.Update(c)
	if err == nil {
		require.Fail(t, "expected error")
	}
	apiErr := &apierror.APIError{}
	if !errors.As(err, &apiErr) {
		require.Failf(t, "前提条件失敗", "err type = %T, want *apierror.APIError", err)
	}
	if apiErr.Code != apierror.CodeBadRequest {
		require.Failf(t, "前提条件失敗", "api error code = %s, want %s", apiErr.Code, apierror.CodeBadRequest)
	}
	if uc.updateCalled {
		require.Fail(t, "usecase Update should not be called")
	}
}

func TestGoalHandlerReorder(t *testing.T) {
	// Given
	e := echo.New()
	e.Validator = &goalTestValidator{validator: validator.New()}
	uc := &mockGoalUsecase{}
	h := NewGoalHandler(uc)
	req := httptest.NewRequest(http.MethodPut, "/internal/me/goals/order", bytes.NewBufferString(`{"goal_ids":[30,10,20]}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 1})

	// When
	err := h.Reorder(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, []uint32{30, 10, 20}, uc.reorderedIDs)
}

func TestGoalHandlerReorderRejectsMissingGoalIDs(t *testing.T) {
	// Given
	e := echo.New()
	uc := &mockGoalUsecase{}
	h := NewGoalHandler(uc)
	req := httptest.NewRequest(http.MethodPut, "/internal/me/goals/order", bytes.NewBufferString(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 1})

	// When
	err := h.Reorder(c)

	// Then
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
	assert.False(t, uc.reorderCalled)
}

func TestGoalHandlerReorderRejectsNullGoalIDs(t *testing.T) {
	// Given: goal_ids を明示的に null で送る（未指定の場合は既存テストで確認済み）。
	e := echo.New()
	uc := &mockGoalUsecase{}
	h := NewGoalHandler(uc)
	req := httptest.NewRequest(http.MethodPut, "/internal/me/goals/order", bytes.NewBufferString(`{"goal_ids":null}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 1})

	// When
	err := h.Reorder(c)

	// Then: 形式不正として拒否し、usecase を呼ばない。
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
	assert.False(t, uc.reorderCalled)
}

func TestGoalHandlerReorderAcceptsEmptyGoalIDs(t *testing.T) {
	// Given: 空配列は未指定・null とは区別され、usecase の順序検証へ渡す。
	e := echo.New()
	uc := &mockGoalUsecase{}
	h := NewGoalHandler(uc)
	req := httptest.NewRequest(http.MethodPut, "/internal/me/goals/order", bytes.NewBufferString(`{"goal_ids":[]}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 1})

	// When
	err := h.Reorder(c)

	// Then: 受理し、空のまま（nil ではなく）usecase へ渡す。
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	require.True(t, uc.reorderCalled)
	assert.NotNil(t, uc.reorderedIDs)
	assert.Empty(t, uc.reorderedIDs)
	assert.Nil(t, uc.reorderedGroupID)
}

func TestGoalHandlerReorderPassesGroupID(t *testing.T) {
	// Given
	groupID := uint32(3)
	tests := []struct {
		name string
		// Given: リクエストボディ
		body string
		// Then: usecase へ渡されるグループ ID（nil は未分類）
		expectedGroupID *uint32
	}{
		{
			name:            "グループIDを指定したときその値を渡す",
			body:            `{"group_id":3,"goal_ids":[5]}`,
			expectedGroupID: &groupID,
		},
		{
			name:            "グループIDを省略したときnilを渡す（未分類）",
			body:            `{"goal_ids":[5]}`,
			expectedGroupID: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			uc := &mockGoalUsecase{}
			h := NewGoalHandler(uc)
			req := httptest.NewRequest(http.MethodPut, "/internal/me/goals/order", bytes.NewBufferString(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("userEntity", &entity.User{ID: 1})

			// When
			err := h.Reorder(c)

			// Then
			require.NoError(t, err)
			assert.Equal(t, http.StatusNoContent, rec.Code)
			require.True(t, uc.reorderCalled)
			assert.Equal(t, tt.expectedGroupID, uc.reorderedGroupID)
			assert.Equal(t, []uint32{5}, uc.reorderedIDs)
		})
	}
}

func TestGoalHandlerUpdatePathIDBoundaries(t *testing.T) {
	// Given: 有効な更新ボディを用意し、パス ID の解釈だけを変えて検証する。
	const validBody = `{"title":"t","achievement_type":"score_count","achievement_params":{"score":1000000,"count":1},"attributes":{}}`
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
			uc := &mockGoalUsecase{}
			h := NewGoalHandler(uc)
			req := httptest.NewRequest(http.MethodPut, "/internal/me/goals/"+tt.pathID, bytes.NewBufferString(validBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/internal/me/goals/:id")
			c.SetPathValues(echo.PathValues{{Name: "id", Value: tt.pathID}})
			c.Set("userEntity", &entity.User{ID: 1})

			// When
			err := h.Update(c)

			// Then
			if tt.wantBadRequest {
				var apiErr *apierror.APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
				assert.False(t, uc.updateCalled)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
			require.True(t, uc.updateCalled)
			assert.Equal(t, tt.expectedID, uc.updatedID)
		})
	}
}

func TestGoalHandlerDeleteRejectsInvalidPathID(t *testing.T) {
	// Given: 削除も更新と同じ32ビット符号なし整数の解釈を通ることを確かめる。
	e := echo.New()
	uc := &mockGoalUsecase{}
	h := NewGoalHandler(uc)
	req := httptest.NewRequest(http.MethodDelete, "/internal/me/goals/-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/internal/me/goals/:id")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "-1"}})
	c.Set("userEntity", &entity.User{ID: 1})

	// When
	err := h.Delete(c)

	// Then: 形式不正として拒否し、usecase を呼ばない。
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
	assert.False(t, uc.deleteCalled)
}

func TestToGoalInput(t *testing.T) {
	req := &internaldto.GoalRequest{
		Title:           "test",
		AchievementType: "score_count",
		AchievementParams: map[string]any{
			"score": 1000000,
			"count": 1,
		},
		Attributes: map[string]any{
			"diff": 4,
		},
		InvertValue:      true,
		InvertPercentage: false,
	}

	in, err := toGoalInput(req)
	if err != nil {
		require.Failf(t, "前提条件失敗", "toGoalInput returned error: %v", err)
	}
	if in.Title != req.Title {
		require.Failf(t, "前提条件失敗", "Title = %s, want %s", in.Title, req.Title)
	}
	if in.AchievementType != req.AchievementType {
		require.Failf(t, "前提条件失敗", "AchievementType = %s, want %s", in.AchievementType, req.AchievementType)
	}
	var gotParams map[string]any
	if err := json.Unmarshal(in.AchievementParams, &gotParams); err != nil {
		require.Failf(t, "前提条件失敗", "unmarshal AchievementParams: %v", err)
	}
	if gotParams["score"].(float64) != 1000000 || gotParams["count"].(float64) != 1 {
		require.Failf(t, "前提条件失敗", "AchievementParams = %#v, want score=1000000,count=1", gotParams)
	}
	var gotAttrs map[string]any
	if err := json.Unmarshal(in.Attributes, &gotAttrs); err != nil {
		require.Failf(t, "前提条件失敗", "unmarshal Attributes: %v", err)
	}
	if gotAttrs["diff"].(float64) != 4 {
		require.Failf(t, "前提条件失敗", "Attributes = %#v, want diff=4", gotAttrs)
	}
	assert.True(t, in.InvertValue)
	assert.False(t, in.InvertPercentage)
}

func TestToGoalResponseKeepsInvertFlagsIndependent(t *testing.T) {
	// Given
	goal := &usecase.GoalOutput{
		InvertValue:      false,
		InvertPercentage: true,
	}

	// When
	response := toGoalResponse(goal)

	// Then
	assert.False(t, response.InvertValue)
	assert.True(t, response.InvertPercentage)

	encoded, err := json.Marshal(response)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(encoded, &body))
	assert.Equal(t, false, body["invert_value"])
	assert.Equal(t, true, body["invert_percentage"])
	assert.NotContains(t, body, "invert")
}

func TestGoalGaugeStartMapping(t *testing.T) {
	t.Run("リクエストのスタート位置と達成率への適用指定をユースケース入力へ渡す", func(t *testing.T) {
		// Given
		req := &internaldto.GoalRequest{
			Title:                       "OP達成率99.8%",
			AchievementType:             "overpower_percent",
			AchievementParams:           map[string]any{"total": 99.8},
			GaugeStart:                  99.6,
			ApplyGaugeStartToPercentage: true,
		}

		// When
		in, err := toGoalInput(req)

		// Then
		require.NoError(t, err)
		assert.Equal(t, 99.6, in.GaugeStart)
		assert.True(t, in.ApplyGaugeStartToPercentage)
	})

	t.Run("レスポンスにスタート位置と達成率への適用指定を含める", func(t *testing.T) {
		// Given
		goal := &usecase.GoalOutput{GaugeStart: 99.6, ApplyGaugeStartToPercentage: true}

		// When
		encoded, err := json.Marshal(toGoalResponse(goal))

		// Then
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(encoded, &body))
		assert.Equal(t, 99.6, body["gauge_start"])
		assert.Equal(t, true, body["apply_gauge_start_to_percentage"])
	})
}
