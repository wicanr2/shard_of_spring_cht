#!/usr/bin/env bash
# IDA Pro 9.4 headless 包裝(CLAUDE.md §3:反組譯一律用 IDA)。
#
#   tools/ida.sh analyze START.EXE            產 .i64 + .asm(-A -B)
#   tools/ida.sh run tools/ida/foo.py START.EXE [腳本參數...]
#                                             對已建好的 .i64 跑 IDAPython
#   tools/ida.sh raw idat --version           直接下 idat 參數
#
# 工作目錄固定 workplace/ida/(gitignore)。原版唯讀,一律複製後才分析。
# image 的來歷與 IDAPython 修法見 ~/.claude/knowledge-base/retro/ida-pro-9.4.md
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${IDA_IMAGE:-ida-pro-9.4-idapython:py312-v1}"
WORK="$ROOT/workplace/ida"
ORIG="$ROOT/game/sharspri"

mkdir -p "$WORK" "$WORK/out"

run() {
  docker run --rm \
    --network none \
    --memory 2g \
    --cpus 2 \
    --pids-limit 256 \
    --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" \
    -v "$WORK:/work" \
    -v "$ROOT/tools:/work/tools:ro" \
    -w /work \
    "$IMAGE" "$@"
}

# 從唯讀原版複製一份到工作目錄(不就地分析)
stage() {
  local bin="$1"
  [ -f "$ORIG/$bin" ] || { echo "[ida.sh] $ORIG/$bin 不存在" >&2; exit 1; }
  [ -f "$WORK/$bin" ] || cp "$ORIG/$bin" "$WORK/$bin"
}

case "${1:-}" in
  analyze)
    bin="${2:?用法: tools/ida.sh analyze <執行檔名>}"
    stage "$bin"
    echo "[ida.sh] $(sha256sum "$WORK/$bin" | cut -c1-16)…  $bin"
    run idat -A -B "$bin"
    [ -f "$WORK/$bin.i64" ] || { echo "[ida.sh] ❌ 沒產出 $bin.i64" >&2; exit 1; }
    ;;

  run)
    script="${2:?用法: tools/ida.sh run <腳本路徑> <執行檔名> [參數...]}"
    bin="${3:?用法: tools/ida.sh run <腳本路徑> <執行檔名> [參數...]}"
    shift 3
    [ -f "$ROOT/$script" ] || { echo "[ida.sh] $ROOT/$script 不存在" >&2; exit 1; }
    [ -f "$WORK/$bin.i64" ] || { echo "[ida.sh] 先跑 analyze $bin" >&2; exit 1; }
    # tools/ 掛在 /work/tools,所以 $script(= tools/ida/foo.py)直接接 /work/ 就對
    spec="-S/work/$script $*"
    run idat -A "$spec" "$bin.i64"
    ;;

  raw)
    shift
    run "$@"
    ;;

  *)
    sed -n '2,10p' "$0"
    exit 1
    ;;
esac
