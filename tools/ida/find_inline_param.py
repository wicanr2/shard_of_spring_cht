"""找「呼叫後接內嵌參數」的共用實作(docs/re/35)。

特徵:函式開頭是 pop <記憶體位址>(把返回位址存起來當參數指標)。
這種函式的每個呼叫端後面都跟著資料,線性反組譯會錯位。
"""
import json, os, sys
import ida_auto, ida_funcs, ida_pro, idautils, idc

def main():
    ida_auto.auto_wait()
    rows = []
    for ea in idautils.Functions():
        m = idc.print_insn_mnem(ea)
        if m != "pop":
            continue
        if idc.get_operand_type(ea, 0) != idc.o_mem:
            continue
        callers = sorted({x.frm for x in idautils.XrefsTo(ea) if x.iscode})
        rows.append({
            "ea": f"0x{ea:X}", "name": ida_funcs.get_func_name(ea),
            "first": idc.GetDisasm(ea), "callers": len(callers),
            "caller_list": [f"0x{c:X}" for c in callers[:20]],
        })
    rows.sort(key=lambda r: -r["callers"])
    json.dump(rows, open(sys.argv[1], "w", encoding="utf-8"), indent=1)
    ida_pro.qexit(0)

main()
