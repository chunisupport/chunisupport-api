package usecase

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrPlayerNotLinked    = errors.New("player not linked to user")
	ErrPlayerNotFound     = errors.New("player not found")
	ErrUserPrivate        = errors.New("user profile is private")
	ErrIncorrectPassword  = errors.New("current password is incorrect")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrOperationFailed    = errors.New("operation failed")
	ErrInternalError      = errors.New("internal error")
	ErrUserAlreadyDeleted = errors.New("user already deleted")
	ErrUserNotDeleted     = errors.New("user not deleted")
)
