#!/bin/sh
set -eu

APP_PORT="${APP_PORT:-8080}"
COMPOSE_FILE="stylometry_docker-compose.yml"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-stylometry_project}"

echo "Освобождаю порт ${APP_PORT}..."

if command -v docker >/dev/null 2>&1; then
  docker ps --format '{{.ID}} {{.Names}} {{.Ports}}' | while IFS= read -r stylometry_container_line; do
    case "$stylometry_container_line" in
      *"0.0.0.0:${APP_PORT}->"*|*"[::]:${APP_PORT}->"*)
        stylometry_container_id=${stylometry_container_line%% *}
        stylometry_container_rest=${stylometry_container_line#* }
        stylometry_container_name=${stylometry_container_rest%% *}
        case "$stylometry_container_name" in
          "${COMPOSE_PROJECT_NAME}"-*)
            echo "Останавливаю контейнер ${stylometry_container_name}, который занимает порт ${APP_PORT}"
            docker stop "$stylometry_container_id" >/dev/null
            ;;
          *)
            if [ "${STYLOMETRY_CLOSE_FOREIGN_8080:-0}" = "1" ]; then
              echo "Останавливаю внешний контейнер ${stylometry_container_name}, который занимает порт ${APP_PORT}"
              docker stop "$stylometry_container_id" >/dev/null
            else
              echo "Порт ${APP_PORT} занят внешним контейнером ${stylometry_container_name}."
              echo "Для остановки запусти: STYLOMETRY_CLOSE_FOREIGN_8080=1 ./stylometry_open.sh"
              exit 1
            fi
            ;;
        esac
        ;;
    esac
  done
fi

if command -v lsof >/dev/null 2>&1; then
  stylometry_pids=$(lsof -tiTCP:"${APP_PORT}" -sTCP:LISTEN || true)
  if [ -n "$stylometry_pids" ]; then
    echo "$stylometry_pids" | xargs kill -9
  fi
fi

echo "Собираю и запускаю приложение на порту ${APP_PORT}..."
docker compose -f "$COMPOSE_FILE" up -d --build

echo "Приложение: http://127.0.0.1:${APP_PORT}/samples"
echo "Adminer: http://127.0.0.1:${ADMINER_PORT:-8082}"
