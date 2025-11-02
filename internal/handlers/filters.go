package handlers

import (
	"html/template"
	"log"
	"net/http"
)

// FilterHandler обрабатывает фильтрацию постов
func FilterHandler(w http.ResponseWriter, r *http.Request) {
	sortBy := r.URL.Query().Get("sort")
	category := r.URL.Query().Get("category")

	log.Printf("Фильтр: sort=%s, category=%s", sortBy, category)

	// Здесь ты позже добавишь логику: выбор постов из БД с учетом фильтров

	tmpl := template.Must(template.ParseFiles(
		"internal/templates/layout.html",
		"internal/templates/home.html",
		"internal/templates/partials/filter.html",
	))

	// Пока просто отображаем фильтры и тестовые данные
	data := struct {
		Title string
	}{
		Title: "Результаты фильтрации",
	}

	tmpl.ExecuteTemplate(w, "layout", data)
}
