.PHONY: help build run test docker-up docker-down d-build d-build-no-cache docker-build docker-tag docker-push docker-run

# Переменные для Docker Hub (укажите ваш username)
DOCKER_USERNAME ?= your-dockerhub-username
IMAGE_NAME = schedule-api
VERSION ?= latest
DOCKER_IMAGE = $(DOCKER_USERNAME)/$(IMAGE_NAME):$(VERSION)

help:
	@echo "Available commands:"
	@echo ""
	@echo "Local development:"
	@echo "  make build       - Build the application"
	@echo "  make run         - Run the application locally"
	@echo "  make test        - Run tests"
	@echo ""
	@echo "Docker Compose (для разработки):"
	@echo "  make d-up        - Start Docker Compose services"
	@echo "  make d-down      - Stop Docker Compose services"
	@echo "  make d-logs      - View API logs"
	@echo "  make d-build     - Build Docker Compose services"
	@echo "  make d-build-no-cache - Build no cache Docker Compose services"
	@echo ""
	@echo "Standalone Docker (для публикации):"
	@echo "  make docker-build       - Build standalone Docker image"
	@echo "  make docker-tag         - Tag image for Docker Hub"
	@echo "  make docker-push        - Push image to Docker Hub"
	@echo "  make docker-run         - Run standalone container (example)"
	@echo "  make docker-publish     - Build, tag and push to Docker Hub"

build:
	go build -o bin/api ./cmd/api

run:
	go run ./cmd/api/main.go

test:
	go test -v ./...

d-up:
	docker-compose up -d

d-down:
	docker-compose down

d-logs: #посмотреть логов api
	docker-compose logs -f api

d-build: #сборка образа api
	docker-compose build api

d-build-no-cache: #сборка образа api без кеша
	docker-compose build --no-cache api

# Standalone Docker commands
docker-build: # Сборка standalone образа
	docker build -t $(IMAGE_NAME):$(VERSION) .
	@echo "✓ Image built: $(IMAGE_NAME):$(VERSION)"

docker-tag: # Тегирование для Docker Hub
	docker tag $(IMAGE_NAME):$(VERSION) $(DOCKER_IMAGE)
	@echo "✓ Image tagged: $(DOCKER_IMAGE)"

docker-push: # Публикация в Docker Hub
	docker push $(DOCKER_IMAGE)
	@echo "✓ Image pushed: $(DOCKER_IMAGE)"

docker-publish: docker-build docker-tag docker-push # Сборка и публикация
	@echo "✓ Image published to Docker Hub: $(DOCKER_IMAGE)"

docker-run: # Пример запуска standalone контейнера
	@echo "Starting standalone API container..."
	docker run -d \
		--name schedule-api \
		-p 8080:8080 \
		-e MINIO_ENDPOINT=host.docker.internal:9000 \
		-e MINIO_ACCESS_KEY=minioadmin \
		-e MINIO_SECRET_KEY=minioadmin \
		-e MINIO_BUCKET=university-schedules \
		-e MINIO_USE_SSL=false \
		-e SOURCE_BUCKET=file-upload \
		-e TARGET_BUCKET=university-schedules \
		-e FILE_PATH_PATTERN=universities/%s/courses/%s/types/%s/files/%s \
		-e CACHE_TTL_MINUTES=10 \
		-e PRESIGNED_URL_TTL_MINUTES=15 \
		-e ENVIRONMENT=production \
		$(IMAGE_NAME):$(VERSION)
	@echo "✓ Container started: schedule-api"
	@echo "  API available at: http://localhost:8080"
	@echo "  View logs: docker logs -f schedule-api"
	@echo "  Stop: docker stop schedule-api"
	@echo "  Remove: docker rm schedule-api"

docker-stop: # Остановка standalone контейнера
	docker stop schedule-api || true
	docker rm schedule-api || true

clean:
	rm -rf bin/
