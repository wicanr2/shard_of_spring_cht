#!/usr/bin/env python3
"""把 `INT 3Fh` 的索引翻成 `BRUN30` 裡的目標位址。

    python3 tools/brun_api.py 23 95 7B        # 查幾個索引
    python3 tools/brun_api.py --all           # 整張表

派工表在 `BRUN30.EXE` 的檔案位移 **0x58D**,每個索引 2 bytes(little-endian),
線性位址 = 表值 + 0x10000。

表的位置是用**兩個已知索引反推**的(docs/re/77:`3F:8D` = 除 → `0xB525`、
`3F:91` = 乘 → `0xB4A0`,兩者在表裡相隔 4 個索引 = 8 bytes),
再用其餘六個已知索引回驗。

⚠ **這張表是判別「同一支常式的第二個進入點」最快的工具。**
`3F:8F` 一度被當成除法(它夾在 `3F:8D` 除與 `3F:91` 乘之間),
查表才看到它指向 `0xB4AB` —— 落在乘法本體內,所以是**乘的另一個進入點**
(docs/re/153 §0)。**相鄰的索引不代表相關的運算。**
"""
import struct
import sys
import pathlib

TABLE_OFF = 0x58D
LINEAR = 0x10000

# 已解出語意的索引。出處:docs/re/77(算術八個)、152 §3、153 §6.2、157。
KNOWN = {
    0x57: "整數 bx → 浮點累加器",
    0x5C: "MID$ 指派",
    0x61: "字串指派",
    0x62: "字串比較",
    0x6A: "印字串",
    0x6E: "字串串接",
    0x71: "累加器 ↔ 變數",
    0x77: "浮點累加器 → 整數 bx",
    0x7D: "指派",
    0x81: "加",
    0x85: "加(另一序)",
    0x8D: "除",
    0x8F: "乘(另一序)",
    0x91: "乘",
    0x95: "乘(第三個進入點)",
    0x9D: "減",
    0xA1: "浮點比較(設旗標)",
    0xB8: "INPUT#",
}


def table(d):
    return lambda i: struct.unpack_from("<H", d, TABLE_OFF + 2 * i)[0]


def main(argv):
    root = pathlib.Path(__file__).resolve().parent.parent
    d = (root / "game/sharspri/BRUN30.EXE").read_bytes()
    at = table(d)
    # 回驗:兩個錨點必須對得上,否則表位置錯了
    for i, want in ((0x8D, 0xB525), (0x91, 0xB4A0)):
        if at(i) != want:
            raise SystemExit(f"派工表位置不對:3F:{i:02X} → 0x{at(i):04X},應為 0x{want:04X}")
    idx = range(0x100) if "--all" in argv else [int(a, 16) for a in argv]
    for i in idx:
        v = at(i)
        if "--all" in argv and (v == 0 or v > len(d)):
            continue
        print(f"3F:{i:02X} → 0x{v:04X}  linear 0x{v + LINEAR:05X}  {KNOWN.get(i, '')}")


if __name__ == "__main__":
    main(sys.argv[1:])
