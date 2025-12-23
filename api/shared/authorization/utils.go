package authorization

import (
	"api/shared/configs"
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type JWTData struct {
	Id string
	jwt.RegisteredClaims
}

func VerifyJWT(tokenString string) (JWTData, bool) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTData{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid-token-method")
		} else {
			return configs.Envs.JWT_KEY, nil
		}
	})

	if err != nil {
		return JWTData{}, false
	}

	if claims, ok := token.Claims.(*JWTData); ok && token.Valid {
		return *claims, true
	}

	return JWTData{}, false

}
