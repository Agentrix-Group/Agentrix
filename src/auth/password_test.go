package auth

import (
	"crypto/sha512"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArgon2PasswordHashing(t *testing.T) {
	r := require.New(t)

	// Use fast params for tests
	SetActiveArgon2Params(FastArgon2Params)
	defer SetActiveArgon2Params(DefaultArgon2Params)

	rawPassword := "SecurePassword123!"

	// 1. Hash generation
	hashed, err := HashPassword(rawPassword)
	r.NoError(err)
	r.NotEmpty(hashed)
	r.True(strings.HasPrefix(hashed, "$argon2id$v=19$"))

	// 2. Successful verification
	match, needsRehash, err := VerifyPassword(rawPassword, hashed)
	r.NoError(err)
	r.True(match)
	r.False(needsRehash)

	// 3. Failed verification with wrong password
	wrongMatch, _, err := VerifyPassword("WrongPassword123!", hashed)
	r.NoError(err)
	r.False(wrongMatch)

	// 4. Verification with legacy SHA-512 triggers lazy rehash
	legacySHA512 := fmt.Sprintf("%x", sha512.Sum512([]byte(rawPassword)))
	legacyMatch, legacyRehash, err := VerifyPassword(rawPassword, legacySHA512)
	r.NoError(err)
	r.True(legacyMatch, "Legacy SHA-512 password should match")
	r.True(legacyRehash, "Legacy SHA-512 password must trigger needsRehash=true for lazy upgrade")

	// 5. Length limits
	_, err = HashPassword("short")
	r.Equal(ErrPasswordTooShort, err)

	veryLong := strings.Repeat("A", 129)
	_, err = HashPassword(veryLong)
	r.Equal(ErrPasswordTooLong, err)

	matchLong, _, err := VerifyPassword(veryLong, hashed)
	r.Equal(ErrPasswordTooLong, err)
	r.False(matchLong)

	// 6. Tampered hash rejected
	tampered := hashed[:len(hashed)-4] + "AAAA"
	tamperedMatch, _, err := VerifyPassword(rawPassword, tampered)
	r.NoError(err)
	r.False(tamperedMatch)

	// 7. Invalid hash format returns error
	_, _, err = VerifyPassword(rawPassword, "$argon2id$invalid$format")
	r.Equal(ErrInvalidHashFormat, err)
}
