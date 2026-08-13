"""找 BASIC 陣列的基底位址:形如 [reg+XXXXh] 的記憶體運算元。

為什麼不是掃 mul:word 陣列的索引是 `add reg, <下界>` + `shl reg, 1`,
**編譯器不會產生乘法**(docs/re/23 §5 當時掃 mul 掃不到,結論下錯了)。
"""
import json, os, re, sys
import ida_auto, ida_bytes, ida_funcs, ida_lines, ida_pro, ida_segment, idautils

PAT = re.compile(r"\[(bx|si|di|bp)([+\-])([0-9A-F]+)h\]")

def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    rows, scanned = [], 0
    prev = []
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        for head in idautils.Heads(seg.start_ea, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(head)):
                continue
            scanned += 1
            line = ida_lines.tag_remove(ida_lines.generate_disasm_line(head, 0) or "")
            m = PAT.search(line)
            if m and m.group(2) == "+":
                disp = int(m.group(3), 16)
                if disp >= 0x1000:                     # 排除區域變數 [bp+xx]
                    rows.append({"ea": f"0x{head:X}", "reg": m.group(1),
                                 "base": disp, "line": line.strip(),
                                 "prev": prev[-3:]})
            prev.append(line.strip())
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    json.dump({"scanned": scanned, "rows": rows}, open(out, "w"))
    ida_pro.qexit(0)

main()
