package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	TypeAccess  TokenType = "access"
	TypeRefresh TokenType = "refresh"
)

type Claims struct {
	jwtlib.RegisteredClaims
	UserID    string    `json:"uid"`
	Role      string    `json:"role"`
	TokenType TokenType `json:"type"`
	TokenID   string    `json:"jti"`
}

type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewManager(accessSecret, refreshSecret string, accessExpiry, refreshExpiry time.Duration) *Manager {
	return &Manager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (m *Manager) GenerateAccessToken(userID uuid.UUID, role string) (string, time.Time, error) {
	expiry := time.Now().Add(m.accessExpiry)
	claims := &Claims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			IssuedAt:  jwtlib.NewNumericDate(time.Now()),
			ExpiresAt: jwtlib.NewNumericDate(expiry),
			Issuer:    "gobank",
		},
		UserID:    userID.String(),
		Role:      role,
		TokenType: TypeAccess,
		TokenID:   uuid.New().String(),
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.accessSecret)
	return signed, expiry, err
}

func (m *Manager) GenerateRefreshToken(userID uuid.UUID, role string) (string, uuid.UUID, time.Time, error) {
	tokenID := uuid.New()
	expiry := time.Now().Add(m.refreshExpiry)
	claims := &Claims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			IssuedAt:  jwtlib.NewNumericDate(time.Now()),
			ExpiresAt: jwtlib.NewNumericDate(expiry),
			Issuer:    "gobank",
		},
		UserID:    userID.String(),
		Role:      role,
		TokenType: TypeRefresh,
		TokenID:   tokenID.String(),
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.refreshSecret)
	return signed, tokenID, expiry, err
}

func (m *Manager) ValidateAccessToken(tokenString string) (*Claims, error) {
	return m.validate(tokenString, m.accessSecret, TypeAccess)
}

func (m *Manager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return m.validate(tokenString, m.refreshSecret, TypeRefresh)
}

func (m *Manager) validate(tokenString string, secret []byte, expectedType TokenType) (*Claims, error) {
	token, err := jwtlib.ParseWithClaims(tokenString, &Claims{}, func(t *jwtlib.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return nil, ErrExpired
		}
		return nil, ErrInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalid
	}
	if claims.TokenType != expectedType {
		return nil, ErrInvalid
	}
	return claims, nil
}

var (
	ErrExpired = errors.New("token expired")
	ErrInvalid = errors.New("token invalid")
)
