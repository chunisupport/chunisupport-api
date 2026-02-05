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
	case errors.Is(err, usecase.ErrIncorrectPassword):
		return ErrInvalidCredentials.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidPassword):
		return ErrInvalidPassword.WithInternal(err) // パスワード関連を統合
	case errors.Is(err, usecase.ErrUserIDMismatch):
		return ErrForbidden.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidSession):
		return ErrInvalidSession.WithInternal(err) // セッション系を統合
	case errors.Is(err, usecase.ErrUserDeleted):
		return ErrUnauthorized.WithInternal(err)
	case errors.Is(err, usecase.ErrUserNotFound):
		return ErrUserNotFound.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidRecoveryCredentials):
		return ErrInvalidRecovery.WithInternal(err)
	case errors.Is(err, usecase.ErrUserPrivate):
		return ErrUserNotFound.WithInternal(err) // 403 → 404 でユーザー存在を隠蔽
	case errors.Is(err, usecase.ErrPlayerNotLinked):
		return ErrUserNotFound.WithInternal(err) // プレイヤー未紐付も404で隠蔽
	case errors.Is(err, usecase.ErrUserAlreadyDeleted):
		return ErrOperationFailed.WithInternal(err) // 409 → 400 で詳細を隠蔽
	case errors.Is(err, usecase.ErrUserNotDeleted):
		return ErrOperationFailed.WithInternal(err) // 409 → 400 で詳細を隠蔽
	case errors.Is(err, usecase.ErrOperationFailed):
		return ErrOperationFailed.WithInternal(err)
	case errors.Is(err, usecase.ErrInvalidAPIToken):
		return ErrInvalidToken.WithInternal(err)
	// 楽曲関連エラー
	case errors.Is(err, repository.ErrSongNotFound):
		return ErrSongNotFound.WithInternal(err)
	// ユーザー名バリデーションエラー
	case errors.Is(err, usecase.ErrUsernameEmpty):
		return ErrUsernameEmpty.WithInternal(err)
	case errors.Is(err, usecase.ErrUsernameTooShort):
		return ErrUsernameTooShort.WithInternal(err)
	case errors.Is(err, usecase.ErrUsernameTooLong):
		return ErrUsernameTooLong.WithInternal(err)
	case errors.Is(err, usecase.ErrUsernameInvalidChar):
		return ErrUsernameInvalidChar.WithInternal(err)
	// パスワードバリデーションエラー
	case errors.Is(err, usecase.ErrPasswordTooShort):
		return ErrPasswordTooShort.WithInternal(err)
	case errors.Is(err, usecase.ErrPasswordTooLong):
		return ErrPasswordTooLong.WithInternal(err)
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
