"""解碼 DG*MAZE.SQZ:81 列 × 51 欄的文字地圖,含跑長編碼。

規則(docs/re/50):
  - 純文字,`\\r\\n` 分行,`0x1A` 結尾,81 列
  - **小寫字母與 `_` 是跑長標記**,下一個字元是計數(值 = ord − 0x2A)
  - 其他字元(數字、大寫、標點)是單一格
  - 每列展開後應為 51 格

⚠ 486 列裡有 20 列展開後是 50 格(短 1),成因未解 —— 見 docs/re/50 §4。
本工具對那些列補一格 `_`,並在回傳值裡標出來。
"""
import pathlib, sys

BASE = 0x2A
WIDTH = 51
ROWS = 81

def is_marker(c):
    return c == "_" or c.islower()

def decode(path):
    raw = pathlib.Path(path).read_bytes().rstrip(b"\x1a")
    lines = [l.decode("latin1") for l in raw.split(b"\r\n") if l]
    grid, short = [], []
    for r, line in enumerate(lines):
        row, i = [], 0
        while i < len(line):
            if is_marker(line[i]) and i + 1 < len(line):
                row += [line[i]] * (ord(line[i + 1]) - BASE)
                i += 2
            else:
                row.append(line[i])
                i += 1
        if len(row) < WIDTH:
            short.append((r, len(row)))
            row += ["_"] * (WIDTH - len(row))
        grid.append(row[:WIDTH])
    return grid, short

if __name__ == "__main__":
    for p in sys.argv[1:]:
        g, short = decode(p)
        print(f"== {pathlib.Path(p).name}  {len(g)} 列 × {WIDTH}  補過的列: {short}")
        for row in g:
            print("  " + "".join(" " if c == "_" else c for c in row))
