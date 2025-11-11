package middleware

import "net/http"

// MethodOverrideMiddleware позволяет использовать PUT и DELETE запросы через HTML-формы.
func MethodOverrideMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Failed to parse form", http.StatusInternalServerError)
				return
			}
			method := r.Form.Get("_method")
			if method == http.MethodPut || method == http.MethodDelete {
				r.Method = method // Изменяем метод запроса
			}
		}
		next.ServeHTTP(w, r)
	})
}
