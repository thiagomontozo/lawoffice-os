package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testError(w http.ResponseWriter, _ *http.Request, status int, _, _ string) {
	w.WriteHeader(status)
}

func TestCORSRejectsForeignMutation(t *testing.T) {
	handler := CORS("https://office.example", testError)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected foreign mutation to be rejected, got %d", response.Code)
	}
}

func TestCORSAcceptsConfiguredOriginAndSetsSecurityHeaders(t *testing.T) {
	handler := CORS("https://office.example", testError)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters", nil)
	request.Header.Set("Origin", "https://office.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected configured origin to pass, got %d", response.Code)
	}
	if response.Header().Get("Content-Security-Policy") == "" || response.Header().Get("Permissions-Policy") == "" {
		t.Fatal("expected security headers")
	}
}

func TestRateLimitStopsExcessRequests(t *testing.T) {
	handler := RateLimit(2, time.Minute, testError)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	for attempt := 1; attempt <= 3; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/login", nil)
		request.RemoteAddr = "192.0.2.1:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if attempt < 3 && response.Code != http.StatusNoContent {
			t.Fatalf("attempt %d unexpectedly rejected", attempt)
		}
		if attempt == 3 && response.Code != http.StatusTooManyRequests {
			t.Fatalf("expected final attempt to be limited, got %d", response.Code)
		}
	}
}

func TestBearerTokenUsesExactCredential(t *testing.T) {
	handler := BearerToken("metrics-secret", testError)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, test := range []struct {
		name   string
		header string
		status int
	}{
		{name: "missing", status: http.StatusUnauthorized},
		{name: "wrong", header: "Bearer metrics-secret-extra", status: http.StatusUnauthorized},
		{name: "valid", header: "Bearer metrics-secret", status: http.StatusNoContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			request.Header.Set("Authorization", test.header)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("got status %d, want %d", response.Code, test.status)
			}
		})
	}
}
