package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	playerdataresult "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
)

// TemporaryPlayerDataPayloadDecoder は一時保存された本文をプレイヤーデータへデコードします。
// 入力JSONの形式はユースケースの外側の層が持つため、外側から注入します。
type TemporaryPlayerDataPayloadDecoder func(stored []byte) (*PlayerDataPayload, error)

// CreateTemporaryPlayerDataInput は一時登録入力です。
// Payload は受信したままの本文で、IPAddress は保持件数を数える送信元の単位です（IPv6 は /64）。
type CreateTemporaryPlayerDataInput struct {
	IPAddress string
	Payload   []byte
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
	Commit(ctx context.Context, input CommitTemporaryPlayerDataInput) (*playerdataresult.Result, error)
}
