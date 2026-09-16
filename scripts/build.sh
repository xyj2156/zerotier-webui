#!/usr/bin/env bash
# 交叉编译单执行产物：windows / linux / darwin × amd64 / arm64。
#
# build.ps1 的 Linux/POSIX 等价版：GitHub Actions（ubuntu runner）与本地 Linux/WSL 用它，
# 避免在 Win11 上构建偏慢；两者产出的目标清单、ldflags、版本戳字段完全一致。
#
# 前置：需装 pnpm（corepack enable 或 npm i -g pnpm）与 Go；脚本会自动构建前端再交叉编译。
# 用法：scripts/build.sh [version] [--skip-web]   # 省略 version 时用 git describe，取不到则回落 dev
#       --skip-web：dist 已是最新时跳过前端构建，只做交叉编译。
set -euo pipefail

# 保险：若本文件被 Windows 编辑器或挂载同步写成 CRLF，去掉 \r 后原地用 bash 重跑，
# 避免 '\r' 混进 shebang / 变量导致诡异的“command not found”。
if grep -q $'\r' "$0" 2>/dev/null; then
  _self="$(mktemp)"
  tr -d '\r' <"$0" >"$_self"
  chmod +x "$_self"
  exec bash "$_self" "$@"
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR/.."

MODULE='github.com/xyj2156/zerotier-webui'
OUT_DIR='bin'

# 参数：位置参数为版本号；--skip-web 跳过前端构建（dist 已最新时提速）；-h 帮助。
VERSION=''
SKIP_WEB=0
for arg in "$@"; do
  case "$arg" in
    --skip-web) SKIP_WEB=1 ;;
    -h|--help) echo '用法：scripts/build.sh [version] [--skip-web]'; exit 0 ;;
    *) VERSION="$arg" ;;
  esac
done

# 1) 前端产物（go:embed 读取 web/dist）：先构建再交叉编译，省去手动 pnpm build。
if [ "$SKIP_WEB" = "1" ]; then
  echo '跳过前端构建（--skip-web）'
else
  command -v pnpm >/dev/null 2>&1 || { echo '未找到 pnpm：先 corepack enable 或 npm i -g pnpm。' >&2; exit 1; }
  INSTALL_FLAGS=''
  if [ -n "${CI:-}" ]; then INSTALL_FLAGS='--frozen-lockfile'; fi
  echo "==> 构建前端：pnpm install ${INSTALL_FLAGS} && pnpm build"
  ( cd web && pnpm install $INSTALL_FLAGS && pnpm build )
fi

if [ ! -f web/dist/index.html ]; then
  echo 'web/dist/index.html 不存在且未跳过前端构建：请查看上面的 pnpm 输出。' >&2
  exit 1
fi

if [ -z "$VERSION" ]; then
  VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
fi
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo none)"
BUILT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# -s -w 剥离符号表与 DWARF，体积比默认构建小约 25~30%；三处 -X 注入构建信息。
LDFLAGS="-s -w \
-X ${MODULE}/internal/buildinfo.Version=${VERSION} \
-X ${MODULE}/internal/buildinfo.Commit=${COMMIT} \
-X ${MODULE}/internal/buildinfo.Date=${BUILT}"

# 纯交叉编译，不链接 C 运行时；关掉 CGO 保证产物自包含、可在目标机直接跑。
export CGO_ENABLED=0
mkdir -p "$OUT_DIR"

# 六个目标：GOOS:GOARCH
for target in windows:amd64 windows:arm64 linux:amd64 linux:arm64 darwin:amd64 darwin:arm64; do
  GOOS="${target%%:*}"
  GOARCH="${target##*:}"
  ext=''
  [ "$GOOS" = "windows" ] && ext='.exe'
  name="zerotier-webui-${GOOS}-${GOARCH}${ext}"
  out="${OUT_DIR}/${name}"

  GOOS="$GOOS" GOARCH="$GOARCH" go build -trimpath -ldflags "$LDFLAGS" -o "$out" ./cmd/zerotier-webui

  bytes="$(wc -c <"$out" | tr -d ' ')"
  mb="$(awk "BEGIN{printf \"%.1f\", ${bytes}/1048576}")"
  printf '%-42s %6s MB\n' "$name" "$mb"
done

printf '\n版本 %s（commit %s，构建于 %s）\n' "$VERSION" "$COMMIT" "$BUILT"
