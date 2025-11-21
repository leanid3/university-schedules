# Docker Quick Start Guide

Быстрый старт для работы со standalone Docker образом Schedule API.

## 🚀 Быстрый запуск

### Для пользователей (скачать и запустить)

```bash
# 1. Скачайте образ
docker pull your-dockerhub-username/schedule-api:latest

# 2. Запустите контейнер
docker run -d \
  --name schedule-api \
  -p 8080:8080 \
  -e MINIO_ENDPOINT=your-minio-host:9000 \
  -e MINIO_ACCESS_KEY=your-access-key \
  -e MINIO_SECRET_KEY=your-secret-key \
  your-dockerhub-username/schedule-api:latest

# 3. Проверьте работу
curl http://localhost:8080/api/v1/health
```

### Для разработчиков (сборка и публикация)

```bash
# 1. Отредактируйте Makefile - укажите ваш DOCKER_USERNAME

# 2. Авторизуйтесь в Docker Hub
docker login

# 3. Соберите и опубликуйте образ
make docker-publish VERSION=1.0.0
```

## 📦 Использование готовых скриптов

### Linux/Mac

```bash
./run-docker.sh your-dockerhub-username/schedule-api latest
```

### Windows

```cmd
run-docker.bat your-dockerhub-username/schedule-api latest
```

Скрипты интерактивно запросят все параметры.

## 🔧 Базовые команды

```bash
# Просмотр логов
docker logs -f schedule-api

# Остановка
docker stop schedule-api

# Запуск
docker start schedule-api

# Удаление
docker rm -f schedule-api

# Перезапуск
docker restart schedule-api
```

## 🌐 Варианты подключения к MinIO

### Локальный MinIO (на хосте машины)

```bash
-e MINIO_ENDPOINT=host.docker.internal:9000
```

### MinIO в отдельном Docker контейнере

```bash
# Создайте сеть
docker network create app-network

# Запустите оба контейнера в этой сети
docker run -d --name minio --network app-network ...
docker run -d --name schedule-api --network app-network \
  -e MINIO_ENDPOINT=minio:9000 ...
```

### Внешний MinIO сервер

```bash
-e MINIO_ENDPOINT=minio.example.com:9000
-e MINIO_USE_SSL=true
```

## ⚙️ Минимальная конфигурация

Обязательно указать:

```bash
-e MINIO_ENDPOINT=<адрес>
-e MINIO_ACCESS_KEY=<ключ>
-e MINIO_SECRET_KEY=<секрет>
```

Остальные параметры имеют значения по умолчанию.

## 📝 Полная конфигурация

```bash
docker run -d \
  --name schedule-api \
  -p 8080:8080 \
  -e MINIO_ENDPOINT=minio:9000 \
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

## 🔍 Проверка работоспособности

```bash
# Health check
curl http://localhost:8080/api/v1/health

# Должен вернуть:
# {"status":"ok","time":"..."}

# Проверка API
curl http://localhost:8080/api/v1/universities

# Логи
docker logs schedule-api
```

## 📚 Дополнительная документация

- `DOCKER_STANDALONE.md` - Полная документация по standalone контейнеру
- `Makefile` - Все доступные команды для сборки и публикации
- `run-docker.sh` / `run-docker.bat` - Скрипты для запуска

## 🆘 Troubleshooting

### API не запускается

```bash
# Проверьте логи
docker logs schedule-api

# Проверьте статус
docker ps -a | grep schedule-api
```

### Не может подключиться к MinIO

1. Проверьте доступность MinIO с хоста:
   ```bash
   curl http://your-minio-host:9000/minio/health/live
   ```

2. Для локального MinIO используйте `host.docker.internal:9000` вместо `localhost:9000`

3. Проверьте, что API и MinIO в одной сети (если оба в Docker)

### Порт уже занят

```bash
# Используйте другой порт
docker run -p 8081:8080 ...
```

## 💡 Советы

1. **Версионирование**: Используйте конкретные версии вместо `latest` для production
   ```bash
   docker pull your-dockerhub-username/schedule-api:1.0.0
   ```

2. **Логирование**: Используйте Docker logging drivers для централизованных логов

3. **Healthcheck**: Настройте healthcheck в вашем оркестраторе:
   ```
   GET /api/v1/health
   ```

4. **Secrets**: Для production используйте Docker secrets или переменные окружения из безопасного хранилища

5. **Сеть**: Для production создайте отдельную Docker сеть для изоляции

