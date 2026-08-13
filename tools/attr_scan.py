"""掃描戰鬥單位陣列 `ds:6822h` 的存取,並把「判不出來」和「屬性 0」分開。

docs/re/85:舊版在找不到 `add reg, imm` 時預設 imm=0,
於是 31% 的失敗被讀成屬性 0 —— 而那正是最高的一筆。
**失敗值不可以和有效輸出撞號。**
"""
import collections, pathlib, struct, sys

BASE = 0x6822
LINEAR = 0xFE00          # linear = file_off + 0xFE00


def scan(exe, width=15, back=24):
    d = pathlib.Path(exe).read_bytes()
    out, undet = [], []
    for k in range(1, len(d) - 1):
        if d[k] != 0x22 or d[k + 1] != 0x68:
            continue
        if not 0x80 <= d[k - 1] <= 0xBF:      # 需要 disp16 形式的 modrm
            continue
        imm = None
        for b in range(3, back):
            j = k - 2 - b
            if j < 0:
                break
            if d[j] == 0x83 and 0xC0 <= d[j + 1] <= 0xC7:
                imm = d[j + 2]; break
            if d[j] == 0x81 and 0xC0 <= d[j + 1] <= 0xC7:
                imm = struct.unpack_from('<H', d, j + 2)[0]; break
            if d[j] in (0xD1,) and d[j + 1] in (0xE7, 0xE6, 0xE3):
                continue                       # shl,略過不當成邊界
        ea = k - 2 + LINEAR
        if imm is None:
            undet.append(ea)                   # ⚠ 不併進屬性 0
        elif imm % width:
            undet.append(ea)                   # 不是屬性存取
        else:
            out.append((ea, imm // width))
    return out, undet


if __name__ == "__main__":
    exe = sys.argv[1]
    hits, undet = scan(exe)
    c = collections.Counter(a for _, a in hits)
    print(f"{exe}: 判定 {len(hits)} 處、**判不出來 {len(undet)} 處**")
    for a in sorted(c):
        print(f"  屬性 {a:>2}: {c[a]:>3}")
    print("\n判不出來的位址(前 20):", " ".join(f"0x{x:X}" for x in undet[:20]))
