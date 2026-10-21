package auth

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestIssuerRequiresSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	if _, err := NewIssuerFromEnv(); err == nil {
		t.Fatal("ожидалась ошибка без JWT_SECRET")
	}
	t.Setenv("JWT_SECRET", "short")
	if _, err := NewIssuerFromEnv(); err == nil {
		t.Fatal("ожидалась ошибка для короткого JWT_SECRET")
	}
}

func TestIssueSignsUniqueOneHourTokens(t *testing.T) {
	var randomSecret [32]byte
	if _, err := rand.Read(randomSecret[:]); err != nil {
		t.Fatal(err)
	}
	secret := hex.EncodeToString(randomSecret[:])
	t.Setenv("JWT_SECRET", secret)
	issuer, err := NewIssuerFromEnv()
	if err != nil {
		t.Fatal(err)
	}

	var previousID string
	for range 2 {
		tokenString, err := issuer.Issue(42)
		if err != nil {
			t.Fatal(err)
		}
		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
		if err != nil || !token.Valid {
			t.Fatalf("токен не прошёл проверку подписи: %v", err)
		}
		if claims.Subject != strconv.Itoa(42) || claims.IssuedAt == nil || claims.ExpiresAt == nil {
			t.Fatalf("отсутствуют обязательные поля JWT: %+v", claims)
		}
		if got := claims.ExpiresAt.Sub(claims.IssuedAt.Time); got != AccessTokenTTL {
			t.Fatalf("срок действия = %v, ожидался %v", got, AccessTokenTTL)
		}
		if time.Since(claims.IssuedAt.Time) > time.Minute {
			t.Fatal("время выдачи токена устарело")
		}
		if claims.ID == "" || claims.ID == previousID {
			t.Fatal("jti отсутствует или повторился")
		}
		verified, err := issuer.Verify(tokenString)
		if err != nil || verified.UserID != 42 || verified.JTI != claims.ID || !verified.ExpiresAt.Equal(claims.ExpiresAt.Time) {
			t.Fatalf("проверенные claims не совпадают: %+v, ошибка: %v", verified, err)
		}
		previousID = claims.ID
	}
}

func TestVerifyRejectsInvalidClaimsAndAlgorithm(t *testing.T) {
	var randomSecret [32]byte
	if _, err := rand.Read(randomSecret[:]); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JWT_SECRET", hex.EncodeToString(randomSecret[:]))
	issuer, err := NewIssuerFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	tests := []struct {
		name   string
		claims jwt.RegisteredClaims
		method jwt.SigningMethod
	}{
		{"expired", jwt.RegisteredClaims{Subject: "42", IssuedAt: jwt.NewNumericDate(now.Add(-2 * time.Hour)), ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)), ID: "test"}, jwt.SigningMethodHS256},
		{"missing jti", jwt.RegisteredClaims{Subject: "42", IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}, jwt.SigningMethodHS256},
		{"missing iat", jwt.RegisteredClaims{Subject: "42", ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)), ID: "test"}, jwt.SigningMethodHS256},
		{"invalid sub", jwt.RegisteredClaims{Subject: "invalid", IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)), ID: "test"}, jwt.SigningMethodHS256},
		{"wrong algorithm", jwt.RegisteredClaims{Subject: "42", IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)), ID: "test"}, jwt.SigningMethodHS384},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := jwt.NewWithClaims(test.method, test.claims).SignedString(issuer.secret)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := issuer.Verify(token); err == nil {
				t.Fatal("ожидался отказ для недействительного JWT")
			}
		})
	}
}
