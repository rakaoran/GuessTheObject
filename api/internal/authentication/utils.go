package authentication

import (
	"api/internal/shared/authorization"
	"api/internal/shared/configs"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

func hashPassword(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	hash := argon2.IDKey([]byte(password),
		salt,
		configs.Argon2id.Time,
		configs.Argon2id.Memory,
		configs.Argon2id.Threads,
		configs.Argon2id.KeyLen,
	)
	b64hash := base64.RawStdEncoding.EncodeToString(hash)
	b64salt := base64.RawStdEncoding.EncodeToString(salt)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		configs.Argon2id.Memory,
		configs.Argon2id.Time,
		configs.Argon2id.Threads,
		b64salt,
		b64hash,
	)
}

func verifyPassword(susPassword string, encodedHash string) bool {
	parts := strings.Split(string(encodedHash), "$")
	salt, _ := base64.RawStdEncoding.DecodeString(parts[4])
	expectedHash, _ := base64.RawStdEncoding.DecodeString(parts[5])
	var time, memory uint32
	var threads uint8
	fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	susHash := argon2.IDKey([]byte(susPassword), salt, time, memory, threads, uint32(len(expectedHash)))

	return subtle.ConstantTimeCompare(susHash, expectedHash) == 1
}

func getJWT(id string) string {
	claim := authorization.JWTData{
		Id: id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * time.Duration(configs.JWTCookie.MaxAge))),
		},
	}
	tokenString, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claim).SignedString(configs.Envs.JWT_KEY)
	return tokenString
}
