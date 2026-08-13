"""從已解鎖的 DB 取出實際的執行期呼叫點 (中斷, 索引) 與次數。

用位元組掃描會把資料區的偶然位元組算進去(docs/re/05 §3 的但書);
這支只認 IDA 判成指令的 int,所以是實數不是上限。
"""
import json, os, sys
import ida_auto, ida_bytes, ida_pro, ida_segment, idautils, idc

ENTRY = 0x10040

def main():
    ida_auto.auto_wait()
    seg = ida_segment.getseg(ENTRY); limit = seg.end_ea
    hits = {}
    for head in idautils.Heads(ENTRY, limit):
        if not ida_bytes.is_code(ida_bytes.get_flags(head)):
            continue
        # ⚠ 不能用 print_insn_mnem == "int" 過濾:IDA 把 CD 3D 標成 "wait"
        # (浮點模擬器呼叫),那個過濾會漏掉全部 227 處。改直接看位元組。
        b = ida_bytes.get_bytes(head, 2) or b""
        if len(b) != 2 or b[0] != 0xCD or b[1] not in (0x3D, 0x3E, 0x3F):
            continue
        vec = b[1]
        param = ida_bytes.get_byte(head + 2)
        key = f"{vec:02X}:{param:02X}"
        hits[key] = hits.get(key, 0) + 1
    out = sys.argv[1]
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    json.dump(hits, open(out, "w"), indent=0)
    ida_pro.qexit(0)

main()
