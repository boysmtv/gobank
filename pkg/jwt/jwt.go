package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (m *Manager) Generate(subject string) (string, error) {
	expiresAt := time.Now().Add(m.ttl).Unix()
	payload := fmt.Sprintf("%s:%d", subject, expiresAt)

	mac := hmac.New(sha256.New, m.secret)
	if _, err := mac.Write([]byte(payload)); err != nil {
		return "", err
	}

	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	token := base64.RawURLEncoding.EncodeToString([]byte(payload))

	return token + "." + signature, nil
}

func (m *Manager) Validate(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errors.New("invalid token payload")
	}

	mac := hmac.New(sha256.New, m.secret)
	if _, err := mac.Write(payload); err != nil {
		return "", err
	}

	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return "", errors.New("invalid token signature")
	}

	values := strings.Split(string(payload), ":")
	if len(values) != 2 {
		return "", errors.New("invalid token claims")
	}

	expiresAt, err := strconv.ParseInt(values[1], 10, 64)
	if err != nil {
		return "", errors.New("invalid token expiry")
	}
	if time.Now().Unix() > expiresAt {
		return "", errors.New("token expired")
	}

	return values[0], nil
}
