package middleware

import "net/http"

// SecureHeadersMiddleware добавляет безопасные HTTP-заголовки для защиты от различных атак.
func SecureHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy (CSP)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline' https://getbootstrap.com https://cdn.jsdelivr.net; script-src 'self' https://getbootstrap.com https://cdn.jsdelivr.net; img-src 'self' data:; font-src 'self' https://getbootstrap.com https://cdn.jsdelivr.net; connect-src 'self' https://getbootstrap.com https://cdn.jsdelivr.net https://*.bootstrap.com; object-src 'none'")

		// X-Frame-Options: Защита от clickjacking.
		w.Header().Set("X-Frame-Options", "DENY")

		// X-XSS-Protection: Включает встроенный XSS-фильтр в браузерах.
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// X-Content-Type-Options: Предотвращает Mime-Type Sniffing.
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Referrer-Policy: Управляет информацией, отправляемой в заголовке Referer.
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Strict-Transport-Security (HSTS): Принудительное использование HTTPS.
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains") // 1 год

		next.ServeHTTP(w, r)
	})
}
