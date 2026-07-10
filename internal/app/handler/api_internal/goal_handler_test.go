package api_internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
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
	createCalled  bool
	updateCalled  bool
	reorderCalled bool
	reorderedIDs  []uint32
	createErr     error
	updateErr     error
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
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return &usecase.GoalOutput{ID: id, Title: input.Title, AchievementType: input.AchievementType, AchievementParams: map[string]any{}, Attributes: map[string]any{}}, nil
}

func (m *mockGoalUsecase) Delete(ctx context.Context, userID int, id uint32) error {
	return nil
}

func (m *mockGoalUsecase) Reorder(ctx context.Context, userID int, orderedGoalIDs []uint32) error {
	m.reorderCalled = true
	m.reorderedIDs = append([]uint32(nil), orderedGoalIDs...)
	return nil
}

func TestDecodeStrictJSONReturnsSpecificErrorForMissingContentType(t *testing.T) {
	body := bytes.NewBufferString(`{"title":"t"}`)
	header := http.Header{}

	var out map[string]any
	err := apphandler.DecodeStrictJSON(body, header, &out)
	if err == nil {
		require.Fail(t, "expected error")
	}
	if err.Error() != "content-type header is missing" {
		require.Failf(t, "前提条件失敗", "error = %q, want %q", err.Error(), "content-type header is missing")
	}
}

func TestDecodeStrictJSONReturnsSpecificErrorForInvalidContentType(t *testing.T) {
	body := bytes.NewBufferString(`{"title":"t"}`)
	header := http.Header{}
	header.Set(echo.HeaderContentType, "text/plain")

	var out map[string]any
	err := apphandler.DecodeStrictJSON(body, header, &out)
	if err == nil {
		require.Fail(t, "expected error")
	}
	if err.Error() != "content-type must be application/json" {
		require.Failf(t, "前提条件失敗", "error = %q, want %q", err.Error(), "content-type must be application/json")
	}
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
		Invert: true,
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
	if !in.Invert {
		require.Fail(t, "Invert = false, want true")
	}
}
