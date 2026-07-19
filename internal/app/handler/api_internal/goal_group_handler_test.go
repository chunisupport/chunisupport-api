package api_internal

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockGoalGroupUsecase struct {
	createdName string
	reordered   []uint32
}

func (m *mockGoalGroupUsecase) List(ctx context.Context, userID int) ([]*usecase.GoalGroupOutput, error) {
	return []*usecase.GoalGroupOutput{}, nil
}

func (m *mockGoalGroupUsecase) Create(ctx context.Context, userID int, name string) (*usecase.GoalGroupOutput, error) {
	m.createdName = name
	return &usecase.GoalGroupOutput{ID: 1, Name: name, SortOrder: 1}, nil
}

func (m *mockGoalGroupUsecase) Update(ctx context.Context, userID int, id uint32, name string) (*usecase.GoalGroupOutput, error) {
	return &usecase.GoalGroupOutput{ID: id, Name: name, SortOrder: 1}, nil
}

func (m *mockGoalGroupUsecase) Delete(ctx context.Context, userID int, id uint32) error {
	return nil
}

func (m *mockGoalGroupUsecase) Reorder(ctx context.Context, userID int, orderedGroupIDs []uint32) error {
	m.reordered = append([]uint32(nil), orderedGroupIDs...)
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
