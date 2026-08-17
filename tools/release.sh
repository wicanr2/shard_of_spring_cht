#!/usr/bin/env bash
# 打包發行版 —— 三平台四架構,一律走 docker(CLAUDE.md §8)。
#
#   tools/release.sh v0.1.0        建出 build/release/ 底下的壓縮檔
#   tools/release.sh v0.1.0 linux  只建其中一個平台(linux / windows / macos)
#
# 產物(每個平台一包):
#   shard[.exe]         引擎
#   shard-convert[.exe] 轉換器(玩家用自備的原版跑它)
#   translations/       譯文 TSV(轉換器要讀)
#   PLAYING.md README.md
#
# ⛔ **不附帶 `assets/`**(CLAUDE.md §1)—— 那是原版資料與美術轉出來的,
#    repo 收錄 ≠ 發行附帶。玩家自備合法原版、自己跑一次轉換器。
# ⛔ **一律用預設 build tag**,不帶 `-tags eten` —— 倚天字型是 1993 年的
#    商業軟體,打包等於散布(docs/spec/21 §G1)。
# ⛔ 本腳本不做任何 docker 清理(禁止 image/system/volume/builder prune、rmi)。
#
# 平台對 cgo 的要求不一樣,這是三條路線分岔的唯一原因:
#   linux   ebiten 要 X11/OpenGL → **要 cgo**,在建置容器裡原生編
#   windows ebiten 走純 Go 的 syscall(DirectX/OpenGL)→ **CGO_ENABLED=0** 就能交叉編
#   macOS   ebiten 要 Cocoa/Metal → **要 cgo**,靠 osxcross 的 SDK,兩弧各編一次再 lipo
#
# macOS 那條線的原理與坑見 skill `osxcross-macos-cross-build`。
# ⚠ Linux 上驗不了 macOS 執行檔能不能跑,只能做靜態檢查(§驗收)。
set -euo pipefail

VER="${1:?用法:tools/release.sh <版本,如 v0.1.0> [linux|windows|macos]}"
ONLY="${2:-all}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/build/release"
STAGE="$ROOT/build/stage"
CACHE="$ROOT/workplace/gocache"
GO_IMAGE="${GO_IMAGE:-shard-go-build:4}"
MAC_IMAGE="${MAC_IMAGE:-wolong-osxcross-go:20260811-event10-r4}"

# 版本字串編進執行檔。ldflags 的 -s -w 去掉除錯符號,binary 小一半。
LDFLAGS="-s -w -X main.version=$VER"

mkdir -p "$OUT" "$STAGE" "$CACHE/build" "$CACHE/mod"
rm -rf "${STAGE:?}/"*

# 共用的 docker 參數。GOTOOLCHAIN=local:go.mod 的 toolchain 指令在
# --network none 底下會想下載,設 local 才會用 image 裡那份。
docker_go() {
  local image="$1"; shift
  docker run --rm \
    --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" \
    -e HOME=/tmp -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
    -e GOFLAGS=-buildvcs=false -e GOTOOLCHAIN=local \
    --memory 4g --pids-limit 512 --network none \
    -v "$ROOT/engine":/src:ro \
    -v "$STAGE":/out \
    -v "$CACHE":/gocache \
    -w /src \
    "$image" "$@"
}

# 每一包共同的非執行檔內容。
common_files() {
  local dir="$1"
  mkdir -p "$dir/translations"
  # 轉換器只讀這四類(engine/cmd/convert/main.go 的 loadLang)。
  # glossary.md / source/ 的其餘檔案是工作用的,不進發行包。
  cp -r "$ROOT/translations/names" "$dir/translations/"
  cp -r "$ROOT/translations/dungeon-text" "$dir/translations/"
  mkdir -p "$dir/translations/source" "$dir/translations/module-text"
  cp "$ROOT/translations/source/towndata.tsv" "$dir/translations/source/"
  cp "$ROOT/translations/module-text/TOWN-rumors.tsv" "$dir/translations/module-text/"
  cp "$ROOT/docs/PLAYING.md" "$dir/"
  cp "$ROOT/README.md" "$dir/"
}

say() { printf '\n\033[1m→ %s\033[0m\n' "$*" >&2; }

# ── Linux ────────────────────────────────────────────────────────────────
build_linux() {
  say "linux/amd64(cgo,原生編)"
  docker_go "$GO_IMAGE" go build -ldflags "$LDFLAGS" -o /out/shard-linux .
  docker_go "$GO_IMAGE" go build -ldflags "$LDFLAGS" -o /out/shard-convert-linux ./cmd/convert

  local d="$STAGE/shard-of-spring-cht-$VER-linux-amd64"
  mkdir -p "$d"
  mv "$STAGE/shard-linux" "$d/shard"
  mv "$STAGE/shard-convert-linux" "$d/shard-convert"
  chmod +x "$d/shard" "$d/shard-convert"
  common_files "$d"
  tar -C "$STAGE" -czf "$OUT/$(basename "$d").tar.gz" "$(basename "$d")"
}

# ── Windows ──────────────────────────────────────────────────────────────
build_windows() {
  say "windows/amd64(CGO_ENABLED=0,純 Go 交叉編)"
  # -H=windowsgui:不要在遊戲後面開一個主控台視窗。
  # ⚠ 轉換器**不能**加這個旗標 —— 它靠 stdout 報告轉了幾筆,
  #   加了之後玩家在 Windows 上看不到任何輸出,失敗也不會有訊息。
  docker_go "$GO_IMAGE" env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
    go build -ldflags "$LDFLAGS -H=windowsgui" -o /out/shard.exe .
  docker_go "$GO_IMAGE" env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
    go build -ldflags "$LDFLAGS" -o /out/shard-convert.exe ./cmd/convert

  local d="$STAGE/shard-of-spring-cht-$VER-windows-amd64"
  mkdir -p "$d"
  mv "$STAGE/shard.exe" "$STAGE/shard-convert.exe" "$d/"
  common_files "$d"
  ( cd "$STAGE" && zip -qr "$OUT/$(basename "$d").zip" "$(basename "$d")" )
}

# ── macOS ────────────────────────────────────────────────────────────────
build_macos() {
  say "darwin arm64 + amd64(cgo via osxcross),之後 lipo 成 universal"
  # 前綴帶 SDK 次版號(darwin24.5),skill §2:讀 OSXCROSS_TARGET,不要寫死。
  local tgt
  # ⚠ 不要寫成 `. osxcross-conf || eval "$(osxcross-conf)"`:`.` 是 POSIX 的
  # **special built-in**,找不到檔案時整個 `sh` 立刻以 2 結束,`||` 輪不到 ——
  # 現象是「零輸出、exit 2」,看不出是哪一句。osxcross-conf 本來就只是印
  # export 行,eval 它就好。
  tgt="$(docker run --rm --log-opt max-size=10m --log-opt max-file=3 --network none \
          "$MAC_IMAGE" sh -c 'eval "$(osxcross-conf)"; echo "$OSXCROSS_TARGET"')"
  [ -n "$tgt" ] || { echo "取不到 OSXCROSS_TARGET" >&2; return 1; }
  echo "   OSXCROSS_TARGET=$tgt" >&2

  local pkg
  for pkg in . ./cmd/convert; do
    local name=shard
    [ "$pkg" = ./cmd/convert ] && name=shard-convert
    local arch cc
    for arch in arm64 amd64; do
      case "$arch" in
        arm64) cc="aarch64-apple-$tgt-clang" ;;
        amd64) cc="x86_64-apple-$tgt-clang" ;;
      esac
      docker_go "$MAC_IMAGE" env GOOS=darwin GOARCH="$arch" CGO_ENABLED=1 \
        CC="$cc" CXX="${cc}++" \
        go build -ldflags "$LDFLAGS" -o "/out/$name-darwin-$arch" "$pkg"
    done
  done

  # lipo 在 osxcross 裡(前綴任一弧都行)。
  local d="$STAGE/shard-of-spring-cht-$VER-macos-universal"
  mkdir -p "$d"
  docker run --rm --log-opt max-size=10m --log-opt max-file=3 --network none \
    -u "$(id -u):$(id -g)" -v "$STAGE":/out -w /out "$MAC_IMAGE" sh -c "
      set -e
      for n in shard shard-convert; do
        x86_64-apple-$tgt-lipo -create \"\$n-darwin-arm64\" \"\$n-darwin-amd64\" \
          -output \"$(basename "$d")/\$n\"
      done"
  chmod +x "$d/shard" "$d/shard-convert"
  rm -f "$STAGE"/shard-darwin-* "$STAGE"/shard-convert-darwin-*
  common_files "$d"
  tar -C "$STAGE" -czf "$OUT/$(basename "$d").tar.gz" "$(basename "$d")"

  # 靜態驗收(skill §5)。Linux 上跑不了 macOS binary,這是能做的全部。
  say "macOS 靜態驗收"
  docker run --rm --log-opt max-size=10m --log-opt max-file=3 --network none \
    -u "$(id -u):$(id -g)" -v "$STAGE":/out -w /out "$MAC_IMAGE" sh -c "
      set -e
      T=x86_64-apple-$tgt
      for n in shard shard-convert; do
        B=\"$(basename "$d")/\$n\"
        echo \"-- \$n\"
        \$T-lipo -info \"\$B\"
        for a in arm64 x86_64; do
          \$T-lipo -thin \$a \"\$B\" -output /tmp/\$a
          # arm64 沒有 ad-hoc 簽章 → Apple Silicon 直接 Killed: 9
          if [ \$a = arm64 ] && ! \$T-otool -l /tmp/\$a | grep -q LC_CODE_SIGNATURE; then
            echo '   ✗ arm64 缺 LC_CODE_SIGNATURE' >&2; exit 1
          fi
          # 連到系統以外的動態庫 = 那台機器才有的路徑,玩家開不起來
          bad=\$(\$T-otool -L /tmp/\$a | tail -n +2 | awk '{print \$1}' \
                 | grep -vE '^(/usr/lib/|/System/Library/)' || true)
          [ -z \"\$bad\" ] || { echo \"   ✗ \$a 外部相依:\$bad\" >&2; exit 1; }
          # ⚠ 兩弧的最低版本用**不同的載入指令**記:arm64 走 LC_BUILD_VERSION
          # (有 minos 欄),x86_64 走舊式的 LC_VERSION_MIN_MACOSX(欄名是 version)。
          # 只 grep minos 會讓 x86_64 看起來像沒設,而它其實設了。
          v=\$(\$T-otool -l /tmp/\$a | grep -E '^ +(minos|version) ' | head -1 | awk '{print \$2}')
          echo \"   \$a ok(最低系統 \${v:-未標})\"
        done
      done"
}

case "$ONLY" in
  all)     build_linux; build_windows; build_macos ;;
  linux)   build_linux ;;
  windows) build_windows ;;
  macos)   build_macos ;;
  *) echo "未知平台:$ONLY" >&2; exit 2 ;;
esac

say "產物"
ls -lh "$OUT"
