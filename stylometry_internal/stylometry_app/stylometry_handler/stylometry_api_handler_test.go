package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"stylometry/stylometry_internal/stylometry_app/stylometry_auth"
)

type testLogoutBlacklist struct {
	saved bool
	err   error
}

func (testLogoutBlacklist) IsRevoked(context.Context, string) (bool, error) {
	return false, nil
}

func (blacklist testLogoutBlacklist) Revoke(context.Context, auth.VerifiedToken) (bool, error) {
	return blacklist.saved, blacklist.err
}

func TestLogoutRequiresSuccessfulBlacklistWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JWT_SECRET", hex.EncodeToString(secret[:]))
	issuer, err := auth.NewIssuerFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	token, err := issuer.Issue(42)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name      string
		blacklist testLogoutBlacklist
		status    int
	}{
		{"Redis write failed", testLogoutBlacklist{err: errors.New("Redis unavailable")}, http.StatusServiceUnavailable},
		{"already revoked", testLogoutBlacklist{}, http.StatusUnauthorized},
		{"saved", testLogoutBlacklist{saved: true}, http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			handler := &SampleHandler{Blacklist: test.blacklist}
			router.POST("/api/auth/logout", auth.RequireJWT(issuer, func(uint) (bool, error) { return true, nil }, test.blacklist), handler.APIPostAuthLogout)
			request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, ожидался %d", response.Code, test.status)
			}
		})
	}
}

func TestStylometrySplitDraftText(t *testing.T) {
	stylometry_author, stylometry_text, stylometry_err := stylometrySplitDraftText("  Лев Толстой  \r\nПервая строка.\r\nВторая строка.")
	if stylometry_err != nil {
		t.Fatalf("неожиданная ошибка: %v", stylometry_err)
	}
	if stylometry_author != "Лев Толстой" {
		t.Fatalf("автор = %q, ожидался %q", stylometry_author, "Лев Толстой")
	}
	if stylometry_text != "Первая строка.\nВторая строка." {
		t.Fatalf("текст = %q", stylometry_text)
	}
}

func TestStylometrySplitDraftTextRejectsMissingPart(t *testing.T) {
	for _, stylometry_input := range []string{"Лев Толстой", "\nТекст", "Лев Толстой\n   "} {
		if _, _, stylometry_err := stylometrySplitDraftText(stylometry_input); stylometry_err == nil {
			t.Fatalf("ожидалась ошибка для %q", stylometry_input)
		}
	}
}
