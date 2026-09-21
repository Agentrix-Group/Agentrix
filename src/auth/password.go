package auth

import (
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrPasswordTooLong   = errors.New("password exceeds maximum allowed length of 128 characters")
	ErrPasswordTooShort  = errors.New("password must be at least 6 characters")
	ErrInvalidHashFormat = errors.New("invalid password hash format")
)

const (
	MaxPasswordLength = 128
	MinPasswordLength = 6
)

// Argon2Params defines the work factors and memory costs for Argon2id hashing.
type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2Params defines production-grade OWASP-compliant parameters.
var DefaultArgon2Params = Argon2Params{
	Memory:      64 * 1024, // 64 MB
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// FastArgon2Params provides low-cost parameters for fast unit testing.
var FastArgon2Params = Argon2Params{
	Memory:      16 * 1024, // 16 MB
	Iterations:  1,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

// activeParams holds the currently active params (defaults to DefaultArgon2Params).
var activeParams = DefaultArgon2Params

// SetActiveArgon2Params allows tests or environment configuration to override work factors.
func SetActiveArgon2Params(p Argon2Params) {
	activeParams = p
}

// IsArgon2idHash checks whether the given hash string conforms to the Argon2id PHC prefix.
func IsArgon2idHash(hash string) bool {
	return strings.HasPrefix(hash, "$argon2id$")
}

// HashPassword hashes a plain-text password using Argon2id with cryptographically
// secure random salt, formatted in standard PHC string format:
// $argon2id$v=19$m=65536,t=3,p=2$<base64-salt>$<base64-hash>
func HashPassword(password string) (string, error) {
	if len(password) > MaxPasswordLength {
		return "", ErrPasswordTooLong
	}
	if len(password) < MinPasswordLength {
		return "", ErrPasswordTooShort
	}

	salt := make([]byte, activeParams.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate cryptographic salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		activeParams.Iterations,
		activeParams.Memory,
		activeParams.Parallelism,
		activeParams.KeyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		activeParams.Memory,
		activeParams.Iterations,
		activeParams.Parallelism,
		b64Salt,
		b64Hash,
	)

	return encoded, nil
}

// VerifyPassword verifies a plaintext password against an encoded hash.
// It supports both modern Argon2id PHC hashes and legacy unsalted SHA-512 hashes.
// Returns (matches, needsRehash, err).
// If a legacy SHA-512 hash matches, needsRehash is true so the caller can lazily migrate the credential.
func VerifyPassword(password, encodedHash string) (bool, bool, error) {
	if len(password) > MaxPasswordLength {
		return false, false, ErrPasswordTooLong
	}

	// 1. Argon2id PHC string format check
	if strings.HasPrefix(encodedHash, "$argon2id$") {
		parts := strings.Split(encodedHash, "$")
		// Expected format: ["", "argon2id", "v=19", "m=65536,t=3,p=2", "<salt>", "<hash>"]
		if len(parts) != 6 {
			return false, false, ErrInvalidHashFormat
		}

		var version int
		if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
			return false, false, ErrInvalidHashFormat
		}

		var memory, iterations uint32
		var parallelism uint8
		if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
			return false, false, ErrInvalidHashFormat
		}

		salt, err := base64.RawStdEncoding.DecodeString(parts[4])
		if err != nil {
			return false, false, ErrInvalidHashFormat
		}

		expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
		if err != nil {
			return false, false, ErrInvalidHashFormat
		}

		calculatedHash := argon2.IDKey(
			[]byte(password),
			salt,
			iterations,
			memory,
			parallelism,
			uint32(len(expectedHash)),
		)

		if subtle.ConstantTimeCompare(expectedHash, calculatedHash) != 1 {
			return false, false, nil
		}

		// Check if work parameters match target configuration
		needsRehash := (memory != activeParams.Memory ||
			iterations != activeParams.Iterations ||
			parallelism != activeParams.Parallelism)

		return true, needsRehash, nil
	}

	// 2. Legacy SHA-512 fallback (128 hex chars)
	legacyHash := fmt.Sprintf("%x", sha512.Sum512([]byte(password)))
	if subtle.ConstantTimeCompare([]byte(encodedHash), []byte(legacyHash)) == 1 {
		// Valid match on legacy hash -> requires lazy migration to Argon2id
		return true, true, nil
	}

	return false, false, nil
}
