"""線性掃描反組譯 bz 模組本體,正確跳過 INT 3Dh/3Eh/3Fh 的內嵌參數。

依 docs/re/06 的 ABI:
    CD 3E nn        3 bytes   派工表 cs:0x243
    CD 3F nn        3 bytes   派工表 cs:0x38D
    CD 3D nn        3 bytes   (nn > 0)  派工表 cs:0x171
    CD 3D 00 oo oo  5 bytes   (nn == 0) 執行期會就地改寫成 9A far call

用法: idat -A "-S/work/tools/ida/unlock_module.py /work/out/x.json" X.EXE.i64
"""
import json, os, sys
import ida_auto, ida_bytes, ida_funcs, ida_pro, ida_segment, ida_ua, idautils

ENTRY = 0x10040

def main():
    ida_auto.auto_wait()
    seg = ida_segment.getseg(ENTRY)
    limit = seg.end_ea
    ida_bytes.del_items(ENTRY, ida_bytes.DELIT_SIMPLE, limit - ENTRY)
    ida_auto.auto_wait()

    pos, ints, patched, stalled = ENTRY, 0, 0, 0
    while pos < limit:
        b0 = ida_bytes.get_byte(pos)
        if b0 == 0xCD and ida_bytes.get_byte(pos + 1) in (0x3D, 0x3E, 0x3F):
            vec = ida_bytes.get_byte(pos + 1)
            param = ida_bytes.get_byte(pos + 2)
            extra = 3 if (vec == 0x3D and param == 0) else 1
            ida_ua.create_insn(pos)                    # 2-byte int
            ida_bytes.create_data(pos + 2, ida_bytes.FF_BYTE, extra, 0)
            ints += 1
            patched += (extra == 3)
            pos += 2 + extra
            continue
        n = ida_ua.create_insn(pos)
        if n <= 0:
            stalled += 1
            ida_bytes.create_data(pos, ida_bytes.FF_BYTE, 1, 0)
            pos += 1
        else:
            pos += n

    ida_funcs.add_func(ENTRY)
    ida_auto.auto_wait()

    code = sum(1 for ea in range(ENTRY, limit)
               if ida_bytes.is_code(ida_bytes.get_flags(ea)))
    funcs = [f for f in idautils.Functions() if ENTRY <= f < limit]
    out = sys.argv[1]
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    with open(out, "w", encoding="utf-8") as fh:
        json.dump({
            "region_bytes": limit - ENTRY,
            "code_bytes": code,
            "code_ratio": round(code / max(1, limit - ENTRY), 3),
            "int_thunks": ints,
            "int3d_farcall_form": patched,
            "undecodable_bytes": stalled,
            "funcs_in_module": len(funcs),
        }, fh, indent=2)
    ida_pro.qexit(0)

main()
