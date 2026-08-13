"""解碼 MONST*.BIN:八張 17×17 的動畫格,以 8 個 word 為間隔交錯。

交錯的成因見 docs/re/48:八張圖被 GET 進同一個二維 BASIC 陣列 A%(7, n),
而 BASIC 的二維陣列是 column-major —— A%(i, j) 在第 j*8+i 個元素,
所以第 i 張圖的資料落在 word i, 8+i, 16+i, …
"""
import pathlib, struct, sys

CH = " .:#"

def decode(path):
    d = pathlib.Path(path).read_bytes()
    if d[0] != 0xFD or d[-1] != 0x1A:
        raise ValueError(f"{path}: 不是 BSAVE 容器")
    seg, off, ln = struct.unpack_from("<HHH", d, 1)
    body = d[7:7 + ln]
    words = struct.unpack_from("<%dH" % (len(body) // 2), body, 0)
    frames = []
    for i in range(8):
        sub = words[i::8]
        wbits, h = sub[0], sub[1]
        data = b"".join(struct.pack("<H", x) for x in sub[2:])
        rb = (wbits + 7) // 8
        rows = []
        for r in range(h):
            row = data[r * rb:(r + 1) * rb]
            s = ""
            for b in row:
                for sh in (6, 4, 2, 0):
                    s += CH[(b >> sh) & 3]
            rows.append(s[:wbits // 2])
        frames.append({"w": wbits // 2, "h": h, "rows": rows})
    return frames

if __name__ == "__main__":
    for p in sys.argv[1:]:
        fr = decode(p)
        print(f"== {pathlib.Path(p).name}  {len(fr)} 格 {fr[0]['w']}×{fr[0]['h']}")
        for r in range(fr[0]["h"]):
            print("  " + " | ".join(f["rows"][r] for f in fr))
