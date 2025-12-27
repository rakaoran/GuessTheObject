package crypto

import (
	"api/auth"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtCustomClaims is an unexported struct used for claims.
// Fields must be exported for JSON serialization.
type jwtCustomClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secretKey []byte
}

func NewJWTManager(secretKey string) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
	}
}

func (m *JWTManager) Generate(username string) string {
	claims := jwtCustomClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(40, 0, 0)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, _ := token.SignedString(m.secretKey)

	return signedToken
}

func (m *JWTManager) Verify(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtCustomClaims{}, func(token *jwt.Token) (any, error) {
		// Validate the signing method is what we expect (HMAC)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, auth.InvalidTokenError
		}
		return m.secretKey, nil
	})

	if err != nil {
		return "", auth.InvalidTokenError
	}

	if claims, ok := token.Claims.(*jwtCustomClaims); ok && token.Valid {
		return claims.Username, nil
	}

	return "", auth.InvalidTokenError
}
