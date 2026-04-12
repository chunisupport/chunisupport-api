package usecase

import "errors"

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrPlayerNotLinked = errors.New("player not linked to user")
	ErrUserPrivate     = errors.New("user profile is private")
	ErrOperationFailed = errors.New("operation failed")
	ErrInternalError   = errors.New("internal error")
)
