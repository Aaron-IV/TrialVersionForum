# Проект: Forum — Полная документация

**Цель проекта:** создать веб-форум, который позволяет:

- общаться пользователям (создавать посты и комментарии);
- ассоциировать категории с постами;
- ставить лайки/дизлайки постам и комментариям;
- фильтровать посты по категориям, по созданным постам (для текущего пользователя), по понравившимся (лайкнутым) постам.

**Хранилище данных:** SQLite.

**Аутентификация:**
- Регистрация: email, username, password.
- Проверка уникальности email (если занят — возвращать ошибку).
- Хранить пароли (рекомендуется хеширование bcrypt — бонус).
- Использовать сессии через cookies (каждый пользователь имеет только одну открытую сессию). Сессия должна иметь дату истечения.
- Дополнительно (бонус): UUID для идентификаторов сессий.

**Коммуникация:**
- Зарегистрированные пользователи создают посты и комментарии.
- Постам можно присоединять одну или несколько категорий.
- Посты и комментарии видны всем (включая гостей).
- Только зарегистрированные могут создавать и лайкать/дизлайкать.
- Количество лайков/дизлайков видно всем.

**Фильтрация:**
- Фильтровать по категориям (как сабфорумы).
- Для зарегистрированных: фильтр по созданным постам и по понравившимся (liked) постам.

**Docker:** приложение должно быть контейнеризовано с использованием Docker.

**Требования к коду:**
- Использовать SQLite.
- Обрабатывать HTTP и технические ошибки, возвращать корректные HTTP статусы.
- Следовать хорошим практикам Go.
- Рекомендуется иметь юнит-тесты.

**Разрешенные пакеты:** стандартные пакеты Go, sqlite3 (github.com/mattn/go-sqlite3), bcrypt (golang.org/x/crypto/bcrypt), gofrs/uuid или google/uuid.

**Запрещено:** фронтенд фреймворки типа React/Angular/Vue.

**Ресурсы (ссылки):**
- SQLite: https://www.sqlite.org/index.html
- ER-диаграммы: https://www.smartdraw.com/entity-relationship-diagram/
- Документация go-sqlite3: https://github.com/mattn/go-sqlite3
- bcrypt: https://pkg.go.dev/golang.org/x/crypto/bcrypt
- uuid: https://github.com/gofrs/uuid или https://github.com/google/uuid
- Docker: https://docs.docker.com/get-started/
- OWASP сессии: https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html#session-management-waf-protections
- Cookie RFC и MDN: https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies

---

## 2. Разбивка проекта на маленькие шаги (микрозадачи)

Я разбил проект на логические блоки и внутри — на мелкие шаги. Выполняйте по порядку, проверяя каждый шаг через `go test` и ручные проверки.

### A. Подготовительный этап
1. Создать репозиторий и инициализировать `go mod init github.com/yourname/forum`.
2. Создать папку проекта и базовую структуру (см. раздел "Структура проекта").
3. Подготовить Dockerfile и `.dockerignore`.

### B. Настройка базы данных (SQLite)
1. Добавить зависимость `github.com/mattn/go-sqlite3`.
2. Создать файл `migrations.sql` с CREATE TABLE запросами.
3. Реализовать инициализацию БД в коде (функция `InitDB()`), которая выполняет миграции при старте.
4. Добавить в миграции хотя бы один `INSERT` тестовый (например, категории).
5. Проверить, что `SELECT` работает (написать маленькую утилиту или тест).

### C. Модели данных и ER-диаграмма
1. Определить сущности: users, sessions, posts, comments, categories, post_categories (many-to-many), likes (для постов и комментариев), возможно отдельные tables: post_likes, comment_likes.
2. Создать ER-диаграмму (на бумаге/в draw.io) и документировать её.
3. Для каждой сущности описать поля и типы.

### D. Аутентификация и сессии
1. Реализовать регистрацию: валидация email/username/password, проверка уникальности email.
2. Хеширование пароля с bcrypt при регистрации.
3. Реализовать вход: проверка email + password, создание записи в sessions с UUID и expiration.
4. Поставить cookie со значением session UUID и `HttpOnly`, `Secure` (в продакшн), `SameSite`.
5. Мидлварь (middleware) для проверки сессии и «прокидывания» текущего пользователя в контекст запроса.
6. Реализовать выход (logout) — удаление сессии и очистка cookie.

### E. CRUD для постов и комментариев
1. Маршруты: создание поста (POST), получение списка постов (GET), получение поста по id (GET), редактирование (PUT/PATCH) и удаление (DELETE) — редактирование/удаление только для автора.
2. Комментарии: POST к `/posts/:id/comments`, получение комментариев для поста, редактирование/удаление — только автор.
3. При создании поста — возможность указать список категорий (в запросе: массив id категорий).

### F. Лайки/Дизлайки
1. Таблицы `post_likes` и `comment_likes` с полями: id, user_id, post_id/comment_id, value (1 = like, -1 = dislike) или отдельные флаги.
2. Эндпоинты: POST `/posts/:id/like` и `/posts/:id/dislike` — переключение/установка.
3. Подсчет лайков/дизлайков при запросе поста/комментариев (агрегация в SQL).

### G. Фильтрация
1. Фильтр по категории: `/posts?category_id=3`.
2. Фильтр по созданным постам пользователя: `/posts?mine=true` (только для залогиненных).
3. Фильтр по liked posts: `/posts?liked=true` (только для залогиненных).
4. Поддержать комбинирование фильтров.

### H. UI (минимальный, без фронтенд-фреймворков)
1. Простые HTML-шаблоны (Go `html/template`) для: главной страницы, формы регистрации/входа, создания поста, просмотра поста.
2. CSS — минимальный, встроенный или через static files.
3. JavaScript — минимальный, только для удобства (fetch для лайков), но все должно работать без JS.

### I. Тестирование
1. Юнит-тесты для логики: хеширования пароля, создания сессии, проверок прав.
2. Интеграционные тесты для маршрутов (используя httptest).

### J. Docker
1. Написать `Dockerfile` и `docker-compose.yml` (опционально) для запуска приложения.
2. Убедиться, что sqlite-файл монтируется в volume или сохраняется в контейнере корректно.

### K. Документация и README
1. Подготовить README с инструкциями по запуску, миграциям, тестам, env-переменным.
2. Сгенерировать этот файл (ниже предоставлен пример).

---

## 3. Структура проекта (файлы и папки)

forum/
│
├── cmd/
│   └── main.go               # Точка входа в приложение
│
├── internal/
│   ├── handlers/             # HTTP-обработчики (контроллеры)
│   │   ├── home.go
│   │   ├── auth.go
│   │   └── posts.go
│   │
│   ├── templates/            # HTML шаблоны
│   │   ├── layout.html
│   │   ├── home.html
│   │   ├── login.html
│   │   ├── register.html
│   │   └── post.html
│   │
│   ├── static/               # Статические файлы (CSS, JS, изображения)
│   │   ├── css/
│   │   │   └── style.css
│   │   ├── js/
│   │   │   └── main.js
│   │   └── img/
│   │       └── logo.png
│   │
│   └── utils/                # Утилиты (например, логирование)
│       └── logger.go
│
├── db/
│   ├── migrations.sql        # SQL-миграции для базы данных
│   └── initdb.go             # Инициализация базы
│
├── docker-compose.yml        # Подключение Docker
├── Dockerfile                # Образ Go-приложения
└── README.md

└─ Forum_Project_Documentation.md  # этот документ
```

**Почему такая структура?**
- `cmd/server/main.go` — стандартный подход для Go-проектов.
- `internal/` — код, не предназначенный для использования внешними пакетами.
- Разделение на `models`, `repository`, `handlers` и `middleware` делает код модульным и легче тестируемым.

---

## 4. Выбор технологий и обоснование

- **Go (Golang)** — выбран по заданию. Быстрый, статически типизированный, удобен для веб-серверов.
- **SQLite** — легковесная встроенная БД, не требует отдельного сервера; идеальна для учебного проекта.
- **github.com/mattn/go-sqlite3** — де-факто адаптер для работы SQLite в Go.
- **bcrypt (golang.org/x/crypto/bcrypt)** — стандарт для хеширования паролей.
- **gofrs/uuid или google/uuid** — для генерации UUID, удобны для идентификаторов сессий.
- **Docker** — контейнеризация для воспроизводимости окружения и удобства развёртывания.
- **html/template** — безопасные шаблоны для рендеринга HTML на сервере (без фронтенд-фреймворков).

**Почему такие компоненты?**
- Они просты, хорошо документированы и покрыты большим сообществом.
- Использование SQLite уменьшает сложность инфраструктуры — нет необходимости поднимать отдельный СУБД при разработке.
- bcrypt защищает пароли от утечки в случае компрометации БД.

---

## 5. Архитектура (логическая и физическая)

### Логическая архитектура (уровни)

1. **HTTP Handlers (handlers/):** принимают HTTP-запросы, валидируют вход, вызывают сервисы/репозиторий.
2. **Service/Repository (repository/):** работа с БД — SQL-запросы, транзакции.
3. **Models (models/):** структуры данных.
4. **Templates/Static:** представление данных пользователю.

Поток простого запроса "создать пост":
- Handler получает POST /posts, валидирует тело и проверяет сессию через middleware;
- Handler вызывает repository.CreatePost(userID, title, body, categories);
- Repository выполняет INSERT в posts и INSERTs в post_categories;
- Handler возвращает 201 Created и редирект или JSON с данными поста.

### Физическая архитектура

- Приложение — один Docker-контейнер.
- Файл SQLite (`forum.db`) хранится либо в volume (рекомендуется для сохранности), либо в каталоге проекта (для простоты).

---

## 6. Схема базы данных (пример SQL)

Ниже — пример миграций (`migrations.sql`). Он содержит `CREATE`, `INSERT`, и пример `SELECT` в комментарии.

```sql
PRAGMA foreign_keys = ON;

CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT NOT NULL UNIQUE,
  username TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
  id TEXT PRIMARY KEY, -- UUID
  user_id INTEGER NOT NULL,
  expires_at DATETIME NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE categories (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE
);

CREATE TABLE posts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE post_categories (
  post_id INTEGER NOT NULL,
  category_id INTEGER NOT NULL,
  PRIMARY KEY (post_id, category_id),
  FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
  FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE CASCADE
);

CREATE TABLE comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  post_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  body TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE post_likes (
  user_id INTEGER NOT NULL,
  post_id INTEGER NOT NULL,
  value INTEGER NOT NULL CHECK(value IN (1, -1)),
  PRIMARY KEY (user_id, post_id),
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE TABLE comment_likes (
  user_id INTEGER NOT NULL,
  comment_id INTEGER NOT NULL,
  value INTEGER NOT NULL CHECK(value IN (1, -1)),
  PRIMARY KEY (user_id, comment_id),
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY(comment_id) REFERENCES comments(id) ON DELETE CASCADE
);

-- Пример INSERT (тестовые категории)
INSERT INTO categories (name) VALUES ('General'), ('News'), ('Programming');

-- Пример SELECT (получить посты с количеством лайков)
-- SELECT p.*, COALESCE(SUM(pl.value), 0) as score FROM posts p
-- LEFT JOIN post_likes pl ON p.id = pl.post_id
-- GROUP BY p.id;
```

---

## 7. Примеры HTTP API (REST)

Все ответы — JSON или HTML (в зависимости от Accept и конечной страницы).

| Метод | Путь | Описание |
|---|---:|---|
| POST | /register | Регистрация (email, username, password) |
| POST | /login | Вход (email, password) -> ставит cookie-сессию |
| POST | /logout | Удалить сессию |
| GET | /posts | Список постов (фильтры: ?category_id=&mine=&liked=) |
| POST | /posts | Создать пост (только залогиненные) |
| GET | /posts/:id | Получить пост и комментарии |
| POST | /posts/:id/comments | Добавить комментарий (только залогиненные) |
| POST | /posts/:id/like | Поставить like/dislike (payload: {value:1 or -1}) |


**Примеры тел запросов:**

- Регистрация `POST /register` JSON:
```json
{ "email": "a@b.com", "username": "alex", "password": "secret" }
```

- Лайк `POST /posts/5/like` JSON:
```json
{ "value": 1 }
```

---

## 8. Примеры кода (фрагменты) — помощь новичку

### A. Инициализация БД (Go, упрощённо)

```go
package db

import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "log"
)

func InitDB(path string) (*sql.DB, error) {
    db, err := sql.Open("sqlite3", path)
    if err != nil { return nil, err }
    // Можно выполнять миграции здесь: читать migrations.sql и exec
    return db, nil
}
```

### B. Хеширование пароля

```go
import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(hash), err
}

func CheckPasswordHash(hash, password string) error {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
```

### C. Создание сессии с UUID

```go
import (
  "time"
  "github.com/google/uuid"
)

func CreateSession(db *sql.DB, userID int) (string, error) {
  id := uuid.New().String()
  expires := time.Now().Add(24 * time.Hour)
  _, err := db.Exec(`INSERT INTO sessions(id, user_id, expires_at) VALUES (?, ?, ?)`, id, userID, expires)
  if err != nil { return "", err }
  return id, nil
}
```

---

## 9. Безопасность и лучшие практики

1. Хешировать пароли с bcrypt (не хранить plaintext).
2. Session cookie: `HttpOnly`, `SameSite=Lax` (или Strict), в продакшне `Secure`.
3. Устанавливать срок жизни сессии и удалять старые сессии.
4. Использовать проверку CSRF для форм (или ограничиться `SameSite` и токенами для важной логики).
5. Валидировать и очищать пользовательский ввод перед отображением (html/template автоматически экранирует).
6. Ограничивать размер загружаемых данных (например тело поста).
7. Логи: не логируйте пароли и секреты.

---

## 10. Dockerfile (пример)

```
# syntax=docker/dockerfile:1
FROM golang:1.20-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /forum ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=build /forum /app/forum
COPY --from=build /app/migrations.sql /app/migrations.sql
# Создаём папку для БД
VOLUME ["/app/data"]
EXPOSE 8080
CMD ["/app/forum"]
```

---

## 11. Тесты (рекомендации)

- Используйте `testing` и `net/http/httptest`.
- Мокируйте DB через временную sqlite в памяти `:memory:` для быстрых тестов.
- Тесты для: регистрации (duplicate email case), логина (wrong password), создание поста (auth required), лайков (toggle).

---

## 12. План по этапам разработки (пошагово)

1. Инициализация проекта, структура, go.mod.
2. Написать миграции и InitDB, убедиться, что БД создаётся.
3. Реализовать модели и репозиторий для users и sessions.
4. Реализовать регистрацию и login, cookie-сессии.
5. Добавить CRUD постов и категорий.
6. Добавить комментарии.
7. Реализовать лайки/дизлайки.
8. Добавить фильтрацию.
9. Создать простые HTML-шаблоны.
10. Написать тесты.
11. Dockerize.
12. Подготовить README.

---

## 13. README (готовый шаблон)

```markdown
# Forum — simple web forum in Go

## Требования
- Go 1.20+
- Docker (опционально)

## Установка и запуск локально
1. Клонируйте репозиторий
2. `go mod download`
3. Создайте БД и выполните миграции (если не реализована автоиммиграция):
   `sqlite3 forum.db < internal/db/migrations.sql`
4. `go run ./cmd/server`
5. Откройте http://localhost:8080

## Docker
`docker build -t forum .`
`docker run -p 8080:8080 -v $(pwd)/data:/app/data forum`

## Тесты
`go test ./...`

## Архитектура и структура проекта
(описание: internal/, cmd/, и т.д.)

## API
(список маршрутов)

## Безопасность
(рекомендации по хранению секретов, cookie и т.д.)
```

---

## 14. Что я сделал сейчас

- Перевёл задание на русский язык.
- Подробно разбил проект на множество мелких шагов.
- Описал структуру проекта, выбор технологий и архитектуру, обосновал выбор.
- Привёл SQL миграции, примеры кода, примеры API и Dockerfile.
- Подготовил готовый шаблон README.

---

## 15. Что дальше (предложения)

- Хотите, чтобы я сгенерировал исходную заготовку (boilerplate) проекта — файлы `main.go`, `db.go`, примеры handlers, migrations.sql, Dockerfile и README? Я могу создать их прямо сейчас в виде архива, чтобы вы скачали и открыли.
- Могу пошагово проводить вас через реализацию каждого шага — писать код, объяснять каждую строчку.

---

## 16. Контакты и ресурсы для чтения
- SQLite documentation: https://www.sqlite.org/index.html
- Go docs: https://golang.org/pkg/
- bcrypt: https://pkg.go.dev/golang.org/x/crypto/bcrypt
- Docker docs: https://docs.docker.com/get-started/


---

Спасибо! Если вы хотите — я могу сейчас сгенерировать проект-скелет из файлов (boilerplate) и упаковать его в архив, который вы сможете скачать. Напишите, что предпочитаете: "только код-скелет" или "код + базовые HTML-шаблоны + миграции".

