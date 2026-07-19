// Package security holds credential primitives kept out of the application
// layer so services depend on the PasswordHasher port instead of a library.
package security

import (
	"github.com/alexedwards/argon2id"
)


type Argon2Hasher struct {
	params *argon2id.Params
}

func NewArgon2Hasher() *Argon2Hasher {
	return &Argon2Hasher{params: argon2id.DefaultParams}
}

func (h *Argon2Hasher) Hash(password string) (string, error) {
	return argon2id.CreateHash(password, h.params)
}

// Verify reports whether the password matches. A malformed stored hash is a
// mismatch, not an error. the caller only distinguishes valid from invalid.
func (h *Argon2Hasher) Verify(password, encodedHash string) bool {
	match, err := argon2id.ComparePasswordAndHash(password, encodedHash)
	if err != nil {
		return false
	}
	return match
}