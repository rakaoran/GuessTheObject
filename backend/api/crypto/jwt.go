package crypto

import (
	"api/domain"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtCustomClaims is an unexported struct used for claims.
// Fields must be exported for JSON serialization.
type jwtCustomClaims struct {
	Id string `json:"id"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secretKey []byte
	maxAge    time.Duration
}

func NewJWTManager(secretKey string, maxAge time.Duration) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
		maxAge:    maxAge,
	}
}

func (m *JWTManager) Generate(id string, now time.Time) (string, error) {
	claims := jwtCustomClaims{
		Id: id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.maxAge)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secretKey)

	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.UnexpectedTokenGenerationError, err)
	}

	return signedToken, nil
}

func (m *JWTManager) Verify(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtCustomClaims{}, func(token *jwt.Token) (any, error) {
		// Validate the signing method is what we expect (HMAC)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrInvalidSigningAlg
		}
		return m.secretKey, nil
	})

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidSigningAlg):
			return "", err
		case errors.Is(err, jwt.ErrTokenExpired):
			return "", domain.ErrExpiredToken
		case errors.Is(err, jwt.ErrSignatureInvalid):
			return "", domain.ErrInvalidTokenSignature
		case errors.Is(err, jwt.ErrTokenMalformed):
			return "", domain.ErrCorruptedToken
		default:
			return "", fmt.Errorf("%w: %w", domain.UnexpectedTokenVerificationError, err)
		}
	}

	if claims, ok := token.Claims.(*jwtCustomClaims); ok && token.Valid {
		return claims.Id, nil
	}

	return "", domain.ErrCorruptedToken
}
