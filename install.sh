#!/usr/bin/env bash
# ARTEX 설치 스크립트: ① 전부 Docker  ② 로컬 컴파일 실행
set -euo pipefail
cd "$(cd "$(dirname "$0")" && pwd)"

info(){ printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok(){   printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn(){ printf '\033[33m[!]\033[0m %s\n' "$*"; }
die(){  printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }
ask(){  local p="$1" d="${2:-}" a; read -rp "$p${d:+ [$d]}: " a; echo "${a:-$d}"; }
rand(){ head -c 18 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 24; }

# ── docker 환경 감지 / 자동 설치 ──────────────────
ensure_docker(){
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    ok "docker 与 docker compose 检查通过"; return
  fi
  warn "找不到 docker / docker compose"
  case "$(uname -s)" in
    Linux)
      if [ "$(ask 'Docker 를 자동으로 설치할까요? (y/n)' y)" = y ]; then
        curl -fsSL https://get.docker.com | sh
        sudo usermod -aG docker "$USER" || true
        ok "Docker 安装完成（用户组变更需重新登录后才能免 sudo 生效）"
      else
        die "请自行安装 docker 后重新运行"
      fi ;;
    Darwin) die "macOS 请安装 Docker Desktop：https://www.docker.com/products/docker-desktop/" ;;
    *)      die "请自行安装 docker 后重新运行" ;;
  esac
}

# ── ① 전부 Docker ──────────────────────────────
install_docker(){
  ensure_docker
  if [ ! -f .env ]; then
    cp .env.example .env 2>/dev/null || true
    local pw key
    pw="$(ask 'Postgres 密码（回车则随机生成）' "$(rand)")"
    key="$(ask 'ANTHROPIC_API_KEY（可留空，稍后在 UI 中配置）' '')"
    sed -i.bak "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=${pw}|" .env
    sed -i.bak "s|^ANTHROPIC_API_KEY=.*|ANTHROPIC_API_KEY=${key}|" .env
    rm -f .env.bak
    ok ".env 파일을 생성했습니다 (POSTGRES_PASSWORD 설정 완료)"
  else
    info "이미 있는 .env 파일을 그대로 사용합니다"
  fi
  info "拉取镜像并启动…"
  docker compose pull || true
  if ! docker compose up -d; then
    warn "상류 이미지 autumn27/artex 를 Docker Hub 에서 받을 수 없어 기동에 실패했습니다(현재 상류에서 내려간 상태라 pull 이 거부됩니다)"
    warn "한국어판 화면·출력은 이 스크립트를 다시 실행해 \"2) 로컬 실행 (go 컴파일)\" 을 고르거나, README \"소스에서 단일 바이너리 컴파일\" 路径构建"
    die "Docker Compose（全 Docker）路径当前无法拉取上游镜像，暂不可用"
  fi
  ok "启动完成 → http://localhost:8787"
  info "查看日志：docker compose logs -f artex"
}

# ── ② 로컬 컴파일 실행 ────────────────────────────
install_local(){
  echo "数据库安装方式："
  echo "  1) 连接已有 PostgreSQL"
  echo "  2) 用 Docker 启动一个 PostgreSQL（需要 docker）"
  case "$(ask '选择' 1)" in
    2)
      ensure_docker
      local pw; pw="$(ask 'Postgres 密码（回车则随机）' "$(rand)")"
      docker run -d --name artex-pg -p 5432:5432 \
        -e POSTGRES_USER=artex -e POSTGRES_PASSWORD="$pw" -e POSTGRES_DB=artex \
        -v artex-pg:/var/lib/postgresql/data postgres:16-alpine
      DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=artex DB_PASS="$pw" DB_NAME=artex DB_SSL=disable ;;
    *)
      DB_HOST="$(ask '数据库地址' 127.0.0.1)"
      DB_PORT="$(ask '端口' 5432)"
      DB_USER="$(ask '账户' artex)"
      DB_PASS="$(ask '密码' '')"
      DB_NAME="$(ask '数据库名' artex)"
      DB_SSL="$(ask 'sslmode (disable/require)' disable)" ;;
  esac

  # config.json 생성
  cat > config.json <<JSON
{
  "database": {
    "host": "${DB_HOST}",
    "port": ${DB_PORT},
    "user": "${DB_USER}",
    "password": "${DB_PASS}",
    "dbname": "${DB_NAME}",
    "sslmode": "${DB_SSL}"
  }
}
JSON
  ok "config.json 파일을 생성했습니다"

  # go 환경 확인
  command -v go >/dev/null 2>&1 || die "找不到 Go，请先安装 Go (>=1.26)：https://go.dev/dl/"
  ok "Go: $(go version)"

  # 프런트엔드를 내장하려면 node 로 정적 산출물을 만들어야 합니다
  if command -v npm >/dev/null 2>&1; then
    info "构建前端静态产物…"
    ( cd web && npm ci && npm run build:static )
    rm -rf server/webui/dist && mkdir -p server/webui && cp -r web/out server/webui/dist
    info "编译内嵌前端的单二进制…"
    CGO_ENABLED=0 go build -tags embedui -trimpath -o artex ./cmd/artex
  else
    warn "npm 을 찾을 수 없습니다: 프런트엔드를 내장하지 않은 백엔드만 컴파일합니다 (프런트엔드는 npm run dev 로 따로 실행)"
    CGO_ENABLED=0 go build -o artex ./cmd/artex
  fi
  ok "컴파일을 완료했습니다 → ./artex"

  info "启动中…（Ctrl-C 退出）"
  ./artex
}

echo "=============================="
echo "  ARTEX 安装"
echo "  1) 全部 Docker 安装"
echo "  2) 本地运行（go 编译）"
echo "=============================="
case "$(ask '选择' 1)" in
  1) install_docker ;;
  2) install_local ;;
  *) die "无效选择" ;;
esac
