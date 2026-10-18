#!/usr/bin/env bash
# ARTEX 업데이트 스크립트: ① Docker 업데이트(새 이미지를 받아 재구성)  ② 로컬 컴파일 업데이트(바이너리 재빌드)
# install.sh 와 짝을 이룹니다. install 은 최초 설치를, update 는 새 버전으로의 업그레이드를 담당합니다.
# DB 마이그레이션은 직접 실행할 필요가 없습니다. artex 는 기동할 때마다 schema.sql 을 멱등하게 다시 돌리므로(ADD COLUMN/CREATE
# INDEX IF NOT EXISTS 포함) "重启即迁移"입니다. 데이터(pgdata 볼륨, ./data, ./skills)는 영향을 받지 않습니다.
set -euo pipefail
cd "$(cd "$(dirname "$0")" && pwd)"

info(){ printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok(){   printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn(){ printf '\033[33m[!]\033[0m %s\n' "$*"; }
die(){  printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }
ask(){  local p="$1" d="${2:-}" a; read -rp "$p${d:+ [$d]}: " a; echo "${a:-$d}"; }

# ── 선택: 저장소를 최신 코드로 동기화합니다(compose·스크립트·로컬 컴파일 소스가 모두 이걸로 갱신됩니다) ───────
sync_repo(){
  [ -d .git ] && command -v git >/dev/null 2>&1 || { warn "不是 git 工作副本，跳过 git pull"; return; }
  [ "$(ask '是否拉取最新代码（git pull --ff-only）？(y/n)' y)" = y ] || return
  if ! git pull --ff-only; then
    warn "git pull 无法快进（存在本地改动或分支分叉）。请处理后重试；本次直接使用当前代码"
  fi
}

# ── ① Docker 업데이트 ───────────────────────────────
update_docker(){
  command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1 \
    || die "找不到 docker / docker compose。请先用 ./install.sh 完成安装部署"
  [ -f .env ] || die "找不到 .env。请先用 ./install.sh 完成首次部署"

  # 선택: 지정한 버전 tag 로 업그레이드합니다(비워 두면 .env 의 ARTEX_TAG 를 따르고, 기본값은 latest 입니다)
  local tag; tag="$(ask '目标镜像 tag（回车则用 .env / latest）' '')"
  if [ -n "$tag" ]; then
    if grep -q '^ARTEX_TAG=' .env; then
      sed -i.bak "s|^ARTEX_TAG=.*|ARTEX_TAG=${tag}|" .env && rm -f .env.bak
    else
      printf '\nARTEX_TAG=%s\n' "$tag" >> .env
    fi
    ok "已设置 ARTEX_TAG 为 ${tag}"
  fi

  # artex 만 건드립니다. postgres 는 16-alpine 로 고정이라 따라 올릴 필요가 없습니다(받아 봐야 대역폭 낭비이고,
  # 메이저 버전이 바뀌면 호환성 위험도 있습니다). artex 는 depends_on postgres 를 선언하므로 서비스명을 붙여
  # up 하면 pg 가 안 떠 있을 때 자동으로 띄우고, 이미 떠 있으면 그대로 두고 재구성하지 않습니다.
  info "拉取新镜像（仅 artex）…"
  if ! docker compose pull artex; then
    warn "上游镜像 autumn27/artex 无法从 Docker Hub 拉取（上游已下架，pull 被拒绝）"
    die "Docker 镜像更新路径当前不可用。请选择 \"2) 本地更新（go 重新编译）\""
  fi
  info "重建后启动（artex 重启时会自动迁移 schema）…"
  docker compose up -d artex
  ok "更新完成 → http://localhost:8787"
  info "查看日志：docker compose logs -f artex"
  info "清理旧镜像（可选）：docker image prune -f"
}

# ── ② 로컬 컴파일 업데이트 ──────────────────────────────
update_local(){
  command -v go >/dev/null 2>&1 || die "找不到 Go (>=1.26)：https://go.dev/dl/"
  [ -f config.json ] || warn "找不到 config.json。首次部署请使用 ./install.sh"
  ok "Go: $(go version)"

  if command -v npm >/dev/null 2>&1; then
    info "重新构建前端静态产物…"
    ( cd web && npm ci && npm run build:static )
    rm -rf server/webui/dist && cp -r web/out server/webui/dist
    info "重新编译内嵌前端的单二进制…"
    CGO_ENABLED=0 go build -tags embedui -trimpath -o artex ./cmd/artex
  else
    warn "找不到 npm。只编译不含前端的后端（前端需另以 npm run dev 运行）"
    CGO_ENABLED=0 go build -o artex ./cmd/artex
  fi
  ok "编译完成 → ./artex"
  warn "重启运行中的 artex 进程以应用变更（重启时自动迁移 schema）"
}

echo "=============================="
echo "  ARTEX 更新"
echo "  1) Docker 更新（拉新镜像重建）"
echo "  2) 本地更新（go 重新编译）"
echo "=============================="
case "$(ask '选择' 1)" in
  1) sync_repo; update_docker ;;
  2) sync_repo; update_local ;;
  *) die "无效选择" ;;
esac
