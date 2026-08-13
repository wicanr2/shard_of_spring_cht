"""列出所有節區與各節區預設的段暫存器值(16-bit 目標要靠這個定 DS)。"""
import os, sys
import ida_auto, ida_pro, ida_segment, idautils, idc

def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    with open(out, "w", encoding="utf-8") as fh:
        for ea in idautils.Segments():
            s = ida_segment.getseg(ea)
            ds = idc.get_sreg(ea, "ds"); cs = idc.get_sreg(ea, "cs")
            es = idc.get_sreg(ea, "es"); ss = idc.get_sreg(ea, "ss")
            fh.write(f"{ida_segment.get_segm_name(s):<10} {s.start_ea:06X}-{s.end_ea:06X} "
                     f"size={s.end_ea-s.start_ea:6d} base={ida_segment.get_segm_para(s):04X} "
                     f"cs={cs:04X} ds={ds:04X} es={es:04X} ss={ss:04X}\n")
        fh.write(f"\n入口 = 0x{idc.get_inf_attr(idc.INF_START_IP):X} "
                 f"CS=0x{idc.get_inf_attr(idc.INF_START_CS):X} "
                 f"SS=0x{idc.get_inf_attr(idc.INF_START_SS):X} "
                 f"SP=0x{idc.get_inf_attr(idc.INF_START_SP):X}\n")
    ida_pro.qexit(0)

main()
