package domain

import "errors"

var UnexpectedDatabaseError = errors.New("unexpected-database-error")
var UnexpectedPasswordHashingError = errors.New("unexpected-password-hashing-error")
var UnexpectedPasswordHashComparisonError = errors.New("unexpected-password-hash-comparison-error")
var UnexpectedTokenGenerationError = errors.New("unexpected-token-generation-error")
var UnexpectedTokenVerificationError = errors.New("unexpected-token-verification-error")

var (
	ErrDuplicateUsername = errors.New("duplicate-username")
	ErrUserNotFound      = errors.New("user-not-found")
	ErrIdNotFound        = errors.New("id-not-found")
)

var (
	ErrInvalidSigningAlg     = errors.New("invalid-signing-algorithm")
	ErrExpiredToken          = errors.New("expired-token")
	ErrInvalidTokenSignature = errors.New("invalid-token-signature")
	ErrCorruptedToken        = errors.New("corrupted-token")
)
