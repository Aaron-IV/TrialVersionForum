package handlers

import (
	"html/template"
	"log"
	"net/http"
)

// HomeHandler — отображает главную страницу форума
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	// Загружаем шаблоны
	tmpl := template.Must(template.ParseFiles(
		"internal/templates/layout.html",
		"internal/templates/home.html",
		"internal/templates/partials/filter.html", // подключаем фильтр
	))

	// В будущем сюда передадим реальные посты из БД
	data := struct {
		Title string
		Posts []string
	}{
		Title: "Главная страница форума",
		Posts: []string{
			"Пост 1: Привет, мир!",
			"Пост 2: Добро пожаловать в мой форум!",
			"Пост 3: Сегодня изучаем Golang!",
		},
	}

	// Выполняем шаблон
	err := tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		log.Println("Ошибка при рендеринге шаблона:", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
	}
}
