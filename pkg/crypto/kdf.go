package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// Argon2id parameters (64MB memory, 3 iterations, 4 threads for memory-hard key derivation)
	Argon2Time    uint32 = 3
	Argon2Memory  uint32 = 64 * 1024
	Argon2Threads uint8  = 4

	// General cryptographic constants
	KeySize          = 32     // 256-bit key size
	SaltSize         = 16     // 128-bit salt
	PBKDF2Iterations = 210000 // OWASP recommended fallback iterations
)

// GenerateSalt creates a cryptographically secure 16-byte random salt
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate secure salt: %w", err)
	}
	return salt, nil
}

// DeriveKeyArgon2id derives a 256-bit key from a passphrase and salt using Argon2id
func DeriveKeyArgon2id(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, Argon2Time, Argon2Memory, Argon2Threads, KeySize)
}

// DeriveKeyPBKDF2 derives a 256-bit key using PBKDF2 with SHA-256 (standard alternative)
func DeriveKeyPBKDF2(passphrase string, salt []byte) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, PBKDF2Iterations, KeySize, sha256.New)
}