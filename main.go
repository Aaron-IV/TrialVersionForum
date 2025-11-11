package main

import (
	"log"

	"forum/config"
	"forum/internal/database"
	"forum/internal/server"
)

func main() {
	// 1. Загружаем конфигурацию приложения
	config.LoadConfig()

	// 2. Инициализируем соединение с базой данных
	err := database.InitDB(config.AppConfig)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize database: %v", err)
	}

	// 3. Откладываем закрытие соединения с базой данных
	defer func() {
		if err := database.DB.Close(); err != nil {
			log.Printf("ERROR: Failed to close database connection: %v", err)
		} else {
			log.Println("Database connection closed successfully.")
		}
	}()

	// 4. Запускаем веб-сервер
	server.StartServer()
}
