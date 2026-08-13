"""解析模組裡的 BASIC 字串表:<長度:2><DS 指標:2><文字>。

驗證方式:長度欄要與後面連續可列印位元組的長度一致,
而且下一筆描述子要緊接在文字之後(自我校驗的鏈)。
"""
import struct, sys, json, pathlib

def parse(path):
    d = pathlib.Path(path).read_bytes()
    hdr = struct.unpack_from('<H', d, 0x08)[0] * 16
    out, i = [], hdr
    while i + 4 < len(d):
        ln, ptr = struct.unpack_from('<HH', d, i)
        if 1 <= ln <= 200 and ptr:
            body = d[i+4:i+4+ln]
            if body and all(32 <= c < 127 or c in (9,) for c in body):
                out.append({"file": i, "seg": 0x10000 + (i - hdr) - 0x10000,
                            "len": ln, "ds": ptr,
                            "text": body.decode('latin1')})
                i += 4 + ln
                continue
        i += 1
    return d, hdr, out

if __name__ == "__main__":
    for p in sys.argv[1:]:
        d, hdr, rows = parse(p)
        name = pathlib.Path(p).name
        print(f"== {name}: {len(rows)} 筆")
        for r in rows[:6]:
            print(f"   file=0x{r['file']:X} ds=0x{r['ds']:04X} len={r['len']:3d}  {r['text']!r}")
