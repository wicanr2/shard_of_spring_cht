"""RE-01:從 IDA 資料庫匯出可比較的初始清冊。

用法(由 tools/ida.sh 呼叫):
    idat -A "-S/work/tools/ida/export_inventory.py /work/out/START.json" START.EXE.i64

輸出是 JSON,欄位固定。**不要在這支腳本裡下任何語意判斷** ——
它只負責把 IDA 已經知道的事實搬出來,判讀留給筆記(CLAUDE.md §2.1)。
"""

import hashlib
import json
import os
import sys

import ida_auto
import ida_bytes
import ida_funcs
import ida_nalt
import ida_pro
import ida_segment
import idaapi
import idautils
import idc


def sha256_of_input():
    path = ida_nalt.get_input_file_path()
    try:
        with open(path, "rb") as fh:
            return hashlib.sha256(fh.read()).hexdigest()
    except OSError:
        # headless 容器裡原始路徑可能不在;退回 IDA 自己記的 MD5
        return None


def segments():
    rows = []
    for ea in idautils.Segments():
        seg = ida_segment.getseg(ea)
        rows.append({
            "name": ida_segment.get_segm_name(seg),
            "start": f"0x{seg.start_ea:X}",
            "end": f"0x{seg.end_ea:X}",
            "size": seg.end_ea - seg.start_ea,
            "class": ida_segment.get_segm_class(seg),
            "bitness": seg.bitness,        # 0 = 16-bit
        })
    return rows


def functions():
    rows = []
    for ea in idautils.Functions():
        f = ida_funcs.get_func(ea)
        name = ida_funcs.get_func_name(ea)
        rows.append({
            "ea": f"0x{ea:X}",
            "name": name,
            "size": f.end_ea - f.start_ea,
            "named": not name.startswith(("sub_", "nullsub_", "j_")),
            "callers": len({x.frm for x in idautils.XrefsTo(ea) if x.iscode}),
        })
    return rows


def strings(min_len=4):
    """IDA 認定的字串項。

    ⚠ 這個數字通常**遠低於**實際可見字串 —— IDA 只列它已經定義成字串項的,
    沒被自動分析碰到的資料區完全不算。所以另外做一次 raw_ascii() 當對照,
    兩個數字差很多是正常的,但差距本身要記錄下來,不要只報其中一個。
    """
    sc = idautils.Strings()
    sc.setup(strtypes=[idaapi.STRTYPE_C], minlen=min_len)
    rows = []
    for s in sc:
        try:
            text = str(s)
        except Exception:
            continue
        rows.append({"ea": f"0x{s.ea:X}", "len": s.length, "text": text})
    return rows


def raw_ascii(min_len=6):
    """獨立掃描:整個 database 的可列印 ASCII 連續段。

    不依賴 IDA 的自動分析結果,所以能抓到 strings() 漏掉的資料區文字。
    這是中文化盤點(CLAUDE.md §7 落點二)的主要輸入。
    """
    rows = []
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        buf = ida_bytes.get_bytes(seg.start_ea, seg.end_ea - seg.start_ea) or b""
        start = None
        for i, b in enumerate(buf):
            printable = 0x20 <= b <= 0x7E
            if printable and start is None:
                start = i
            elif not printable and start is not None:
                if i - start >= min_len:
                    rows.append({
                        "ea": f"0x{seg.start_ea + start:X}",
                        "len": i - start,
                        "text": buf[start:i].decode("ascii", "replace"),
                    })
                start = None
        if start is not None and len(buf) - start >= min_len:
            rows.append({
                "ea": f"0x{seg.start_ea + start:X}",
                "len": len(buf) - start,
                "text": buf[start:].decode("ascii", "replace"),
            })
    return rows


def entries():
    return [
        {"ordinal": ordinal, "ea": f"0x{ea:X}", "name": name}
        for _, ordinal, ea, name in idautils.Entries()
    ]


def imports():
    rows = []

    def cb(ea, name, ordinal):
        rows.append({"ea": f"0x{ea:X}", "name": name, "ordinal": ordinal})
        return True

    for i in range(idaapi.get_import_module_qty()):
        mod = idaapi.get_import_module_name(i) or f"<module {i}>"
        current = len(rows)
        idaapi.enum_import_names(i, cb)
        for row in rows[current:]:
            row["module"] = mod
    return rows


def far_calls():
    """跨段呼叫的目的地統計。

    16-bit DOS 的模組間／runtime 呼叫都是 far call,
    彙總目的地就能看出「誰是共用入口」——RE-02 要用。
    """
    hits = {}
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        for head in idautils.Heads(seg.start_ea, seg.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_flags(head)):
                continue
            mnem = idc.print_insn_mnem(head)
            if mnem not in ("call", "jmp"):
                continue
            op = idc.print_operand(head, 0)
            if "far" not in idc.GetDisasm(head).lower() and ":" not in op:
                continue
            hits[op] = hits.get(op, 0) + 1
    return sorted(
        ({"target": k, "count": v} for k, v in hits.items()),
        key=lambda r: -r["count"],
    )


def main():
    ida_auto.auto_wait()
    out = sys.argv[1] if len(sys.argv) > 1 else "/work/out/inventory.json"

    func_rows = functions()
    str_rows = strings()
    ascii_rows = raw_ascii()

    data = {
        "input": os.path.basename(ida_nalt.get_input_file_path()),
        "sha256": sha256_of_input(),
        "ida_version": idaapi.get_kernel_version(),
        "processor": idaapi.inf_get_procname(),
        "counts": {
            "functions": len(func_rows),
            "named_functions": sum(1 for r in func_rows if r["named"]),
            "segments": len(segments()),
            "strings_ida": len(str_rows),
            "strings_raw_ascii": len(ascii_rows),
            "entries": len(entries()),
            "imports": len(imports()),
        },
        "segments": segments(),
        "entries": entries(),
        "imports": imports(),
        "functions": func_rows,
        "strings_ida": str_rows,
        "strings_raw_ascii": ascii_rows,
        "far_call_targets": far_calls(),
    }

    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    with open(out, "w", encoding="utf-8") as fh:
        json.dump(data, fh, indent=2, ensure_ascii=False)
    ida_pro.qexit(0)


main()
