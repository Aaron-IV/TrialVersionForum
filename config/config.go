package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config содержит все конфигурационные параметры приложения.
type Config struct {
	Server struct {
		Port         string
		CertFile     string
		KeyFile      string
		CookieSecure bool
	}
	Database struct {
		DSN string // Data Source Name, например: "forum.db?_foreign_keys=on"
	}
	Session struct {
		Expiration time.Duration
	}
}

// AppConfig - это глобальная переменная для хранения загруженной конфигурации,
// доступная для всего приложения.
var AppConfig *Config

// LoadConfig загружает конфигурацию из переменных окружения или устанавливает значения по умолчанию.
// Эту функцию нужно вызвать один раз при старте приложения в main.go.
func LoadConfig() {
	AppConfig = &Config{}

	// --- Конфигурация Сервера ---
	AppConfig.Server.Port = getEnv("FORUM_PORT", "8080")
	AppConfig.Server.CertFile = getEnv("FORUM_CERT_FILE", "./certs/server.crt")
	AppConfig.Server.KeyFile = getEnv("FORUM_KEY_FILE", "./certs/server.key")
	// Cookie Secure: default false for local HTTP; set FORUM_COOKIE_SECURE=true in prod
	AppConfig.Server.CookieSecure = getEnv("FORUM_COOKIE_SECURE", "false") == "true"

	// --- Конфигурация Базы Данных ---
	dbName := getEnv("FORUM_DB_NAME", "forum.db")
	AppConfig.Database.DSN = dbName + "?_foreign_keys=on"

	// --- Конфигурация Сессий ---
	sessionHours, err := strconv.Atoi(getEnv("FORUM_SESSION_HOURS", "24"))
	if err != nil {
		log.Printf("WARNING: Invalid session duration specified. Using default 24 hours. Error: %v", err)
		sessionHours = 24
	}
	AppConfig.Session.Expiration = time.Duration(sessionHours) * time.Hour

	log.Println("Configuration loaded successfully.")
}

// getEnv - это вспомогательная функция для чтения переменной окружения.
// Если переменная не установлена, возвращается значение по умолчанию (fallback).
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
