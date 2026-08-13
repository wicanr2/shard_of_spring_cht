"""倒出某支「讀內嵌參數」共用實作的每個呼叫端與其後的 N bytes。"""
import json, sys
import ida_auto, ida_bytes, ida_pro, idautils, idc
def main():
    ida_auto.auto_wait()
    out, tgt, n = sys.argv[1], int(sys.argv[2],0), int(sys.argv[3])
    rows=[]
    for x in idautils.XrefsTo(tgt):
        if not x.iscode: continue
        ea=x.frm; sz=ida_bytes.get_item_size(ea)
        rows.append({"call":f"0x{ea:X}","after":f"0x{ea+sz:X}",
                     "bytes":(ida_bytes.get_bytes(ea+sz,n) or b"").hex(" ")})
    rows.sort(key=lambda r:int(r["call"],16))
    json.dump(rows, open(out,"w"), indent=1)
    ida_pro.qexit(0)
main()
