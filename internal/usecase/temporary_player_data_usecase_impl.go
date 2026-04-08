package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
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
	if len(input.Payload) == 0 {
		return nil, &PlayerDataValidationError{Field: "payload", Message: "is required"}
	}
	if input.IPAddress == "" {
		return nil, &PlayerDataValidationError{Field: "ip_address", Message: "is required"}
	}

	token := uuid.NewString()
	now := time.Now().UTC()
	entry := &entity.TemporaryPlayerData{
		Token:     token,
		IPAddress: input.IPAddress,
		Payload:   append([]byte(nil), input.Payload...),
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

	entry, err := u.repo.ConsumeByToken(ctx, input.UploadToken)
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
		restoreErr := u.repo.Create(ctx, entry)
		if restoreErr != nil {
			slog.Warn("temporary player data restore failed after register error", "token", input.UploadToken, "error", restoreErr)
		}
		return nil, err
	}

	return result, nil
}

var _ TemporaryPlayerDataUsecase = (*temporaryPlayerDataUsecase)(nil)
