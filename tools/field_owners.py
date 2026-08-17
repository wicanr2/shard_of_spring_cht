#!/usr/bin/env python3
"""量「`*Game` 的每個欄位被哪些檔案碰」——C2 重構的判準要靠數字,不靠感覺。

    python3 tools/field_owners.py            # 摘要 + 跨檔最嚴重的前 15 名
    python3 tools/field_owners.py --all      # 每個欄位一行

`docs/spec/14-remake-worklist.md` §4 把 C2(把狀態切進各自的套件)標成暫緩,
重啟判準是「**改一個場景時不小心動到另一個場景的狀態**」。那個判準有一個問題:
它只有在**已經出事**的時候才觸發。這支工具量的是它的前置條件 ——
**一個欄位被幾個檔案碰**。碰的檔案越多,誤傷的機會越大。

⚠ 這是**粗略的樣式比對**,不是型別分析:
`g.foo` 只認 `g.` 這個接收器名(`engine/` 全域統一用 `g`),
而測試檔一律不算(測試本來就會跨場景擺狀態)。
數字用來排序與比較,不要拿去當「呼叫次數」。
"""
import argparse
import collections
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
ENGINE = ROOT / "engine"

# `*Game` 的欄位宣告:從 main.go 的 struct 裡取,而不是從用法反推 ——
# 反推會把方法名(g.play(...))混進來。
STRUCT = re.compile(r"type Game struct \{(.*?)\n\}", re.S)
FIELD = re.compile(r"^\t([A-Za-z_][A-Za-z0-9_]*)(?:,\s*[A-Za-z0-9_]+)*\s+\S", re.M)


def game_fields() -> list[str]:
    src = (ENGINE / "main.go").read_text(encoding="utf-8")
    m = STRUCT.search(src)
    if not m:
        print("找不到 type Game struct —— main.go 改過了?", file=sys.stderr)
        sys.exit(2)
    out = []
    for line in m.group(1).split("\n"):
        fm = re.match(r"\t([A-Za-z_][A-Za-z0-9_]*(?:, *[A-Za-z_][A-Za-z0-9_]*)*) +[^/\s]", line)
        if fm:
            out += [n.strip() for n in fm.group(1).split(",")]
    return out


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--all", action="store_true", help="每個欄位都印")
    args = ap.parse_args()

    fields = game_fields()
    files = sorted(p for p in ENGINE.glob("*.go") if not p.name.endswith("_test.go"))
    texts = {p.name: p.read_text(encoding="utf-8") for p in files}

    touched: dict[str, set[str]] = collections.defaultdict(set)
    for f in fields:
        pat = re.compile(r"\bg\." + re.escape(f) + r"\b")
        for name, text in texts.items():
            if pat.search(text):
                touched[f].add(name)

    rows = sorted(((len(touched[f]), f) for f in fields), reverse=True)
    print(f"# `*Game` 有 {len(fields)} 個欄位;掃了 {len(files)} 個非測試檔\n")
    spread = [n for n, _ in rows]
    print(f"跨檔數:中位數 {sorted(spread)[len(spread)//2]}、"
          f"最大 {max(spread)}、只被 1 個檔案碰的有 {sum(1 for n in spread if n <= 1)} 個\n")
    show = rows if args.all else rows[:15]
    for n, f in show:
        if n == 0:
            continue
        print(f"{n:2d}  g.{f:<18} {' '.join(sorted(touched[f]))}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
