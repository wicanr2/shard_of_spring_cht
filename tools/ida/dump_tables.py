"""倒出執行期三張派工表(docs/re/06 §1),含每個進入點的目標與所屬函式。"""
import json, os, sys
import ida_auto, ida_bytes, ida_funcs, ida_pro

TABLES = {"3D": 0x10171, "3E": 0x10243, "3F": 0x1038D}

def main():
    ida_auto.auto_wait()
    out, n = sys.argv[1], int(sys.argv[2], 0)
    res = {}
    for vec, base in TABLES.items():
        rows = []
        for i in range(n):
            off = ida_bytes.get_word(base + i * 2)
            tgt = 0x10000 + off
            rows.append({"i": i, "off": f"0x{off:04X}",
                         "func": ida_funcs.get_func_name(tgt) or ""})
        res[vec] = {"base": f"0x{base:X}", "entries": rows}
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    json.dump(res, open(out, "w", encoding="utf-8"), indent=1)
    ida_pro.qexit(0)

main()
