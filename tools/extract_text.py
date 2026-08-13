"""抽出模組裡的文字:三個 4-byte 描述子 + 文字本體(docs/re/46 §1)。

判準:第二個描述子的長度欄 == 文字實際長度(七組驗證)。
只認這個形狀,不猜 —— 抓不到的寧可漏,不要假陽性。
"""
import json, pathlib, struct, sys

def extract(path):
    d = pathlib.Path(path).read_bytes()
    hdr = struct.unpack_from('<H', d, 0x08)[0] * 16
    rows, i = [], hdr
    end = len(d)
    while i + 12 < end:
        l1, p1, l2, p2, l3, p3 = struct.unpack_from('<HHHHHH', d, i)
        if l1 == 8 and 1 <= l2 <= 200 and p1 and p2 and p3 and l3 in (l2 + 4, l2 + 5):
            body = d[i + 12: i + 12 + l2]
            if len(body) == l2 and all(32 <= c < 127 for c in body):
                rows.append({"file_off": i + 12, "seg_off": (i + 12) - hdr,
                             "len": l2, "ds": p2,
                             "text": body.decode('latin1')})
                i += 12 + l2
                continue
        i += 1
    return rows

def extract_data_stmts(path, taken):
    """第二種形式:BASIC 的 DATA 敘述是 null 結尾的純文字(docs/re/46 §6)。

    taken = 已被描述子形式吃掉的位元組範圍,避免重複計。
    """
    d = pathlib.Path(path).read_bytes()
    hdr = struct.unpack_from('<H', d, 0x08)[0] * 16
    rows, i, n = [], hdr, len(d)
    while i < n:
        if 32 <= d[i] < 127 and i not in taken:
            j = i
            while j < n and 32 <= d[j] < 127:
                j += 1
            if j < n and d[j] == 0 and (j - i) >= 4 and not (set(range(i, j)) & taken):
                rows.append({"file_off": i, "seg_off": i - hdr,
                             "len": j - i, "text": d[i:j].decode('latin1')})
            i = j + 1
        else:
            i += 1
    return rows


if __name__ == "__main__":
    out = {}
    for p in sys.argv[1:]:
        name = pathlib.Path(p).name
        rows = extract(p)
        taken = set()
        for r in rows:
            taken.update(range(r['file_off'], r['file_off'] + r['len']))
        data = extract_data_stmts(p, taken)
        out[name] = {"descriptor": rows, "data_stmt": data}
        t1 = sum(r['len'] for r in rows)
        t2 = sum(r['len'] for r in data)
        print(f"{name:14s} 描述子 {len(rows):4d} 段/{t1:5d}B   DATA {len(data):4d} 段/{t2:5d}B")
    pathlib.Path("docs/re/generated-text-inventory.json").write_text(
        json.dumps(out, ensure_ascii=False, indent=1))
