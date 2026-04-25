#!/usr/bin/env bash

set -euo pipefail

# 中文说明：
# 1. 当前版本不再在 .env 中生成 JWT_SECRET / DATA_ENCRYPTION_KEY / RSA_PRIVATE_KEY。
# 2. 首次启动后，服务会在后端日志中输出管理员登录密钥，并在本地 config/keys/ 生成根密钥对。
# 3. 请务必备份 config/keys/ 与 backup/ 目录，否则重装容器后将无法解密历史敏感配置。

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

detect_compose_cmd() {
    if docker compose version >/dev/null 2>&1; then
        COMPOSE_CMD="docker compose"
    elif command -v docker-compose >/dev/null 2>&1; then
        COMPOSE_CMD="docker-compose"
    else
        print_error "Docker Compose not found."
        exit 1
    fi
}

check_docker() {
    if ! command -v docker >/dev/null 2>&1; then
        print_error "Docker not found. Please install Docker first."
        exit 1
    fi

    detect_compose_cmd
}

check_env() {
    if [ ! -f ".env" ]; then
        if [ -f ".env.example" ]; then
            print_warning ".env not found, copying from .env.example"
            cp .env.example .env
        else
            print_warning ".env not found, creating an empty one"
            : > .env
        fi
    fi

    chmod 600 .env 2>/dev/null || true
}

read_env_vars() {
    if [ -f ".env" ]; then
        NOFX_FRONTEND_PORT=$(grep "^NOFX_FRONTEND_PORT=" .env 2>/dev/null | cut -d'=' -f2- || echo "3000")
        NOFX_BACKEND_PORT=$(grep "^NOFX_BACKEND_PORT=" .env 2>/dev/null | cut -d'=' -f2- || echo "8080")
        NOFX_FRONTEND_PORT=$(echo "${NOFX_FRONTEND_PORT:-3000}" | tr -d '"' | tr -d ' ')
        NOFX_BACKEND_PORT=$(echo "${NOFX_BACKEND_PORT:-8080}" | tr -d '"' | tr -d ' ')
    else
        NOFX_FRONTEND_PORT=3000
        NOFX_BACKEND_PORT=8080
    fi
}

ensure_runtime_dirs() {
    mkdir -p data config/keys backup
    chmod 700 data config/keys backup 2>/dev/null || true
}

show_first_run_notes() {
    echo ""
    echo -e "${CYAN}NOFX Offline Mode${NC}"
    echo "  1. Use ./start.sh logs nofx to read the first-run admin key from backend logs."
    echo "  2. Back up ./config/keys immediately. Losing this directory means encrypted data cannot be recovered."
    echo "  3. Open the web dashboard and log in with the admin key."
    echo ""
    echo -e "  Web dashboard: ${BLUE}http://localhost:${NOFX_FRONTEND_PORT}${NC}"
    echo -e "  Backend logs:  ${YELLOW}./start.sh logs nofx${NC}"
    echo -e "  Stop service:  ${YELLOW}./start.sh stop${NC}"
    echo ""
    echo "  Admin key reset:  ./start.sh reset-admin-key"
    echo "  Root key reset:   ./start.sh reset-root-key"
    echo "  Restore backup:   ./start.sh restore-backup <timestamp>"
    echo ""
}

start() {
    read_env_vars
    check_env
    ensure_runtime_dirs

    print_info "Starting services..."
    if [ "${1:-}" = "--build" ]; then
        $COMPOSE_CMD up -d --build
    else
        $COMPOSE_CMD up -d
    fi

    print_success "Services started"
    show_first_run_notes
}

stop() {
    print_info "Stopping services..."
    $COMPOSE_CMD stop
    print_success "Services stopped"
}

restart() {
    print_info "Restarting services..."
    $COMPOSE_CMD restart
    print_success "Services restarted"
}

logs() {
    if [ -n "${2:-}" ]; then
        $COMPOSE_CMD logs -f "$2"
    else
        $COMPOSE_CMD logs -f
    fi
}

status() {
    read_env_vars

    print_info "Container status:"
    $COMPOSE_CMD ps

    echo ""
    if command -v curl >/dev/null 2>&1; then
        print_info "Backend health:"
        curl -fsS "http://localhost:${NOFX_BACKEND_PORT}/api/health" || print_warning "Backend health endpoint is not ready yet"
    else
        print_warning "curl not found, skipping health check"
    fi
}

clean() {
    print_warning "This will delete all containers and volumes created by docker-compose."
    read -r -p "Confirm? (yes/no): " confirm
    if [ "$confirm" = "yes" ]; then
        $COMPOSE_CMD down -v
        print_success "Docker resources removed"
    else
        print_info "Cancelled"
    fi
}

update() {
    print_info "Pulling latest code and rebuilding containers..."
    git pull
    $COMPOSE_CMD up -d --build
    print_success "Update complete"
}

require_running_backend() {
    if ! $COMPOSE_CMD ps nofx >/dev/null 2>&1; then
        print_error "Backend container is not running. Start it first with ./start.sh start"
        exit 1
    fi
}

reset_admin_key() {
    require_running_backend
    print_warning "Resetting admin key. Existing login sessions will be invalidated."
    $COMPOSE_CMD exec nofx ./nofx reset-admin-key
}

reset_root_key() {
    require_running_backend
    print_warning "Resetting root key. The service will back up current data before re-encrypting sensitive fields."
    $COMPOSE_CMD exec nofx ./nofx reset-root-key
}

restore_backup() {
    if [ -z "${2:-}" ]; then
        print_error "Usage: ./start.sh restore-backup <timestamp>"
        exit 1
    fi

    require_running_backend
    print_warning "Restoring backup ${2}. The backend will use the selected backup snapshot."
    $COMPOSE_CMD exec nofx ./nofx restore-backup "$2"
}

show_help() {
    cat <<'EOF'
NOFX Docker helper

Usage:
  ./start.sh [command] [options]

Commands:
  start [--build]             Start services
  stop                        Stop services
  restart                     Restart services
  logs [service]              Follow logs
  status                      Show container status
  clean                       Remove containers and docker volumes
  update                      Pull latest code and rebuild
  reset-admin-key             Reset the offline admin login key inside the backend container
  reset-root-key              Rotate RSA root key and re-encrypt stored sensitive data
  restore-backup <timestamp>  Restore a backup created under ./backup/<timestamp>
  regenerate-keys             Deprecated alias, use reset-root-key instead
  help                        Show this help

Notes:
  - First login key is printed by the backend on first startup.
  - Back up ./config/keys immediately after first startup.
  - Do not rely on .env to store application secrets anymore.
EOF
}

main() {
    check_docker

    case "${1:-start}" in
        start)
            start "${2:-}"
            ;;
        stop)
            stop
            ;;
        restart)
            restart
            ;;
        logs)
            logs "$@"
            ;;
        status)
            status
            ;;
        clean)
            clean
            ;;
        update)
            update
            ;;
        reset-admin-key)
            reset_admin_key
            ;;
        reset-root-key)
            reset_root_key
            ;;
        restore-backup)
            restore_backup "$@"
            ;;
        regenerate-keys)
            print_warning "regenerate-keys is deprecated. Use ./start.sh reset-root-key instead."
            reset_root_key
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "Unknown command: ${1}"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
