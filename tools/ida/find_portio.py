"""找 I/O 埠存取(in/out)與緊接在前的埠號/值 —— CGA 暫存器設定用。"""
import os, sys
import ida_auto, ida_bytes, ida_lines, ida_pro, ida_segment, idautils, idc

def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    rows, prev = [], []
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        for head in idautils.Heads(seg.start_ea, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(head)):
                continue
            line = ida_lines.tag_remove(ida_lines.generate_disasm_line(head, 0) or "").strip()
            m = idc.print_insn_mnem(head)
            if m in ("out", "in"):
                rows.append((head, line, prev[-4:]))
            prev.append(line)
    with open(out, "w", encoding="utf-8") as fh:
        for ea, line, p in rows:
            fh.write(f"0x{ea:X}  {line}\n")
            for q in p:
                fh.write(f"        前: {q}\n")
    ida_pro.qexit(0)

main()
