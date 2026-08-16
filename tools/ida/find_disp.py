"""按**位移文字**找存取點,不限 `ds:XXXXh` 這一種形式。

用法: idat -A "-S/work/tools/ida/find_disp.py /work/out/x.txt 79A4 798E ..." X.i64
      (參數是十六進位的位移,不帶 0x、不帶 h)

為什麼需要這支(而 `find_dsref.py` 不夠):
陣列存取寫成 `mov bx, [di+79A4h]` —— 位移在**基底加索引**的運算元裡,
不是 `ds:79A4h`。`find_dsref.py` 只認後者,對前者回**零命中**,
而零命中與「真的沒人碰」長得一模一樣(CLAUDE.md §3.3 的同一個坑)。

輸出附前後各三條指令,因為陣列存取的意義幾乎都在鄰近的索引計算裡。
"""
import os
import re
import sys

import ida_auto
import ida_bytes
import ida_lines
import ida_pro
import ida_segment
import idautils


def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    pats = [a.upper().lstrip("0X") for a in sys.argv[2:]]
    if not pats:
        ida_pro.qexit(2)
    # 位移在運算元裡一律帶 h 結尾,前面可能有 0(如 0A3Ah)。
    rx = [(p, re.compile(r"\b0?" + p + r"h\b", re.I)) for p in pats]

    lines, hits, scanned = [], {p: [] for p in pats}, 0
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        for head in idautils.Heads(seg.start_ea, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(head)):
                continue
            txt = ida_lines.tag_remove(
                ida_lines.generate_disasm_line(head, 0) or "").strip()
            lines.append((head, txt))
            scanned += 1
            for p, r in rx:
                if r.search(txt):
                    hits[p].append(len(lines) - 1)

    with open(out, "w", encoding="utf-8") as fh:
        fh.write("# 掃了 %d 條指令\n" % scanned)
        for p in pats:
            fh.write("\n## %s —— %d 處\n" % (p, len(hits[p])))
            for i in hits[p]:
                fh.write("\n")
                for j in range(max(0, i - 3), min(len(lines), i + 4)):
                    ea, txt = lines[j]
                    fh.write("%s%06X  %s\n" % ("→ " if j == i else "  ", ea, txt))
    ida_pro.qexit(0)


main()
