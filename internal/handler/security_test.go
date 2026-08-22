package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSecurityHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	plainRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	plainResponse := httptest.NewRecorder()
	SecurityHeaders(next).ServeHTTP(plainResponse, plainRequest)

	for _, header := range []string{
		"Content-Security-Policy",
		"Referrer-Policy",
		"Permissions-Policy",
		"X-Content-Type-Options",
		"X-Frame-Options",
	} {
		if plainResponse.Header().Get(header) == "" {
			t.Errorf("missing %s header", header)
		}
	}
	if got := plainResponse.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("HSTS on plain HTTP response = %q, want empty", got)
	}

	secureRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	secureRequest.Header.Set("X-Forwarded-Proto", "https")
	secureResponse := httptest.NewRecorder()
	SecurityHeaders(next).ServeHTTP(secureResponse, secureRequest)
	if got := secureResponse.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatal("missing HSTS on forwarded HTTPS response")
	}
}

func TestRateLimiterWindow(t *testing.T) {
	limiter := newRateLimiter(2, time.Minute)
	now := time.Date(2026, time.August, 22, 10, 0, 0, 0, time.UTC)
	if !limiter.Allow("client", now) || !limiter.Allow("client", now.Add(time.Second)) {
		t.Fatal("first two requests should be allowed")
	}
	if limiter.Allow("client", now.Add(2*time.Second)) {
		t.Fatal("third request in the window should be rejected")
	}
	if !limiter.Allow("client", now.Add(time.Minute)) {
		t.Fatal("request after the window should be allowed")
	}
}

func TestQuizClientKeyTrustsProxyIPOnlyFromLoopback(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/quiz/aqidah/leaderboard", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	request.Header.Set("X-Real-IP", "198.51.100.20")
	if got := quizClientKey(request); got != "192.0.2.10" {
		t.Fatalf("quizClientKey on direct request = %q, want remote IP", got)
	}

	request.RemoteAddr = "127.0.0.1:1234"
	if got := quizClientKey(request); got != "198.51.100.20" {
		t.Fatalf("quizClientKey behind loopback proxy = %q, want forwarded IP", got)
	}
}
