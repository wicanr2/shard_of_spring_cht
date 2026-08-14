"""把 BASIC 編譯器產生的布林運算式還原成可讀的判斷式。

MS BASIC 把 `IF A AND B THEN` 編成一個**極度規律**的形狀:

    cmp  <運算元>, <立即數>
    mov  <暫存器>, 0
    j<cc> short +1          ; 跳過下一行
    dec  <暫存器>           ; 暫存器 = -1(TRUE)當且僅當 j<cc> **沒有**跳

所以 `mov reg,0 / jge L / dec reg / L:` 的意思是 **reg = TRUE 當 值 < imm**
—— 條件與助憶碼是**相反**的。這正是人眼最容易讀錯的地方
(docs/re/131:我第一次就把 jge/jl 讀反)。

接著用 `or` / `and` / `not` 組合,最後 `and reg,reg` + `jnz` 決定分支。

用法:
    tools/ida.sh run tools/ida/dump_range.py X.EXE /work/out/d.txt 0xLO 0xHI
    python3 tools/bool_chain.py workplace/ida/out/d.txt

⚠ 本工具只認這一種形狀。**認不出來的行會原樣印出**,不會安靜略過 ——
「沒有輸出」與「這裡沒有布林運算」必須看得出差別。
"""
import re
import sys
import pathlib

# j<cc> 沒跳時 → TRUE。所以 TRUE 的條件是助憶碼的**反面**。
NEGATED = {
    "jge": "<", "jl": ">=", "jle": ">", "jg": "<=",
    "jnz": "==", "jne": "==", "jz": "!=", "je": "!=",
    "ja": "<=", "jb": ">=", "jae": "<", "jbe": ">",
}

LINE = re.compile(r"^([0-9A-F]{6})\s+\S+\s+(.*?)(?:\s+<<<.*)?$")


def norm(s):
    """把 IDA 的立即數寫法轉成十進位;`ds:0CD5Ah` 之類原樣保留。"""
    s = s.strip().rstrip(";").strip()
    s = re.sub(r"\s*;.*$", "", s)
    m = re.fullmatch(r"([0-9A-F]+)h", s)
    if m:
        return str(int(m.group(1), 16))
    if re.fullmatch(r"\d+", s):
        return s
    return s


def parse(path):
    rows = []
    for ln in pathlib.Path(path).read_text(encoding="utf-8").splitlines():
        m = LINE.match(ln.strip())
        if m:
            rows.append((m.group(1), m.group(2).strip()))
    out, i = [], 0
    expr = {}          # 暫存器 → 可讀的判斷式
    while i < len(rows):
        addr, ins = rows[i]

        # 形狀一:cmp X, imm / mov reg,0 / j<cc> / dec reg
        m = re.match(r"cmp\s+(?:word ptr\s+)?(.+?),\s*(.+)$", ins)
        if m and i + 3 < len(rows):
            lhs, rhs = m.group(1).strip(), norm(m.group(2))
            m2 = re.match(r"mov\s+(\w\w),\s*0$", rows[i + 1][1])
            m3 = re.match(r"(j\w+)\s", rows[i + 2][1])
            m4 = re.match(r"dec\s+(\w\w)$", rows[i + 3][1])
            if m2 and m3 and m4 and m2.group(1) == m4.group(1):
                op = NEGATED.get(m3.group(1))
                if op:
                    reg = m2.group(1)
                    expr[reg] = f"{lhs} {op} {rhs}"
                    out.append(f"{addr}  {reg} := ({expr[reg]})")
                    i += 4
                    continue

        # 形狀四(必須先判,否則 `and dx,dx` 會被形狀二當成合併,
        # 把整個式子複製一份 —— 輸出看起來像 `X AND X`)
        m = re.match(r"and\s+(\w\w),\s*(\w\w)$", ins)
        if m and m.group(1) == m.group(2) and m.group(1) in expr and i + 1 < len(rows):
            r = m.group(1)
            m2 = re.match(r"(j\w+)\s+(?:short\s+)?(\S+)", rows[i + 1][1])
            if m2:
                sense = "成立" if m2.group(1) in ("jnz", "jne") else "不成立"
                out.append(f"{addr}  ⇒ 若 {expr[r]} {sense} → {m2.group(2)}")
                i += 2
                continue

        # 形狀二:or / and 兩個**不同**暫存器
        m = re.match(r"(or|and)\s+(\w\w),\s*(\w\w)$", ins)
        if m and m.group(2) != m.group(3) and m.group(2) in expr and m.group(3) in expr:
            op, a, b = m.group(1), m.group(2), m.group(3)
            joiner = " OR " if op == "or" else " AND "
            expr[a] = f"({expr[a]}{joiner}{expr[b]})"
            out.append(f"{addr}  {a} := {expr[a]}")
            i += 1
            continue

        # 形狀三:not reg
        m = re.match(r"not\s+(\w\w)$", ins)
        if m and m.group(1) in expr:
            r = m.group(1)
            expr[r] = f"NOT {expr[r]}"
            out.append(f"{addr}  {r} := {expr[r]}")
            i += 1
            continue

        out.append(f"{addr}  {ins}")
        i += 1
    return out


if __name__ == "__main__":
    if len(sys.argv) < 2:
        sys.exit(__doc__)
    for line in parse(sys.argv[1]):
        print(line)
