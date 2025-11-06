package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

// глобальная переменная для хранения шаблонов
var tmpl *template.Template

// Главная страница форума
func indexHandler(w http.ResponseWriter, r *http.Request) {
	err := tmpl.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Страница входа
func loginHandler(w http.ResponseWriter, r *http.Request) {
	err := tmpl.ExecuteTemplate(w, "login.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Страница регистрации
func registerHandler(w http.ResponseWriter, r *http.Request) {
	err := tmpl.ExecuteTemplate(w, "register.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Страница создания поста
func createPostHandler(w http.ResponseWriter, r *http.Request) {
	err := tmpl.ExecuteTemplate(w, "create_post.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Страница просмотра поста
func postHandler(w http.ResponseWriter, r *http.Request) {
	err := tmpl.ExecuteTemplate(w, "post.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Фильтрация постов
func filterHandler(w http.ResponseWriter, r *http.Request) {
	err := tmpl.ExecuteTemplate(w, "filter.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Главная функция
func main() {
	// Путь к HTML-шаблонам (относительно текущего файла)
	templatesDir := "../../web/templates"

	// Загружаем все шаблоны
	tmpl = template.Must(template.ParseGlob(filepath.Join(templatesDir, "*.html")))
	tmpl = template.Must(tmpl.ParseGlob(filepath.Join(templatesDir, "filter", "*.html")))

	// Подключаем CSS и JS
	fs := http.FileServer(http.Dir("../../web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Обработчики страниц
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/create", createPostHandler)
	http.HandleFunc("/post", postHandler)
	http.HandleFunc("/filter", filterHandler)

	// favicon (чтобы браузер не выдавал 404 на иконку)
	http.Handle("/favicon.ico", http.FileServer(http.Dir("../../web/assets")))

	// Запуск сервера
	log.Println("✅ Сервер запущен: http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("❌ Ошибка запуска сервера:", err)
	}
}
