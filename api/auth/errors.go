package auth

import "errors"

// Signup errors
var (
	ErrWeakPassword          = errors.New("weak-password")
	ErrPasswordTooLong       = errors.New("weak-password")
	ErrInvalidUsernameFormat = errors.New("invalid-username-format")
)

// Login errors
var (
	ErrIncorrectPassword = errors.New("incorrect-password")
)
