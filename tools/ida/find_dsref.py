"""按運算元文字找 DS 相對存取(ds:XXXXh)。

為什麼需要這支:16-bit DOS 的資料段基底 IDA 並不知道,`ds:0A3Ah` 這類存取
**不會產生 xref**(range_xref 對它只會回假命中)。CLAUDE.md §2.1 條件 2
要的「用 xref 確認讀寫端點」在這種存取上結構性地做不到,只能掃運算元文字。

用法: idat -A "-S/work/tools/ida/find_dsref.py /work/out/x.txt 0A3A 1142 ..." X.i64
      (參數是十六進位的位移,不帶 0x、不帶 h)
"""
import os, re, sys
import ida_auto, ida_bytes, ida_funcs, ida_lines, ida_pro, ida_segment, idautils, idc

def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    # ds:0A3Ah / ds:A3Ah 兩種寫法都要中;IDA 會補前導 0 到偶數位數
    pats = []
    for t in sys.argv[2:]:
        v = int(t, 16)
        pats.append((v, re.compile(r"\bds:0*%Xh\b" % v, re.I)))
    hits, scanned = [], 0
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        for head in idautils.Heads(seg.start_ea, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(head)):
                continue
            scanned += 1
            line = ida_lines.tag_remove(ida_lines.generate_disasm_line(head, 0) or "")
            for v, p in pats:
                if p.search(line):
                    hits.append((v, head, ida_funcs.get_func_name(head) or "<無函式>", line.strip()))
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    with open(out, "w", encoding="utf-8") as fh:
        fh.write(f"# 掃了 {scanned} 條指令,命中 {len(hits)} 處\n")
        for v, ea, fn, line in sorted(hits):
            fh.write(f"{v:04X}  0x{ea:X}  {fn:<16} {line}\n")
    ida_pro.qexit(0)

main()
