#!/usr/bin/env bash
# 用 DOSBox 跑原版當 oracle(headless,全程 docker)。
#
#   tools/dosbox_run.sh "<timeline>" [cycles]
#
# timeline 用 ';' 分隔:
#   wait:N        等 N 秒
#   key:KEYSYM    送一個按鍵(xdotool keysym:Return / Escape / Up / Down …)
#   type:STRING   打字(不含 Enter)
#   shot:NAME     截圖存成 workplace/dosbox/shots/NAME.png
#
# 例:
#   tools/dosbox_run.sh "wait:5;shot:01-boot"
#   tools/dosbox_run.sh "wait:3;shot:01;key:Return;wait:2;shot:02"
#
# ⚠ cycles 預設寫死 "fixed 4000" —— `cycles=auto` 是可重現性的敵人
#    (~/.claude/knowledge-base/retro/dosbox-game-configs.md)。
#
# 顯示模式固定 cga(docs/re/20 已確認 320×200 2bpp)。
# 遊戲檔第一次跑會從唯讀的 game/sharspri/ 複製到 workplace/dosbox/game/;
# 要還原乾淨狀態就刪掉那個目錄再跑。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${DOSBOX_IMAGE:-demwin-dosbox:latest}"
GAME="$ROOT/workplace/dosbox/game"
SHOTS="$ROOT/workplace/dosbox/shots"
ORIG="$ROOT/game/sharspri"

TIMELINE="${1:-wait:5;shot:default}"
CYCLES="${2:-fixed 4000}"

docker image inspect "$IMAGE" >/dev/null 2>&1 || {
  echo "[dosbox] 找不到 image $IMAGE" >&2; exit 1; }

if [ ! -f "$GAME/START.EXE" ]; then
  echo "[dosbox] 建立可寫副本 $GAME"
  mkdir -p "$GAME"; cp -r "$ORIG"/. "$GAME"/; chmod -R u+w "$GAME"
fi
# 沿用的 image 其 entrypoint 寫死執行 DEMON,用轉接批次檔導到 START.EXE
printf 'START.EXE\r\n' > "$GAME/DEMON.BAT"
mkdir -p "$SHOTS"

echo "[dosbox] timeline=$TIMELINE  cycles=\"$CYCLES\""
docker run --rm --log-opt max-size=10m --log-opt max-file=3 \
  -v "$GAME:/game" -v "$SHOTS:/shots" \
  "$IMAGE" cga "$TIMELINE" "$CYCLES"
echo "[dosbox] 截圖在 $SHOTS"
ls -1 "$SHOTS" 2>/dev/null | tail -5
