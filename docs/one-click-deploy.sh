#!/usr/bin/env bash
# BYOK 一键部署：克隆（可选）→ 构建本仓镜像 → SQLite 启动 :3000
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/mimaweimahan/tongyiapi.git}"
ROOT="${ROOT:-}"
BYOK_PORT="${BYOK_PORT:-3000}"

need() { command -v "$1" >/dev/null 2>&1 || { echo "缺少命令: $1"; exit 1; }; }
need git
need docker
docker compose version >/dev/null 2>&1 || { echo "需要 docker compose 插件"; exit 1; }

if [[ -z "$ROOT" ]]; then
  if [[ -f Dockerfile && -f docs/docker-compose.byok.yml ]]; then
    ROOT="$(pwd)"
  else
    ROOT="${HOME}/tongyiapi"
    if [[ ! -d "$ROOT/.git" ]]; then
      echo "==> 克隆 $REPO_URL → $ROOT"
      git clone --depth 1 "$REPO_URL" "$ROOT"
    else
      echo "==> 使用已有目录 $ROOT"
      git -C "$ROOT" pull --ff-only || true
    fi
  fi
fi

cd "$ROOT"
[[ -f Dockerfile ]] || { echo "未找到 Dockerfile，目录不对: $ROOT"; exit 1; }
[[ -f docs/docker-compose.byok.yml ]] || { echo "未找到 docs/docker-compose.byok.yml"; exit 1; }

mkdir -p data logs
if [[ ! -f .env.byok ]]; then
  SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"
  cat > .env.byok <<EOF
SESSION_SECRET=${SECRET}
BYOK_PORT=${BYOK_PORT}
EOF
  echo "==> 已写入 .env.byok（含 SESSION_SECRET）"
else
  echo "==> 复用已有 .env.byok"
fi

set -a
# shellcheck disable=SC1091
source .env.byok
set +a

echo "==> 构建并启动（首次编译较久，请耐心等待）"
docker compose --project-directory "$ROOT" -f "$ROOT/docs/docker-compose.byok.yml" --env-file "$ROOT/.env.byok" up -d --build

echo
echo "========================================"
echo " 部署完成"
echo " 目录: $ROOT"
echo " 访问: http://127.0.0.1:${BYOK_PORT}"
echo " 首次打开请完成安装向导（创建超管）"
echo " 然后设置 Server Address 为你的公网 HTTPS"
echo " 功能说明: docs/FEATURES.md"
echo " 部署说明: docs/DEPLOY.md"
echo " 日志: docker compose -f docs/docker-compose.byok.yml logs -f"
echo "========================================"
