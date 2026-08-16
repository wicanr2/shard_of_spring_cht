#!/usr/bin/env python3
"""倚天中文系統 3.53 的 24×24 點陣字型 → 引擎資產(docs/spec/04 §4)。

    tools/eten_font.py <倚天字型目錄> <輸出.bin>

倚天那三個檔:

| 檔案 | 內容 | 版面 |
|---|---|---|
| `STDFONT.24` | 漢字 13,094 字(Big5 A440 起,含倚天擴充 41 字) | 24×24,72 B/字 |
| `SPCFONT.24` | 符號 408 字(Big5 A140–A3BF,全形標點在這裡) | 24×24,72 B/字 |
| `ASCFONT.24` | 半形 256 字(ASCII) | 12×24,存成 2 B/列 → 48 B/字 |

⚠ `STDFONT.24` 在光碟上是 `STD.24M`,`ETUNPACK V1.00` 壓縮格式,
**要先用光碟 DISK2 的 `ETUNPACK.EXE` 解包**(DOSBox 即可,見 docs/spec/21)。
壓縮資料是高熵的(壓縮比 0.70),不是 RLE —— 不要試圖直接讀。

索引方式:**Big5 碼的緊密編號**。低位元組的合法範圍是 `0x40–0x7E` 與
`0xA1–0xFE`(每個高位元組 157 個),所以
`索引 = (hi − 起始hi) × 157 + (lo ≤ 0x7E ? lo − 0x40 : lo − 0xA1 + 63)`。
起始碼:漢字 `A440`、符號 `A140`。**索引 0 畫出來是「一」**(驗證過)。

⚠ **漢字檔跳過常用區與次常用區之間的空洞。** Big5 常用字止於 `C67E`、
次常用字始於 `C940`,中間的 `C6A1–C8FE`(倚天放日文假名、俄文、線條符號)
**不在 `STDFONT.24` 裡**,而是另一個檔 `SPCFSUPP.24`。
所以 `code ≥ C940` 的索引要減掉那 408 個碼位 —— 不減的話尾端 367 個字
(驉…鰲)會算到檔案外面,而**前面 12,000 多字全部正確**,
症狀只出現在最罕用的那一段。

本專案的文字**沒有任何一個字落在空洞區**(掃過 assets/、engine/、
translations/ 共 1,422 個非 ASCII 字種),所以不載入 `SPCFSUPP.24`。

輸出格式(小端序,`internal/render/bitmap.go` 讀它):

    magic   "ETEN24\0\0"   8 B
    version u32 = 1
    full_w  u32 = 24     全形寬(像素)
    full_h  u32 = 24
    half_w  u32 = 12     半形寬 —— ⚠ 來源是 12 位元存在 16 位元裡,右邊 4 位是墊的
    half_h  u32 = 24
    count   u32          字數
    索引 × count(依 rune 排序):
        rune u32 | offset u32 | width u8 | row_bytes u8 | pad ×2
    bitmaps:MSB 先,每字 row_bytes × 高

⚠ **`row_bytes` 要明寫,不能讓讀取端從寬度推。** 全形 24 位元剛好 3 B,
半形卻是 12 位元**存在 2 B 裡**(右邊 4 位是墊的)—— 同一條公式湊不出這兩個,
而推錯的症狀是漢字變成雜訊、ASCII 完全正常(半形那條剛好推對)。

⚠ **這個輸出是倚天的著作物**,不進版控、不隨發行版散布
(`CLAUDE.md` §1、`docs/spec/04` §4:發行版用開源字型)。
"""
import sys
import os
import struct

FULL_W = FULL_H = 24
HALF_W, HALF_H = 12, 24


def packed_index(code: int, start: int) -> int:
    """Big5 碼 → 緊密編號(不含空洞修正)。"""
    hi, lo = code >> 8, code & 0xFF
    off = lo - 0x40 if lo <= 0x7E else lo - 0xA1 + 63
    return (hi - (start >> 8)) * 157 + off


# GAP 是常用區與次常用區之間被跳過的碼位數(C67F–C93F)。
GAP = packed_index(0xC940, 0xA440) - packed_index(0xC67E, 0xA440) - 1


def hanzi_index(code: int) -> int:
    """漢字檔的索引:次常用區要減掉空洞。"""
    i = packed_index(code, 0xA440)
    return i - GAP if code >= 0xC940 else i


def big5(ch: str):
    """回傳 Big5 碼;編不出來回 None。"""
    try:
        b = ch.encode("big5")
    except UnicodeEncodeError:
        return None
    return (b[0] << 8 | b[1]) if len(b) == 2 else None


def main() -> int:
    if len(sys.argv) != 3:
        print(__doc__)
        return 2
    src, out = sys.argv[1], sys.argv[2]

    def read(name):
        p = os.path.join(src, name)
        if not os.path.exists(p):
            sys.exit(f"找不到 {p} —— STDFONT.24 要先用 ETUNPACK 解包 STD.24M")
        return open(p, "rb").read()

    std, spc, asc = read("STDFONT.24"), read("SPCFONT.24"), read("ASCFONT.24")
    if len(std) % 72 or len(spc) % 72 or len(asc) % 48:
        sys.exit("字型檔長度不是每字位元組數的整數倍 —— 檔案可能不完整")

    # rune → (bitmap, 寬度)。半形先放,全形後放時會覆蓋(不會撞,範圍不重疊)。
    glyphs = {}
    for code in range(0x20, 0x7F):  # 可見的 ASCII
        glyphs[chr(code)] = (asc[code * 48:(code + 1) * 48], HALF_W, 2)

    missing = []
    # 走一遍 Big5 全部合法碼位,把有字的收進來。
    for hi in range(0xA1, 0xFA):
        for lo in list(range(0x40, 0x7F)) + list(range(0xA1, 0xFF)):
            code = hi << 8 | lo
            try:
                ch = bytes([hi, lo]).decode("big5")
            except UnicodeDecodeError:
                continue
            if len(ch) != 1:
                continue
            if 0xC67F <= code < 0xC940:
                continue  # 空洞區(倚天的日文/俄文/線條符號),在 SPCFSUPP.24
            if code >= 0xA440:
                d, i = std, hanzi_index(code)
            else:
                d, i = spc, packed_index(code, 0xA140)
            if i < 0 or (i + 1) * 72 > len(d):
                missing.append(ch)
                continue
            g = d[i * 72:(i + 1) * 72]
            if any(g):  # 全零 = 這個碼位沒有字
                glyphs[ch] = (g, FULL_W, 3)

    items = sorted(glyphs.items(), key=lambda kv: ord(kv[0]))
    header = struct.pack("<8s6I", b"ETEN24\0\0", 1,
                         FULL_W, FULL_H, HALF_W, HALF_H, len(items))
    index, blob, off = bytearray(), bytearray(), 0
    for ch, (bmp, w, rb) in items:
        index += struct.pack("<IIBB2x", ord(ch), off, w, rb)
        blob += bmp
        off += len(bmp)
    with open(out, "wb") as f:
        f.write(header)
        f.write(index)
        f.write(blob)

    print(f"→ {out}:{len(items)} 字,{os.path.getsize(out)} bytes")
    if missing:
        print(f"⚠ {len(missing)} 個碼位在字型檔範圍外:{''.join(missing[:20])}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
