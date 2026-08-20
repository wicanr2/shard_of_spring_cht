#!/usr/bin/env bash
# 把「發行要交出去的東西」集中到一個目錄:三平台包、release notes、推廣片。
#
#   tools/dist_all.sh            用最新的 release
#   tools/dist_all.sh v0.4.0     指定版本
#
# 產出:dist-all/(gitignore —— 裡面是二進位,repo 不收)
#
# ⚠ **三個包一律從 GitHub release 下載,不從 `build/release/` 複製。**
#    要集中的是「已經發出去的那一份」,而本機那份可能被後來的重編蓋過 ——
#    兩者長得一模一樣,但只有前者是玩家真的會拿到的東西。
#    本機若也有同名檔,會逐檔比 SHA-256;不一致就中止(表示本機重編過)。
#
# ⛔ 不附帶 `assets/`(CLAUDE.md §1)。腳本會**把三個包都實際翻開**檢查,
#    不是靠 release.sh 的承諾 —— 承諾與事實要分開驗。
#    ⚠ AppImage 用 `--appimage-extract`,**那條路不需要 FUSE**;
#      以為要 FUSE 而略過它,等於三個包只驗了兩個。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/dist-all"
PROMO="$ROOT/docs/promo/shard-of-spring-cht-promo.mp4"

# CLAUDE.md §10:碰 repo 之前先問一次可見性,那是狀態不是設定。
VIS=$(gh repo view --json visibility -q .visibility)
echo "[dist] repo 可見性:$VIS"
[ "$VIS" = PRIVATE ] || { echo "⛔ repo 不是 PRIVATE,停手(CLAUDE.md §10)" >&2; exit 1; }

VER="${1:-$(gh release view --json tagName -q .tagName)}"
echo "[dist] 版本:$VER"

rm -rf "$OUT"; mkdir -p "$OUT"

echo "[dist] 下載三平台包"
gh release download "$VER" -D "$OUT" \
  -p '*-linux-x86_64.AppImage' -p '*-windows-amd64.zip' -p '*-macos-universal.tar.gz'
chmod +x "$OUT"/*.AppImage

# 與本機建出來的那份對照(有的話)。不一致 = 本機重編過,集中的東西會誤導。
local_dir="$ROOT/build/release"
for f in "$OUT"/*; do
  b=$(basename "$f")
  [ -f "$local_dir/$b" ] || continue
  a=$(sha256sum "$f" | cut -d' ' -f1)
  c=$(sha256sum "$local_dir/$b" | cut -d' ' -f1)
  if [ "$a" = "$c" ]; then echo "   ✓ $b 與本機建出來的相同"
  else echo "   ⛔ $b 與 build/release/ 不一致 —— 本機重編過?" >&2; exit 1; fi
done

echo "[dist] 檢查包裡沒有 assets/"
python3 - "$OUT" <<'PY'
import sys, pathlib, zipfile, tarfile
d = pathlib.Path(sys.argv[1])
bad = []
for f in sorted(d.iterdir()):
    if f.suffix == ".zip":
        names = zipfile.ZipFile(f).namelist()
    elif f.name.endswith(".tar.gz"):
        names = tarfile.open(f).getnames()
    elif f.suffix == ".AppImage":
        continue                      # 下面用 --appimage-extract 單獨查
    else:
        continue
    hits = [n for n in names if "/assets/" in n or n.startswith("assets/")]
    print(f"   {f.name}:{len(names)} 項,assets/ {len(hits)} 項")
    bad += hits
if bad:
    print("⛔ 包裡有 assets/:", bad[:5], file=sys.stderr); sys.exit(1)
PY
# AppImage 是 squashfs,但 **`--appimage-extract` 不需要 FUSE**(那是 runtime
# 自己實作的)—— 所以三個包都翻得開,不必拿「出自同一個 stage」當理由略過。
tmpx=$(mktemp -d)
( cd "$tmpx" && "$OUT"/*.AppImage --appimage-extract >/dev/null )
n=$(find "$tmpx/squashfs-root" -type f | wc -l)
hits=$(find "$tmpx/squashfs-root" -path '*assets*' | wc -l)
echo "   $(basename "$OUT"/*.AppImage):$n 項,assets/ $hits 項"
rm -rf "$tmpx"
[ "$hits" = 0 ] || { echo "⛔ AppImage 裡有 assets/" >&2; exit 1; }

echo "[dist] release notes 與推廣片"
gh release view "$VER" --json body -q .body > "$OUT/RELEASE-NOTES-$VER.md"
[ -f "$PROMO" ] || { echo "⛔ 找不到推廣片 $PROMO(先跑 tools/promo.sh)" >&2; exit 1; }
cp "$PROMO" "$OUT/"

cat > "$OUT/README.md" <<EOF
# 春之石 Shard of Spring 繁體中文重製版 — $VER 發行包

| 檔案 | 給誰 |
|---|---|
| \`shard-of-spring-cht-$VER-windows-amd64.zip\` | Windows 64 位元:解壓縮,跑 \`shard.exe\` |
| \`shard-of-spring-cht-$VER-linux-x86_64.AppImage\` | Linux 64 位元:\`chmod +x\` 之後直接執行,不用解壓縮 |
| \`shard-of-spring-cht-$VER-macos-universal.tar.gz\` | macOS(Intel 與 Apple Silicon 共用):**沒有簽章**,第一次要右鍵 →「打開」 |
| \`RELEASE-NOTES-$VER.md\` | 這一版有什麼 |
| \`shard-of-spring-cht-promo.mp4\` | 推廣片 |
| \`SHA256SUMS\` | 校驗:\`sha256sum -c SHA256SUMS\` |

## 開始玩之前要做一件事

包裡**沒有遊戲資料**。原版的執行檔、資料檔、美術不隨引擎散布,
所以第一次要拿自己那份合法原版(MS-DOS 版的 \`sharspri\` 資料夾)跑一次轉換器。
**三個平台的指令不一樣**:

\`\`\`
# Linux(AppImage 是唯讀掛載,所以資產寫到 ~/.local/share/shard-of-spring/assets)
./shard-of-spring-cht-$VER-linux-x86_64.AppImage --convert /路徑/sharspri

# Windows
.\\shard-convert.exe -in C:\\路徑\\sharspri -out assets

# macOS
./shard-convert -in /路徑/sharspri -out assets
\`\`\`

轉完再開 \`shard\`(Linux 直接再跑一次 AppImage)。玩法見包裡的 \`PLAYING.md\`。

## 這份包是怎麼來的

三個平台包從 GitHub release **原樣下載**(不是本機重建的),
推廣片是 \`tools/promo.sh\` 錄的。重建這個目錄:\`tools/dist_all.sh $VER\`。

⚠ 推廣片與 release notes 含原版美術與原版措辭,與 repo 同一個地位:
**對外散布要專案負責人另行決定**。
EOF

# ⚠ 校驗檔要**最後**才產:排在 README 之前的話,清單裡就少了 README 那一行,
#    而 `sha256sum -c` 只驗清單上有的,少一行不會紅。
( cd "$OUT" && sha256sum -- * > SHA256SUMS.tmp && mv SHA256SUMS.tmp SHA256SUMS )

echo
ls -lh "$OUT"
echo
echo "[dist] 完成:$OUT"
