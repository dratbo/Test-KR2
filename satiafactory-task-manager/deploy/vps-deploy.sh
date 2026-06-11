#!/usr/bin/env bash
# Деплой Satisfactory Task Manager на VPS (git clone + docker build)
# Использование на сервере:
#   curl -fsSL https://raw.githubusercontent.com/dratbo/Test-KR2/main/satiafactory-task-manager/deploy/vps-deploy.sh | bash
# или после git clone:
#   cd Test-KR2/satiafactory-task-manager && bash deploy/vps-deploy.sh

set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/dratbo/Test-KR2.git}"
INSTALL_DIR="${INSTALL_DIR:-/opt/satisfactory-task-manager}"
BRANCH="${BRANCH:-main}"
COMPOSE_FILE="docker-compose.vps.yml"
ENV_FILE="deploy/.env"

log() { echo "[vps-deploy] $*"; }
die() { echo "[vps-deploy] ERROR: $*" >&2; exit 1; }

need_root_for_docker() {
  if ! docker info >/dev/null 2>&1; then
    if groups "$USER" 2>/dev/null | grep -q docker; then
      die "Docker установлен, но текущая сессия не в группе docker. Выполните: newgrp docker"
    fi
    die "Docker не доступен. Установите Docker и добавьте пользователя в группу docker."
  fi
}

ensure_swap() {
  local mem_mb
  mem_mb=$(awk '/MemTotal/ {print int($2/1024)}' /proc/meminfo)
  if swapon --show | grep -q .; then
    log "swap уже включён"
    return
  fi
  if [ "$mem_mb" -lt 3500 ]; then
    log "RAM ${mem_mb}MB — добавляю swap 2G (для docker build)"
    if [ ! -f /swapfile ]; then
      sudo fallocate -l 2G /swapfile || sudo dd if=/dev/zero of=/swapfile bs=1M count=2048
      sudo chmod 600 /swapfile
      sudo mkswap /swapfile
    fi
    sudo swapon /swapfile || true
    grep -q '/swapfile' /etc/fstab 2>/dev/null || echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
  fi
}

clone_or_update() {
  if [ -d "$INSTALL_DIR/.git" ]; then
    log "обновление $INSTALL_DIR"
    git -C "$INSTALL_DIR" fetch origin "$BRANCH"
    git -C "$INSTALL_DIR" checkout "$BRANCH"
    git -C "$INSTALL_DIR" pull --ff-only origin "$BRANCH"
    cd "$INSTALL_DIR/satiafactory-task-manager"
  elif [ -d "$INSTALL_DIR/satiafactory-task-manager" ]; then
    cd "$INSTALL_DIR/satiafactory-task-manager"
  else
    log "клонирование $REPO_URL -> $INSTALL_DIR"
    sudo mkdir -p "$(dirname "$INSTALL_DIR")"
    if [ ! -d "$INSTALL_DIR" ]; then
      sudo git clone --branch "$BRANCH" "$REPO_URL" "$INSTALL_DIR"
      sudo chown -R "$USER:$USER" "$INSTALL_DIR"
    fi
    cd "$INSTALL_DIR/satiafactory-task-manager"
  fi
}

prepare_env() {
  if [ ! -f "$ENV_FILE" ]; then
    cp deploy/.env.example "$ENV_FILE"
    log "создан $ENV_FILE — ОБЯЗАТЕЛЬНО смените пароли: nano $ENV_FILE"
    read -r -p "Нажмите Enter после редактирования .env (или Ctrl+C)..." _
  fi
}

check_port() {
  local port="${GATEWAY_PORT:-8080}"
  if command -v ss >/dev/null 2>&1; then
    if ss -tln | awk '{print $4}' | grep -q ":${port}$"; then
      log "ВНИМАНИЕ: порт ${port} уже занят. Проверьте Telegram proxy и GATEWAY_PORT в deploy/.env"
    fi
  fi
}

main() {
  need_root_for_docker
  ensure_swap
  clone_or_update
  prepare_env
  # shellcheck disable=SC1091
  set -a && source "$ENV_FILE" && set +a
  check_port

  log "сборка и запуск (это может занять 10–20 мин на 1 ядре)..."
  docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --build

  log "готово. UI: http://$(curl -fsS ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}'):${GATEWAY_PORT:-8080}"
  log "статус: docker compose -f $COMPOSE_FILE ps"
  log "логи gateway: docker logs gateway --tail 30"
}

main "$@"
