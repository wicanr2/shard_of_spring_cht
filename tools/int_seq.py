"""列出某段位址裡的執行期 API 呼叫序列(位元組層,不靠 IDA 的指令邊界)。

IDA 在 BASIC 模組的算術區常常對不齊(docs/re/77 §3),但 `CD 3D/3E/3F nn`
這個形狀是定長的,直接掃位元組就不會錯。

    python3 tools/int_seq.py game/sharspri/CMBT.EXE 0x13842 0x139E4

⚠ **只印 API 呼叫看不出算式**——`INT 3F:81`(加)沒說「加什麼」,
運算元是它前面那幾條 `mov`。所以本工具把兩者**交錯印出來**
(docs/re/152 就是這樣讀出來的)。

⚠ 這是一個「盡力而為」的解碼器,不是完整的 x86 反組譯器:
認得出 BASIC 算術區會用到的那十來種形狀,認不出的印成 `?? xx`。
**`??` 不代表那裡沒有指令**,只代表本工具沒認出來 ——
下結論之前該段要能整條讀通,不能跳過 `??`。
"""
import pathlib
import sys

# `bz` 模組的載入基底:線性位址 = 檔案位移 + 0xFE00(docs/re/03/04)
BASE = 0xFE00

API = {
    (0x3F, 0x57): "整數→浮點累加器(bx)", (0x3F, 0x77): "浮點累加器→整數(bx)",
    (0x3F, 0x81): "加",                  (0x3F, 0x85): "加(另一序)",
    (0x3F, 0x9D): "減",                  (0x3F, 0x91): "乘",
    (0x3F, 0x8D): "除",                  (0x3F, 0x71): "累加器↔變數",
    (0x3F, 0x61): "字串指派",            (0x3F, 0x62): "字串比較",
    (0x3F, 0xB8): "INPUT#",              (0x3F, 0x6E): "字串串接",
    (0x3F, 0x7D): "指派",                (0x3F, 0x5C): "MID$ 指派",
    (0x3F, 0x6A): "印字串",              (0x3F, 0xA1): "浮點比較(設旗標)",
    # docs/re/152 §3:前三個是 `INT(RND × N) + 1` 這個成語的零件。
    # ⚠ 3D:03 一度被讀成亂數 —— 它是取整,亂數是 3D:34。
    (0x3D, 0x03): "INT()(向下取整)",   (0x3D, 0x34): "RND",
    (0x3D, 0x0A): "MID$",                (0x3D, 0x19): "CVS(4 bytes → 單精度)",
    (0x3D, 0x1C): "MKS$(單精度 → 4 bytes)",
}

REG = {0xB8: "ax", 0xB9: "cx", 0xBA: "dx", 0xBB: "bx", 0xBE: "si", 0xBF: "di"}
# `mov r, r` 的 modrm 高兩位 = 11,這裡只列算術區常見的幾個
MOVRR = {
    0xC3: "ax, bx", 0xC2: "ax, dx", 0xD8: "bx, ax", 0xD3: "dx, bx",
    0xDA: "bx, dx", 0xD9: "bx, cx", 0xF8: "di, ax", 0xFB: "di, bx",
    0xFA: "di, dx", 0xF3: "si, bx", 0xDF: "bx, di", 0xF9: "di, cx",
    0xCB: "cx, bx", 0xCA: "cx, dx", 0xDE: "bx, si", 0xD6: "dx, si",
    0xD7: "dx, di", 0xF2: "si, dx", 0xC1: "ax, cx",
}


def decode(b, ea, i):
    """回傳 (文字, 長度);認不出來回 (None, 1)。"""
    o = b[i]
    w = lambda k: b[k] | b[k + 1] << 8

    if o == 0x90:
        return "", 1
    if o == 0xCD and b[i + 1] in (0x3D, 0x3E, 0x3F):
        v, p = b[i + 1], b[i + 2]
        return f"INT {v:02X}:{p:02X}  {API.get((v, p), '')}", 3
    if o in REG:
        return f"mov {REG[o]}, {w(i + 1):04X}h", 3
    if o == 0x05:
        return f"add ax, {w(i + 1):04X}h", 3
    if o == 0x81 and b[i + 1] in (0xC0, 0xC1, 0xC2, 0xC3, 0xC6, 0xC7):
        r = ["ax", "cx", "dx", "bx", "sp", "bp", "si", "di"][b[i + 1] & 7]
        return f"add {r}, {w(i + 2):04X}h", 4
    if o == 0xA1:
        return f"mov ax, ds:{w(i + 1):04X}h", 3
    if o == 0xA3:
        return f"mov ds:{w(i + 1):04X}h, ax", 3
    if o == 0x01 and b[i + 1] == 0x06:
        return f"add ds:{w(i + 2):04X}h, ax   ← 累加", 4
    if o == 0xFF and b[i + 1] == 0x06:
        return f"inc word ptr ds:{w(i + 2):04X}h", 4
    if o == 0xC7 and b[i + 1] == 0x06:
        return f"mov word ptr ds:{w(i + 2):04X}h, {w(i + 4):04X}h", 6
    if o == 0xC7 and b[i + 1] in (0x85, 0x84):
        r = "di" if b[i + 1] == 0x85 else "si"
        return f"mov word ptr [{r}+{w(i + 2):04X}h], {w(i + 4):04X}h", 6
    if o == 0x8B and b[i + 1] in (0x85, 0x9D, 0x95, 0x8D):
        dst = {0x85: "ax", 0x9D: "bx", 0x95: "dx", 0x8D: "cx"}[b[i + 1]]
        return f"mov {dst}, [di+{w(i + 2):04X}h]   ← 讀陣列", 4
    if o == 0x89 and b[i + 1] in (0x94, 0x84, 0x9C, 0x8C):
        src = {0x94: "dx", 0x84: "ax", 0x9C: "bx", 0x8C: "cx"}[b[i + 1]]
        return f"mov [si+{w(i + 2):04X}h], {src}   ← 寫陣列", 4
    if o == 0x8B and b[i + 1] == 0x1E:
        return f"mov bx, ds:{w(i + 2):04X}h", 4
    if o == 0x8B and b[i + 1] == 0x16:
        return f"mov dx, ds:{w(i + 2):04X}h", 4
    if o == 0x8B and b[i + 1] == 0x3E:
        return f"mov di, ds:{w(i + 2):04X}h", 4
    if o == 0x8B and b[i + 1] == 0x0E:
        return f"mov cx, ds:{w(i + 2):04X}h", 4
    if o == 0x8B and b[i + 1] in (0xBD, 0xB5):
        r = "di" if b[i + 1] == 0xBD else "si"
        return f"mov {r}, [di+{w(i + 2):04X}h]   ← 讀陣列", 4
    if o == 0x2B and b[i + 1] in (0x9D, 0x85, 0x95):
        dst = {0x9D: "bx", 0x85: "ax", 0x95: "dx"}[b[i + 1]]
        return f"sub {dst}, [di+{w(i + 2):04X}h]", 4
    if o == 0x2B and b[i + 1] in (0x1E, 0x06, 0x16):
        dst = {0x1E: "bx", 0x06: "ax", 0x16: "dx"}[b[i + 1]]
        return f"sub {dst}, ds:{w(i + 2):04X}h", 4
    if o == 0x03 and b[i + 1] in (0x1E, 0x06, 0x16, 0x3E):
        dst = {0x1E: "bx", 0x06: "ax", 0x16: "dx", 0x3E: "di"}[b[i + 1]]
        return f"add {dst}, ds:{w(i + 2):04X}h", 4
    if o == 0x83 and b[i + 1] in (0xC0, 0xC1, 0xC2, 0xC3, 0xC6, 0xC7,
                                  0xE8, 0xE9, 0xEA, 0xEB, 0xEE, 0xEF):
        r = ["ax", "cx", "dx", "bx", "sp", "bp", "si", "di"][b[i + 1] & 7]
        op = "add" if b[i + 1] < 0xE0 else "sub"
        v = b[i + 2] - 256 if b[i + 2] > 127 else b[i + 2]
        return f"{op} {r}, {v}", 3
    if o == 0x83 and b[i + 1] in (0xBD, 0xBC):
        r = "di" if b[i + 1] == 0xBD else "si"
        return f"cmp word ptr [{r}+{w(i + 2):04X}h], {b[i + 4]:02X}h", 5
    if o == 0x83 and b[i + 1] == 0x3E:
        return f"cmp word ptr ds:{w(i + 2):04X}h, {b[i + 4]:02X}h", 5
    if o == 0x3B and b[i + 1] == 0x06:
        return f"cmp ax, ds:{w(i + 2):04X}h", 4
    if o == 0x3D:
        return f"cmp ax, {w(i + 1):04X}h", 3
    if o == 0x8B and b[i + 1] in MOVRR:
        return f"mov {MOVRR[b[i + 1]]}", 2
    if o == 0xD1:
        r = {0xE0: "ax", 0xE1: "cx", 0xE2: "dx", 0xE3: "bx",
             0xE6: "si", 0xE7: "di"}.get(b[i + 1])
        if r:
            return f"shl {r}, 1", 2
    if o == 0xE9:
        d = w(i + 1)
        return f"jmp {ea + 3 + (d - 0x10000 if d > 0x7FFF else d):06X}", 3
    if o in (0x74, 0x75, 0x7C, 0x7D, 0x7E, 0x7F, 0xEB):
        d = b[i + 1]
        name = {0x74: "jz", 0x75: "jnz", 0x7C: "jl", 0x7D: "jge",
                0x7E: "jle", 0x7F: "jg", 0xEB: "jmp"}[o]
        return f"{name} {ea + 2 + (d - 256 if d > 127 else d):06X}", 2
    if o in (0x40, 0x41, 0x42, 0x43):
        return f"inc {['ax', 'cx', 'dx', 'bx'][o - 0x40]}", 1
    if o in (0x48, 0x49, 0x4A, 0x4B):
        return f"dec {['ax', 'cx', 'dx', 'bx'][o - 0x48]}", 1
    if o == 0x33 and b[i + 1] == 0xC0:
        return "xor ax, ax", 2
    if o == 0x23 and b[i + 1] in (0xDA, 0xDB):
        return f"and bx, {'dx' if b[i + 1] == 0xDA else 'bx'}", 2
    return None, 1


def main(exe, lo, hi):
    d = pathlib.Path(exe).read_bytes()
    b = d[lo - BASE:hi - BASE]
    i = 0
    while i < len(b):
        ea = lo + i
        try:
            text, n = decode(b, ea, i)
        except IndexError:
            break
        if text is None:
            print(f"  {ea:06X}  ?? {b[i]:02X}")
            i += 1
            continue
        if text:
            print(f"  {ea:06X}  {text}")
        i += n


if __name__ == "__main__":
    main(sys.argv[1], int(sys.argv[2], 0), int(sys.argv[3], 0))
