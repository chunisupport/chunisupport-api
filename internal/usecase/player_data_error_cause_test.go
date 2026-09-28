package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername/playernametest"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username/usernametest"
	"github.com/stretchr/testify/assert"
)

func TestPlayerDataErrors_原因エラーを保持する(t *testing.T) {
	cause := errors.New("cause")
	tests := []struct {
		name string
		// Given: 原因エラーを保持した独自エラー
		err error
		// Then: 原因エラーを保持してもメッセージは変わらない
		wantMessage string
	}{
		{name: "検証エラー", err: &PlayerDataValidationError{Field: "name", Message: cause.Error(), Err: cause}, wantMessage: "name: cause"},
		{name: "競合エラー", err: &PlayerDataConflictError{Reason: cause.Error(), Err: cause}, wantMessage: "cause"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := errors.Is(tt.err, cause)

			// Then
			assert.True(t, got)
			assert.Equal(t, tt.wantMessage, tt.err.Error())
		})
	}
}

func TestEnsurePlayer_古い取得日時は原因エラーを保持する(t *testing.T) {
	// Given
	collectedAt := time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)
	playerRepo := &stubPlayerRepositoryForPlayerData{foundPlayer: &entity.Player{
		ID: 10, UserID: 1, Name: playernametest.New(t, "登録済み"), Level: 1,
		DataCollectedAt: &collectedAt, CreatedAt: collectedAt, UpdatedAt: collectedAt,
	}}
	uc := &playerDataUsecase{playerRepo: playerRepo, userRepo: new(MockUserRepository)}
	user := &entity.User{ID: 1, Username: usernametest.New(t, "playerdatatest")}

	// When
	_, _, err := uc.ensurePlayer(context.Background(), nil, user, &PlayerDataSummaryInput{
		Name: "登録済み", Level: 1, OfficialRating: 17.25, OfficialOverpower: 12345.67,
	}, collectedAt.Add(-time.Hour))

	// Then
	var conflictErr *PlayerDataConflictError
	assert.ErrorAs(t, err, &conflictErr)
	assert.ErrorIs(t, err, entity.ErrStalePlayerData)
}

func TestTemporaryPlayerDataUsecase_Create_生成失敗は原因エラーを保持する(t *testing.T) {
	// Given: 有効期限が作成日時より後にならない保持期間
	uc := NewTemporaryPlayerDataUsecase(nil, &stubTemporaryPlayerDataRepository{}, &stubPlayerDataUsecase{}, 0)

	// When
	_, err := uc.Create(context.Background(), CreateTemporaryPlayerDataInput{
		IPAddress: "127.0.0.1",
		Payload:   []byte(`{"name":"TEST"}`),
		BodyHash:  "hash",
	})

	// Then
	var validationErr *PlayerDataValidationError
	assert.ErrorAs(t, err, &validationErr)
	assert.ErrorIs(t, err, entity.ErrTemporaryPlayerDataExpiresAtInvalid)
}
