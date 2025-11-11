package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"forum/internal/auth"
)

// AuthMiddleware проверяет сессию пользователя и добавляет объект User в контекст запроса.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_token")
		if err != nil {
			// Куки нет, пользователь не аутентифицирован.
			// Просто продолжаем, User будет nil в контексте.
			next.ServeHTTP(w, r)
			return
		}

		user, err := auth.GetUserBySession(sessionCookie.Value)
		if err != nil {
			// Сессия недействительна или истекла. Очищаем куки.
			auth.ClearSessionCookie(w)
			log.Printf("Invalid or expired session for UUID: %s, error: %v", sessionCookie.Value, err)
			next.ServeHTTP(w, r) // Продолжаем без пользователя в контексте
			return
		}

		// Если сессия валидна, добавляем пользователя в контекст запроса
		ctx := context.WithValue(r.Context(), auth.UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuthMiddleware проверяет наличие аутентифицированного пользователя
// и перенаправляет на страницу логина, если пользователь не аутентифицирован.
// Для AJAX запросов возвращает JSON ошибку вместо редиректа.
func RequireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())
		if user == nil {
			// Проверяем, является ли запрос AJAX (по заголовку Accept или X-Requested-With)
			accept := r.Header.Get("Accept")
			isAJAX := r.Header.Get("X-Requested-With") == "XMLHttpRequest" ||
				strings.Contains(accept, "application/json")

			if isAJAX {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"Unauthorized"}`))
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}
