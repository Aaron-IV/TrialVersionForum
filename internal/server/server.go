// Файл: internal/server/server.go

package server

import (
	"log"
	"net/http"
	"time"

	"forum/config"
	"forum/internal/database"
	"forum/internal/handlers"
	"forum/internal/middleware"
)

// applyMiddleware (остается без изменений)
func applyMiddleware(h http.Handler, m ...func(http.Handler) http.Handler) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}

func StartServer() {
	go database.CleanupExpiredSessions()

	mux := http.NewServeMux()

	// Регистрация маршрутов (остается без изменений)
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", handlers.ProtectStatic(fs)))

	mux.HandleFunc("/post/", handlers.PostDetailHandler)
	mux.HandleFunc("/login", handlers.LoginHandler)
	mux.HandleFunc("/register", handlers.RegisterHandler)

	mux.Handle("/logout", middleware.RequireAuthMiddleware(http.HandlerFunc(handlers.LogoutHandler)))
	mux.Handle("/create_post", middleware.RequireAuthMiddleware(http.HandlerFunc(handlers.CreatePostHandler)))

	editPostHandler := middleware.RequireAuthMiddleware(middleware.MethodOverrideMiddleware(http.HandlerFunc(handlers.EditPostHandler)))
	mux.Handle("/edit_post", editPostHandler)

	deletePostHandler := middleware.RequireAuthMiddleware(middleware.MethodOverrideMiddleware(http.HandlerFunc(handlers.DeletePostHandler)))
	mux.Handle("/delete_post", deletePostHandler)

	// soft delete removed

	editCommentHandler := middleware.RequireAuthMiddleware(middleware.MethodOverrideMiddleware(http.HandlerFunc(handlers.EditCommentHandler)))
	mux.Handle("/edit_comment", editCommentHandler)

	deleteCommentHandler := middleware.RequireAuthMiddleware(middleware.MethodOverrideMiddleware(http.HandlerFunc(handlers.DeleteCommentHandler)))
	mux.Handle("/delete_comment", deleteCommentHandler)

	likePostHandler := middleware.RequireAuthMiddleware(http.HandlerFunc(handlers.LikePostHandler))
	mux.Handle("/like_post", likePostHandler)

	likeCommentHandler := middleware.RequireAuthMiddleware(http.HandlerFunc(handlers.LikeCommentHandler))
	mux.Handle("/like_comment", likeCommentHandler)

	mux.HandleFunc("/", handlers.HomeHandler)

	// Применение глобальных middleware (остается без изменений)
	globalChain := applyMiddleware(mux,
		middleware.LoggerMiddleware,
		middleware.SecureHeadersMiddleware,
		middleware.AuthMiddleware,
	)

	cfg := config.AppConfig
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      globalChain,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// ----- ИЗМЕНЕНИЯ ЗДЕСЬ -----

	// 1. Меняем сообщение в логе - выводим прямую ссылку
	serverURL := "http://localhost" + srv.Addr
	log.Printf("Server starting on %s", serverURL)

	// 2. Заменяем ListenAndServeTLS на ListenAndServe
	err := srv.ListenAndServe() // БЫЛО: srv.ListenAndServeTLS(...)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
