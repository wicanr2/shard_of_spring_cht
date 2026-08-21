#!/usr/bin/env bash
# 把錄好的影格剪成推廣影片。
#
#   tools/promo.sh                     錄影格 + 剪片
#   tools/promo.sh --frames-only       只錄影格
#   tools/promo.sh --cut-only          只剪片(用既有影格)
#
# 產出:build/promo/shard-of-spring-cht-promo.mp4
#
# 影格由 `engine/promo_test.go` 錄:**走真的按鍵**驅動遊戲、每格寫一張 PNG。
# ⚠ 不接受靜態 fixture —— 推廣片要證明的是「玩得動」,
#    而擺好狀態拍出來的畫面看起來一樣,卻不是那個證據。
#
# 配樂用 `internal/music/assets/` 的 remake 曲子(本專案自己寫的譜)。
# ⛔ **不用原版那兩首**:通關曲配在世界地圖上就是 docs/spec/13 §5 擋掉的錯位置,
#    而且推廣片的觀眾沒有上下文,聽到就會以為原版長這樣。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FRAMES="$ROOT/workplace/promo-frames"
OUT="$ROOT/build/promo"
MUSIC="$ROOT/engine/internal/music/assets"
IMAGE="${GO_IMAGE:-shard-go-build:6}"
FPS=30
FONT=/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc

MODE="${1:-all}"

record() {
  echo "→ 錄影格" >&2
  rm -rf "$FRAMES"
  SHARD_NOSOUND=1 PROMO_DIR=/workplace/promo-frames \
    "$ROOT/tools/go.sh" test -count=1 -run TestPromoFrames
  echo "   $(find "$FRAMES" -name 'frame-*.png' | wc -l) 格" >&2
}

# 把某一段(以及它後面接的所有段)從影格序列裡拆掉,讓錄製步驟可以重跑。
#
# ⚠ **拆的是「該段起點之後的全部影格」,不是只拆那一段。** 後面接的段
#    (比較段 → 貢獻卡)編號都在它之後,只刪自己那一段會在序列中間留下空號,
#    而 `-i frame-%06d.png` 讀到空號就**安靜地停在那裡** ——
#    症狀是片子莫名其妙短了一截,不會有任何錯誤訊息。
drop_from() {
  local name="$1"
  [ -f "$FRAMES/beats.tsv" ] || return 0
  grep -q "	$name\$" "$FRAMES/beats.tsv" || return 0

  local start
  start=$(awk -F'\t' -v n="$name" '$2==n{print $1; exit}' "$FRAMES/beats.tsv")
  local f num
  for f in "$FRAMES"/frame-*.png; do
    num=${f##*frame-}; num=${num%.png}
    if [ "$((10#$num))" -ge "$start" ]; then rm -f "$f"; fi
  done
  awk -F'\t' -v s="$start" '$1 < s' "$FRAMES/beats.tsv" > "$FRAMES/beats.tsv.tmp"
  mv "$FRAMES/beats.tsv.tmp" "$FRAMES/beats.tsv"
}

# ── 原版 vs 重製版的比較畫面 ──────────────────────────────────────────
#
# 原版畫面是 `tools/dosbox_run.sh` 實跑抓的(路線見 docs/re/139),
# **不是掃描或網路上的圖** —— 比較段要成立,兩邊都得是自己跑出來的。
#
# ⚠ DOSBox 的截圖是 1024×768 的視窗,CGA 的 320×200 內容擺在左上角,
#    所以要先 `crop=320:200:0:0` 再放大。⛔ 放大一律 `flags=neighbor`:
#    雙線性會把 CGA 的硬邊糊掉,那就不是「原版長什麼樣」了。
ORIG_SHOTS="$ROOT/workplace/dosbox/shots"
REMAKE_SHOTS="$ROOT/docs/images"

compare() {
  local pairs=(
    "orig-01-title:01-title:標題畫面 —— 同一張 CGA 美術"
    "orig-02-menu:02-menu:主選單"
    "orig-03-world:03-world:世界地圖 —— 同一支隊伍、同一個位置"
  )
  # ⚠ 版面高度要跟 cut() 的字卡底板對齊:底板從 `ih-116`(= y 652)開始,
  #    所以最高的那塊面板(重製版 364 px)底邊不能超過它 —— y=268 → 632。
  #    先前放 y=300 會被切掉 12 px,而切掉的正好是提示列的下緣,看起來像沒切乾淨。
  local outdir="$FRAMES/compare"
  mkdir -p "$outdir"

  local p i=0
  for p in "${pairs[@]}"; do
    local o="${p%%:*}" rest="${p#*:}"
    local r="${rest%%:*}" title="${rest#*:}"
    [ -f "$ORIG_SHOTS/$o.png" ] || { echo "缺原版截圖:$o.png(先跑 tools/dosbox_run.sh)" >&2; return 1; }
    [ -f "$REMAKE_SHOTS/$r.png" ] || { echo "缺重製版截圖:$r.png" >&2; return 1; }

    printf '%s' "$title" > "$outdir/title-$i.txt"
    printf '%s' "原版 CGA 320×200(DOSBox 實跑)" > "$outdir/lo-$i.txt"
    printf '%s' "重製版 1024×768" > "$outdir/ro-$i.txt"

    docker run --rm \
      --log-opt max-size=10m --log-opt max-file=3 \
      -u "$(id -u):$(id -g)" \
      --memory 2g --pids-limit 256 --network none \
      -v "$ORIG_SHOTS":/orig:ro -v "$REMAKE_SHOTS":/remake:ro -v "$outdir":/out \
      -v /usr/share/fonts:/usr/share/fonts:ro -e HOME=/tmp \
      "$IMAGE" \
      ffmpeg -hide_banner -loglevel error -y \
        -i "/orig/$o.png" -i "/remake/$r.png" \
        -filter_complex "\
          [0:v]crop=320:200:0:0,scale=480:300:flags=neighbor,pad=484:304:2:2:0x606060[l];\
          [1:v]scale=480:360:flags=bicubic,pad=484:364:2:2:0x606060[r];\
          color=c=0x0b0f14:s=1024x768:d=1[bg];\
          [bg][l]overlay=22:268[a];[a][r]overlay=518:268[b];\
          [b]drawtext=fontfile=${FONT}:textfile=/out/title-$i.txt:fontcolor=0xf0f0f0:fontsize=40:x=(w-text_w)/2:y=138,\
             drawtext=fontfile=${FONT}:textfile=/out/lo-$i.txt:fontcolor=0xa8b4c0:fontsize=24:x=22+(484-text_w)/2:y=226,\
             drawtext=fontfile=${FONT}:textfile=/out/ro-$i.txt:fontcolor=0xa8b4c0:fontsize=24:x=518+(484-text_w)/2:y=226" \
        -frames:v 1 -update 1 "/out/compare-$i.png"
    i=$((i + 1))
  done

  drop_from 07-compare

  # 把比較畫面接到影格序列後面:一張定格 120 格(4 秒)。
  # ⚠ 直接續編號 + 在 beats.tsv 補一行,剪接那一段就不必特別處理 ——
  #    段落表是**唯一**決定配樂與字卡的地方(cut() 讀它)。
  local total start j
  total=$(find "$FRAMES" -maxdepth 1 -name 'frame-*.png' | wc -l)
  start=$total
  for ((j = 0; j < i; j++)); do
    local k
    for ((k = 0; k < 120; k++)); do
      cp "$outdir/compare-$j.png" "$(printf '%s/frame-%06d.png' "$FRAMES" "$total")"
      total=$((total + 1))
    done
  done
  printf '%d\t07-compare\n' "$start" >> "$FRAMES/beats.tsv"
  echo "   比較段 $((total - start)) 格" >&2
}

# ── 收尾:貢獻卡 ──────────────────────────────────────────────────────
#
# 片子最後兩張定格,講這個專案到底做了什麼。
#
# ⚠ **卡上的每個數字都要當場量得出來**,不要憑印象寫。
#    量法寫在各行後面的註解裡;改了之後重量一次再改字,
#    因為推廣片是對外的,而寫錯的數字沒有人會回頭查。
# ⚠ 卡片自己就是完整版面,所以 cut() 給它**空字卡** ——
#    再壓一條底板上去只會把最後一行蓋掉。
credits() {
  local outdir="$FRAMES/credits"
  mkdir -p "$outdir"
  rm -f "$outdir"/line-*.txt "$outdir"/credits-*.png

  # 一行一筆:「y|字級|顏色|對齊(c 置中 / l 靠左)|文字」
  local card0=(
    "120|46|0xf5f5f5|c|這個專案做了什麼"
    "196|26|0x7f8c99|c|Shard of Spring(SSI, 1986)—— 逆向、重製、繁體中文化"
    "286|30|0xdfe6ec|l|232 篇逆向工程筆記，十二個子系統全部收斂"          # ls docs/re/*.md | wc -l
    "336|30|0xdfe6ec|l|命中、傷害、經驗、升級、法術、遭遇 —— 規則從機器碼讀出來"
    "386|30|0xdfe6ec|l|33 個法術、74 種怪物、57 件道具，欄位語意逐欄對過"  # assets/data/*.json
    "436|30|0xdfe6ec|l|1,476 段原版文字全數盤點，繁體中文化"              # CLAUDE.md 2.2 L
    "486|30|0xdfe6ec|l|Go + Ebitengine 重寫，Windows / Linux / macOS 三平台"
    "600|30|0xc7a86a|c|定位是文化資產保存，不是「能跑就好」"
  )
  local card1=(
    "120|46|0xf5f5f5|c|規則不是猜的"
    "212|30|0xdfe6ec|l|每一條結論都要：在 IDA 裡讀到原始指令、確認讀寫端、"
    "262|30|0xdfe6ec|l|有一份獨立資料互相印證、筆記標明位址與信心等級"
    "342|30|0xdfe6ec|l|解不出來的欄位就標「未解」，不填一個看起來合理的值"
    "422|30|0xdfe6ec|l|被推翻的斷言集中記一處，正文永遠只寫現況"
    "502|30|0xdfe6ec|l|20,039 行引擎、14,348 行測試碼、509 條測試"           # find engine -name *.go
    "600|30|0xc7a86a|c|原版執行檔與資料不隨引擎散布，玩家自備合法原版"
  )

  local cards=(card0 card1)
  local c i=0
  for c in "${cards[@]}"; do
    local -n lines="$c"
    local draw="" l j=0
    for l in "${lines[@]}"; do
      local y="${l%%|*}"; local r1="${l#*|}"
      local sz="${r1%%|*}"; local r2="${r1#*|}"
      local col="${r2%%|*}"; local r3="${r2#*|}"
      local al="${r3%%|*}"; local txt="${r3#*|}"
      # 文字寫成檔案,不走 `text=` —— 逗號與冒號會把 filtergraph 切散
      # (全形的也一樣),而 ffmpeg 報的是「Both text and text file provided」。
      printf '%s' "$txt" > "$outdir/line-$i-$j.txt"
      local x="(w-text_w)/2"
      [ "$al" = l ] && x=150
      draw="${draw}drawtext=fontfile=${FONT}:textfile=/out/line-$i-$j.txt:fontcolor=${col}:fontsize=${sz}:x=${x}:y=${y},"
      j=$((j + 1))
    done
    draw="${draw%,}"

    docker run --rm \
      --log-opt max-size=10m --log-opt max-file=3 \
      -u "$(id -u):$(id -g)" \
      --memory 2g --pids-limit 256 --network none \
      -v "$outdir":/out -v /usr/share/fonts:/usr/share/fonts:ro -e HOME=/tmp \
      "$IMAGE" \
      ffmpeg -hide_banner -loglevel error -y \
        -f lavfi -i "color=c=0x0b0f14:s=1024x768:d=1" \
        -filter_complex "[0:v]drawbox=x=150:y=176:w=724:h=3:color=0x2b3742:t=fill,${draw}[v]" \
        -map '[v]' -frames:v 1 -update 1 "/out/credits-$i.png"
    i=$((i + 1))
  done

  drop_from 08-credits

  # 一張定格 150 格(5 秒)—— 比比較段慢,讓人讀得完。
  local total start k
  total=$(find "$FRAMES" -maxdepth 1 -name 'frame-*.png' | wc -l)
  start=$total
  local j
  for ((j = 0; j < i; j++)); do
    for ((k = 0; k < 150; k++)); do
      cp "$outdir/credits-$j.png" "$(printf '%s/frame-%06d.png' "$FRAMES" "$total")"
      total=$((total + 1))
    done
  done
  printf '%d\t08-credits\n' "$start" >> "$FRAMES/beats.tsv"
  echo "   貢獻卡 $((total - start)) 格" >&2
}

cut() {
  [ -f "$FRAMES/beats.tsv" ] || { echo "沒有影格,先跑 --frames-only" >&2; exit 1; }
  mkdir -p "$OUT"

  local total
  total=$(find "$FRAMES" -name 'frame-*.png' | wc -l)

  # 每一段配一首曲子。段落起點讀 beats.tsv,**不要在這裡重寫一份** ——
  # 兩邊各自寫死的話,改了錄製腳本這裡不會紅,只會配錯。
  # 段名 → 曲名 + 字卡
  local -a seg_name seg_start seg_track seg_caption
  while IFS=$'\t' read -r start name; do
    seg_start+=("$start")
    seg_name+=("$name")
    case "$name" in
      01-title)  seg_track+=(remake-title);  seg_caption+=("SSI 1986 —— 繁體中文重製版") ;;
      02-menu)   seg_track+=(remake-title);  seg_caption+=("原版沒有「開新遊戲」:先造角色,再載入隊伍") ;;
      03-modern-ui) seg_track+=(remake-title); seg_caption+=("F6 獨立切換 —— 原版／手繪冒險書現代 UI") ;;
      03-world)  seg_track+=(remake-world);  seg_caption+=("世界地圖 —— 121 × 103 格") ;;
      04-town)   seg_track+=(remake-town);   seg_caption+=("十三座城鎮 —— 商店、旅店、酒館、訓練所、治療所") ;;
      05-maze)   seg_track+=(remake-maze);   seg_caption+=("六座地城 —— 光源決定你看得到多遠") ;;
      06-combat) seg_track+=(remake-combat); seg_caption+=("最終戰 —— 巨龍 ×2 與希瑞雅妮") ;;
      07-compare) seg_track+=(remake-title); seg_caption+=("左:原版(DOSBox 實跑)  右:重製版") ;;
      08-credits) seg_track+=(remake-title); seg_caption+=("") ;;
      *)         seg_track+=(remake-world);  seg_caption+=("") ;;
    esac
  done < "$FRAMES/beats.tsv"

  # ── 音軌:每一段把對應的曲子循環到該段長度,再接起來 ──────────────
  # ⚠ `-stream_loop -1` 要放在 `-i` **前面**(它是輸入選項)。
  # ⚠ **截長度要用 filtergraph 裡的 `atrim`,不要用輸入端的 `-t`。**
  #    `-stream_loop -1 -t D -i file` 不會把循環後的輸入截成 D ——
  #    症狀是**音軌比影像長一大截**(量到 66.5 秒 vs 37.2 秒),
  #    而 ffmpeg 一聲不吭,mp4 的長度取兩者較長的那個。
  local aparts=() atrims="" amaps="" i n dur
  n=${#seg_start[@]}
  for ((i = 0; i < n; i++)); do
    local frames
    if ((i + 1 < n)); then frames=$((seg_start[i+1] - seg_start[i]));
    else frames=$((total - seg_start[i])); fi
    dur=$(awk -v f="$frames" -v r="$FPS" 'BEGIN{printf "%.3f", f/r}')
    # 路徑用**容器裡**的 /music,不要用主機路徑再事後代換 ——
    # 代換寫在展開式裡看不出來,而錯了只會變成 ffmpeg 說「找不到檔」。
    aparts+=("-stream_loop" "-1" "-i" "/music/${seg_track[i]}.ogg")
    # 輸入 0 是影格序列,所以音訊輸入從 1 開始。
    atrims="${atrims}[$((i + 1)):a]atrim=duration=${dur},asetpts=PTS-STARTPTS[a${i}];"
    amaps="${amaps}[a${i}]"
  done

  # ── 字卡:每一段底部一行 ──────────────────────────────────────────
  # ⚠ **一定要用 `textfile=` 而不是 `text=`。** `filter_complex` 先用**逗號**
  #    切濾鏡、drawtext 再用**冒號**切選項,所以字卡裡只要有這兩個字元
  #    (中文全形的也一樣會被切)整條 filtergraph 就散掉,
  #    而 ffmpeg 報的是「Both text and text file provided」——
  #    那句話完全看不出是標點造成的。寫成檔案就沒有跳脫問題。
  local capdir="$FRAMES/captions"
  mkdir -p "$capdir"
  local draw="" t0 t1
  for ((i = 0; i < n; i++)); do
    [ -n "${seg_caption[i]}" ] || continue
    printf '%s' "${seg_caption[i]}" > "$capdir/$i.txt"
    t0=$(awk -v s="${seg_start[i]}" -v r="$FPS" 'BEGIN{printf "%.3f", s/r}')
    if ((i + 1 < n)); then
      t1=$(awk -v s="${seg_start[i+1]}" -v r="$FPS" 'BEGIN{printf "%.3f", s/r}')
    else
      t1=$(awk -v s="$total" -v r="$FPS" 'BEGIN{printf "%.3f", s/r}')
    fi
    # ⚠ **`drawbox` 的 `h` 是「方塊自己的高度」,不是畫面高度**(畫面高度是 `ih`)。
    #    寫成 `y=h-116` 會算成 `y=0`,底板畫到畫面**頂端** —— 而字卡文字用的
    #    `drawtext` 剛好相反,它的 `h` 就是畫面高度。兩個濾鏡同名不同義,
    #    症狀是「文字在下面、底色在上面」,看起來像畫面上方莫名其妙暗了一條。
    draw="${draw}drawbox=x=0:y=ih-116:w=iw:h=116:color=0x000000:t=fill:enable='between(t\,${t0}\,${t1})',"
    draw="${draw}drawtext=fontfile=${FONT}:textfile=/frames/captions/${i}.txt:fontcolor=0xf0f0f0:fontsize=32:x=(w-text_w)/2:y=h-74:enable='between(t\,${t0}\,${t1})',"
  done
  draw="${draw%,}"

  echo "→ 剪片($total 格 / $n 段)" >&2
  docker run --rm \
    --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" \
    --memory 4g --pids-limit 512 --network none \
    -v "$FRAMES":/frames:ro -v "$MUSIC":/music:ro -v "$OUT":/out \
    -v /usr/share/fonts:/usr/share/fonts:ro \
    -e HOME=/tmp \
    "$IMAGE" \
    ffmpeg -hide_banner -loglevel error -y \
      -framerate "$FPS" -i /frames/frame-%06d.png \
      "${aparts[@]}" \
      -filter_complex "[0:v]${draw}[v];${atrims}${amaps}concat=n=${n}:v=0:a=1[a]" \
      -map '[v]' -map '[a]' -shortest \
      -c:v libx264 -preset slow -crf 20 -pix_fmt yuv420p \
      -c:a aac -b:a 160k -movflags +faststart \
      /out/shard-of-spring-cht-promo.mp4

  ls -lh "$OUT"
}

case "$MODE" in
  all)           record; compare; credits; cut ;;
  --frames-only) record; compare; credits ;;
  --cut-only)    cut ;;
  --compare-only) compare; credits ;;
  --credits-only) credits ;;
  *) echo "未知選項:$MODE" >&2; exit 2 ;;
esac
