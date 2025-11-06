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
	// Загружаем все HTML шаблоны из internal/web/templates
	pattern := filepath.Join("internal", "web", "templates", "*.html")
	templates = template.Must(template.ParseGlob(pattern))

	// Настраиваем маршруты
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/create", handleCreatePost)
	http.HandleFunc("/post", handleViewPost)

	// Подключаем статические файлы (CSS, JS, изображения)
	fs := http.FileServer(http.Dir("internal/web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Запускаем сервер
	port := ":8080"
	fmt.Println("✅ Server started on http://localhost" + port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}

// ================== HANDLERS ==================

func handleIndex(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index.html", nil)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		renderTemplate(w, "login.html", nil)
	case "POST":
		// Здесь будет логика авторизации (позже)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		renderTemplate(w, "register.html", nil)
	case "POST":
		// Здесь будет логика регистрации (позже)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func handleCreatePost(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		renderTemplate(w, "create_post.html", nil)
	case "POST":
		// Здесь будет логика создания поста (позже)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func handleViewPost(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "post.html", nil)
}

// ================== UTILS ==================

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
