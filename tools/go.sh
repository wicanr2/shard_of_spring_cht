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
#   只掛 engine/ build/ workplace/ game/ assets/ 與快取,不掛整個 repo
#   game/ 一律 **:ro** —— CLAUDE.md §8「game/ 與 original/ 唯讀」
#   assets/ 也是 **:ro** —— T1 整合測試(docs/spec/14 §8)要讀真正的資產,
#     但存檔會寫回 <assets>/save/*.DAT,所以測試自己複製一份到 t.TempDir(),
#     版控裡的那份只讀不寫。路徑由 SHARD_ASSETS 傳進去。
#
# ⛔ 本腳本**不做任何 docker 清理**。要空間請人工列候選清單再決定,
#    禁止 image/system/volume/builder prune 與 rmi(CLAUDE.md §8)。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${GO_IMAGE:-shard-go-build:6}"
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
    -e SHARD_ASSETS=/assets \
    -e SHOT_DIR -e SHOT_ASSETS -e PROMO_DIR -e SHARD_NOSOUND \
    --memory 4g --pids-limit 512 \
    ${NET_ARGS[@]+"${NET_ARGS[@]}"} \
    -v "$ROOT/engine":/src \
    -v "$ROOT/build":/out \
    -v "$ROOT/workplace":/workplace \
    -v "$ROOT/game":/game:ro \
    -v "$ROOT/assets":/assets:ro \
    -v "$ROOT/translations":/translations:ro \
    -v /usr/share/fonts:/usr/share/fonts:ro \
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
    # ⚠ 測試需要一個 X server:`package main` 匯入 ebiten 時,`internal/ui` 的
    # `init()` 會呼叫 glfw.Init(),沒有 DISPLAY 就 panic —— 那發生在任何測試
    # 跑起來之前,連空測試都擋。細節見 Dockerfile.go-build 的註解。
    #
    # ⚠ **不要用 `xvfb-run`。** 它是個 shell script,在容器裡當 PID 1 時
    # `go test` 結束之後它自己不會退 —— 現象是**容器永遠掛著、零輸出**,
    # 看起來像測試跑很久,實際上 `ps` 裡連 `go` 都沒有了。
    # 自己起 Xvfb 再設 DISPLAY,行為明確得多。
    # ⚠ 順手建一個空的 .Xauthority:少了它 ebiten 的 XGB 每次都會噴兩行
    # 「Could not get authority info」——無害,但它出現在每一次 test 輸出裡,
    # 會蓋掉真正的訊息,也讓人以為 X 沒接上。
    run sh -c ': > "$HOME/.Xauthority"
               Xvfb :99 -screen 0 640x480x24 -nolisten tcp >/dev/null 2>&1 &
               for i in $(seq 1 50); do [ -e /tmp/.X11-unix/X99 ] && break; sleep 0.1; done
               DISPLAY=:99 go test ./... "$@"' -- "$@"
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
