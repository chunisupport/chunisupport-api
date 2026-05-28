package apierror

import (
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

// FromUsecaseError はユースケース層のエラーをAPIErrorに変換します
func FromUsecaseError(err error) *APIError {
	if err == nil {
		return nil
	}

	// usecase層の既知エラーをマッピング
	// セキュリティ上の理由により、詳細なエラーは汎用的なエラーにマッピングされます
	switch {
	case errors.Is(err, usecase.ErrUsernameTaken):
		return ErrRegistrationFailed.WithInternal(err) // 409 Conflict → 400 Bad Request
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return ErrInvalidCredentials.WithInternal(err)
	case errors.Is(err, usecase.ErrRecentSignInAuthTimeMissing):
		return ErrRecentSignInRequired.WithInternal(err)
	case errors.Is(err, usecase.ErrRecentSignInRequired):
		return ErrRecentSignInRequired.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidIDToken):
		return ErrInvalidToken.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidTurnstileToken):
		return ErrInvalidTurnstileToken.WithInternal(err)
	case errors.Is(err, usecase.ErrFirebaseUIDAlreadyLinked):
		return ErrFirebaseUIDAlreadyLinked.WithInternal(err)
	case errors.Is(err, usecase.ErrUserNotFound):
		return ErrUserNotFound.WithInternal(err)
	case errors.Is(err, usecase.ErrUserPrivate):
		return ErrUserNotFound.WithInternal(err) // 403 → 404 でユーザー存在を隠蔽
	case errors.Is(err, usecase.ErrPlayerNotLinked):
		return ErrPlayerNotLinked.WithInternal(err)

	case errors.Is(err, usecase.ErrOperationFailed):
		return ErrOperationFailed.WithInternal(err)
	case errors.Is(err, usecase.ErrInternalError):
		return ErrInternalError.WithInternal(err)
	case errors.Is(err, usecase.ErrAdminRequired):
		return ErrForbidden.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidAPIToken):
		return ErrInvalidToken.WithInternal(err)
	// 楽曲関連エラー
	case errors.Is(err, repository.ErrSongNotFound):
		return ErrSongNotFound.WithInternal(err)
	case errors.Is(err, repository.ErrDuplicateOfficialIdx):
		return ErrDuplicateOfficialIdx.WithInternal(err)
	case errors.Is(err, repository.ErrHonorNotFound):
		return ErrNotFound.WithInternal(err)
	case errors.Is(err, repository.ErrHonorConflict):
		return ErrConflict.WithInternal(err)
	// 難易度関連エラー
	case errors.Is(err, usecase.ErrInvalidDifficulty):
		return ErrInvalidDifficulty.WithInternal(err)
	case errors.Is(err, usecase.ErrChartNotFound):
		return ErrChartNotFound.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidPlayerName):
		return ErrValidationFailed.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidWorldsendInput):
		return ErrValidationFailed.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidHonorInput):
		return ErrValidationFailed.WithInternal(err)
	// ユーザー名バリデーションエラー
	case errors.Is(err, usecase.ErrUsernameEmpty):
		return ErrUsernameEmpty.WithInternal(err)
	case errors.Is(err, usecase.ErrUsernameTooShort):
		return ErrUsernameTooShort.WithInternal(err)
	case errors.Is(err, usecase.ErrUsernameTooLong):
		return ErrUsernameTooLong.WithInternal(err)
	case errors.Is(err, usecase.ErrUsernameInvalidChar):
		return ErrUsernameInvalidChar.WithInternal(err)
	// アプリバージョンバリデーションエラー
	case errors.Is(err, usecase.ErrAppVersionUnsupported):
		return ErrAppVersionUnsupported.WithInternal(err)
	case errors.Is(err, usecase.ErrGoalNotFound):
		return ErrGoalNotFound.WithInternal(err)
	case errors.Is(err, usecase.ErrGoalLimitExceeded):
		return ErrGoalLimitExceeded.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidGoalInput):
		return ErrInvalidGoalInput.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidGoalTitle):
		return ErrGoalInvalidTitle.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidAchievementType):
		return ErrGoalInvalidAchievementType.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidAchievementParam):
		return ErrGoalInvalidAchievementParams.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidGoalAttributes):
		return ErrGoalInvalidAttributes.WithInternal(err)
	}

	// PlayerDataValidationError
	var validationErr *usecase.PlayerDataValidationError
	if errors.As(err, &validationErr) {
		return ErrValidationFailed.WithInternal(err)
	}

	// PlayerDataNotFoundError
	var notFoundErr *usecase.PlayerDataNotFoundError
	if errors.As(err, &notFoundErr) {
		return ErrResourceNotFound.WithInternal(err)
	}

	// PlayerDataConflictError
	var conflictErr *usecase.PlayerDataConflictError
	if errors.As(err, &conflictErr) {
		return ErrConflict.WithInternal(err)
	}

	// 未知のエラーは内部エラーとして扱う
	return ErrInternalError.WithInternal(err)
}
