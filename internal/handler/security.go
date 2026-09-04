package handler

import (
	"net/http"
	"strings"
)

// SecurityHeaders applies the baseline response headers shared by HTML, static,
// and API responses. HSTS is only emitted when the request is known to be HTTPS.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		// The vendored standard Alpine build evaluates x-* expressions. Replace it
		// with Alpine's CSP build before removing 'unsafe-eval' from this policy.
		header.Set("Content-Security-Policy", "default-src 'self'; base-uri 'self'; frame-ancestors 'none'; object-src 'none'; form-action 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://analytics.srmdn.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; media-src 'self' https:; connect-src 'self' https://analytics.srmdn.com; frame-src 'none'")
		header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(self), accelerometer=(self)")
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}
