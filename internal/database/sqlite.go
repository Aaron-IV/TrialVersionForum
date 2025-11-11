package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"forum/config" // Импортируем наш пакет config

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB // Глобальная переменная для подключения к БД

// InitDB инициализирует подключение к базе данных и создает необходимые таблицы.
// Теперь функция принимает конфиг.
func InitDB(cfg *config.Config) error {
	var err error
	DB, err = sql.Open("sqlite3", cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}

	log.Printf("Successfully connected to SQLite database using DSN: %s", cfg.Database.DSN)

	return createTables()
}

// createTables создает все необходимые таблицы в базе данных.
func createTables() error {
	schema := `
	PRAGMA foreign_keys = ON;

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		username TEXT NOT NULL UNIQUE COLLATE NOCASE,
		password TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS post_categories (
		post_id INTEGER NOT NULL,
		category_id INTEGER NOT NULL,
		PRIMARY KEY (post_id, category_id),
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
		FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS post_reactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		is_like BOOLEAN NOT NULL, -- 1 for like, 0 for dislike
		dislike BOOLEAN NOT NULL DEFAULT 0, -- 1 for dislike, 0 for like (synced with is_like)
		UNIQUE (post_id, user_id),
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS comment_reactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		comment_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		is_like BOOLEAN NOT NULL, -- 1 for like, 0 for dislike
		dislike BOOLEAN NOT NULL DEFAULT 0, -- 1 for dislike, 0 for like (synced with is_like)
		UNIQUE (comment_id, user_id),
		FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		uuid TEXT NOT NULL UNIQUE,
		expires DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- Note: Length validation is handled in Go code using utf8.RuneCountInString()
	-- to properly count Unicode characters (including emoji, Chinese, etc.)
	-- SQLite length() counts bytes, not characters, so triggers are removed

	-- Helpful indexes
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_username_nocase ON users(username COLLATE NOCASE);
	CREATE INDEX IF NOT EXISTS idx_posts_user_created ON posts(user_id, created_at);
	CREATE INDEX IF NOT EXISTS idx_post_categories_category ON post_categories(category_id);
	CREATE INDEX IF NOT EXISTS idx_comments_post_created ON comments(post_id, created_at);
	CREATE INDEX IF NOT EXISTS idx_sessions_uuid ON sessions(uuid);
	CREATE INDEX IF NOT EXISTS idx_post_reactions_post_user ON post_reactions(post_id, user_id);
	CREATE INDEX IF NOT EXISTS idx_comment_reactions_comment_user ON comment_reactions(comment_id, user_id);
	`

	_, err := DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("error creating tables: %w", err)
	}

	log.Println("Database tables created or already exist.")

	// Исправляем последовательности ID для всех таблиц с AUTOINCREMENT
	if err := fixSequences(); err != nil {
		log.Printf("Warning: Failed to fix sequences: %v", err)
	}

	// Применяем миграции
	if err := applyMigrations(); err != nil {
		log.Printf("Warning: Failed to apply migrations: %v", err)
	}

	return insertDefaultCategories()
}

// fixSequences исправляет последовательности ID в sqlite_sequence,
// чтобы они соответствовали максимальным ID в таблицах.
// Это предотвращает проблемы, когда последовательность не синхронизирована с данными.
func fixSequences() error {
	tables := []string{"users", "categories", "posts", "comments", "post_reactions", "comment_reactions", "sessions"}

	for _, table := range tables {
		// Проверяем, существует ли запись в sqlite_sequence
		var exists bool
		err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_sequence WHERE name = ?)", table).Scan(&exists)
		if err != nil {
			return fmt.Errorf("error checking sequence for %s: %w", table, err)
		}

		if exists {
			// Обновляем последовательность до максимального ID
			query := fmt.Sprintf(`
				UPDATE sqlite_sequence 
				SET seq = (SELECT COALESCE(MAX(id), 0) FROM %s) 
				WHERE name = ?`, table)
			_, err := DB.Exec(query, table)
			if err != nil {
				return fmt.Errorf("error fixing sequence for %s: %w", table, err)
			}
		}
	}

	log.Println("Sequences fixed.")
	return nil
}

// applyMigrations применяет миграции базы данных
func applyMigrations() error {
	// Миграция: добавление колонки dislike в post_reactions
	if err := addColumnIfNotExists("post_reactions", "dislike", "BOOLEAN NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("error adding dislike column to post_reactions: %w", err)
	}

	// Миграция: добавление колонки dislike в comment_reactions
	if err := addColumnIfNotExists("comment_reactions", "dislike", "BOOLEAN NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("error adding dislike column to comment_reactions: %w", err)
	}

	// Синхронизируем данные: dislike = NOT is_like
	if err := syncDislikeColumn("post_reactions"); err != nil {
		return fmt.Errorf("error syncing dislike column in post_reactions: %w", err)
	}

	if err := syncDislikeColumn("comment_reactions"); err != nil {
		return fmt.Errorf("error syncing dislike column in comment_reactions: %w", err)
	}

	// Миграция: удаление старых триггеров для валидации длины
	// Валидация теперь выполняется в Go коде с правильным подсчетом Unicode символов
	triggersToDrop := []string{
		"posts_title_len_ins",
		"posts_title_len_upd",
		"posts_content_len_ins",
		"posts_content_len_upd",
	}
	for _, triggerName := range triggersToDrop {
		_, err := DB.Exec(fmt.Sprintf("DROP TRIGGER IF EXISTS %s", triggerName))
		if err != nil {
			// Логируем, но не останавливаем выполнение, если триггер не существует
			log.Printf("Note: Could not drop trigger %s (may not exist): %v", triggerName, err)
		}
	}

	log.Println("Migrations applied successfully.")
	return nil
}

// addColumnIfNotExists добавляет колонку в таблицу, если она не существует
func addColumnIfNotExists(tableName, columnName, columnDef string) error {
	// Проверяем, существует ли колонка
	var exists int
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = '%s'
	`, tableName, columnName)
	err := DB.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("error checking column existence: %w", err)
	}

	if exists == 0 {
		// Колонка не существует, добавляем её
		alterQuery := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, columnDef)
		_, err := DB.Exec(alterQuery)
		if err != nil {
			return fmt.Errorf("error adding column: %w", err)
		}
		log.Printf("Added column %s to table %s", columnName, tableName)
	} else {
		log.Printf("Column %s already exists in table %s", columnName, tableName)
	}

	return nil
}

// syncDislikeColumn синхронизирует колонку dislike с is_like (dislike = NOT is_like)
func syncDislikeColumn(tableName string) error {
	// Обновляем все записи, где dislike не синхронизирован с is_like
	query := fmt.Sprintf(`
		UPDATE %s 
		SET dislike = CASE WHEN is_like = 0 THEN 1 ELSE 0 END
		WHERE dislike != CASE WHEN is_like = 0 THEN 1 ELSE 0 END
	`, tableName)
	result, err := DB.Exec(query)
	if err != nil {
		return fmt.Errorf("error syncing dislike column: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Synced %d rows in table %s", rowsAffected, tableName)
	}
	return nil
}

// insertDefaultCategories добавляет стандартные категории, если их нет.
func insertDefaultCategories() error {
	categories := []string{"General", "Programming", "Offtopic", "Tech", "Science"}
	for _, name := range categories {
		_, err := DB.Exec("INSERT OR IGNORE INTO categories (name) VALUES (?)", name)
		if err != nil {
			return fmt.Errorf("error inserting category '%s': %w", name, err)
		}
	}
	log.Println("Default categories ensured.")
	return nil
}

// CleanupExpiredSessions удаляет просроченные сессии из БД.
func CleanupExpiredSessions() {
	ticker := time.NewTicker(30 * time.Minute) // Проверять каждые 30 минут
	defer ticker.Stop()

	for range ticker.C {
		result, err := DB.Exec("DELETE FROM sessions WHERE expires < ?", time.Now())
		if err != nil {
			log.Printf("Error cleaning up expired sessions: %v", err)
			continue
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			log.Printf("Cleaned up %d expired sessions.", rowsAffected)
		}
	}
}

// HashPassword хеширует пароль с использованием bcrypt.
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

// CheckPasswordHash сравнивает хешированный пароль с обычным.
func CheckPasswordHash(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
