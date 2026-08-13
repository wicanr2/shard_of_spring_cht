"""一次執行 dump 多個位址範圍。

為什麼要有這支:連續快速開關同一個 .i64(19 次 idat 執行)會把資料庫弄壞
(error code 4,本專案第三次踩到)。把多段合併成單次執行既快又安全。

用法: idat -A "-S/work/tools/ida/dump_many.py /work/out/x.txt 0x10303:0x18 0x105B4:0x18 ..." X.i64
"""
import os, sys
import ida_auto, ida_bytes, ida_lines, ida_pro, idautils, idc

def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    with open(out, "w", encoding="utf-8") as fh:
        for spec in sys.argv[2:]:
            a, _, n = spec.partition(":")
            lo = int(a, 0); hi = lo + int(n, 0)
            fh.write(f"=== 0x{lo:X}\n")
            for head in idautils.Heads(lo, hi):
                line = ida_lines.tag_remove(ida_lines.generate_disasm_line(head, 0) or "")
                raw = ida_bytes.get_bytes(head, ida_bytes.get_item_size(head)) or b""
                fh.write(f"  {head:06X}  {raw.hex():<12} {line}\n")
    ida_pro.qexit(0)

main()
