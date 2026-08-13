"""補齊 BRUN30 未被自動分析的區域(docs/re/13 §2)。

與 unlock_module.py 的差別:**不 del_items**,只在目前未定義的位址上補指令,
既有的 709 個函式與其分析結果保留。

BRUN30 內部會用 INT 3Dh(遊戲模組不用,見 docs/re/07 §2),
所以三種中斷的內嵌參數都要處理。
"""
import json, os, sys
import ida_auto, ida_bytes, ida_funcs, ida_pro, ida_segment, ida_ua, idautils

def main():
    ida_auto.auto_wait()
    seg = ida_segment.getseg(0x10000)
    lo, hi = seg.start_ea, seg.end_ea
    before_code = sum(1 for ea in range(lo, hi)
                      if ida_bytes.is_code(ida_bytes.get_flags(ea)))
    before_f = len([f for f in idautils.Functions() if lo <= f < hi])

    pos, made, ints, bad = lo, 0, {}, 0
    while pos < hi:
        fl = ida_bytes.get_flags(pos)
        if ida_bytes.is_code(fl):                 # 已分析,跳過整個項目
            pos += max(1, ida_bytes.get_item_size(pos))
            continue
        b0 = ida_bytes.get_byte(pos)
        if b0 == 0xCD and ida_bytes.get_byte(pos + 1) in (0x3D, 0x3E, 0x3F):
            vec, prm = ida_bytes.get_byte(pos + 1), ida_bytes.get_byte(pos + 2)
            extra = 3 if (vec == 0x3D and prm == 0) else 1
            if ida_ua.create_insn(pos):
                ida_bytes.create_data(pos + 2, ida_bytes.FF_BYTE, extra, 0)
                k = f"{vec:02X}:{prm:02X}"
                ints[k] = ints.get(k, 0) + 1
                made += 1
                pos += 2 + extra
                continue
        n = ida_ua.create_insn(pos)
        if n <= 0:
            bad += 1
            pos += 1
        else:
            made += 1
            pos += n

    ida_auto.auto_wait()
    after_code = sum(1 for ea in range(lo, hi)
                     if ida_bytes.is_code(ida_bytes.get_flags(ea)))
    after_f = len([f for f in idautils.Functions() if lo <= f < hi])
    json.dump({
        "seg": f"0x{lo:X}-0x{hi:X}", "size": hi - lo,
        "code_before": before_code, "code_after": after_code,
        "ratio_before": round(before_code / (hi - lo), 3),
        "ratio_after": round(after_code / (hi - lo), 3),
        "funcs_before": before_f, "funcs_after": after_f,
        "insns_made": made, "undecodable": bad,
        "int3d_sites": sum(v for k, v in ints.items() if k.startswith("3D")),
        "int_kinds": len(ints),
    }, open(sys.argv[1], "w"), indent=1)
    ida_pro.qexit(0)

main()
