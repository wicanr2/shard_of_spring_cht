"""強制 IDA 把 bz 模組本體當程式碼分析(docs/re/04)。

模組區 = 載入映像 [0, bz+0x16 * 16);bz 標頭在 +0x10,長 0x30;
所以程式碼從 linear 0x10000 + 0x10 + 0x30 = 0x10040 開始。

用法: idat -A "-S/work/tools/ida/unlock_module.py /work/out/x.json" X.EXE.i64
"""
import json, os, sys
import ida_auto, ida_bytes, ida_funcs, ida_segment, ida_ua, idaapi, idautils, idc

ENTRY = 0x10040          # 模組程式碼起點(linear)

def main():
    ida_auto.auto_wait()
    before = len(list(idautils.Functions()))

    seg = ida_segment.getseg(ENTRY)
    limit = seg.end_ea if seg else ENTRY

    # 先把整段標成未定義,免得殘留的資料項擋住指令生成
    ida_bytes.del_items(ENTRY, ida_bytes.DELIT_SIMPLE, limit - ENTRY)
    ida_auto.auto_wait()

    ida_ua.create_insn(ENTRY)
    ida_funcs.add_func(ENTRY)
    ida_auto.plan_and_wait(ENTRY, limit)
    ida_auto.auto_wait()

    funcs = list(idautils.Functions())
    # 只算落在模組區裡的
    inmod = [f for f in funcs if ENTRY <= f < limit]
    coverage = sum(ida_funcs.get_func(f).end_ea - ida_funcs.get_func(f).start_ea
                   for f in inmod)
    # 指令流自洽度:區段內有多少 byte 被判成指令
    code_bytes = sum(1 for ea in range(ENTRY, limit)
                     if ida_bytes.is_code(ida_bytes.get_flags(ea)))

    out = sys.argv[1]
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    with open(out, "w", encoding="utf-8") as fh:
        json.dump({
            "entry": f"0x{ENTRY:X}",
            "seg_end": f"0x{limit:X}",
            "region_bytes": limit - ENTRY,
            "funcs_before": before,
            "funcs_after": len(funcs),
            "funcs_in_module": len(inmod),
            "func_bytes_in_module": coverage,
            "code_bytes_in_region": code_bytes,
            "code_ratio": round(code_bytes / max(1, limit - ENTRY), 3),
            "first_funcs": [f"0x{f:X}" for f in inmod[:10]],
        }, fh, indent=2)
    ida_pro = __import__("ida_pro"); ida_pro.qexit(0)

main()
