package authentication

import (
	"strings"
	"testing"
)

func TestVerifyHash(t *testing.T) {
	x := hashPassword("password1")
	v := verifyPassword("password1", x)

	if !v {
		t.Errorf("Hashing verification problem, comparing password (%s) and (%s) expecting true, got false", "password1", "password2")
	}

	v2 := verifyPassword("password2", x)

	if v2 {
		t.Errorf("Hashing verification problem, comparing password (%s) and (%s) expecting false, got true", "password1", "password2")
	}
}

func TestHashPassword(t *testing.T) {
	hashed := hashPassword("some password")

	parts := strings.Split(hashed, "$")

	if len(parts) != 6 {
		t.Errorf("Hash format is not right, expected 6 after splitting by '$', got %d", len(parts))
	}

	if parts[1] != "argon2id" {
		t.Errorf("Hash algorithm is not right or not written in the right place, expected argon2id, got %s", parts[1])
	}
}
