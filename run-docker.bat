@echo off
REM Скрипт для запуска standalone контейнера Schedule API в Windows
REM Использование: run-docker.bat [имя_образа] [версия]

setlocal EnableDelayedExpansion

set IMAGE_NAME=%1
if "%IMAGE_NAME%"=="" set IMAGE_NAME=your-dockerhub-username/schedule-api

set VERSION=%2
if "%VERSION%"=="" set VERSION=latest

set CONTAINER_NAME=schedule-api

echo =============================================
echo Schedule API - Standalone Docker Container
echo =============================================
echo.

REM Проверяем, запущен ли контейнер
docker ps -q -f name=%CONTAINER_NAME% >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo Контейнер %CONTAINER_NAME% уже запущен
    echo Остановить контейнер? (y/n)
    set /p response=
    if /i "!response!"=="y" (
        docker stop %CONTAINER_NAME%
        docker rm %CONTAINER_NAME%
        echo Контейнер остановлен
    ) else (
        exit /b 0
    )
)

REM Удаляем старый контейнер если он существует
docker ps -aq -f name=%CONTAINER_NAME% >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    docker rm %CONTAINER_NAME% >nul 2>&1
)

echo.
echo Введите параметры подключения к MinIO:
echo (Нажмите Enter для использования значений по умолчанию)
echo.

set /p MINIO_ENDPOINT="MinIO Endpoint [host.docker.internal:9000]: "
if "%MINIO_ENDPOINT%"=="" set MINIO_ENDPOINT=host.docker.internal:9000

set /p MINIO_ACCESS_KEY="MinIO Access Key [minioadmin]: "
if "%MINIO_ACCESS_KEY%"=="" set MINIO_ACCESS_KEY=minioadmin

set /p MINIO_SECRET_KEY="MinIO Secret Key [minioadmin]: "
if "%MINIO_SECRET_KEY%"=="" set MINIO_SECRET_KEY=minioadmin

set /p SOURCE_BUCKET="Source Bucket [file-upload]: "
if "%SOURCE_BUCKET%"=="" set SOURCE_BUCKET=file-upload

set /p TARGET_BUCKET="Target Bucket [university-schedules]: "
if "%TARGET_BUCKET%"=="" set TARGET_BUCKET=university-schedules

set /p API_PORT="API Port [8080]: "
if "%API_PORT%"=="" set API_PORT=8080

echo.
echo Запуск контейнера...
echo Образ: %IMAGE_NAME%:%VERSION%
echo.

docker run -d ^
    --name %CONTAINER_NAME% ^
    -p %API_PORT%:8080 ^
    -e MINIO_ENDPOINT=%MINIO_ENDPOINT% ^
    -e MINIO_ACCESS_KEY=%MINIO_ACCESS_KEY% ^
    -e MINIO_SECRET_KEY=%MINIO_SECRET_KEY% ^
    -e MINIO_BUCKET=%TARGET_BUCKET% ^
    -e MINIO_USE_SSL=false ^
    -e SOURCE_BUCKET=%SOURCE_BUCKET% ^
    -e TARGET_BUCKET=%TARGET_BUCKET% ^
    -e FILE_PATH_PATTERN=universities/%%s/courses/%%s/types/%%s/files/%%s ^
    -e CACHE_TTL_MINUTES=10 ^
    -e PRESIGNED_URL_TTL_MINUTES=15 ^
    -e ENVIRONMENT=production ^
    %IMAGE_NAME%:%VERSION%

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ✓ Контейнер успешно запущен!
    echo.
    echo Информация о контейнере:
    echo   Имя: %CONTAINER_NAME%
    echo   API: http://localhost:%API_PORT%
    echo   Health check: http://localhost:%API_PORT%/api/v1/health
    echo.
    echo Полезные команды:
    echo   Просмотр логов:  docker logs -f %CONTAINER_NAME%
    echo   Остановка:       docker stop %CONTAINER_NAME%
    echo   Удаление:        docker rm %CONTAINER_NAME%
    echo   Перезапуск:      docker restart %CONTAINER_NAME%
    echo.
    
    timeout /t 3 >nul
    echo Проверка состояния...
    
    curl -s http://localhost:%API_PORT%/api/v1/health >nul 2>&1
    if %ERRORLEVEL% EQU 0 (
        echo ✓ API работает корректно
    ) else (
        echo ⚠ API еще запускается, проверьте логи: docker logs %CONTAINER_NAME%
    )
) else (
    echo ✗ Ошибка при запуске контейнера
    exit /b 1
)

endlocal

