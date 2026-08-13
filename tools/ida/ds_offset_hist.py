"""抽出所有 DS 相對存取的位移,用分布判定 DS 指向哪個節區。

原理:若 DS 指向一個 N bytes 的資料節區,所有位移都會 < N。
位移分布的上界就是資料段大小的估計值。
"""
import json, os, re, sys
import ida_auto, ida_bytes, ida_lines, ida_pro, ida_segment, idautils

PAT = re.compile(r"\bds:([0-9A-F]+)h\b")

def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    offs, scanned = [], 0
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        for head in idautils.Heads(seg.start_ea, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(head)):
                continue
            scanned += 1
            line = ida_lines.tag_remove(ida_lines.generate_disasm_line(head, 0) or "")
            for m in PAT.finditer(line):
                offs.append(int(m.group(1), 16))
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    json.dump({"scanned": scanned, "offsets": offs}, open(out, "w"))
    ida_pro.qexit(0)

main()
