"""檢查引擎的**使用者可見字串**有沒有漏掉中文化。

為什麼需要:`docs/re/62` 的 1,476 段是**原版文本的盤點**,
而 remake 有自己的介面字串 —— 兩者不是同一件事
(`docs/spec/11` §6:逐字搬原版字串會搬出對不上的東西)。
真正該量的是「玩家在畫面上看得到英文嗎」。

做法:掃 `engine/**/*.go` 裡傳給繪製函式的字串常數,
挑出含拉丁字母而**不含任何中日韓字元**的那些,逐條列出。

⚠ **會有合理的例外**:角色名(玩家自取)、`HP`/`SP` 這類欄位標題、
除錯用的旗標說明。所以本工具**不自動判定通過與否**,只列出清單 ——
判斷留給人,而清單讓「有沒有變多」看得出來。
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
# 會把字串畫到畫面上的呼叫
DRAW = re.compile(r'\b(?:Draw|DrawRight)\(\s*[^,]+,\s*(.+)$')
STRLIT = re.compile(r'"((?:[^"\\]|\\.)*)"')
CJK = re.compile(r'[　-〿一-鿿＀-￯]')
LATIN = re.compile(r'[A-Za-z]')

# 明列的例外:這些英文是刻意留的。
ALLOW = {
    "HP", "SP", "#", "%c) %s", "%s", "",
}


def main():
    found = []
    for p in sorted(ROOT.joinpath("engine").rglob("*.go")):
        if p.name.endswith("_test.go"):
            continue
        for n, line in enumerate(p.read_text(encoding="utf-8").splitlines(), 1):
            m = DRAW.search(line)
            if not m:
                continue
            for lit in STRLIT.findall(m.group(1)):
                if lit in ALLOW or not LATIN.search(lit) or CJK.search(lit):
                    continue
                found.append((p.relative_to(ROOT), n, lit))
    for f, n, lit in found:
        print(f"{f}:{n}  {lit!r}")
    print(f"\n畫到畫面上、含英文且不含中文的字串常數:{len(found)} 條")
    print("⚠ 本工具不自動判定通過與否 —— 角色名與欄位標題是合理的例外。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
