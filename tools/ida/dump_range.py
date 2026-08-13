"""倒出指定線性位址範圍的反組譯(用於 IDA 沒建函式邊界的區域)。

用法: idat -A "-S/work/tools/ida/dump_range.py /work/out/x.txt 0x11500 0x11600" X.i64

CLAUDE.md §3.3:不要用 grep -v 過濾組合語言,濾掉的常常正是索引計算。
"""
import os, sys
import ida_auto, ida_bytes, ida_funcs, ida_lines, ida_pro, idautils, idc

def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    lo, hi = int(sys.argv[2], 0), int(sys.argv[3], 0)
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    with open(out, "w", encoding="utf-8") as fh:
        fh.write(f"# 0x{lo:X}–0x{hi:X}\n")
        for head in idautils.Heads(lo, hi):
            raw = ida_bytes.get_bytes(head, ida_bytes.get_item_size(head)) or b""
            line = ida_lines.tag_remove(ida_lines.generate_disasm_line(head, 0) or "")
            fn = ida_funcs.get_func_name(head) or ""
            xr = [x.frm for x in idautils.XrefsTo(head)]
            mark = f"   <<< {len(xr)} 處跳入" if xr else ""
            fh.write(f"{head:06X}  {raw.hex():<14} {line}{mark}\n")
    ida_pro.qexit(0)

main()
