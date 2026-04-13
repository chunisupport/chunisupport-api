package usecase

import "errors"

var (
	ErrUsernameTaken            = errors.New("this username is already taken")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrInvalidIDToken           = errors.New("invalid firebase id token")
	ErrFirebaseUIDAlreadyLinked = errors.New("firebase uid already linked to another user")
	ErrRecentSignInRequired     = errors.New("recent sign-in required")
	ErrRecentSignInExpired      = errors.New("recent sign-in expired")
	ErrReauthUIDMismatch        = errors.New("reauth uid does not match user firebase uid")
	ErrFirebaseUIDNotLinked     = errors.New("firebase uid is not linked to user")

	ErrUsernameEmpty       = errors.New("username cannot be empty")
	ErrUsernameTooShort    = errors.New("username must be at least 5 characters")
	ErrUsernameTooLong     = errors.New("username must be 50 characters or less")
	ErrUsernameInvalidChar = errors.New("username can only contain lowercase letters and numbers")
)
