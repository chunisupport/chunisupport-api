package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	playerdataresult "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
)

type temporaryPlayerDataUsecase struct {
	exec              domainrepo.Executor
	repo              domainrepo.TemporaryPlayerDataRepository
	playerDataUsecase PlayerDataUsecase
	ttl               time.Duration
}

// NewTemporaryPlayerDataUsecase は TemporaryPlayerDataUsecase の実装を返します。
func NewTemporaryPlayerDataUsecase(exec domainrepo.Executor, repo domainrepo.TemporaryPlayerDataRepository, playerDataUsecase PlayerDataUsecase, ttl time.Duration) TemporaryPlayerDataUsecase {
	return &temporaryPlayerDataUsecase{
		exec:              exec,
		repo:              repo,
		playerDataUsecase: playerDataUsecase,
		ttl:               ttl,
	}
}

func (u *temporaryPlayerDataUsecase) Create(ctx context.Context, input CreateTemporaryPlayerDataInput) (*CreateTemporaryPlayerDataOutput, error) {
	if len(input.Payload) == 0 {
		return nil, &PlayerDataValidationError{Field: "payload", Message: "is required"}
	}
	if input.IPAddress == "" {
		return nil, &PlayerDataValidationError{Field: "ip_address", Message: "is required"}
	}

	token := uuid.NewV4().String()
	now := time.Now().UTC()
	entry, err := entity.NewTemporaryPlayerData(
		token,
		input.IPAddress,
		input.Payload,
		input.BodyHash,
		now,
		now.Add(u.ttl),
	)
	if err != nil {
		return nil, &PlayerDataValidationError{Field: "payload", Message: err.Error(), Err: err}
	}

	if err := u.repo.Create(ctx, u.exec, entry); err != nil {
		switch {
		case errors.Is(err, domainrepo.ErrTemporaryPlayerDataPerIPLimitExceeded):
			return nil, ErrTempDataPerIPLimitExceeded
		case errors.Is(err, domainrepo.ErrTemporaryPlayerDataTotalSizeLimitExceeded):
			return nil, ErrTempDataCapacityExceeded
		default:
			return nil, fmt.Errorf("temporary player data create failed: %w", err)
		}
	}

	return &CreateTemporaryPlayerDataOutput{
		UploadToken: token,
		ExpiresAt:   entry.ExpiresAt,
	}, nil
}

// Commit は一時データを確定保存します。
// 一時的なDB障害やクライアント切断で再アップロードを強いないよう、処理中は予約だけ行い、
// 登録に成功した場合に削除し、失敗した場合は予約を解除して同じトークンで再試行できるようにします。
func (u *temporaryPlayerDataUsecase) Commit(ctx context.Context, input CommitTemporaryPlayerDataInput) (*playerdataresult.Result, error) {
	if input.UploadToken == "" {
		return nil, &PlayerDataValidationError{Field: "upload_token", Message: "is required"}
	}

	entry, err := u.repo.Claim(ctx, u.exec, input.UploadToken)
	if err != nil {
		switch {
		case errors.Is(err, domainrepo.ErrTemporaryPlayerDataNotFound):
			return nil, ErrTemporaryPlayerDataNotFound
		case errors.Is(err, domainrepo.ErrTemporaryPlayerDataInUse):
			return nil, ErrTemporaryPlayerDataInProgress
		default:
			return nil, fmt.Errorf("temporary player data claim failed: %w", err)
		}
	}

	result, err := u.register(ctx, input.User, entry)
	// クライアント切断後も予約の解除・削除を確実に行うため、キャンセルを伝播させないcontextを使う。
	finishCtx := context.WithoutCancel(ctx)
	if err != nil {
		if releaseErr := u.repo.Release(finishCtx, u.exec, input.UploadToken); releaseErr != nil && !errors.Is(releaseErr, domainrepo.ErrTemporaryPlayerDataNotFound) {
			slog.WarnContext(finishCtx, "一時プレイヤーデータの予約解除に失敗しました", "error", releaseErr)
		}
		return nil, err
	}

	// 登録は完了しているため、削除に失敗しても結果は返す（未削除のデータはTTLで破棄される）。
	if deleteErr := u.repo.Delete(finishCtx, u.exec, input.UploadToken); deleteErr != nil && !errors.Is(deleteErr, domainrepo.ErrTemporaryPlayerDataNotFound) {
		slog.WarnContext(finishCtx, "確定済みの一時プレイヤーデータの削除に失敗しました", "error", deleteErr)
	}
	return result, nil
}

func (u *temporaryPlayerDataUsecase) register(ctx context.Context, user *entity.User, entry *entity.TemporaryPlayerData) (*playerdataresult.Result, error) {
	var payload PlayerDataPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTempDataPayloadInvalidJSON, err)
	}

	return u.playerDataUsecase.Register(ctx, user, &payload, entry.BodyHash)
}

var _ TemporaryPlayerDataUsecase = (*temporaryPlayerDataUsecase)(nil)
