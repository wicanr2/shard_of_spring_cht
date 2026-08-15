#!/usr/bin/env python3
"""把模組裡的字串常數對回它們的 **DGROUP 位址**。

    python3 tools/dgroup_strings.py CAMP.EXE                 # 整張表
    python3 tools/dgroup_strings.py CAMP.EXE hunt lore       # 只印含關鍵字的
    python3 tools/dgroup_strings.py CAMP.EXE --at 72F6       # 查一個位址

**為什麼需要這個**:`docs/re/42` §3 的結論是「模組的變數大半不在檔案裡,
靜態分析看得到存取、看不到內容」—— 而 `mov bx, 72F2h` 這種運算元因此讀不出語意。
**字串常數是例外**:它們有初始值,而且初始值連同 DGROUP 位址一起寫在檔案裡。

記錄格式(`docs/re/162` §4):

    <長度:word> <DGROUP 位址:word> <間距:word> <DGROUP 位址:word> <文字>

間距 = 長度湊成偶數再 + 4,而**程式碼拿到的是描述子位址 = 文字位址 − 4**
(執行期的 DGROUP 是 `[長度][位址][文字]` 連續排列)。所以在反組譯裡看到
`mov bx, 72F2h` 要查的是 `ds:72F6`。本工具兩種都印。

⚠ 這是**樣式比對**,不是解析器:條件是「ptr 出現兩次 + 間距對得上長度」。
連續三個條件同時成立的機率低,但**沒有校驗碼** —— 印出來的東西要用肉眼確認
像不像字串。找不到某一段不代表它不存在,只代表它不符合這個樣式。
"""
import pathlib
import struct
import sys

BASE = 0xFE00  # `bz` 模組:線性位址 = 檔案位移 + 0xFE00(docs/re/03/04)


def records(d):
    """掃出所有 (DGROUP 位址, 檔案線性位址, 文字)。"""
    out, i = [], 0
    while i < len(d) - 8:
        ln, ptr, stride, ptr2 = struct.unpack_from("<HHHH", d, i)
        if 0 < ln < 250 and ptr == ptr2 and stride == ((ln + 1) // 2) * 2 + 4:
            body = d[i + 8:i + 8 + ln]
            # 只收看得懂的:全部可列印,或含 CR/LF
            if all(32 <= c < 127 or c in (9, 10, 13) for c in body):
                out.append((ptr, i + BASE, body.decode("latin-1")))
                i += 8 + ((ln + 1) // 2) * 2
                continue
        i += 1
    return out


def main(argv):
    if not argv:
        raise SystemExit(__doc__)
    root = pathlib.Path(__file__).resolve().parent.parent
    name = argv[0]
    p = pathlib.Path(name)
    if not p.exists():
        p = root / "game/sharspri" / name
    recs = records(p.read_bytes())

    if "--at" in argv:
        want = int(argv[argv.index("--at") + 1], 16)
        for ptr, a, t in recs:
            # 描述子位址 = 文字位址 − 4,兩種都比
            if ptr == want or ptr - 4 == want:
                print(f"ds:{ptr:04X}(描述子 ds:{ptr - 4:04X})  {t!r}")
        return

    keys = [a.lower() for a in argv[1:]]
    for ptr, a, t in recs:
        if keys and not any(k in t.lower() for k in keys):
            continue
        print(f"ds:{ptr:04X}  描述子 ds:{ptr - 4:04X}  檔案 0x{a:X}  {t!r}")
    if not keys:
        print(f"\n共 {len(recs)} 筆")


if __name__ == "__main__":
    main(sys.argv[1:])
