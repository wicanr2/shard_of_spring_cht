"""驗證 translations/module-text/*.tsv 的 `wired` 欄有沒有跟引擎脫鉤。

為什麼需要:F1(docs/spec/19-module-text.md)把 TSV 的 `translation` 逐字抄進
`engine/**/*.go` 之後,**單一真相就變成兩份**——TSV 與原始碼。兩份會漂:
有人改字只改了一邊,測試不會抓到(測試斷言的是行為,不是字面),而漂移的症狀是
「翻譯明明改過,畫面卻沒變」,不會報錯,只會被忽略。

做法:每一列 `wired` 非空的,把它當成「這段譯文接到哪個/哪些 .go 檔」,
驗證 `translation` 欄的內容(去頭尾空白)**逐字**出現在那個檔案的原始碼文字裡。
對不上就報出來 —— 不管是譯文改了沒同步、還是 `wired` 標錯了地方。

⚠ **只驗證「有沒有出現」,不驗證「出現的地方對不對」。** 逐字比對抓不到
「這句剛好在別的地方出現,但不是我以為的那個呼叫點」這種假陽性 ——
那是 code review 的工作,不是這支工具的範圍。

`wired` 欄可以填一個檔案,也可以填多個(分號 `;` 分隔),路徑相對於 repo 根目錄。
去頭尾空白是刻意的:原版字串常常帶著定寬欄位的補位空白(例如 CAMP 第 6 列
`      Camp:` 前面 6 格空白),那是 DOS 文字模式的排版殘留,不是譯文本身要
逐字重現的內容——移植後的版面用像素定位,不用空白補位。
"""
import csv
import pathlib
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
MODULE_TEXT_DIR = ROOT / "translations" / "module-text"


def load_rows(path: pathlib.Path):
    with path.open(encoding="utf-8", newline="") as f:
        r = csv.reader(f, delimiter="\t")
        header = next(r)
        for row in r:
            if not row:
                continue
            yield dict(zip(header, row))


def main():
    file_cache: dict[str, str] = {}

    def read(rel: str) -> str | None:
        if rel not in file_cache:
            p = ROOT / rel
            file_cache[rel] = p.read_text(encoding="utf-8") if p.is_file() else None
        return file_cache[rel]

    wired_ok = 0
    mismatches = []  # (tsv名, row/id, wired欄, translation, 沒找到的檔案清單)

    for path in sorted(MODULE_TEXT_DIR.glob("*.tsv")):
        for row in load_rows(path):
            wired = (row.get("wired") or "").strip()
            if not wired:
                continue
            translation = (row.get("translation") or "").strip()
            key = row.get("row") or row.get("id") or "?"
            if not translation:
                mismatches.append((path.name, key, wired, translation,
                                    ["(translation 欄是空的)"]))
                continue
            missing = []
            found_anywhere = False
            for rel in [x.strip() for x in wired.split(";") if x.strip()]:
                content = read(rel)
                if content is None:
                    missing.append(f"{rel}(檔案不存在)")
                    continue
                if translation in content:
                    found_anywhere = True
                else:
                    missing.append(rel)
            if found_anywhere:
                wired_ok += 1
            else:
                mismatches.append((path.name, key, wired, translation, missing))

    for tsv, key, wired, translation, missing in mismatches:
        print(f"{tsv}:{key}  wired={wired!r}  translation={translation!r}")
        print(f"  沒在裡面找到:{', '.join(missing)}")

    total = wired_ok + len(mismatches)
    print(f"\n已接 {wired_ok} 段、對不上 {len(mismatches)} 段"
          f"(wired 非空共 {total} 段)。")
    wiring()
    rc = coverage()
    return 1 if (mismatches or rc) else 0


def wiring() -> None:
    """F3 的接線進度:哪幾個模組還有多少段沒接回引擎。

    ⚠ **未接不等於沒做完翻譯**(那是 F1,已經 381/381),也不等於缺功能 ——
    三種都混在這個數字裡:引擎真的還沒用原版措辭、原版把同一句拆成好幾段
    而引擎併成一句、以及**引擎根本沒有那個畫面**(後者是 docs/spec/19-coverage.md
    的工作)。⛔ 所以這個數字**不要當成完成度**,它只說「還有多少段沒有落點」。
    """
    done = todo = 0
    per: dict[str, int] = {}
    for path in sorted(MODULE_TEXT_DIR.glob("*.tsv")):
        for row in load_rows(path):
            if not (row.get("translation") or "").strip():
                continue
            if (row.get("wired") or "").strip():
                done += 1
            else:
                todo += 1
                per[path.stem] = per.get(path.stem, 0) + 1
    detail = "、".join(f"{k} {v}" for k, v in sorted(per.items(), key=lambda kv: -kv[1]))
    print(f"F3 接線進度:{done} / {done + todo} 段已接回引擎;未接 {todo} 段({detail})。")


def coverage() -> int:
    """F1 的覆蓋率:清冊裡每一段 `ui` 是不是都有譯文。

    ⚠ **不要用「總數相等」判斷完成。** 譯文分在兩個地方:主檔
    `module-text/<模組>.tsv` 用清冊的 `row` 對應,而 11 段酒館傳聞在
    `module-text/TOWN-rumors.tsv`,那個檔有自己的 `id`(1–11)、對到清冊的
    TOWN 第 62–72 列。兩邊總數剛好都是 381 —— **相加相等是巧合**,
    逐段比對才看得出主檔其實少了那 11 段。
    """
    src = {}
    for path in sorted((ROOT / "translations" / "source").glob("module-*.tsv")):
        mod = path.stem[len("module-"):]
        with path.open(encoding="utf-8") as fh:
            for row in csv.DictReader(fh, delimiter="\t"):
                if row.get("category") == "ui":
                    src[(mod, row["row"])] = row["original"]

    done = set()
    for path in sorted((ROOT / "translations" / "module-text").glob("*.tsv")):
        with path.open(encoding="utf-8") as fh:
            for row in csv.DictReader(fh, delimiter="\t"):
                if not (row.get("translation") or "").strip():
                    continue
                if path.stem == "TOWN-rumors":
                    # 傳聞的 id 1–11 對到清冊 TOWN 的第 62–72 列(docs/re/138 §4)。
                    done.add(("TOWN", str(61 + int(row["id"]))))
                else:
                    done.add((path.stem, row["row"]))

    missing = sorted(src.keys() - done)
    print(f"F1 覆蓋率:清冊 ui {len(src)} 段,有譯文 {len(src) - len(missing)} 段。")
    for mod, row in missing[:20]:
        print(f"  缺:{mod} 第 {row} 列  {src[(mod, row)][:40]!r}")
    if missing:
        print(f"  ⚠ 還有 {len(missing)} 段沒有譯文")
    return 1 if missing else 0


if __name__ == "__main__":
    sys.exit(main())
