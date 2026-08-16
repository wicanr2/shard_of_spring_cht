"""線性掃描反組譯 bz 模組本體,正確跳過 INT 3Dh/3Eh/3Fh 的內嵌參數。

依 docs/re/06 的 ABI:
    CD 3E nn        3 bytes   派工表 cs:0x243
    CD 3F nn        3 bytes   派工表 cs:0x38D
    CD 3D nn        3 bytes   (nn > 0)  派工表 cs:0x171
    CD 3D 00 oo oo  5 bytes   (nn == 0) 執行期會就地改寫成 9A far call

⚠ **18 個 `3F` 索引多帶一個運算元位元組**(docs/re/185 §1.2:
`71 72 85 86 8D 8E 95 96 9D 9E A5 A6 AB AD AE B5 CA CB`),而**它擺在哪要看模組**:
CMBT / MENU / MAZEMOVE / WSIO / MIO2 / MTEST 的每個 `CD 3F xx` 後面接 `90 90`,
運算元在 **+5**;其餘模組沒有那兩個 nop,運算元在 **+3**。

不處理這一類的症狀:那個運算元位元組被當成指令開頭,**接下來整段錯位** ——
而錯位的反組譯讀起來是通順的(`or bp, 343Dh` 之類),不會有任何錯誤訊息。

用法: idat -A "-S/work/tools/ida/unlock_module.py /work/out/x.json" X.EXE.i64
"""
import json, os, sys
import ida_auto, ida_bytes, ida_funcs, ida_pro, ida_segment, ida_ua, idautils

ENTRY = 0x10040

# 多帶一個運算元位元組的 3F 索引(docs/re/185 §1.2)。
FOUR_BYTE_3F = {0x71, 0x72, 0x85, 0x86, 0x8D, 0x8E, 0x95, 0x96,
                0x9D, 0x9E, 0xA5, 0xA6, 0xAB, 0xAD, 0xAE, 0xB5, 0xCA, 0xCB}

def main():
    ida_auto.auto_wait()
    seg = ida_segment.getseg(ENTRY)
    limit = seg.end_ea
    ida_bytes.del_items(ENTRY, ida_bytes.DELIT_SIMPLE, limit - ENTRY)
    ida_auto.auto_wait()

    pos, ints, patched, stalled, extras = ENTRY, 0, 0, 0, 0
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
            # 多運算元的 3F:運算元在 nop 墊片之後(+5)或緊接著(+3),
            # 兩種擺法都要吃掉,否則那個位元組會被當成下一條指令的開頭。
            if vec == 0x3F and param in FOUR_BYTE_3F:
                if (ida_bytes.get_byte(pos) == 0x90
                        and ida_bytes.get_byte(pos + 1) == 0x90):
                    ida_bytes.create_data(pos, ida_bytes.FF_BYTE, 2, 0)
                    pos += 2
                ida_bytes.create_data(pos, ida_bytes.FF_BYTE, 1, 0)
                pos += 1
                extras += 1
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
            "int3f_with_extra_operand": extras,
            "undecodable_bytes": stalled,
            "funcs_in_module": len(funcs),
        }, fh, indent=2)
    ida_pro.qexit(0)

main()
