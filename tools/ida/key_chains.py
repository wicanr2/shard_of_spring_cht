"""抽出按鍵派工鏈:mov bx,<字元描述子> 後面緊接 mov ax,<輸入描述子>。

依「輸入描述子」分組 —— 每個畫面有自己的輸入變數,所以同一組就是同一個畫面。
(docs/re/58:選單鍵是單字元字串常數 + 字串比對,不是數值比較。)
"""
import json, os, sys
import ida_auto, ida_bytes, ida_lines, ida_pro, ida_segment, idautils, idc

def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    seq = []
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        for head in idautils.Heads(seg.start_ea, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(head)):
                continue
            if idc.print_insn_mnem(head) != "mov":
                continue
            if idc.get_operand_type(head, 1) != idc.o_imm:
                continue
            reg = idc.print_operand(head, 0)
            if reg not in ("bx", "ax"):
                continue
            seq.append((head, reg, idc.get_operand_value(head, 1)))
    pairs = []
    for i in range(len(seq) - 1):
        ea1, r1, v1 = seq[i]
        ea2, r2, v2 = seq[i + 1]
        if r1 == "bx" and r2 == "ax" and 0 < ea2 - ea1 <= 8:
            pairs.append({"ea": ea1, "key_desc": v1, "input_desc": v2})
    json.dump(pairs, open(out, "w"))
    ida_pro.qexit(0)

main()
