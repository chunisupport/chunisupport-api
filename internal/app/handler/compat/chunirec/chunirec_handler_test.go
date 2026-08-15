package chunirec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

type stubChunirecUserUsecase struct{}

func (stubChunirecUserUsecase) GetUserProfile(context.Context, string, *entity.User) (*usecase.UserProfileOutput, error) {
	return nil, nil
}
func (stubChunirecUserUsecase) GetUserUpdatedAt(context.Context, string, *entity.User) (*usecase.UserUpdatedAtOutput, error) {
	return nil, nil
}
func (stubChunirecUserUsecase) GetUserProfileWithRecords(context.Context, string, *entity.User, bool) (*usecase.UserProfileWithRecordsOutput, error) {
	return &usecase.UserProfileWithRecordsOutput{}, nil
}
func (stubChunirecUserUsecase) GetUserProfileRatingView(context.Context, string, *entity.User) (*usecase.UserProfileRatingViewOutput, error) {
	return nil, nil
}
func (stubChunirecUserUsecase) GetUserProfileRecordView(context.Context, string, *entity.User, bool) (*usecase.UserProfileRecordViewOutput, error) {
	return nil, nil
}
func (stubChunirecUserUsecase) GetUserSongRecord(context.Context, string, *entity.User, string, bool, string) (*usecase.UserSongRecordOutput, error) {
	return nil, nil
}
func (stubChunirecUserUsecase) GetUserWorldsendSongRecord(context.Context, string, *entity.User, string, bool) (*usecase.UserWorldsendSongRecordOutput, error) {
	return nil, nil
}
func (stubChunirecUserUsecase) GetAllUsersForAdmin(context.Context, int, int, string) ([]usecase.AdminUserOutput, error) {
	return nil, nil
}
func (stubChunirecUserUsecase) DeleteUser(context.Context, *entity.User, string) error { return nil }

var _ usecase.UserUsecase = stubChunirecUserUsecase{}

func TestChunirecHandler_GetUserShow_プレイヤー未連携ではHTTP200とnullを返す(t *testing.T) {
	// Given
	e := echo.New()
	handler := NewChunirecHandler(nil, stubChunirecUserUsecase{}, nil, time.UTC)
	e.GET("/compat/chunirec/2.0/users/show", handler.GetUserShow)
	req := httptest.NewRequest(http.MethodGet, "/compat/chunirec/2.0/users/show?user_name=testuser", nil)
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `null`, rec.Body.String())
}
