package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/passwordhash"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/utils"
)

// RecoveryUsecase はリカバリーコード関連機能を扱います。
type RecoveryUsecase interface {
	IssueRecoveryCodes(ctx context.Context, userID int) ([]string, error)
	RecoverWithRecoveryCode(ctx context.Context, recoveryCode, newPassword string) error
}

type recoveryUsecaseImpl struct {
	db               repository.Executor
	tm               TransactionManager
	userRepo         repository.UserRepository
	recoveryCodeRepo repository.RecoveryCodeRepository
	pepper           string
}

func NewRecoveryUsecase(db repository.Executor, tm TransactionManager, userRepo repository.UserRepository, recoveryCodeRepo repository.RecoveryCodeRepository, pepper string) RecoveryUsecase {
	return &recoveryUsecaseImpl{db: db, tm: tm, userRepo: userRepo, recoveryCodeRepo: recoveryCodeRepo, pepper: pepper}
}

func (s *recoveryUsecaseImpl) IssueRecoveryCodes(ctx context.Context, userID int) ([]string, error) {
	if _, err := s.userRepo.FindByID(ctx, s.db, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	displayCodes := make([]string, 0, info.RecoveryCodeCount)
	recoveryCodes := make([]*entity.RecoveryCode, 0, info.RecoveryCodeCount)
	seen := make(map[string]struct{}, info.RecoveryCodeCount)
	maxGenerationAttempts := info.RecoveryCodeCount * 10
	for attempts := 0; len(recoveryCodes) < info.RecoveryCodeCount; attempts++ {
		if attempts >= maxGenerationAttempts {
			return nil, fmt.Errorf("failed to generate unique recovery codes after %d attempts", maxGenerationAttempts)
		}
		display, normalized, hashBytes, err := generateRecoveryCode()
		if err != nil {
			return nil, err
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		recoveryCodes = append(recoveryCodes, &entity.RecoveryCode{UserID: userID, CodeHash: hashBytes})
		displayCodes = append(displayCodes, display)
	}

	if err := s.tm.Transactional(ctx, func(tx repository.Executor) error {
		if err := s.recoveryCodeRepo.DeleteByUserID(ctx, tx, userID); err != nil {
			return err
		}
		return s.recoveryCodeRepo.CreateBatch(ctx, tx, recoveryCodes)
	}); err != nil {
		return nil, err
	}
	return displayCodes, nil
}

func (s *recoveryUsecaseImpl) RecoverWithRecoveryCode(ctx context.Context, recoveryCode, newPassword string) error {
	if len(newPassword) < info.PasswordMinLength {
		return ErrPasswordTooShort
	}
	if len(newPassword) > info.PasswordMaxLength {
		return ErrPasswordTooLong
	}
	normalized := normalizeRecoveryCode(recoveryCode)
	hashBytes := hashRecoveryCode(normalized)
	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		code, err := s.recoveryCodeRepo.FindByHashForUpdate(ctx, tx, hashBytes)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvalidRecoveryCredentials
			}
			return err
		}
		user, err := s.userRepo.FindByID(ctx, tx, code.UserID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvalidRecoveryCredentials
			}
			return err
		}
		if !user.IsActive() {
			return ErrInvalidRecoveryCredentials
		}
		if utils.CheckPasswordHashWithPepper(newPassword, s.pepper, user.PasswordHash.String()) {
			return ErrInvalidPassword
		}
		hashed, err := utils.HashPasswordWithPepper(newPassword, s.pepper)
		if err != nil {
			return err
		}
		newHash, err := passwordhash.NewPasswordHash(hashed)
		if err != nil {
			return err
		}
		user.ChangePassword(newHash)
		if err := s.userRepo.Save(ctx, tx, user); err != nil {
			return err
		}
		return s.recoveryCodeRepo.DeleteByID(ctx, tx, code.ID)
	})
}

func generateRecoveryCode() (string, string, []byte, error) {
	if info.RecoveryCodeSegmentCount <= 0 || info.RecoveryCodeSegmentLength <= 0 {
		return "", "", nil, fmt.Errorf("invalid recovery code format configuration")
	}
	totalLength := info.RecoveryCodeSegmentCount * info.RecoveryCodeSegmentLength
	displayBuilder := strings.Builder{}
	displayBuilder.Grow(totalLength + info.RecoveryCodeSegmentCount - 1)
	normalizedBuilder := strings.Builder{}
	normalizedBuilder.Grow(totalLength)
	charsetMax := big.NewInt(int64(len(info.RecoveryCodeCharset)))
	for i := 0; i < totalLength; i++ {
		index, err := rand.Int(rand.Reader, charsetMax)
		if err != nil {
			return "", "", nil, err
		}
		ch := info.RecoveryCodeCharset[int(index.Int64())]
		if i > 0 && i%info.RecoveryCodeSegmentLength == 0 {
			displayBuilder.WriteByte('-')
		}
		displayBuilder.WriteByte(ch)
		normalizedBuilder.WriteByte(ch)
	}
	normalized := normalizedBuilder.String()
	sum := sha256.Sum256([]byte(normalized))
	return displayBuilder.String(), normalized, sum[:], nil
}

func normalizeRecoveryCode(raw string) string {
	return strings.ToUpper(strings.ReplaceAll(raw, "-", ""))
}

func hashRecoveryCode(normalized string) []byte {
	sum := sha256.Sum256([]byte(normalized))
	return sum[:]
}
