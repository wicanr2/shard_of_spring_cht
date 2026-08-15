"""列出某段位址裡的執行期 API 呼叫序列(位元組層,不靠 IDA 的指令邊界)。

IDA 在 BASIC 模組的算術區常常對不齊(docs/re/77 §3),但 `CD 3D/3E/3F nn`
這個形狀是定長的,直接掃位元組就不會錯。
"""
import pathlib, sys

API = {
    (0x3F, 0x57): "整數→浮點累加器(bx)", (0x3F, 0x77): "浮點累加器→整數(bx)",
    (0x3F, 0x81): "加",                  (0x3F, 0x85): "加(另一序)",
    (0x3F, 0x9D): "減",                  (0x3F, 0x91): "乘",
    (0x3F, 0x8D): "除",                  (0x3F, 0x71): "累加器↔變數",
    (0x3F, 0x61): "字串指派",            (0x3F, 0x62): "字串比較",
    (0x3F, 0xB8): "INPUT#",              (0x3F, 0x6E): "字串串接",
    (0x3F, 0x7D): "指派",                (0x3F, 0x5C): "MID$ 指派",
    (0x3F, 0x6A): "印字串",
    # docs/re/152 §3:這三個是 `INT(RND × N) + 1` 這個成語的零件,
    # ⚠ 3D:03 一度被讀成亂數 —— 它是取整,亂數是 3D:34。
    (0x3D, 0x03): "INT()(向下取整)",   (0x3D, 0x34): "RND",
    (0x3D, 0x0A): "MID$",                (0x3D, 0x19): "CVS(4 bytes → 單精度)",
    (0x3D, 0x1C): "MKS$(單精度 → 4 bytes)",
}

def main(exe, lo, hi):
    d = pathlib.Path(exe).read_bytes()
    for k in range(lo - 0xFE00, hi - 0xFE00):
        if d[k] == 0xCD and d[k + 1] in (0x3D, 0x3E, 0x3F):
            v, p = d[k + 1], d[k + 2]
            print(f"  0x{k + 0xFE00:X}  INT {v:02X}:{p:02X}  {API.get((v, p), '')}")

if __name__ == "__main__":
    main(sys.argv[1], int(sys.argv[2], 0), int(sys.argv[3], 0))
