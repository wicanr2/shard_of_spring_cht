"""從 bz+0x30 做流程追蹤(recursive descent),只把真正到得了的位址標成程式碼。

取代 unlock_module.py 的線性掃描 —— 後者分不出程式碼與資料,
會在資料區造出假指令(docs/re/07 §3 有硬證據)。

INT 3Eh/3Fh 是 3 bytes、會返回,所以直接跨過去繼續追。
"""
import json, os, sys
import ida_auto, ida_bytes, ida_funcs, ida_pro, ida_segment, ida_ua, idautils, idc

ENTRY = 0x10040
STOP = {"retn", "retf", "ret", "iret", "iretw"}

def main():
    ida_auto.auto_wait()
    seg = ida_segment.getseg(ENTRY)
    lo, hi = seg.start_ea, seg.end_ea
    ida_bytes.del_items(lo, ida_bytes.DELIT_SIMPLE, hi - lo)
    ida_auto.auto_wait()

    seen, work, funcs = set(), [ENTRY], {ENTRY}
    ints, bad = {}, 0
    breaks = []          # 追蹤斷點:(位址, 原因, 前一個 thunk 的索引與距離)
    last_thunk = [None]

    while work:
        ea = work.pop()
        while lo <= ea < hi and ea not in seen:
            seen.add(ea)
            # 執行期 thunk:3 bytes,跨過去
            if (ida_bytes.get_byte(ea) == 0xCD
                    and ida_bytes.get_byte(ea + 1) in (0x3E, 0x3F)):
                ida_ua.create_insn(ea)
                ida_bytes.create_data(ea + 2, ida_bytes.FF_BYTE, 1, 0)
                k = f"{ida_bytes.get_byte(ea+1):02X}:{ida_bytes.get_byte(ea+2):02X}"
                ints[k] = ints.get(k, 0) + 1
                last_thunk[0] = (ea, k)
                seen.add(ea + 1); seen.add(ea + 2)
                ea += 3
                continue
            n = ida_ua.create_insn(ea)
            if n <= 0:
                bad += 1
                lt = last_thunk[0]
                breaks.append({"ea": f"0x{ea:X}", "why": "解不開",
                               "thunk": lt[1] if lt else None,
                               "dist": (ea - lt[0]) if lt else None,
                               "bytes": ida_bytes.get_bytes(ea, 6).hex(" ")})
                break
            for i in range(1, n):
                seen.add(ea + i)
            mnem = idc.print_insn_mnem(ea)
            # 分支目標:問 IDA 的 code xref,不要自己判斷運算元型別
            for t in idautils.CodeRefsFrom(ea, 0):
                if lo <= t < hi:
                    if t not in seen:
                        work.append(t)
                    if mnem == "call":
                        funcs.add(t)
            if mnem in STOP or mnem == "jmp":
                lt = last_thunk[0]
                breaks.append({"ea": f"0x{ea:X}", "why": mnem,
                               "thunk": lt[1] if lt else None,
                               "dist": (ea - lt[0]) if lt else None,
                               "bytes": ida_bytes.get_bytes(ea, 6).hex(" ")})
                break
            ea += n

    for f in sorted(funcs):
        ida_funcs.add_func(f)
    ida_auto.auto_wait()

    code = sum(1 for ea in range(lo, hi)
               if ida_bytes.is_code(ida_bytes.get_flags(ea)))
    got = [f for f in idautils.Functions() if lo <= f < hi]
    json.dump({
        "region": hi - lo, "reached_bytes": len(seen),
        "code_bytes": code, "code_ratio": round(code / max(1, hi - lo), 3),
        "funcs": len(got), "undecodable": bad,
        "int_calls": ints, "breaks": breaks[:400],
    }, open(sys.argv[1], "w"), indent=1)
    ida_pro.qexit(0)

main()
