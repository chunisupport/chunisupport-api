package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

// CreateTemporaryPlayerDataInput は一時登録入力です。
type CreateTemporaryPlayerDataInput struct {
	IPAddress string
	Payload   *PlayerDataPayload
	BodyHash  string
}

// CreateTemporaryPlayerDataOutput は一時登録結果です。
type CreateTemporaryPlayerDataOutput struct {
	UploadToken string
	ExpiresAt   time.Time
}

// CommitTemporaryPlayerDataInput は確定保存入力です。
type CommitTemporaryPlayerDataInput struct {
	User        *entity.User
	UploadToken string
}

// TemporaryPlayerDataUsecase は一時プレイヤーデータの登録・確定保存ユースケースです。
type TemporaryPlayerDataUsecase interface {
	Create(ctx context.Context, input CreateTemporaryPlayerDataInput) (*CreateTemporaryPlayerDataOutput, error)
	Commit(ctx context.Context, input CommitTemporaryPlayerDataInput) (*api_internal.PlayerDataResult, error)
}
