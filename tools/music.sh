#!/usr/bin/env bash
# 把樂譜渲染成 OGG,放進 engine/internal/music/assets/(進版控,由 go:embed 收進執行檔)。
#
#   tools/music.sh              重新產生全部八首
#   tools/music.sh --wav-only   只倒 WAV 到 workplace/music/(試聽用,不進版控)
#
# 為什麼要有這一步,而不是每次開機現算:
#   - 場景配樂是**循環播放**的,現算要把整首的 PCM 攤在記憶體裡;
#     OGG 壓完是十分之一以下,而且解碼是串流的
#   - 譜一旦定稿就不該每次啟動重算 —— 定稿的東西該是資產
#
# ⚠ 這一步**不是**建置的一環,是**產生資產**:改了 internal/music/cue.go 的譜
#    才要跑它,而且跑完要把 assets/*.ogg 一起 commit。
#    (少了這一步的症狀是「改了譜但遊戲裡沒變」,而且完全不會報錯。)
#
# ⚠ 原版那兩首(通關曲、死亡曲)的譜是**原版檔案裡的字串**(docs/re/148)。
#    轉成 OGG 只是換容器,不改變它是原版內容這件事 —— 與 score.go 同一個地位。
#
# 編碼器是 `oggenc`(vorbis-tools,在 tools/Dockerfile.go-build 裡)。
# Go 沒有成熟的純 Go Vorbis **編碼器**:ebiten 的 vorbis 套件只解不編。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST="$ROOT/engine/internal/music/assets"
WAVDIR="$ROOT/workplace/music"

mkdir -p "$WAVDIR"

echo "→ 渲染成 WAV" >&2
"$ROOT/tools/go.sh" run ./cmd/musicdump -out /workplace/music

if [ "${1:-}" = "--wav-only" ]; then
  echo "→ 只倒 WAV:$WAVDIR" >&2
  exit 0
fi

mkdir -p "$DEST"
echo "→ 壓成 OGG" >&2
# -q3 ≈ 112 kbps。⚠ 音源是**方波**,壓太狠會在方波的邊緣產生前迴響
# (pre-echo),聽起來像「音頭糊掉」—— 那是壓縮的假影,不是譜寫壞了。
docker run --rm \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  --memory 1g --pids-limit 128 --network none \
  -v "$WAVDIR":/wav -v "$DEST":/ogg \
  "${GO_IMAGE:-shard-go-build:5}" \
  sh -c 'set -e
         for f in /wav/*.wav; do
           n=$(basename "$f" .wav)
           oggenc -Q -q3 -o "/ogg/$n.ogg" "$f"
         done'

echo "→ 產物" >&2
ls -l "$DEST"
