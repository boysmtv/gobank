package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Hasher struct {
	pepper string
}

func NewHasher(pepper string) *Hasher {
	return &Hasher{pepper: pepper}
}

func (h *Hasher) Hash(raw string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(raw+h.pepper), salt, 1, 64*1024, 4, 32)

	return fmt.Sprintf(
		"argon2id$%s$%s",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func (h *Hasher) Compare(raw, encoded string) error {
	parts := strings.Split(encoded, "$")
	if len(parts) != 3 {
		return errors.New("invalid password hash")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return err
	}

	currentHash := argon2.IDKey([]byte(raw+h.pepper), salt, 1, 64*1024, 4, 32)
	if subtle.ConstantTimeCompare(currentHash, expectedHash) != 1 {
		return errors.New("password mismatch")
	}

	return nil
}
