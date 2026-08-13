"""抽出 CD 3D 00 oo oo 形式的 5-byte 延遲繫結 far call 及其索引(docs/re/06 §3)。"""
import collections, json, os, sys
import ida_auto, ida_bytes, ida_pro, ida_segment, idautils

def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    hits = collections.Counter()
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        for head in idautils.Heads(seg.start_ea, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(head)):
                continue
            b = ida_bytes.get_bytes(head, 5) or b""
            if len(b) == 5 and b[0] == 0xCD and b[1] == 0x3D and b[2] == 0x00:
                hits[b[3] | (b[4] << 8)] += 1
    json.dump({f"0x{k:04X}": v for k, v in sorted(hits.items())}, open(out, "w"), indent=0)
    ida_pro.qexit(0)

main()
