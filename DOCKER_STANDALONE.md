# Schedule API - Standalone Docker Container

Документация по использованию Schedule API как независимого Docker контейнера.

## Публикация образа в Docker Hub

### 1. Настройка

Отредактируйте `Makefile` и укажите ваш Docker Hub username:

```makefile
DOCKER_USERNAME = your-dockerhub-username
```

### 2. Сборка и публикация

```bash
# Авторизация в Docker Hub
docker login

# Сборка, тегирование и публикация одной командой
make docker-publish VERSION=1.0.0

# Или поэтапно:
make docker-build VERSION=1.0.0    # Сборка образа
make docker-tag VERSION=1.0.0      # Тегирование
make docker-push VERSION=1.0.0     # Публикация в Docker Hub
```

Образ будет опубликован как: `your-dockerhub-username/schedule-api:1.0.0`

## Запуск standalone контейнера

### Вариант 1: Использование скрипта (рекомендуется)

```bash
# Сделайте скрипт исполняемым
chmod +x run-docker.sh

# Запустите скрипт
./run-docker.sh your-dockerhub-username/schedule-api latest
```

Скрипт интерактивно запросит все необходимые параметры.

### Вариант 2: Docker Run вручную

```bash
docker run -d \
  --name schedule-api \
  -p 8080:8080 \
  -e MINIO_ENDPOINT=your-minio-host:9000 \
  -e MINIO_ACCESS_KEY=minioadmin \
  -e MINIO_SECRET_KEY=minioadmin \
  -e MINIO_BUCKET=university-schedules \
  -e MINIO_USE_SSL=false \
  -e SOURCE_BUCKET=file-upload \
  -e TARGET_BUCKET=university-schedules \
  -e FILE_PATH_PATTERN="universities/%s/courses/%s/types/%s/files/%s" \
  -e CACHE_TTL_MINUTES=10 \
  -e PRESIGNED_URL_TTL_MINUTES=15 \
  -e ENVIRONMENT=production \
  your-dockerhub-username/schedule-api:latest
```

### Вариант 3: Использование Makefile

```bash
# Отредактируйте параметры в Makefile, затем:
make docker-run
```

## Переменные окружения

### Обязательные переменные

| Переменная | Описание | Пример |
|------------|----------|--------|
| `MINIO_ENDPOINT` | Адрес MinIO сервера | `minio.example.com:9000` |
| `MINIO_ACCESS_KEY` | Ключ доступа к MinIO | `minioadmin` |
| `MINIO_SECRET_KEY` | Секретный ключ MinIO | `minioadmin` |

### Опциональные переменные (с значениями по умолчанию)

| Переменная | Описание | Значение по умолчанию |
|------------|----------|----------------------|
| `SERVER_PORT` | Порт API внутри контейнера | `8080` |
| `MINIO_BUCKET` | Основной бакет | `university-schedules` |
| `MINIO_USE_SSL` | Использовать SSL для MinIO | `false` |
| `SOURCE_BUCKET` | Бакет с исходными XLSX | `file-upload` |
| `TARGET_BUCKET` | Бакет с JSON | `university-schedules` |
| `FILE_PATH_PATTERN` | Паттерн пути к файлам | `universities/%s/courses/%s/types/%s/files/%s` |
| `CACHE_TTL_MINUTES` | Время жизни кэша (мин) | `10` |
| `PRESIGNED_URL_TTL_MINUTES` | Время жизни presigned URL (мин) | `15` |
| `ENVIRONMENT` | Окружение (development/production) | `development` |

## Примеры использования

### Подключение к локальному MinIO (на хосте)

```bash
docker run -d \
  --name schedule-api \
  -p 8080:8080 \
  -e MINIO_ENDPOINT=host.docker.internal:9000 \
  -e MINIO_ACCESS_KEY=minioadmin \
  -e MINIO_SECRET_KEY=minioadmin \
  your-dockerhub-username/schedule-api:latest
```

**Note:** `host.docker.internal` - специальный DNS имя для доступа к хосту из контейнера (работает на Windows/Mac).

### Подключение к внешнему MinIO

```bash
docker run -d \
  --name schedule-api \
  -p 8080:8080 \
  -e MINIO_ENDPOINT=minio.example.com:9000 \
  -e MINIO_ACCESS_KEY=your-access-key \
  -e MINIO_SECRET_KEY=your-secret-key \
  -e MINIO_USE_SSL=true \
  your-dockerhub-username/schedule-api:latest
```

### Запуск с кастомными бакетами

```bash
docker run -d \
  --name schedule-api \
  -p 8080:8080 \
  -e MINIO_ENDPOINT=minio:9000 \
  -e MINIO_ACCESS_KEY=minioadmin \
  -e MINIO_SECRET_KEY=minioadmin \
  -e SOURCE_BUCKET=xlsx-files \
  -e TARGET_BUCKET=json-schedules \
  your-dockerhub-username/schedule-api:latest
```

### Запуск в той же сети с MinIO

```bash
# Создаем сеть
docker network create schedule-network

# Запускаем MinIO
docker run -d \
  --name minio \
  --network schedule-network \
  -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio:latest \
  server /data --console-address ":9001"

# Запускаем API
docker run -d \
  --name schedule-api \
  --network schedule-network \
  -p 8080:8080 \
  -e MINIO_ENDPOINT=minio:9000 \
  -e MINIO_ACCESS_KEY=minioadmin \
  -e MINIO_SECRET_KEY=minioadmin \
  your-dockerhub-username/schedule-api:latest
```

## Управление контейнером

```bash
# Просмотр логов
docker logs -f schedule-api

# Остановка контейнера
docker stop schedule-api

# Запуск остановленного контейнера
docker start schedule-api

# Перезапуск
docker restart schedule-api

# Удаление контейнера
docker rm schedule-api

# Удаление контейнера с остановкой
docker rm -f schedule-api
```

## Проверка работоспособности

```bash
# Health check
curl http://localhost:8080/api/v1/health

# Список университетов
curl http://localhost:8080/api/v1/universities

# Статистика контейнера
docker stats schedule-api
```

## Troubleshooting

### Контейнер не запускается

```bash
# Проверьте логи
docker logs schedule-api

# Проверьте, что порт не занят
netstat -an | grep 8080

# Проверьте статус контейнера
docker ps -a | grep schedule-api
```

### API не может подключиться к MinIO

1. Проверьте, что `MINIO_ENDPOINT` правильный:
   - Для локального MinIO: `host.docker.internal:9000` (Windows/Mac) или IP хоста (Linux)
   - Для MinIO в той же Docker сети: `minio:9000`
   - Для внешнего MinIO: полный адрес

2. Проверьте, что MinIO доступен:
```bash
# Изнутри контейнера
docker exec -it schedule-api sh
wget -qO- http://your-minio-endpoint:9000/minio/health/live
```

### Ошибка "connection refused"

- API не может достучаться до MinIO
- Проверьте `MINIO_ENDPOINT` - не используйте `localhost` внутри контейнера
- Убедитесь, что MinIO запущен и доступен

## Docker Compose vs Standalone

### Используйте Docker Compose когда:
- Разработка и тестирование
- Нужно запустить все сервисы (API + MinIO) вместе
- Локальная разработка

### Используйте Standalone контейнер когда:
- Production deployment
- MinIO уже запущен отдельно
- Нужна гибкость в настройке
- Деплой в Kubernetes/облако
- Нужно распространять только API

## Интеграция с оркестраторами

### Docker Swarm

```yaml
version: '3.8'
services:
  api:
    image: your-dockerhub-username/schedule-api:latest
    ports:
      - "8080:8080"
    environment:
      - MINIO_ENDPOINT=minio:9000
      - MINIO_ACCESS_KEY=minioadmin
      - MINIO_SECRET_KEY=minioadmin
    deploy:
      replicas: 3
      restart_policy:
        condition: on-failure
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: schedule-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: schedule-api
  template:
    metadata:
      labels:
        app: schedule-api
    spec:
      containers:
      - name: api
        image: your-dockerhub-username/schedule-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: MINIO_ENDPOINT
          value: "minio-service:9000"
        - name: MINIO_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: minio-secret
              key: access-key
        - name: MINIO_SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: minio-secret
              key: secret-key
```

## Версионирование

Рекомендуется использовать semantic versioning:

```bash
# Разработка
make docker-publish VERSION=dev

# Релизные версии
make docker-publish VERSION=1.0.0
make docker-publish VERSION=1.0.1

# Latest (для production)
make docker-publish VERSION=latest
```

