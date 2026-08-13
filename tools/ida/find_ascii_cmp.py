"""找與 ASCII 可列印字元比較的指令,並依位址聚類 —— 按鍵派工表會成叢出現。"""
import os, re, sys
import ida_auto, ida_bytes, ida_lines, ida_pro, ida_segment, idautils, idc

PAT = re.compile(r"\bcmp\b.*,\s*([0-9A-F]+)h\s*(?:;.*)?$")

def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    hits = []
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        for head in idautils.Heads(seg.start_ea, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(head)):
                continue
            line = ida_lines.tag_remove(ida_lines.generate_disasm_line(head, 0) or "").strip()
            m = PAT.search(line.split(";")[0].strip())
            if not m:
                continue
            v = int(m.group(1), 16)
            if 0x21 <= v <= 0x7E:
                hits.append((head, v, line))
    # 聚類:相鄰 <= 0x40 bytes 算同一叢
    with open(out, "w", encoding="utf-8") as fh:
        cluster = []
        for ea, v, line in hits:
            if cluster and ea - cluster[-1][0] > 0x40:
                if len(cluster) >= 4:
                    fh.write(f"=== 叢集 0x{cluster[0][0]:X}–0x{cluster[-1][0]:X}({len(cluster)} 個)\n")
                    for e, vv, l in cluster:
                        fh.write(f"    0x{e:X}  {chr(vv)!r}  {l}\n")
                cluster = []
            cluster.append((ea, v, line))
        if len(cluster) >= 4:
            fh.write(f"=== 叢集 0x{cluster[0][0]:X}–0x{cluster[-1][0]:X}({len(cluster)} 個)\n")
            for e, vv, l in cluster:
                fh.write(f"    0x{e:X}  {chr(vv)!r}  {l}\n")
    ida_pro.qexit(0)

main()
