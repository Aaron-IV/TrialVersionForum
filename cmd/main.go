package main

import (
	"fmt"
	"net/http"

	"forum/internal/handlers"
)

func main() {
	// Подключаем статические файлы (CSS, JS, картинки и т.д.)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("internal/static"))))

	// Главная страница
	http.HandleFunc("/", handlers.HomeHandler)

	// Фильтры (для фильтрации постов)
	http.HandleFunc("/filters", handlers.FilterHandler)

	// Страница входа / регистрации
	// http.HandleFunc("/login", handlers.LoginHandler)
	// http.HandleFunc("/register", handlers.RegisterHandler)

	fmt.Println("🚀 Сервер запущен на http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Ошибка при запуске сервера:", err)
	}
}
