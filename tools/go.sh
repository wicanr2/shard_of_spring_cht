#!/usr/bin/env bash
# Go 建置包裝 —— 一律走 docker(CLAUDE.md §8),不裝系統 Go。
#
#   tools/go.sh test            跑單元測試
#   tools/go.sh build           建出 build/shard(Linux)
#   tools/go.sh tidy            go mod tidy
#   tools/go.sh <任意 go 子指令>
#
# 邊界寫在腳本裡而不是對話裡,因為下一個 session 讀的是腳本:
#   --rm                用完即拆
#   --log-opt           daemon 預設的 json-file 沒有 rotation(370 GB 事故)
#   -u $(id -u)         不留 root-owned 檔案
#   --memory/--pids     上限
#   只掛 engine/ build/ workplace/ game/ 與快取,不掛整個 repo
#   game/ 一律 **:ro** —— CLAUDE.md §8「game/ 與 original/ 唯讀」
#
# ⛔ 本腳本**不做任何 docker 清理**。要空間請人工列候選清單再決定,
#    禁止 image/system/volume/builder prune 與 rmi(CLAUDE.md §8)。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${GO_IMAGE:-shard-go-build:2}"
CACHE="$ROOT/workplace/gocache"

mkdir -p "$CACHE/build" "$CACHE/mod" "$ROOT/build"

# image 不存在就建。只建本專案自己的那顆。
if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
  echo "→ 建置 $IMAGE(首次執行,約需數分鐘)" >&2
  docker build -f "$ROOT/tools/Dockerfile.go-build" -t "$IMAGE" "$ROOT/tools/"
fi

# NET_ARGS=(--network none) 給不需要網路的指令(test/build);
# tidy 要抓套件所以留空。
# ⚠ 空陣列展開要用 ${A[@]+"${A[@]}"};寫成 "${A[@]:-}" 會多出一個空字串參數,
#   docker 收到之後回 "invalid reference format",訊息完全看不出是這個原因。
NET_ARGS=()
run() {
  docker run --rm \
    --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" \
    -e HOME=/tmp -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
    -e GOFLAGS=-buildvcs=false \
    --memory 4g --pids-limit 512 \
    ${NET_ARGS[@]+"${NET_ARGS[@]}"} \
    -v "$ROOT/engine":/src \
    -v "$ROOT/build":/out \
    -v "$ROOT/workplace":/workplace \
    -v "$ROOT/game":/game:ro \
    -v "$CACHE":/gocache \
    -w /src \
    "$IMAGE" "$@"
}

case "${1:-}" in
  build)
    shift
    NET_ARGS=(--network none)
    run go build -o /out/shard "$@" .
    echo "→ build/shard"
    ;;
  test)
    shift
    NET_ARGS=(--network none)
    run go test ./... "$@"
    ;;
  tidy)
    shift
    NET_ARGS=()          # 需要網路
    run go mod tidy "$@"
    ;;
  *)
    NET_ARGS=()
    run go "$@"
    ;;
esac
