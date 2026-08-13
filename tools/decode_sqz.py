"""解碼 DG*MAZE.SQZ。規則直接取自 MAZEMOVE.EXE 的解碼迴圈(docs/re/55)。

  v = 字元碼 − 42                      (ds:964Eh = 0x2A)
  v ≤ −29(即 CR)      → 換列          (cmp ax, 0FFE3h)
  v ≥ 53               → 跑長:圖塊 = v − 53,長度 = 下一字元 − 42   (ds:964Ch = 0x35)
  否則                 → 字面:圖塊 = v

同一個圖塊值有兩種寫法:字面(字元 = 值 + 42)或跑長(字元 = 值 + 95)。
陣列索引是 `欄 × 81 + 列`(column-major,81 列)。
解碼器**沒有欄數檢查**,所以各列長度可以不同,未寫到的格子保持 0。
"""
import pathlib, sys

CBASE, THRESH, ROWS = 42, 53, 81

def decode(path):
    raw = pathlib.Path(path).read_bytes().rstrip(b"\x1a")
    rows, i = [[]], 0
    while i < len(raw):
        v = raw[i] - CBASE
        if v <= -29:
            if raw[i] == 13:
                rows.append([])
            i += 1
            continue
        if v >= THRESH:
            if i + 1 >= len(raw):
                break
            rows[-1] += [v - THRESH] * (raw[i + 1] - CBASE)
            i += 2
        else:
            rows[-1].append(v)
            i += 1
    return [r for r in rows if r]

if __name__ == "__main__":
    import collections
    for p in sys.argv[1:]:
        g = decode(p)
        c = collections.Counter(v for r in g for v in r)
        print(f"== {pathlib.Path(p).name}  {len(g)} 列,寬 {sorted({len(r) for r in g})}")
        print("   圖塊值:", dict(c.most_common()))
