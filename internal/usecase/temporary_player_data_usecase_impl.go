package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/google/uuid"
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
	var payload PlayerDataPayload
	if err := unmarshalPlayerDataPayload(input.Payload, &payload); err != nil {
		return nil, &PlayerDataValidationError{Field: "payload", Message: "must be valid json"}
	}

	token := uuid.NewString()
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
		return nil, &PlayerDataValidationError{Field: "payload", Message: err.Error()}
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

func (u *temporaryPlayerDataUsecase) Commit(ctx context.Context, input CommitTemporaryPlayerDataInput) (*api_internal.PlayerDataResult, error) {
	if input.UploadToken == "" {
		return nil, &PlayerDataValidationError{Field: "upload_token", Message: "is required"}
	}

	entry, err := u.repo.ConsumeByToken(ctx, u.exec, input.UploadToken)
	if err != nil {
		if errors.Is(err, domainrepo.ErrTemporaryPlayerDataNotFound) {
			return nil, ErrTemporaryPlayerDataNotFound
		}
		return nil, fmt.Errorf("temporary player data consume failed: %w", err)
	}

	var payload PlayerDataPayload
	if err := unmarshalPlayerDataPayload(entry.Payload, &payload); err != nil {
		return nil, fmt.Errorf("temporary player data payload decode failed: %w", err)
	}

	result, err := u.playerDataUsecase.Register(ctx, input.User, &payload, entry.BodyHash)
	if err != nil {
		return nil, err
	}
	return result, nil
}

var _ TemporaryPlayerDataUsecase = (*temporaryPlayerDataUsecase)(nil)
