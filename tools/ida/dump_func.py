"""把指定函式(或位址範圍)的反組譯逐行倒出來,附註解與 xref。

用法:
    idat -A "-S/work/tools/ida/dump_func.py /work/out/x.txt <名稱或0x位址> [...]" X.EXE.i64

不做任何過濾。**不要為了讓輸出短而濾掉「看起來是樣板」的行** ——
被濾掉的常常正是索引計算(見 kb ida-pro-9.4.md 工作紀律)。
"""

import os
import sys

import ida_auto
import ida_bytes
import ida_funcs
import ida_lines
import ida_name
import ida_pro
import idautils
import idc


def resolve(token):
    if token.lower().startswith("0x"):
        return int(token, 16)
    ea = ida_name.get_name_ea(idc.BADADDR, token)
    return ea if ea != idc.BADADDR else None


def dump_one(fh, token):
    ea = resolve(token)
    if ea is None:
        fh.write(f"### {token}: 找不到\n\n")
        return
    f = ida_funcs.get_func(ea)
    if f is None:
        fh.write(f"### {token} @ 0x{ea:X}: 不在任何函式裡\n\n")
        return

    name = ida_funcs.get_func_name(f.start_ea)
    callers = sorted({x.frm for x in idautils.XrefsTo(f.start_ea) if x.iscode})
    fh.write(f"### {name}  0x{f.start_ea:X}–0x{f.end_ea:X}  ({f.end_ea - f.start_ea} bytes)\n")
    fh.write(f"# 呼叫端 {len(callers)} 處: " +
             ", ".join(f"0x{c:X}({ida_funcs.get_func_name(c)})" for c in callers[:12]) + "\n\n")

    for head in idautils.Heads(f.start_ea, f.end_ea):
        raw = ida_bytes.get_bytes(head, ida_bytes.get_item_size(head)) or b""
        line = ida_lines.tag_remove(ida_lines.generate_disasm_line(head, 0) or "")
        cmt = idc.get_cmt(head, 0) or idc.get_cmt(head, 1) or ""
        # 這一行有沒有被別的地方跳進來(基本區塊邊界)
        tgt = "  <<<" if any(True for _ in idautils.XrefsTo(head)) and head != f.start_ea else ""
        fh.write(f"{head:06X}  {raw.hex():<14} {line}{('   ; ' + cmt) if cmt else ''}{tgt}\n")
    fh.write("\n")


def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    with open(out, "w", encoding="utf-8") as fh:
        for token in sys.argv[2:]:
            dump_one(fh, token)
    ida_pro.qexit(0)


main()
