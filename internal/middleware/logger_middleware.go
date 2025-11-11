package middleware

import (
	"log"
	"net/http"
	"time"
)

// LoggerMiddleware логирует информацию о каждом входящем HTTP-запросе.
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r) // Передаем запрос следующему обработчику
		log.Printf("Method: %s | URL: %s | Duration: %s | From: %s", r.Method, r.URL.Path, time.Since(start), r.RemoteAddr)
	})
}
