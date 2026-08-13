"""直接數:有多少指令起點的頭兩個位元組是 CD 3D / CD 3E / CD 3F,以及 IDA 給的助憶碼。

docs/re/07 §2 的清點用 print_insn_mnem(head) != "int" 過濾,
若 IDA 把 CD 3D 解成別的助憶碼(浮點模擬器呼叫),那個過濾會漏掉全部。
"""
import collections, json, os, sys
import ida_auto, ida_bytes, ida_pro, ida_segment, idautils, idc

def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    cnt = collections.Counter(); heads = collections.Counter(); ex = {}
    total = 0
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        for head in idautils.Heads(seg.start_ea, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(head)):
                continue
            total += 1
            b = ida_bytes.get_bytes(head, 2) or b""
            if len(b) == 2 and b[0] == 0xCD and b[1] in (0x3D, 0x3E, 0x3F):
                key = f"CD{b[1]:02X}"
                heads[key] += 1
                m = idc.print_insn_mnem(head)
                cnt[f"{key}/{m}"] += 1
                ex.setdefault(f"{key}/{m}", f"0x{head:X}  {idc.generate_disasm_line(head,0)}")
    # 位元組層(不管是不是指令起點)
    raw = collections.Counter()
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        data = ida_bytes.get_bytes(seg.start_ea, seg.end_ea - seg.start_ea) or b""
        for v in (0x3D, 0x3E, 0x3F):
            pat = bytes([0xCD, v]); i = 0
            while True:
                i = data.find(pat, i)
                if i < 0: break
                raw[f"CD{v:02X}"] += 1; i += 1
    json.dump({"total_insns": total, "as_head": dict(heads),
               "by_mnem": dict(cnt), "examples": ex, "raw_bytes": dict(raw)},
              open(out, "w"), indent=1)
    ida_pro.qexit(0)

main()
