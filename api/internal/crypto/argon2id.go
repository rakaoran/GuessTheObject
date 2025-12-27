package crypto

import (
	"github.com/alexedwards/argon2id"
)

type Argon2idHasher struct {
	params *argon2id.Params
}

// NewArgon2idHasher creates a new hasher with the specified difficulty parameters.
//
// memory must be provided in Kilobytes (KB).
func NewArgon2idHasher(time, memory, keyLength, saltLength uint32, parallelism uint8) *Argon2idHasher {
	return &Argon2idHasher{
		params: &argon2id.Params{
			Memory:      memory,
			Iterations:  time,
			Parallelism: parallelism,
			SaltLength:  saltLength,
			KeyLength:   keyLength,
		},
	}
}

func (h *Argon2idHasher) Hash(password string) string {
	hash, _ := argon2id.CreateHash(password, h.params)
	return hash
}

// Compare verifies a password against a hash.
func (h *Argon2idHasher) Compare(hash, password string) bool {
	match, _ := argon2id.ComparePasswordAndHash(password, hash)
	return match
}
