# Многоэтапная сборка Docker образа для Forum приложения

# Этап 1: Сборка приложения
FROM golang:1.24-alpine AS builder

# Устанавливаем необходимые пакеты для сборки
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем весь исходный код
COPY . .

# Обновляем зависимости
RUN go mod tidy

# Собираем приложение
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o forum ./main.go

# Этап 2: Финальный образ
FROM alpine:latest

# Устанавливаем необходимые пакеты для запуска
RUN apk --no-cache add ca-certificates sqlite

# Создаем пользователя для запуска приложения
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем бинарник из этапа сборки
COPY --from=builder /app/forum .

# Копируем статические файлы
COPY --from=builder /app/static ./static

# Копируем шаблоны
COPY --from=builder /app/internal/templates ./internal/templates

# Создаем директорию для базы данных с правильными правами
RUN mkdir -p /app/data && \
    chown -R appuser:appgroup /app && \
    chmod -R 755 /app/data

# Переключаемся на непривилегированного пользователя
USER appuser

# Открываем порт
EXPOSE 8080

# Переменные окружения
ENV FORUM_PORT=8080
ENV FORUM_DB_NAME=/app/data/forum.db
ENV FORUM_SESSION_HOURS=24
ENV FORUM_COOKIE_SECURE=false

# Создаем entrypoint скрипт для проверки директории и прав доступа
RUN echo '#!/bin/sh' > /app/entrypoint.sh && \
    echo 'set -e' >> /app/entrypoint.sh && \
    echo 'mkdir -p /app/data' >> /app/entrypoint.sh && \
    echo 'chmod 755 /app/data 2>/dev/null || true' >> /app/entrypoint.sh && \
    echo 'exec ./forum' >> /app/entrypoint.sh && \
    chmod +x /app/entrypoint.sh

# Запускаем приложение через entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]

