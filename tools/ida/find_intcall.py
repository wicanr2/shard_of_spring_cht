"""找指定 (中斷, 索引) 的呼叫點,並印出前幾條指令。"""
import os, sys
import ida_auto, ida_bytes, ida_lines, ida_pro, ida_segment, idautils, idc

def main():
    ida_auto.auto_wait()
    out = sys.argv[1]; vec = int(sys.argv[2], 16); idx = int(sys.argv[3], 16)
    rows, prev = [], []
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        for head in idautils.Heads(seg.start_ea, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(head)):
                continue
            b = ida_bytes.get_bytes(head, 3) or b""
            line = ida_lines.tag_remove(ida_lines.generate_disasm_line(head, 0) or "").strip()
            if len(b) == 3 and b[0] == 0xCD and b[1] == vec and b[2] == idx:
                rows.append((head, prev[-8:]))
            prev.append(f"{head:06X}  {line}")
    with open(out, "w", encoding="utf-8") as fh:
        for ea, p in rows:
            fh.write(f"=== 0x{ea:X}\n")
            for q in p: fh.write("    "+q+"\n")
    ida_pro.qexit(0)

main()
