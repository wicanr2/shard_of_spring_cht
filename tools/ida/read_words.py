"""讀出指定線性位址上的 word,用來查派工表的表項。

用法: idat -A "-S/work/tools/ida/read_words.py /work/out/w.txt 0x101D6 8" X.i64
第三個參數是要讀幾個 word。
"""
import os, sys
import ida_auto, ida_bytes, ida_pro

def main():
    ida_auto.auto_wait()
    out, lo, n = sys.argv[1], int(sys.argv[2], 0), int(sys.argv[3], 0)
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    with open(out, "w", encoding="utf-8") as fh:
        for i in range(n):
            ea = lo + i * 2
            fh.write(f"{ea:06X}  {ida_bytes.get_word(ea):04X}\n")
    ida_pro.qexit(0)

main()
