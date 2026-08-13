"""列出所有遠端(far)呼叫/跳躍,尤其是間接的 —— 執行期跳進模組必須經過它們。"""
import json, os, sys, collections
import ida_auto, ida_bytes, ida_funcs, ida_pro, idautils, idc

def main():
    ida_auto.auto_wait()
    rows = []
    for seg in idautils.Segments():
        import ida_segment
        s = ida_segment.getseg(seg)
        for ea in idautils.Heads(s.start_ea, s.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(ea)):
                continue
            m = idc.print_insn_mnem(ea)
            if m not in ("call", "jmp"):
                continue
            dis = idc.GetDisasm(ea)
            if "far" not in dis.lower():
                continue
            op = idc.print_operand(ea, 0)
            indirect = "[" in op
            rows.append({"ea": f"0x{ea:X}", "mnem": m, "op": op,
                         "indirect": indirect,
                         "func": ida_funcs.get_func_name(ea) or ""})
    json.dump(rows, open(sys.argv[1], "w", encoding="utf-8"), indent=1)
    ida_pro.qexit(0)

main()
