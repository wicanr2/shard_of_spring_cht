"""對一段線性位址逐 byte 查 xref(kb ida-pro-9.4.md:結構化記憶體只有起點有名字)。

用法: idat -A "-S/work/tools/ida/range_xref.py /work/out/x.json 0x10C26 0x10C32" X.i64
"""
import json, os, sys
import ida_auto, ida_funcs, ida_pro, idautils, idc

def main():
    ida_auto.auto_wait()
    out, lo, hi = sys.argv[1], int(sys.argv[2], 0), int(sys.argv[3], 0)
    rows = []
    for ea in range(lo, hi):
        for x in idautils.XrefsTo(ea):
            rows.append({"target": f"0x{ea:X}", "off": ea - lo,
                         "from": f"0x{x.frm:X}", "type": x.type,
                         "func": ida_funcs.get_func_name(x.frm) or "",
                         "dis": idc.GetDisasm(x.frm)})
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    json.dump(rows, open(out, "w", encoding="utf-8"), indent=1)
    ida_pro.qexit(0)

main()
