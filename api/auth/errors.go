package auth

import "errors"

// Signup errors
var (
	UsernameAlreadyExistsErr = errors.New("username-already-exists")
	WeakPasswordErr          = errors.New("weak-password")
	InvalidUsernameFormatErr = errors.New("invalid-username-format")
)

// Login errors
var (
	UsernameNotFoundErr  = errors.New("username-not-found")
	IncorrectPasswordErr = errors.New("incorrect-password")
)

var (
	InvalidTokenError = errors.New("invalid-token")
)

var UnknownErr = errors.New("unknown-error")
