package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

// глобальная переменная для шаблонов
var templates *template.Template

func main() {
	// Загружаем все шаблоны
	pattern := filepath.Join("internal", "web", "templates", "*.html")
	templates = template.Must(template.ParseGlob(pattern))

	// Роуты
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/create", handleCreatePost)
	http.HandleFunc("/post", handleViewPost)

	// Подключаем CSS, JS и т.д.
	fs := http.FileServer(http.Dir("internal/web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	port := ":8080"
	fmt.Println("✅ Сервер запущен на http://localhost" + port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("❌ Ошибка запуска: %v", err)
	}
}

// ================== HANDLERS ==================

func handleIndex(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Форум сообщества",
		"Posts": []map[string]interface{}{
			{
				"ID":         1,
				"Title":      "Добро пожаловать на форум!",
				"Content":    "Это ваш первый тестовый пост. Позже вы сможете добавлять собственные.",
				"Author":     "Admin",
				"Categories": []string{"Новости", "Объявления"},
				"Likes":      10,
				"Dislikes":   2,
			},
			{
				"ID":         2,
				"Title":      "Начинаем изучать Go!",
				"Content":    "Форум скоро будет поддерживать создание и редактирование постов.",
				"Author":     "GoFan",
				"Categories": []string{"GoLang", "Учеба"},
				"Likes":      5,
				"Dislikes":   0,
			},
		},
	}

	renderTemplate(w, "index.html", data)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "login.html", nil)
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "register.html", nil)
}

func handleCreatePost(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "create_post.html", nil)
}

func handleViewPost(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "post.html", nil)
}

// ================== UTILS ==================

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	// layout.html всегда подключается первым, а потом конкретный шаблон
	files := []string{
		filepath.Join("internal", "web", "templates", "layout.html"),
		filepath.Join("internal", "web", "templates", tmpl),
	}

	t, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.ExecuteTemplate(w, "layout", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
