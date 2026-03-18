package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/dto"
)

func TestUserService_GetUserUpdatedAt_Success(t *testing.T) {
	now := time.Now()
	un, _ := username.NewUserName("tester")
	user := &entity.User{
		ID:       1,
		Username: un,
		PlayerID: intPointer(1),
	}
	player := &dto.PlayerDTO{
		Name:      "TestPlayer",
		Level:     10,
		UpdatedAt: now,
	}
	service := NewUserService(nil, &stubUserRepository{user: user}, &stubPlayerRecordRepository{}, nil, &stubPlayerService{player: player}, nil, nil, nil)

	result, err := service.GetUserUpdatedAt(context.Background(), "tester", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.UpdatedAt.Equal(now) {
		t.Fatalf("expected updated_at %v, got %v", now, result.UpdatedAt)
	}
}

func TestUserService_GetUserUpdatedAt_PrivateUserBlocked(t *testing.T) {
	un, _ := username.NewUserName("privateuser")
	user := &entity.User{
		ID:        1,
		Username:  un,
		PlayerID:  intPointer(1),
		IsPrivate: true,
	}
	player := &dto.PlayerDTO{
		Name:      "PrivatePlayer",
		Level:     1,
		UpdatedAt: time.Now(),
	}
	service := NewUserService(nil, &stubUserRepository{user: user}, &stubPlayerRecordRepository{}, nil, &stubPlayerService{player: player}, nil, nil, nil)

	_, err := service.GetUserUpdatedAt(context.Background(), "privateuser", nil)
	if !errors.Is(err, ErrUserPrivate) {
		t.Fatalf("expected ErrUserPrivate, got %v", err)
	}
}
