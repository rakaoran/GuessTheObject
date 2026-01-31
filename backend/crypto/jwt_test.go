package crypto_test

import (
	"api/crypto"
	"api/domain"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	var JWTManager = crypto.NewJWTManager("supersupersecretkey don't share it with anyone i tell you bruh", time.Hour)
	now := time.Now()
	token, _ := JWTManager.Generate("123-456-789", now)

	tokenParts := strings.Split(token, ".")
	println(token)
	jwtHead, _ := base64.RawURLEncoding.DecodeString(tokenParts[0])
	jwtBody, _ := base64.RawURLEncoding.DecodeString(tokenParts[1])
	jwtSignature, _ := base64.RawURLEncoding.DecodeString(tokenParts[2])

	assert.JSONEq(t, `{"alg": "HS256","typ": "JWT"}`, string(jwtHead))
	assert.JSONEq(t, fmt.Sprintf(`{"id": "123-456-789","exp": %d }`, now.Add(time.Hour).Unix()), string(jwtBody))
	assert.Len(t, jwtSignature, 256/8, "256 bits of sha256")
}

func TestVerify(t *testing.T) {
	var JWTManager = crypto.NewJWTManager("supersupersecretkey don't share it with anyone i tell you bruh", 2*time.Hour)

	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)
	threeHoursAgo := now.Add(-3 * time.Hour)

	token, _ := JWTManager.Generate("idid", threeHoursAgo)

	_, err := JWTManager.Verify(token)

	assert.ErrorIs(t, err, domain.ErrExpiredToken)

	token, _ = JWTManager.Generate("idid", oneHourAgo)
	id, err := JWTManager.Verify(token)

	assert.ErrorIs(t, err, nil)
	assert.Equal(t, "idid", id)

	token2 := token + "lol"

	id, err = JWTManager.Verify(token2)
	assert.ErrorIs(t, err, domain.ErrInvalidTokenSignature)

	tokenNonHS256Alg := "eyJhbGciOiJFUzUxMiIsInR5cCI6IkpXVCJ9" + "." + strings.Split(token, ".")[1] + "." + strings.Split(token, ".")[2]
	id, err = JWTManager.Verify(tokenNonHS256Alg)
	assert.ErrorIs(t, err, domain.ErrInvalidSigningAlg)

	tokenNoneAlg := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0" + "." + strings.Split(token, ".")[1] + "."
	id, err = JWTManager.Verify(tokenNoneAlg)
	assert.ErrorIs(t, err, domain.ErrInvalidSigningAlg)

	corruptedToken := "stemretmretm"

	id, err = JWTManager.Verify(corruptedToken)
	assert.ErrorIs(t, err, domain.ErrCorruptedToken)

}
