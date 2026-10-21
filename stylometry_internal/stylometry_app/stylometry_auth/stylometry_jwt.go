package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const AccessTokenTTL = time.Hour

// Issuer хранит ключ подписи и выпускает access token после проверки пароля.
type Issuer struct {
	secret []byte
}

// VerifiedToken содержит только проверенные данные, нужные middleware и logout.
type VerifiedToken struct {
	UserID    uint
	JTI       string
	ExpiresAt time.Time
}

// NewIssuerFromEnv требует секрет из окружения до запуска HTTP-сервера.
func NewIssuerFromEnv() (*Issuer, error) {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET должен быть задан и содержать не менее 32 байт")
	}
	return &Issuer{secret: []byte(secret)}, nil
}

// Issue создаёт JWT для реального ID пользователя из базы данных.
func (issuer *Issuer) Issue(userID uint) (string, error) {
	if userID == 0 {
		return "", errors.New("нельзя выпустить токен без ID пользователя")
	}

	var randomID [16]byte
	if _, err := rand.Read(randomID[:]); err != nil {
		return "", err
	}

	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(uint64(userID), 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
		ID:        hex.EncodeToString(randomID[:]),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(issuer.secret)
}

// Verify проверяет подпись, алгоритм и обязательные поля access token.
func (issuer *Issuer) Verify(tokenString string) (VerifiedToken, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return issuer.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !token.Valid {
		return VerifiedToken{}, errors.New("недействительный JWT")
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil || strings.TrimSpace(claims.ID) == "" {
		return VerifiedToken{}, errors.New("отсутствуют обязательные поля JWT")
	}
	if !claims.ExpiresAt.After(claims.IssuedAt.Time) {
		return VerifiedToken{}, errors.New("некорректный срок действия JWT")
	}
	parsedID, err := strconv.ParseUint(claims.Subject, 10, strconv.IntSize)
	if err != nil || parsedID == 0 {
		return VerifiedToken{}, errors.New("некорректный sub JWT")
	}
	return VerifiedToken{UserID: uint(parsedID), JTI: claims.ID, ExpiresAt: claims.ExpiresAt.Time}, nil
}
