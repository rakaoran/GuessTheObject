package crypto_test

import (
	"api/crypto"
	"api/domain"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	hasher := crypto.NewArgon2idHasher(1, 15*1024, 32, 16, 1)

	hash, err := hasher.Hash("supersecretpassword")

	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.True(t, strings.HasPrefix(hash, "$argon2id"), "Hash should start with argon2id prefix")
}

func TestCompare(t *testing.T) {
	hasher := crypto.NewArgon2idHasher(1, 15*1024, 32, 16, 1)
	password := "my_password_123"

	hash, _ := hasher.Hash(password)

	// 1. Correct password
	match, err := hasher.Compare(hash, password)
	assert.NoError(t, err)
	assert.True(t, match, "Password should match")

	// 2. Wrong password
	match, err = hasher.Compare(hash, "wrong_password")
	assert.NoError(t, err)
	assert.False(t, match, "Password should not match")

	// 3. Malformed hash (should trigger the wrapped domain error)
	match, err = hasher.Compare("invalid-hash-string", password)
	assert.ErrorIs(t, err, domain.UnexpectedPasswordHashComparisonError)
	assert.False(t, match)
}

func TestHasherParams(t *testing.T) {
	// Custom params to verify they are passed correctly
	iter := uint32(2)
	memory := uint32(12 * 1024)
	parallelism := uint8(2)
	keyLen := uint32(32)
	saltLen := uint32(16)

	hasher := crypto.NewArgon2idHasher(iter, memory, keyLen, saltLen, parallelism)

	hash, err := hasher.Hash("test_param_check")
	assert.NoError(t, err)

	// Format: $argon2id$v=19$m=12288,t=2,p=2$salt$key
	parts := strings.Split(hash, "$")
	assert.Len(t, parts, 6, "Hash format should have 6 parts (including empty start)")

	// 1. Check Params String
	expectedParams := fmt.Sprintf("m=%d,t=%d,p=%d", memory, iter, parallelism)
	assert.Equal(t, expectedParams, parts[3], "Params part matches config")

	// 2. Check Salt Length
	// using RawStdEncoding because argon2 string doesn't use padding
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	assert.NoError(t, err, "Salt should be valid base64")
	assert.Len(t, salt, int(saltLen), "Salt length should match config")

	// 3. Check Key (Hash) Length
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	assert.NoError(t, err, "Key should be valid base64")
	assert.Len(t, key, int(keyLen), "Key length should match config")
}
