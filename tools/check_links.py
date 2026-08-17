"""檢查 markdown 之間的相對連結指不指得到檔案。

用法: python3 tools/check_links.py     (exit 1 = 有斷掉的連結)

為什麼需要:這個專案的文件互相引用得很密 —— `docs/re/` 兩百多篇、
規格與 `CONTEXT.md` 又逐條指回去。**筆記改名的時候引用不會跟著改**,
而斷掉的連結在 GitHub 上長得跟正常連結一模一樣,只有點下去才知道。

⚠ 只檢查**相對路徑的 `.md`**:外部網址、錨點(`#section`)、程式碼路徑
都不在範圍內。錨點不檢查是刻意的 —— 中文標題產生的 anchor 規則
在不同 renderer 之間不一致,檢查它會製造假警報。
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
# 連結目標:排除 http(s)、mailto、純錨點
LINK = re.compile(r"\]\((?!https?:|mailto:|#)([^)\s]+\.md)(?:#[^)]*)?\)")


def targets():
    """要掃的檔案:docs/ 底下全部,加上根目錄與 translations/ 的幾份。"""
    seen = []
    seen += sorted(ROOT.glob("docs/**/*.md"))
    for rel in ("README.md", "CONTEXT.md", "CLAUDE.md", "translations/README.md"):
        p = ROOT / rel
        if p.is_file():
            seen.append(p)
    return seen


def main() -> int:
    bad = []
    files = targets()
    links = 0
    for p in files:
        for m in LINK.finditer(p.read_text(encoding="utf-8")):
            links += 1
            if not (p.parent / m.group(1)).resolve().is_file():
                bad.append((p.relative_to(ROOT), m.group(1)))
    print(f"掃了 {len(files)} 個 markdown、{links} 條 .md 連結。")
    if bad:
        for p, t in bad:
            print(f"  ✗ {p}: {t}")
        print(f"⚠ 有 {len(bad)} 條連結指不到檔案")
        return 1
    print("所有相對連結都指得到。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
