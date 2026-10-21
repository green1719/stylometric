package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeBlacklist struct {
	revoked bool
	err     error
}

func (blacklist fakeBlacklist) IsRevoked(context.Context, string) (bool, error) {
	return blacklist.revoked, blacklist.err
}

func (fakeBlacklist) Revoke(context.Context, VerifiedToken) (bool, error) {
	return true, nil
}

func TestJWTMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var randomSecret [32]byte
	if _, err := rand.Read(randomSecret[:]); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JWT_SECRET", hex.EncodeToString(randomSecret[:]))
	issuer, err := NewIssuerFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	token, err := issuer.Issue(42)
	if err != nil {
		t.Fatal(err)
	}
	knownUser := func(id uint) (bool, error) { return id == 42, nil }
	unknownUser := func(uint) (bool, error) { return false, nil }
	lastDot := strings.LastIndex(token, ".")
	invalidToken := token[:lastDot+1] + "x" + token[lastDot+2:]
	if token[lastDot+1] == 'x' {
		invalidToken = token[:lastDot+1] + "y" + token[lastDot+2:]
	}

	tests := []struct {
		name      string
		header    string
		required  bool
		lookup    UserExists
		blacklist Blacklist
		status    int
		userID    uint
	}{
		{"guest public", "", false, knownUser, fakeBlacklist{}, 200, 0},
		{"guest protected", "", true, knownUser, fakeBlacklist{}, 401, 0},
		{"valid protected", "Bearer " + token, true, knownUser, fakeBlacklist{}, 200, 42},
		{"invalid signature", "Bearer " + invalidToken, true, knownUser, fakeBlacklist{}, 401, 0},
		{"missing database user", "Bearer " + token, true, unknownUser, fakeBlacklist{}, 401, 0},
		{"invalid header", "Basic " + token, true, knownUser, fakeBlacklist{}, 401, 0},
		{"revoked", "Bearer " + token, true, knownUser, fakeBlacklist{revoked: true}, 401, 0},
		{"Redis unavailable", "Bearer " + token, true, knownUser, fakeBlacklist{err: errors.New("unavailable")}, 503, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			middleware := OptionalJWT(issuer, test.lookup, test.blacklist)
			if test.required {
				middleware = RequireJWT(issuer, test.lookup, test.blacklist)
			}
			router.GET("/probe", middleware, func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{"user_id": UserID(ctx)})
			})
			request := httptest.NewRequest(http.MethodGet, "/probe", nil)
			if test.header != "" {
				request.Header.Set("Authorization", test.header)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, ожидался %d", response.Code, test.status)
			}
			if response.Code == 200 && response.Body.String() != "{\"user_id\":"+strconv.FormatUint(uint64(test.userID), 10)+"}" {
				t.Fatalf("ответ: %s", response.Body.String())
			}
		})
	}
}
