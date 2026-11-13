package api_internal

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
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
	createUserID     int
	createInput      *usecase.GoalInput
	createResult     *usecase.GoalOutput
	updateCalled     bool
	updateUserID     int
	updateInput      *usecase.GoalInput
	updateResult     *usecase.GoalOutput
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
	m.createUserID = userID
	m.createInput = input
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.createResult != nil {
		return m.createResult, nil
	}
	return &usecase.GoalOutput{ID: 1, Title: input.Title, AchievementType: input.AchievementType, AchievementParams: map[string]any{}, Attributes: map[string]any{}}, nil
}

func (m *mockGoalUsecase) Update(ctx context.Context, userID int, id uint32, input *usecase.GoalInput) (*usecase.GoalOutput, error) {
	m.updateCalled = true
	m.updateUserID = userID
	m.updatedID = id
	m.updateInput = input
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if m.updateResult != nil {
		return m.updateResult, nil
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

func TestGoalHandlerCreateSuccess(t *testing.T) {
	// Given: 全入力を含む固定リクエストと、usecase が返す固定の作成結果。
	// 反転フラグは true/false を非対称にして取り違えを検出できるようにする。
	// ゲージは小数を含む 99.6/true で、小数の受け渡しと応答への含め方を確かめる。
	fixedTime := time.Date(2026, time.January, 1, 9, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	groupID := uint32(3)
	e := echo.New()
	e.Validator = &goalTestValidator{validator: validator.New()}
	uc := &mockGoalUsecase{
		createResult: &usecase.GoalOutput{
			ID:                          7,
			GroupID:                     &groupID,
			Title:                       "OP達成率99.8%",
			AchievementType:             "overpower_percent",
			AchievementParams:           map[string]any{"total": 99.8},
			Attributes:                  map[string]any{"diff": float64(4)},
			InvertValue:                 true,
			InvertPercentage:            false,
			GaugeStart:                  99.6,
			ApplyGaugeStartToPercentage: true,
			SortOrder:                   2,
			CreatedAt:                   fixedTime,
		},
	}
	h := NewGoalHandler(uc)
	body := `{"group_id":3,"title":"OP達成率99.8%","achievement_type":"overpower_percent","achievement_params":{"total":99.8},"attributes":{"diff":4},"invert_value":true,"invert_percentage":false,"gauge_start":99.6,"apply_gauge_start_to_percentage":true}`
	req := httptest.NewRequest(http.MethodPost, "/internal/me/goals", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 1})

	// When: 公開ハンドラ配線を通して作成する。
	err := h.Create(c)

	// Then: ステータスと usecase へ渡る全入力を固定値で検証する。
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	require.True(t, uc.createCalled)
	assert.Equal(t, 1, uc.createUserID)
	require.NotNil(t, uc.createInput)
	require.NotNil(t, uc.createInput.GroupID)
	assert.Equal(t, uint32(3), *uc.createInput.GroupID)
	assert.Equal(t, "OP達成率99.8%", uc.createInput.Title)
	assert.Equal(t, "overpower_percent", uc.createInput.AchievementType)
	assert.JSONEq(t, `{"total":99.8}`, string(uc.createInput.AchievementParams))
	assert.JSONEq(t, `{"diff":4}`, string(uc.createInput.Attributes))
	assert.True(t, uc.createInput.InvertValue)
	assert.False(t, uc.createInput.InvertPercentage)
	assert.Equal(t, 99.6, uc.createInput.GaugeStart)
	assert.True(t, uc.createInput.ApplyGaugeStartToPercentage)

	// Then: JSON 応答全体を固定値で検証する。キー名の変更や項目の欠落を検出するため。
	assert.JSONEq(t, `{"id":7,"group_id":3,"title":"OP達成率99.8%","achievement_type":"overpower_percent","achievement_params":{"total":99.8},"attributes":{"diff":4},"invert_value":true,"invert_percentage":false,"gauge_start":99.6,"apply_gauge_start_to_percentage":true,"sort_order":2,"created_at":"2026-01-01T09:00:00+09:00"}`, rec.Body.String())
}

func TestGoalHandlerUpdateSuccess(t *testing.T) {
	// Given: 作成時とは逆向きの反転フラグ (false/true) で独立性を確かめる。
	// group_id・attributes・gauge 系を省略し、未分類・空条件・既定値の受け渡しも固定値で検証する。
	fixedTime := time.Date(2026, time.January, 1, 9, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	e := echo.New()
	e.Validator = &goalTestValidator{validator: validator.New()}
	uc := &mockGoalUsecase{
		updateResult: &usecase.GoalOutput{
			ID:                          5,
			GroupID:                     nil,
			Title:                       "test",
			AchievementType:             "score_count",
			AchievementParams:           map[string]any{"score": float64(1000000), "count": float64(1)},
			Attributes:                  map[string]any{},
			InvertValue:                 false,
			InvertPercentage:            true,
			GaugeStart:                  0,
			ApplyGaugeStartToPercentage: false,
			SortOrder:                   1,
			CreatedAt:                   fixedTime,
		},
	}
	h := NewGoalHandler(uc)
	body := `{"title":"test","achievement_type":"score_count","achievement_params":{"score":1000000,"count":1},"invert_value":false,"invert_percentage":true}`
	req := httptest.NewRequest(http.MethodPut, "/internal/me/goals/5", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/internal/me/goals/:id")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "5"}})
	c.Set("userEntity", &entity.User{ID: 1})

	// When: 公開ハンドラ配線を通して更新する。
	err := h.Update(c)

	// Then: パス ID と usecase へ渡る全入力を固定値で検証する。
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	require.True(t, uc.updateCalled)
	assert.Equal(t, 1, uc.updateUserID)
	assert.Equal(t, uint32(5), uc.updatedID)
	require.NotNil(t, uc.updateInput)
	assert.Nil(t, uc.updateInput.GroupID)
	assert.Equal(t, "test", uc.updateInput.Title)
	assert.Equal(t, "score_count", uc.updateInput.AchievementType)
	assert.JSONEq(t, `{"score":1000000,"count":1}`, string(uc.updateInput.AchievementParams))
	assert.JSONEq(t, `{}`, string(uc.updateInput.Attributes))
	assert.False(t, uc.updateInput.InvertValue)
	assert.True(t, uc.updateInput.InvertPercentage)
	assert.Equal(t, float64(0), uc.updateInput.GaugeStart)
	assert.False(t, uc.updateInput.ApplyGaugeStartToPercentage)

	// Then: JSON 応答全体を固定値で検証する。invert の独立と gauge 既定値の保持を含む。
	assert.JSONEq(t, `{"id":5,"group_id":null,"title":"test","achievement_type":"score_count","achievement_params":{"score":1000000,"count":1},"attributes":{},"invert_value":false,"invert_percentage":true,"gauge_start":0,"apply_gauge_start_to_percentage":false,"sort_order":1,"created_at":"2026-01-01T09:00:00+09:00"}`, rec.Body.String())
}
