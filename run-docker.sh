#!/bin/bash

# Скрипт для запуска standalone контейнера Schedule API
# Использование: ./run-docker.sh [имя_образа] [версия]

IMAGE_NAME=${1:-"your-dockerhub-username/schedule-api"}
VERSION=${2:-"latest"}
CONTAINER_NAME="schedule-api"

# Цвета для вывода
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Schedule API - Standalone Docker Container${NC}"
echo "=============================================="
echo ""

# Проверяем, запущен ли контейнер
if [ "$(docker ps -q -f name=$CONTAINER_NAME)" ]; then
    echo -e "${YELLOW}Контейнер $CONTAINER_NAME уже запущен${NC}"
    echo "Остановить контейнер? (y/n)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        docker stop $CONTAINER_NAME
        docker rm $CONTAINER_NAME
        echo -e "${GREEN}Контейнер остановлен${NC}"
    else
        exit 0
    fi
fi

# Удаляем старый контейнер если он существует
if [ "$(docker ps -aq -f name=$CONTAINER_NAME)" ]; then
    docker rm $CONTAINER_NAME > /dev/null 2>&1
fi

# Запрашиваем параметры MinIO
echo ""
echo "Введите параметры подключения к MinIO:"
echo -e "${YELLOW}(Нажмите Enter для использования значений по умолчанию)${NC}"
echo ""

read -p "MinIO Endpoint [host.docker.internal:9000]: " MINIO_ENDPOINT
MINIO_ENDPOINT=${MINIO_ENDPOINT:-host.docker.internal:9000}

read -p "MinIO Access Key [minioadmin]: " MINIO_ACCESS_KEY
MINIO_ACCESS_KEY=${MINIO_ACCESS_KEY:-minioadmin}

read -p "MinIO Secret Key [minioadmin]: " MINIO_SECRET_KEY
MINIO_SECRET_KEY=${MINIO_SECRET_KEY:-minioadmin}

read -p "Source Bucket [file-upload]: " SOURCE_BUCKET
SOURCE_BUCKET=${SOURCE_BUCKET:-file-upload}

read -p "Target Bucket [university-schedules]: " TARGET_BUCKET
TARGET_BUCKET=${TARGET_BUCKET:-university-schedules}

read -p "API Port [8080]: " API_PORT
API_PORT=${API_PORT:-8080}

# Запускаем контейнер
echo ""
echo -e "${GREEN}Запуск контейнера...${NC}"
echo "Образ: $IMAGE_NAME:$VERSION"
echo ""

docker run -d \
    --name $CONTAINER_NAME \
    -p $API_PORT:8080 \
    -e MINIO_ENDPOINT=$MINIO_ENDPOINT \
    -e MINIO_ACCESS_KEY=$MINIO_ACCESS_KEY \
    -e MINIO_SECRET_KEY=$MINIO_SECRET_KEY \
    -e MINIO_BUCKET=$TARGET_BUCKET \
    -e MINIO_USE_SSL=false \
    -e SOURCE_BUCKET=$SOURCE_BUCKET \
    -e TARGET_BUCKET=$TARGET_BUCKET \
    -e FILE_PATH_PATTERN="universities/%s/courses/%s/types/%s/files/%s" \
    -e CACHE_TTL_MINUTES=10 \
    -e PRESIGNED_URL_TTL_MINUTES=15 \
    -e ENVIRONMENT=production \
    $IMAGE_NAME:$VERSION

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✓ Контейнер успешно запущен!${NC}"
    echo ""
    echo "Информация о контейнере:"
    echo "  Имя: $CONTAINER_NAME"
    echo "  API: http://localhost:$API_PORT"
    echo "  Health check: http://localhost:$API_PORT/api/v1/health"
    echo ""
    echo "Полезные команды:"
    echo "  Просмотр логов:  docker logs -f $CONTAINER_NAME"
    echo "  Остановка:       docker stop $CONTAINER_NAME"
    echo "  Удаление:        docker rm $CONTAINER_NAME"
    echo "  Перезапуск:      docker restart $CONTAINER_NAME"
    echo ""
    
    # Ждем несколько секунд и проверяем health
    echo "Проверка состояния..."
    sleep 3
    
    if curl -s http://localhost:$API_PORT/api/v1/health > /dev/null 2>&1; then
        echo -e "${GREEN}✓ API работает корректно${NC}"
    else
        echo -e "${YELLOW}⚠ API еще запускается, проверьте логи: docker logs $CONTAINER_NAME${NC}"
    fi
else
    echo -e "${RED}✗ Ошибка при запуске контейнера${NC}"
    exit 1
fi

