"""替 `translations/module-text/*.tsv` 的 `wired` 欄批次填值(F3)。

用法:

    python3 tools/set_wired.py CMBT:69-72,74 engine/internal/combat/field.go
    python3 tools/set_wired.py CAMP:145-149 engine/internal/original/chars.go;engine/internal/rules/race.go

第一個參數是 `模組:列號清單`(逗號分隔,支援 `a-b` 區間),第二個是接線的檔案
(分號分隔,相對 repo 根目錄)。**只填欄位,不驗證** —— 驗證是
`tools/check_module_text.py` 的工作,填完一定要跑它。

⚠ 這支工具存在的理由是「手改 TSV 容易改錯欄位」,不是「可以少讀一次程式碼」。
`wired` 的意思是**這段譯文真的被那個檔案組進畫面**,而檢查工具只驗得了
「這串字有出現在檔案裡」——「是」「否」這種一兩個字的譯文在任何檔案裡都找得到,
逐字比對永遠會過。填之前要自己讀過呼叫端。
"""
import csv
import pathlib
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent


def parse_rows(spec: str) -> tuple[str, set[str]]:
    mod, _, rows = spec.partition(":")
    want: set[str] = set()
    for part in rows.split(","):
        part = part.strip()
        if not part:
            continue
        if "-" in part:
            a, b = part.split("-")
            want.update(str(i) for i in range(int(a), int(b) + 1))
        else:
            want.add(part)
    return mod, want


def main() -> int:
    if len(sys.argv) != 3:
        print(__doc__)
        return 2
    mod, want = parse_rows(sys.argv[1])
    wired = sys.argv[2]

    path = ROOT / "translations" / "module-text" / f"{mod}.tsv"
    rows = list(csv.reader(path.open(encoding="utf-8"), delimiter="\t"))
    header = rows[0]
    wi = header.index("wired")
    key = "row" if "row" in header else "id"
    ki = header.index(key)

    hit = set()
    for r in rows[1:]:
        if not r or r[ki] not in want:
            continue
        while len(r) <= wi:
            r.append("")
        r[wi] = wired
        hit.add(r[ki])

    with path.open("w", encoding="utf-8", newline="") as f:
        csv.writer(f, delimiter="\t", lineterminator="\n").writerows(rows)

    miss = sorted(want - hit, key=int)
    print(f"{mod}:填了 {len(hit)} 列")
    if miss:
        print(f"  ⚠ 這幾列在 {path.name} 裡不存在:{', '.join(miss)}")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
