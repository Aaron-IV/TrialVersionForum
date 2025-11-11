#!/bin/bash

# Скрипт для сборки Docker образов и контейнеров форума

set -e

IMAGE_NAME="forum"
CONTAINER_NAME="forum-container"
PORT="${FORUM_PORT:-8080}"

# Функция для вывода справки
show_help() {
    echo "Использование: ./docker.sh [команда]"
    echo ""
    echo "Команды:"
    echo "  build       - Собрать Docker образ"
    echo "  run         - Запустить контейнер"
    echo "  stop        - Остановить контейнер"
    echo "  rm          - Удалить контейнер"
    echo "  logs        - Показать логи контейнера"
    echo "  restart     - Перезапустить контейнер"
    echo "  rebuild     - Пересобрать образ и перезапустить контейнер"
    echo "  clean       - Удалить контейнер и образ"
    echo "  help        - Показать эту справку"
    echo ""
    echo "Примеры:"
    echo "  ./docker.sh build       # Собрать образ"
    echo "  ./docker.sh run         # Запустить контейнер"
    echo "  ./docker.sh rebuild     # Пересобрать и перезапустить"
}

# Функция для сборки образа
build_image() {
    echo "🔨 Сборка Docker образа..."
    docker build -t $IMAGE_NAME .
    echo "✅ Образ $IMAGE_NAME успешно собран"
}

# Функция для запуска контейнера
run_container() {
    echo "🚀 Запуск контейнера..."
    
    # Создаем директорию data, если она не существует
    mkdir -p "$(pwd)/data"
    
    # Если база данных существует в корне, но не в data, копируем её
    if [ -f "$(pwd)/forum.db" ] && [ ! -f "$(pwd)/data/forum.db" ]; then
        echo "📦 Копирование базы данных из корня в data/..."
        cp "$(pwd)/forum.db" "$(pwd)/data/forum.db"
        echo "✅ База данных скопирована"
    fi
    
    # Проверяем, не запущен ли уже контейнер
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
            echo "⚠️  Контейнер $CONTAINER_NAME уже запущен"
            return
        else
            echo "🔄 Запуск существующего контейнера..."
            docker start $CONTAINER_NAME
        fi
    else
        echo "🆕 Создание нового контейнера..."
        docker run -d \
            --name $CONTAINER_NAME \
            -p $PORT:8080 \
            -v "$(pwd)/data:/app/data" \
            $IMAGE_NAME
    fi
    
    echo "✅ Контейнер $CONTAINER_NAME запущен"
    echo "🌐 Сервер доступен по адресу: http://localhost:$PORT"
}

# Функция для остановки контейнера
stop_container() {
    echo "🛑 Остановка контейнера..."
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        docker stop $CONTAINER_NAME
        echo "✅ Контейнер $CONTAINER_NAME остановлен"
    else
        echo "⚠️  Контейнер $CONTAINER_NAME не запущен"
    fi
}

# Функция для удаления контейнера
remove_container() {
    echo "🗑️  Удаление контейнера..."
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        docker stop $CONTAINER_NAME 2>/dev/null || true
        docker rm $CONTAINER_NAME
        echo "✅ Контейнер $CONTAINER_NAME удален"
    else
        echo "⚠️  Контейнер $CONTAINER_NAME не существует"
    fi
}

# Функция для просмотра логов
show_logs() {
    echo "📋 Логи контейнера $CONTAINER_NAME:"
    docker logs -f $CONTAINER_NAME
}

# Функция для перезапуска контейнера
restart_container() {
    echo "🔄 Перезапуск контейнера..."
    stop_container
    run_container
}

# Функция для пересборки и перезапуска
rebuild_all() {
    echo "🔄 Пересборка образа и перезапуск контейнера..."
    stop_container
    remove_container
    build_image
    run_container
}

# Функция для очистки (удаление контейнера и образа)
clean_all() {
    echo "🧹 Очистка Docker ресурсов..."
    stop_container
    remove_container
    if docker images --format '{{.Repository}}' | grep -q "^${IMAGE_NAME}$"; then
        docker rmi $IMAGE_NAME
        echo "✅ Образ $IMAGE_NAME удален"
    else
        echo "⚠️  Образ $IMAGE_NAME не существует"
    fi
}

# Основная логика
case "${1:-help}" in
    build)
        build_image
        ;;
    run)
        # Проверяем наличие образа перед запуском
        if ! docker images --format '{{.Repository}}' | grep -q "^${IMAGE_NAME}$"; then
            echo "⚠️  Образ не найден. Собираю образ..."
            build_image
        fi
        run_container
        ;;
    stop)
        stop_container
        ;;
    rm)
        remove_container
        ;;
    logs)
        show_logs
        ;;
    restart)
        restart_container
        ;;
    rebuild)
        rebuild_all
        ;;
    clean)
        clean_all
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo "❌ Неизвестная команда: $1"
        echo ""
        show_help
        exit 1
        ;;
esac

