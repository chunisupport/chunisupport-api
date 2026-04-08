package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/google/uuid"
)

type temporaryPlayerDataUsecase struct {
	repo              domainrepo.TemporaryPlayerDataRepository
	playerDataUsecase PlayerDataUsecase
	ttl               time.Duration
}

// NewTemporaryPlayerDataUsecase は TemporaryPlayerDataUsecase の実装を返します。
func NewTemporaryPlayerDataUsecase(repo domainrepo.TemporaryPlayerDataRepository, playerDataUsecase PlayerDataUsecase, ttl time.Duration) TemporaryPlayerDataUsecase {
	return &temporaryPlayerDataUsecase{
		repo:              repo,
		playerDataUsecase: playerDataUsecase,
		ttl:               ttl,
	}
}

func (u *temporaryPlayerDataUsecase) Create(ctx context.Context, input CreateTemporaryPlayerDataInput) (*CreateTemporaryPlayerDataOutput, error) {
	if input.Payload == nil {
		return nil, &PlayerDataValidationError{Field: "payload", Message: "is required"}
	}
	if input.IPAddress == "" {
		return nil, &PlayerDataValidationError{Field: "ip_address", Message: "is required"}
	}

	token := uuid.NewString()
	now := time.Now().UTC()
	payloadBytes, err := marshalPlayerDataPayload(input.Payload)
	if err != nil {
		return nil, fmt.Errorf("temporary player data payload encode failed: %w", err)
	}

	entry := &entity.TemporaryPlayerData{
		Token:     token,
		IPAddress: input.IPAddress,
		Payload:   payloadBytes,
		BodyHash:  input.BodyHash,
		CreatedAt: now,
		ExpiresAt: now.Add(u.ttl),
	}

	if err := u.repo.Create(ctx, entry); err != nil {
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
	if input.User == nil {
		return nil, ErrUnauthorizedOperation
	}
	if input.UploadToken == "" {
		return nil, &PlayerDataValidationError{Field: "upload_token", Message: "is required"}
	}

	entry, err := u.repo.FindByToken(ctx, input.UploadToken)
	if err != nil {
		if errors.Is(err, domainrepo.ErrTemporaryPlayerDataNotFound) {
			return nil, ErrTemporaryPlayerDataNotFound
		}
		return nil, fmt.Errorf("temporary player data find failed: %w", err)
	}

	var payload PlayerDataPayload
	if err := unmarshalPlayerDataPayload(entry.Payload, &payload); err != nil {
		return nil, fmt.Errorf("temporary player data payload decode failed: %w", err)
	}

	bodyHash := entry.BodyHash
	if bodyHash == "" {
		hash := sha256.Sum256(entry.Payload)
		bodyHash = hex.EncodeToString(hash[:])
	}

	result, err := u.playerDataUsecase.Register(ctx, input.User, &payload, bodyHash)
	if err != nil {
		return nil, err
	}

	if err := u.repo.Delete(ctx, input.UploadToken); err != nil {
		return nil, fmt.Errorf("temporary player data delete failed: %w", err)
	}

	return result, nil
}

var _ TemporaryPlayerDataUsecase = (*temporaryPlayerDataUsecase)(nil)
