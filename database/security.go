package database

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/pbkdf2"
)

const argon2idTime = 2
const argon2idMemory = 19456
const argon2idThreads = 1

func HashPassword(password string) (string, error) {
	salt, err := generateRandomBytes(16)
	if err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argon2idTime, argon2idMemory, argon2idThreads, 32)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argon2idMemory, argon2idTime, argon2idThreads, b64Salt, b64Hash)

	return encodedHash, nil
}

func VerifyPassword(expectedHash string, providedPassword string) (bool, error) {
	parts := strings.Split(expectedHash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid expected hash")
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, fmt.Errorf("invalid expected hash: %w", err)
	}
	if version != argon2.Version {
		return false, fmt.Errorf("argon2id version mismatch: expected %d, got %d", argon2.Version, version)
	}

	var memory uint32
	var time uint32
	var threads uint8
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, fmt.Errorf("invalid expected hash: %w", err)
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("invalid expected hash: %w", err)
	}

	hash, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("invalid expected hash: %w", err)
	}

	providedHash := argon2.IDKey([]byte(providedPassword), salt, time, memory, threads, uint32(len(hash)))

	return subtle.ConstantTimeCompare(hash, providedHash) == 1, nil
}

type HashAlg string

const (
	HashAlg_SHA256             HashAlg = "sha256"
	HashAlgPBKDF2_SHA256_10000 HashAlg = "pbkdf2-sha256_10000"
	HashAlg_DEFAULT                    = HashAlg_SHA256
)

func HashAuthKey(authKey string, alg HashAlg) string {
	var hash []byte
	switch alg {
	case HashAlg_SHA256:
		h := sha256.Sum256([]byte(authKey))
		hash = h[:]
	case HashAlgPBKDF2_SHA256_10000:
		hash = pbkdf2.Key([]byte(authKey), []byte("constant-salt"), 10000, 256, sha256.New)
	default:
		panic("unknown hash algorithm")
	}
	return base64.RawStdEncoding.EncodeToString(hash)
}

const (
	authKeyChars = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ123456789"
	authKeyLen   = 64
)

func GenerateAuthKey() (string, error) {
	authKey := make([]byte, authKeyLen)
	for i := range authKey {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(authKeyChars))))
		if err != nil {
			return "", fmt.Errorf("get random char: %w", err)
		}
		authKey[i] = authKeyChars[n.Int64()]
	}
	return string(authKey), nil
}

func generateRandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
