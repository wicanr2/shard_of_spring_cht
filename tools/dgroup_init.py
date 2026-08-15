#!/usr/bin/env python3
"""解析模組的 **DGROUP 初始資料串**,把任何一個 `ds:xxxx` 的初始值讀出來。

    python3 tools/dgroup_init.py CAMP.EXE                # 整串
    python3 tools/dgroup_init.py CAMP.EXE --at 731E      # 查一個位址
    python3 tools/dgroup_init.py CAMP.EXE --floats       # 只印 4-byte MBF 常數

**格式**(`docs/re/177`):初始資料是一串區塊,每塊

    <總長度:word> <DGROUP 位址:word> <初始內容:總長度−4 bytes>

字串常數因此**佔兩塊**:先一塊 4 bytes 的描述子(`<長度><文字位址>`),
再一塊文字本身。`tools/dgroup_strings.py` 認的「`<len><ptr><stride><ptr><文字>`」
其實就是這兩塊相接 —— 那支是樣式比對,這支是照格式走,
所以**數字常數也讀得到**,不再只有字串。

⚠ 這支解的是**初始值**,不是執行期的值。被程式改寫過的變數讀到的是初始內容;
`docs/re/42` §3 的「模組變數大半不在檔案裡」對**沒有初始值**的變數仍然成立。
判準:區塊裡有它 → 那是編譯期就決定的常數或初值;沒有 → 靜態讀不到。
"""
import argparse
import pathlib
import struct
import sys

BASE = 0xFE00  # `bz` 模組:線性位址 = 檔案位移 + 0xFE00(docs/re/03/04)
MIN_CHAIN = 20  # 判定「這裡是資料串」的最短連續塊數


def mbf(b):
    """Microsoft Binary Format 單精度 → float。位元組序:尾數低/中/高(bit7=符號)/指數。"""
    if b[3] == 0:
        return 0.0
    exp = b[3] - 128
    sign = -1.0 if b[2] & 0x80 else 1.0
    frac = ((b[2] & 0x7F) << 16 | b[1] << 8 | b[0]) / 2.0 ** 24
    return sign * (0.5 + frac) * 2.0 ** exp


def chain(d, start):
    """從 start 起照格式走,回傳 [(位址, 檔案位移, 內容)];遇到不合格式就停。"""
    out, i = [], start
    while i + 4 <= len(d):
        cnt, addr = struct.unpack_from("<HH", d, i)
        if cnt < 4 or cnt % 2 or i + cnt > len(d) or addr == 0:
            break
        if out and addr <= out[-1][0]:
            break  # 位址必須嚴格遞增
        out.append((addr, i, d[i + 4:i + cnt]))
        i += cnt
    return out


def find_chain(d):
    """掃全檔找最長的一條資料串。

    ⚠ **每個偶數位移都要試**。第一版找到一條就跳過它的長度再繼續,
    結果被前面一條較短的串卡住,真正的 316 塊主串(起點在它後面)整條看不到 ——
    而回報的 120 塊看起來完全正常。**貪心的跳躍會製造「像對的」局部解。**
    """
    best = []
    for i in range(0, len(d) - 8, 2):
        c = chain(d, i)
        if len(c) > len(best):
            best = c
    return best if len(best) >= MIN_CHAIN else []


def describe(payload):
    if len(payload) == 4:
        v = mbf(payload)
        w0, w1 = struct.unpack("<HH", payload)
        # 4 bytes 有兩種讀法:MBF 單精度,或「字串描述子 <長度><文字位址>」
        return f"MBF {v:<14g} | 或描述子 長度={w0} 文字@{w1:04X}"
    if len(payload) == 2:
        return f"word {struct.unpack('<H', payload)[0]}"
    txt = "".join(chr(c) if 32 <= c < 127 else "." for c in payload)
    return f"{len(payload):3d}B  {txt!r}"


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("exe", help="workplace/ida/ 底下的模組檔名或路徑")
    ap.add_argument("--at", help="只查這一個 DGROUP 位址(十六進位)")
    ap.add_argument("--floats", action="store_true", help="只印 4-byte 的數值常數")
    a = ap.parse_args()

    p = pathlib.Path(a.exe)
    if not p.exists():
        p = pathlib.Path("workplace/ida") / a.exe
    d = p.read_bytes()

    ch = find_chain(d)
    if not ch:
        sys.exit(f"[dgroup_init] {p} 裡找不到資料串(最短 {MIN_CHAIN} 塊)")
    print(f"# {p.name}: {len(ch)} 塊,DGROUP {ch[0][0]:04X}–{ch[-1][0]:04X}")

    if a.at:
        want = int(a.at, 16)
        for addr, off, payload in ch:
            if addr <= want < addr + len(payload):
                d_ = payload[want - addr:]
                print(f"{addr:04X}+{want - addr}  檔案 {off + BASE:05X}  "
                      f"{d_[:8].hex(' ')}  {describe(d_[:4]) if len(d_) >= 4 else ''}")
                return
        print(f"ds:{want:04X} 不在初始資料串裡 —— 它沒有編譯期初值")
        return

    for addr, off, payload in ch:
        if a.floats and len(payload) != 4:
            continue
        print(f"{addr:04X}  {off + BASE:05X}  {describe(payload)}")


main()
