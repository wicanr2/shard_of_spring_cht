"""在**原始位元組**裡找樣式,不經過 IDA 的指令切分。

用法: idat -A "-S/work/tools/ida/find_bytes.py /work/out/x.txt a17e70 8b1e7e70" X.i64
      (參數是十六進位位元組串,不帶空白;`??` 代表任意一個 byte)

為什麼需要這支(而 `find_disp.py` 不夠):
`find_disp.py` 走 `idautils.Heads()`,只看得到 **IDA 認成程式碼**的部分。
`INT 3D/3E/3F` 帶內嵌運算元會讓線性反組譯從那裡開始切錯(CLAUDE.md §3.3),
切壞的那幾行在 `Heads()` 裡是 `db`,於是那裡的存取**一條都掃不到** ——
而零命中與「真的沒人碰」長得一模一樣。

⚠ 反過來也要小心:這支**不判斷命中處是不是指令**,資料區裡剛好同樣的
兩個 byte 也會命中。拿它定位,再用 `raw_bytes.py` / `dump_many.py` 判讀。

輸出附每一處的前 8 / 後 16 個 byte,以及 IDA 對該位址的看法(code/data)。
"""
import os
import sys

import ida_auto
import ida_bytes
import ida_lines
import ida_pro
import ida_segment
import idautils


def parse(pat: str):
    """把 'a17e70' / 'a1??70' 轉成 [(值 or None), ...]。"""
    pat = pat.replace(" ", "").lower()
    if len(pat) % 2:
        raise ValueError(f"位元組串長度要是偶數:{pat}")
    out = []
    for i in range(0, len(pat), 2):
        tok = pat[i:i + 2]
        out.append(None if tok == "??" else int(tok, 16))
    return out


def main():
    ida_auto.auto_wait()
    out = sys.argv[1]
    pats = [(p, parse(p)) for p in sys.argv[2:]]
    if not pats:
        ida_pro.qexit(2)
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)

    blobs = []
    for seg_ea in idautils.Segments():
        seg = ida_segment.getseg(seg_ea)
        data = ida_bytes.get_bytes(seg.start_ea, seg.end_ea - seg.start_ea)
        if data:
            blobs.append((seg.start_ea, data))

    with open(out, "w", encoding="utf-8") as fh:
        total = sum(len(d) for _, d in blobs)
        fh.write(f"# 掃了 {total} bytes / {len(blobs)} 個節區\n")
        for text, pat in pats:
            hits = []
            n = len(pat)
            for base, data in blobs:
                for i in range(len(data) - n + 1):
                    if all(p is None or data[i + j] == p for j, p in enumerate(pat)):
                        hits.append(base + i)
            fh.write(f"\n## {text} —— {len(hits)} 處\n")
            for ea in hits:
                kind = "code" if ida_bytes.is_code(ida_bytes.get_flags(ea)) else "data"
                pre = ida_bytes.get_bytes(ea - 8, 8) or b""
                post = ida_bytes.get_bytes(ea, 16 + n) or b""
                line = ida_lines.tag_remove(
                    ida_lines.generate_disasm_line(ea, 0) or "")
                fh.write(f"\n  {ea:06X}  [{kind}]  {line}\n")
                fh.write(f"    前 {pre.hex()} | {post.hex()}\n")
    ida_pro.qexit(0)


main()
