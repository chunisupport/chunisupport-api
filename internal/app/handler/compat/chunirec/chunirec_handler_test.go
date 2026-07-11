package chunirec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

type stubChunirecUserUsecase struct{}

func (stubChunirecUserUsecase) GetUserProfile(context.Context, string, *entity.User) (*internaldto.UserProfileDTO, error) {
	return nil, nil
}
func (stubChunirecUserUsecase) GetUserUpdatedAt(context.Context, string, *entity.User) (*internaldto.UserUpdatedAtDTO, error) {
	return nil, nil
}
func (stubChunirecUserUsecase) GetUserProfileWithRecords(context.Context, string, *entity.User, bool) (*internaldto.UserProfileWithRecordsDTO, error) {
	return &internaldto.UserProfileWithRecordsDTO{}, nil
}
func (stubChunirecUserUsecase) GetUserProfileRatingView(context.Context, string, *entity.User) (*internaldto.UserProfileRatingViewDTO, error) {
	return nil, nil
}
func (stubChunirecUserUsecase) GetUserProfileRecordView(context.Context, string, *entity.User, bool) (*internaldto.UserProfileRecordViewDTO, error) {
	return nil, nil
}
func (stubChunirecUserUsecase) GetUserSongRecord(context.Context, string, *entity.User, string, bool, string) (*internaldto.UserSongRecordDTO, error) {
	return nil, nil
}
func (stubChunirecUserUsecase) GetUserWorldsendSongRecord(context.Context, string, *entity.User, string, bool) (*internaldto.UserWorldsendSongRecordDTO, error) {
	return nil, nil
}
func (stubChunirecUserUsecase) GetAllUsersForAdmin(context.Context, int, int, string) ([]internaldto.AdminUserListResponse, error) {
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
