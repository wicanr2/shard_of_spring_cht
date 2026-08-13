"""掃全部指令的立即數,命中指定值就報出來(含所屬函式)。

用法: idat -A "-S/work/tools/ida/find_imm.py /work/out/x.txt 0x7A62 [更多值...]" X.i64

xref 圖看不到「把值當純數字用」的地方,所以這支是補 xref 的盲點
(見 kb ida-pro-9.4.md「掃立即數」)。慢,但十萬條指令等級可接受。
"""
import os, sys
import ida_auto, ida_bytes, ida_funcs, ida_pro, ida_segment, idautils, idc

def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    wanted = {int(t, 0) for t in sys.argv[2:]}
    hits, scanned = [], 0
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        for head in idautils.Heads(seg.start_ea, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(head)):
                continue
            scanned += 1
            for i in range(8):
                t = idc.get_operand_type(head, i)
                if t == idc.o_void:
                    break
                if t == idc.o_imm and idc.get_operand_value(head, i) in wanted:
                    hits.append((head, idc.GetDisasm(head), ida_funcs.get_func_name(head)))
                    break
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    with open(out, "w", encoding="utf-8") as fh:
        fh.write(f"# 掃了 {scanned} 條指令,命中 {len(hits)} 處\n")
        for ea, dis, fn in hits:
            fh.write(f"0x{ea:X}  {fn or '<無函式>':<16} {dis}\n")
    ida_pro.qexit(0)

main()
