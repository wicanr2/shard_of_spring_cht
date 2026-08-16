"""倒出一段線性位址的原始位元組(不解碼)。

用法: idat -A "-S/work/tools/ida/raw_bytes.py /work/out/x.txt 0x11640 0x116E0" X.i64

⚠ 存在的理由:`INT 3D/3E/3F` 帶內嵌參數,IDA 的 item 邊界在那一帶不可信
(CLAUDE.md §3.3)。要判讀就得看**位元組本身**,不是看它切出來的指令。
"""
import os, sys
import ida_auto, ida_bytes, ida_pro

def main():
    ida_auto.auto_wait()
    out, lo, hi = sys.argv[1], int(sys.argv[2], 0), int(sys.argv[3], 0)
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    data = ida_bytes.get_bytes(lo, hi - lo) or b""
    with open(out, "w", encoding="utf-8") as fh:
        for i in range(0, len(data), 16):
            chunk = data[i:i+16]
            fh.write("%06X  %s\n" % (lo + i, ' '.join('%02x' % c for c in chunk)))
    ida_pro.qexit(0)

main()
