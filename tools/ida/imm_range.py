"""列出落在指定範圍內的所有立即數(含所屬指令),用來找字串描述子的引用。"""
import os, sys, collections
import ida_auto, ida_bytes, ida_lines, ida_pro, ida_segment, idautils, idc

def main():
    ida_auto.auto_wait()
    out, lo, hi = sys.argv[1], int(sys.argv[2], 0), int(sys.argv[3], 0)
    rows, scanned = [], 0
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        for head in idautils.Heads(seg.start_ea, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(head)):
                continue
            scanned += 1
            for i in range(3):
                if idc.get_operand_type(head, i) != idc.o_imm:
                    continue
                v = idc.get_operand_value(head, i)
                if lo <= v < hi:
                    rows.append((head, v,
                        ida_lines.tag_remove(ida_lines.generate_disasm_line(head, 0) or "").strip()))
    with open(out, "w", encoding="utf-8") as fh:
        fh.write(f"# 掃了 {scanned} 條指令,命中 {len(rows)} 處\n")
        for ea, v, l in rows:
            fh.write(f"0x{ea:X}  0x{v:04X}  {l}\n")
    ida_pro.qexit(0)

main()
