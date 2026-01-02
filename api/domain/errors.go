package domain

import "errors"

var (
	DatabaseError        = errors.New("database-error")
	ErrDuplicateUsername = errors.New("duplicate-username")
	ErrUsernameNotFound  = errors.New("username-not-found")
	ErrIdNotFound        = errors.New("id-not-found")
)

var HashingError = errors.New("hashing-error")

var (
	TokenError               = errors.New("token-error")
	ErrInvalidSigningMethod  = errors.New("invalid-signing-method")
	ErrExpiredToken          = errors.New("expired-token")
	ErrInvalidTokenSignature = errors.New("invalid-token-signature")
	ErrCorruptedToken        = errors.New("corrupted-token")
)
