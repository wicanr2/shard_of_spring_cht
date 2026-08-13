"""抽出每支模組的單字元按鍵比對鏈(docs/re/70)。

形狀:mov bx, <字串描述子>; mov ax, <暫存字串>; int 3F 62(字串比較)。
描述子位址 = 文字位址 − 4(docs/re/70 §2;⚠ 不是 +2)。

純位元組掃描,不經 IDA —— 因為 IDA 只把 40% 的模組區判成程式碼,
落在未判定區的比較會整批漏掉(docs/re/70 §5)。
"""
import json, pathlib, struct, sys

def chains(exe, inv):
    d = pathlib.Path(exe).read_bytes()
    txt = {r["ds"]: r["text"] for r in inv["descriptor"]}
    seq, i = [], 0
    while True:
        k = d.find(b'\xcd\x3f\x62', i)
        if k < 0:
            break
        i = k + 1
        for back in range(3, 22):          # bx 可能隔著 mov ax 之類的指令
            if k - back >= 0 and d[k - back] in (0xBB, 0xB8):
                imm = struct.unpack_from('<H', d, k - back + 1)[0]
                if imm + 4 in txt:
                    seq.append((0x10000 + k - 0x200, txt[imm + 4]))
                    break
    return seq

if __name__ == "__main__":
    inv = json.load(open("docs/re/generated-text-inventory.json"))
    out = {}
    for name in sorted(inv):
        s = chains(f"game/sharspri/{name}", inv[name])
        if s:
            out[name] = [[hex(a), t] for a, t in s]
            print(f"{name:14s} " + " ".join(t for _, t in s))
    pathlib.Path("docs/re/generated-key-chains.json").write_text(
        json.dumps(out, ensure_ascii=False, indent=1))
