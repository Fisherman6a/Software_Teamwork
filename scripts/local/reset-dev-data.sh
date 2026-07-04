#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CONFIG_LOADER="$ROOT_DIR/scripts/config/load-profile.sh"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.yml"
CURRENT_STEP="initializing"
YES=0
STOP_BACKEND=1
REMOVE_ORPHANS=1

COLOR_RESET=""
COLOR_BLUE=""
COLOR_GREEN=""
COLOR_YELLOW=""
COLOR_RED=""
COLOR_CYAN=""
if [[ -z "${NO_COLOR:-}" && ( -t 1 || "${FORCE_COLOR:-0}" == "1" || "${CLICOLOR_FORCE:-0}" == "1" ) ]]; then
  COLOR_RESET=$'\033[0m'
  COLOR_BLUE=$'\033[1;34m'
  COLOR_GREEN=$'\033[1;32m'
  COLOR_YELLOW=$'\033[1;33m'
  COLOR_RED=$'\033[1;31m'
  COLOR_CYAN=$'\033[1;36m'
fi

log_info() {
  printf '%b%s %s%b\n' "$COLOR_BLUE" "[reset-dev-data]" "$*" "$COLOR_RESET"
}

log_success() {
  printf '%b%s %s%b\n' "$COLOR_GREEN" "[ok]" "$*" "$COLOR_RESET"
}

log_warn() {
  printf '%b%s %s%b\n' "$COLOR_YELLOW" "[warn]" "$*" "$COLOR_RESET" >&2
}

log_error() {
  printf '%b%s %s%b\n' "$COLOR_RED" "[fail]" "$*" "$COLOR_RESET" >&2
}

log_hint() {
  printf '%b%s %s%b\n' "$COLOR_CYAN" "[hint]" "$*" "$COLOR_RESET" >&2
}

usage() {
  cat <<'EOF'
Usage: ./scripts/local/reset-dev-data.sh [options]

Stops host-run backend services and removes local development infrastructure
containers plus Compose-managed data volumes for PostgreSQL, MinIO, and
Elasticsearch. Images are not removed.

Options:
  -y, --yes
            Skip the interactive confirmation prompt.
  --skip-stop-backend
            Do not run ./scripts/local/stop-backend.sh before resetting data.
  --keep-orphans
            Do not pass --remove-orphans to docker compose down.
  --profile <name>
            Set CONFIG_PROFILE for scripts/config/load-profile.sh.
  --secret-file <path>
            Set CONFIG_SECRET_FILE for scripts/config/load-profile.sh.
            Defaults to .env.local.
  -h, --help
            Show this help.

After reset, run ./scripts/local/dev-up.sh to recreate databases, buckets,
indexes, migrations, and local seed data.
EOF
}

parse_args() {
  while (($# > 0)); do
    case "$1" in
      -y|--yes|--force)
        YES=1
        ;;
      --skip-stop-backend)
        STOP_BACKEND=0
        ;;
      --keep-orphans)
        REMOVE_ORPHANS=0
        ;;
      --profile)
        if [[ $# -lt 2 || -z "$2" ]]; then
          log_error "--profile requires a value"
          exit 2
        fi
        export CONFIG_PROFILE="$2"
        shift
        ;;
      --secret-file)
        if [[ $# -lt 2 || -z "$2" ]]; then
          log_error "--secret-file requires a value"
          exit 2
        fi
        export CONFIG_SECRET_FILE="$2"
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        log_error "unknown argument: $1"
        usage >&2
        exit 2
        ;;
    esac
    shift
  done
}

on_exit() {
  status=$?
  local compose_env_hint="${CONFIG_COMPOSE_ENV_FILE:-.local/config/${CONFIG_PROFILE:-dev}.env}"
  if (( status == 0 )); then
    log_success "completed successfully"
  else
    log_error "failed during ${CURRENT_STEP} (exit ${status})"
    case "$CURRENT_STEP" in
      "loading local config")
        log_hint "Create local secrets first: cp .env.example .env.local"
        ;;
      "validating Docker Compose config"|"removing local development data volumes")
        log_hint "Inspect Docker status: docker compose -f deploy/docker-compose.yml --env-file $compose_env_hint ps"
        ;;
      *)
        log_hint "Rerun with --help for supported options."
        ;;
    esac
  fi
}

run_step() {
  CURRENT_STEP="$1"
  shift
  log_info "${CURRENT_STEP}"
  "$@"
  log_success "${CURRENT_STEP} succeeded"
}

confirm_reset() {
  if (( YES )); then
    return
  fi

  log_warn "This will delete local development database/object/index data volumes."
  log_warn "Affected Compose-managed volumes: postgres_data, minio_data, elasticsearch_data."
  log_warn "Docker images and source files will not be removed."

  if [[ ! -t 0 ]]; then
    log_error "refusing to reset without --yes in a non-interactive shell"
    return 2
  fi

  printf 'Type "reset" to continue: ' >&2
  local answer
  read -r answer
  if [[ "$answer" != "reset" ]]; then
    log_error "aborted"
    return 2
  fi
}

stop_backend() {
  if (( ! STOP_BACKEND )); then
    log_info "skipping host-run backend stop"
    return
  fi
  "$ROOT_DIR/scripts/local/stop-backend.sh"
}

compose_down_with_volumes() {
  local args=(down -v)
  if (( REMOVE_ORPHANS )); then
    args+=(--remove-orphans)
  fi
  "${compose[@]}" "${args[@]}"
}

parse_args "$@"
trap on_exit EXIT

log_info "starting local development data reset"

CURRENT_STEP="loading local config"
export SOFTWARE_TEAMWORK_ROOT="$ROOT_DIR"
# shellcheck disable=SC1090
. "$CONFIG_LOADER"
log_success "${CURRENT_STEP} succeeded"

compose=(docker compose -f "$COMPOSE_FILE" --env-file "$CONFIG_COMPOSE_ENV_FILE")

run_step "validating Docker Compose config" "${compose[@]}" config --quiet
confirm_reset
run_step "stopping host-run backend services" stop_backend
run_step "removing local development data volumes" compose_down_with_volumes

log_success "local development data cleared; run ./scripts/local/dev-up.sh to rebuild"
