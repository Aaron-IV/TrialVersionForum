package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"forum/config"
	"forum/internal/database" // <--- ИСПРАВЛЕНО
	"forum/internal/models"   // <--- ИСПРАВЛЕНО

	"github.com/google/uuid"
)

const (
	SessionExpiration = 24 * time.Hour // Сессии действительны 24 часа
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
	ErrEmailExists     = errors.New("email already exists")
	ErrUsernameExists  = errors.New("username already exists")
	ErrInvalidInput    = errors.New("invalid input")
	ErrSessionNotFound = errors.New("session not found or expired")
)

// Regex patterns for validation
var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[\p{L}0-9_]{3,20}$`) // Unicode letters, numbers, underscore
	passwordRegex = regexp.MustCompile(`^.{6,32}$`)           // 6-32 characters
)

// ValidateUserCredentials проверяет входные данные при регистрации
func ValidateUserCredentials(email, username, password string) error {
	if !emailRegex.MatchString(email) || len(email) < 5 || len(email) > 50 {
		return fmt.Errorf("%w: invalid email format or length (5-50 characters)", ErrInvalidInput)
	}
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("%w: invalid username format or length (3-20 characters, letters, numbers, underscore only)", ErrInvalidInput)
	}
	if !passwordRegex.MatchString(password) {
		return fmt.Errorf("%w: invalid password format or length (6-32 characters)", ErrInvalidInput)
	}
	return nil
}

// RegisterUser регистрирует нового пользователя.
func RegisterUser(email, username, password string) (*models.User, error) {
	if err := ValidateUserCredentials(email, username, password); err != nil {
		return nil, err
	}

	hashedPassword, err := database.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to hash password: %w", err)
	}

	// Check if email or username already exists (case-insensitive for username)
	var count int
	err = database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ? OR username = ? COLLATE NOCASE", email, username).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to check existing user: %w", err)
	}
	if count > 0 {
		// More specific check to return the correct error
		err = database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("auth: failed to check existing email: %w", err)
		}
		if count > 0 {
			return nil, ErrEmailExists
		}
		return nil, ErrUsernameExists
	}

	res, err := database.DB.Exec("INSERT INTO users (email, username, password) VALUES (?, ?, ?)", email, username, hashedPassword)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to insert user: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("auth: failed to get last insert ID: %w", err)
	}

	return &models.User{ID: int(id), Email: email, Username: username}, nil
}

// LoginUser аутентифицирует пользователя и создает новую сессию.
func LoginUser(login, password string) (*models.User, *models.Session, error) {
	var user models.User
	query := "SELECT id, email, username, password FROM users WHERE email = ? OR username = ? COLLATE NOCASE"
	err := database.DB.QueryRow(query, login, login).Scan(&user.ID, &user.Email, &user.Username, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrUserNotFound
		}
		return nil, nil, fmt.Errorf("auth: failed to query user: %w", err)
	}

	// Проверяем пароль
	if err = database.CheckPasswordHash(user.Password, password); err != nil {
		// Логируем ошибку для отладки (не логируем сам пароль)
		fmt.Printf("DEBUG: Password check failed for user %s (ID: %d), error: %v\n", user.Username, user.ID, err)
		return nil, nil, ErrInvalidPassword
	}

	// Удаляем старые сессии для этого пользователя
	_, err = database.DB.Exec("DELETE FROM sessions WHERE user_id = ?", user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: failed to delete old sessions: %w", err)
	}

	// Создаем новую сессию
	sessionUUID := uuid.New().String()
	expiresAt := time.Now().Add(SessionExpiration)

	res, err := database.DB.Exec("INSERT INTO sessions (user_id, uuid, expires) VALUES (?, ?, ?)", user.ID, sessionUUID, expiresAt)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: failed to create new session: %w", err)
	}

	sessionID, err := res.LastInsertId()
	if err != nil {
		return nil, nil, fmt.Errorf("auth: failed to get session ID: %w", err)
	}

	session := &models.Session{ID: int(sessionID), UserID: user.ID, UUID: sessionUUID, Expires: expiresAt}
	return &user, session, nil
}

// LogoutUser удаляет сессию из базы данных.
func LogoutUser(sessionUUID string) error {
	result, err := database.DB.Exec("DELETE FROM sessions WHERE uuid = ?", sessionUUID)
	if err != nil {
		return fmt.Errorf("auth: failed to delete session: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("auth: failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrSessionNotFound
	}
	return nil
}

// GetUserBySession проверяет сессию и возвращает пользователя.
func GetUserBySession(sessionUUID string) (*models.User, error) {
	var session models.Session
	err := database.DB.QueryRow("SELECT user_id, expires FROM sessions WHERE uuid = ?", sessionUUID).Scan(&session.UserID, &session.Expires)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("auth: failed to query session: %w", err)
	}

	if time.Now().After(session.Expires) {
		// Сессия истекла, удаляем ее из БД
		_, _ = database.DB.Exec("DELETE FROM sessions WHERE uuid = ?", sessionUUID)
		return nil, ErrSessionNotFound
	}

	var user models.User
	err = database.DB.QueryRow("SELECT id, email, username FROM users WHERE id = ?", session.UserID).Scan(&user.ID, &user.Email, &user.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound // Пользователь сессии не найден, возможно, удален
		}
		return nil, fmt.Errorf("auth: failed to query user by session: %w", err)
	}

	return &user, nil
}

// SetSessionCookie устанавливает HTTP-cookie для сессии.
func SetSessionCookie(w http.ResponseWriter, sessionUUID string, expirationTime time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionUUID,
		Path:     "/",
		Expires:  expirationTime,
		HttpOnly: true,
		Secure:   config.AppConfig != nil && config.AppConfig.Server.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie очищает HTTP-cookie сессии.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Удаляет cookie немедленно
		HttpOnly: true,
		Secure:   config.AppConfig != nil && config.AppConfig.Server.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ContextKey для хранения User в контексте запроса
type contextKey string

const UserContextKey contextKey = "user"

// GetUserFromContext извлекает пользователя из контекста запроса.
func GetUserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}
