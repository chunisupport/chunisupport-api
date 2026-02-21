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
	"github.com/labstack/echo/v4"
)

type goalTestValidator struct {
	validator *validator.Validate
}

func (tv *goalTestValidator) Validate(i any) error {
	return tv.validator.Struct(i)
}

type mockGoalUsecase struct {
	createCalled bool
	updateCalled bool
	createErr    error
	updateErr    error
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

func TestDecodeStrictJSONReturnsSpecificErrorForMissingContentType(t *testing.T) {
	body := bytes.NewBufferString(`{"title":"t"}`)
	header := http.Header{}

	var out map[string]any
	err := apphandler.DecodeStrictJSON(body, header, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "content-type header is missing" {
		t.Fatalf("error = %q, want %q", err.Error(), "content-type header is missing")
	}
}

func TestDecodeStrictJSONReturnsSpecificErrorForInvalidContentType(t *testing.T) {
	body := bytes.NewBufferString(`{"title":"t"}`)
	header := http.Header{}
	header.Set(echo.HeaderContentType, "text/plain")

	var out map[string]any
	err := apphandler.DecodeStrictJSON(body, header, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "content-type must be application/json" {
		t.Fatalf("error = %q, want %q", err.Error(), "content-type must be application/json")
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
		t.Fatal("expected error")
	}
	apiErr := &apierror.APIError{}
	if !errors.As(err, &apiErr) {
		t.Fatalf("err type = %T, want *apierror.APIError", err)
	}
	if apiErr.Code != apierror.CodeBadRequest {
		t.Fatalf("api error code = %s, want %s", apiErr.Code, apierror.CodeBadRequest)
	}
	if uc.createCalled {
		t.Fatal("usecase Create should not be called")
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
		t.Fatal("expected error")
	}
	apiErr := &apierror.APIError{}
	if !errors.As(err, &apiErr) {
		t.Fatalf("err type = %T, want *apierror.APIError", err)
	}
	if apiErr.Code != apierror.CodeBadRequest {
		t.Fatalf("api error code = %s, want %s", apiErr.Code, apierror.CodeBadRequest)
	}
	if uc.createCalled {
		t.Fatal("usecase Create should not be called")
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
		t.Fatal("expected error")
	}
	apiErr := &apierror.APIError{}
	if !errors.As(err, &apiErr) {
		t.Fatalf("err type = %T, want *apierror.APIError", err)
	}
	if apiErr.Code != apierror.CodeBadRequest {
		t.Fatalf("api error code = %s, want %s", apiErr.Code, apierror.CodeBadRequest)
	}
	if uc.createCalled {
		t.Fatal("usecase Create should not be called")
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
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Set("userEntity", &entity.User{ID: 1})

	err := h.Update(c)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr := &apierror.APIError{}
	if !errors.As(err, &apiErr) {
		t.Fatalf("err type = %T, want *apierror.APIError", err)
	}
	if apiErr.Code != apierror.CodeBadRequest {
		t.Fatalf("api error code = %s, want %s", apiErr.Code, apierror.CodeBadRequest)
	}
	if uc.updateCalled {
		t.Fatal("usecase Update should not be called")
	}
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
		t.Fatalf("toGoalInput returned error: %v", err)
	}
	if in.Title != req.Title {
		t.Fatalf("Title = %s, want %s", in.Title, req.Title)
	}
	if in.AchievementType != req.AchievementType {
		t.Fatalf("AchievementType = %s, want %s", in.AchievementType, req.AchievementType)
	}
	var gotParams map[string]any
	if err := json.Unmarshal(in.AchievementParams, &gotParams); err != nil {
		t.Fatalf("unmarshal AchievementParams: %v", err)
	}
	if gotParams["score"].(float64) != 1000000 || gotParams["count"].(float64) != 1 {
		t.Fatalf("AchievementParams = %#v, want score=1000000,count=1", gotParams)
	}
	var gotAttrs map[string]any
	if err := json.Unmarshal(in.Attributes, &gotAttrs); err != nil {
		t.Fatalf("unmarshal Attributes: %v", err)
	}
	if gotAttrs["diff"].(float64) != 4 {
		t.Fatalf("Attributes = %#v, want diff=4", gotAttrs)
	}
	if !in.Invert {
		t.Fatal("Invert = false, want true")
	}
}
