"""全面清點模組區裡被判成指令的 int,以及殘留的 CD 3D 位元組。"""
import json, os, sys, collections
import ida_auto, ida_bytes, ida_pro, ida_segment, idautils, idc
ENTRY=0x10040
def main():
    ida_auto.auto_wait()
    seg=ida_segment.getseg(ENTRY); limit=seg.end_ea
    vecs=collections.Counter(); 
    for head in idautils.Heads(ENTRY, limit):
        if ida_bytes.is_code(ida_bytes.get_flags(head)) and idc.print_insn_mnem(head)=="int":
            vecs[f"int {idc.get_operand_value(head,0):02X}"]+=1
    # 位元組層:CD 3D 出現幾次、其中幾次落在被判成指令的位址
    raw=code=0
    for ea in range(ENTRY, limit-1):
        if ida_bytes.get_byte(ea)==0xCD and ida_bytes.get_byte(ea+1)==0x3D:
            raw+=1
            if ida_bytes.is_code(ida_bytes.get_flags(ea)): code+=1
    json.dump({"decoded_ints":dict(vecs),"CD3D_raw":raw,"CD3D_at_code":code},
              open(sys.argv[1],"w"), indent=1)
    ida_pro.qexit(0)
main()
