package middleware

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	maxRequests = 20          // Максимальное количество запросов за окно
	window      = time.Minute // Окно времени для сброса счетчика
)

// clientState хранит состояние rate limiter'а для каждого клиента
type clientState struct {
	lastRequest  time.Time
	requestCount int
	mu           sync.Mutex
}

var (
	clients = make(map[string]*clientState)
	mu      sync.Mutex // Мьютекс для доступа к map clients
	once    sync.Once
)

// RateLimitMiddleware ограничивает количество запросов от одного IP-адреса.
func RateLimitMiddleware(next http.Handler) http.Handler {
	// Запускаем горутину для очистки старых записей только один раз
	once.Do(func() {
		go cleanupClientStates()
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Исключаем реакции из rate limiting - пользователи должны иметь возможность быстро ставить/убирать лайки
		if r.URL.Path == "/like_post" || r.URL.Path == "/like_comment" {
			next.ServeHTTP(w, r)
			return
		}

		// Исключаем создание постов из rate limiting - валидационные ошибки не должны блокировать
		if r.URL.Path == "/create_post" && r.Method == http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Printf("Error splitting host port: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		mu.Lock()
		state, exists := clients[ip]
		if !exists {
			state = &clientState{}
			clients[ip] = state
		}
		mu.Unlock()

		state.mu.Lock()
		defer state.mu.Unlock()

		if time.Since(state.lastRequest) > window {
			// Сбросить счетчик, если окно времени прошло
			state.requestCount = 0
			state.lastRequest = time.Now()
		}

		state.requestCount++

		if state.requestCount > maxRequests {
			// Проверяем, является ли запрос AJAX
			accept := r.Header.Get("Accept")
			isAJAX := r.Header.Get("X-Requested-With") == "XMLHttpRequest" ||
				strings.Contains(accept, "application/json")

			if isAJAX {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "Too Many Requests",
				})
			} else {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

// cleanupClientStates периодически очищает map от старых записей, чтобы избежать утечек памяти
func cleanupClientStates() {
	for range time.Tick(window) { // Очищаем map каждое `window`
		mu.Lock()
		for ip, state := range clients {
			state.mu.Lock()
			// Удаляем, если не было активности в течение 2-х окон
			if time.Since(state.lastRequest) > 2*window {
				delete(clients, ip)
			}
			state.mu.Unlock()
		}
		mu.Unlock()
	}
}
