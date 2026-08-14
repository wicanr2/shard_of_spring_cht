"""把「程式碼裡的 mov r16, imm16」接到「那個 imm 指向的字串」。

原理(docs/re/70 §2 + docs/re/46 §1):
  執行期字串區塊是 [len:2][ptr:2][文字],程式碼引用的是**區塊起點 = 文字位址 − 4**。
  而 extract_text.py 抽出的每一段都帶 `ds` 欄(= 描述子自己的指標,指向文字)。
  所以 `mov bx, X` 引用的就是 ds == X+4 的那一段。**不需要任何位移↔DS 的公式。**

為什麼用位元組掃描而不是 IDA(docs/re/70 §6):
  解鎖腳本只把模組區的 ~40% 判成程式碼,以 IDA 判定為前提的工具回「0 命中」
  不能當成「不存在」。`mov r16, imm16` 是定長 3 bytes 且前導位元組明確(B8–BF),
  位元組掃描沒有分母問題。代價是會掃到資料裡的巧合 —— 所以要報假陽性率。
"""
import json, pathlib, struct, sys, collections

REG = {0xB8: 'ax', 0xB9: 'cx', 0xBA: 'dx', 0xBB: 'bx',
       0xBC: 'sp', 0xBD: 'bp', 0xBE: 'si', 0xBF: 'di'}


def scan(path, by_ds):
    """回傳 (命中清單, 掃描到的 mov 總數)。"""
    d = pathlib.Path(path).read_bytes()
    hdr = struct.unpack_from('<H', d, 0x08)[0] * 16
    hits, total = [], 0
    for i in range(hdr, len(d) - 2):
        if d[i] not in REG:
            continue
        total += 1
        imm = struct.unpack_from('<H', d, i + 1)[0]
        s = by_ds.get(imm + 4)
        if s is not None:
            hits.append({"file_off": i, "seg_off": i - hdr, "reg": REG[d[i]],
                         "imm": imm, "text": s['text'], "len": s['len']})
    return hits, total


if __name__ == "__main__":
    inv = json.loads(pathlib.Path("docs/re/generated-text-inventory.json").read_text())
    out, summary = {}, []
    for p in sys.argv[1:]:
        name = pathlib.Path(p).name
        segs = inv.get(name, {}).get('descriptor', [])
        by_ds = {s['ds']: s for s in segs}
        hits, total = scan(p, by_ds)
        linked = {h['imm'] + 4 for h in hits}
        out[name] = hits
        summary.append((name, len(segs), len(linked), len(hits), total))

    print(f"{'模組':16s} {'字串段':>6s} {'被引用':>6s} {'覆蓋':>6s} {'引用點':>6s} {'掃到的 mov':>10s}")
    ts = tl = th = tt = 0
    for name, nsegs, nlink, nhit, total in summary:
        pct = f"{nlink / nsegs * 100:.0f}%" if nsegs else "—"
        print(f"{name:16s} {nsegs:6d} {nlink:6d} {pct:>6s} {nhit:6d} {total:10d}")
        ts += nsegs; tl += nlink; th += nhit; tt += total
    print(f"{'合計':14s} {ts:6d} {tl:6d} {tl/ts*100:5.0f}% {th:6d} {tt:10d}")

    pathlib.Path("docs/re/generated-string-refs.json").write_text(
        json.dumps(out, ensure_ascii=False, indent=1))
