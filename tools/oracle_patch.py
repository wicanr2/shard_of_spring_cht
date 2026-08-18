#!/usr/bin/env python3
"""替 oracle 量測改造 `workplace/dosbox/game/` 裡的存檔複本。

    python3 tools/oracle_patch.py hunt      # 給第 1 位隊員 Hunting 技能
    python3 tools/oracle_patch.py strong    # 全隊生命值與命中拉高,好打得贏
    python3 tools/oracle_patch.py hunt strong
    python3 tools/oracle_patch.py place 53,68,1   # 把 PARTY #5 放到某格、面向北

⚠ **只動 `workplace/` 的複本**,`game/sharspri/` 唯讀(`CLAUDE.md` §8)。
複本不存在時先跑一次 `tools/dosbox_run.sh` 讓它建起來。

**為什麼可以這樣量**:要量的兩件事都與被改的欄位無關 ——

- 打獵的收穫量:程式先過技能閘門(docs/re/166 §2)才擲骰,
  擲骰那一段不讀技能旗標,所以「把旗標打開」不影響收穫的分佈;
- 戰後金幣:公式吃的是**怪物的難度階級**(docs/re/152 §2.3),
  不吃隊伍的屬性,所以把隊伍變強不影響金額。

⛔ 反過來說,**不要用這支腳本去量任何會讀到這些欄位的東西** ——
例如命中率(改了命中能力)或「打獵成不成功」(若成功率其實與技能等級有關)。
"""
import pathlib
import sys

L = 94
OFF_TOHIT, OFF_MAXHP, OFF_HP = 24, 26, 28   # 1-based(docs/formats/01)
OFF_SKILLS = 42                              # 位移 42–51,技能 n → 位移 41+n
HUNTING = 9

GL = 90                                      # GROUPS.DAT 每筆 90 bytes
OFF_WX, OFF_WY, OFF_FACING = 35, 37, 41      # 1-based(docs/formats/02)
OFF_MAZENUM = 83                             # 99 = 不在迷宮(docs/re/169)
PARTY5 = 4                                   # 出貨檔唯一有內容的那一筆

# ⚠ `place` 改的是**隊伍在世界地圖的座標**。
# 為什麼可以這樣量:要量的是「站在某個地城入口按方向鍵會印出哪個名字」,
# 而那個名字由 `MAZEDATA` 的入口索引決定(docs/re/222)——
# 與隊伍是怎麼走到那一格的無關。
# ⛔ 反過來:任何**吃路程**的東西(遭遇倒數、時鐘、食糧消耗)不能這樣量。


def put16(b, pos1, v):
    b[pos1 - 1] = v & 0xFF
    b[pos1] = (v >> 8) & 0xFF


def place(root, spec):
    """把 PARTY #5 放到 `x,y[,facing]`(facing 1北 2東 3南 4西)。"""
    parts = [int(v) for v in spec.split(",")]
    x, y = parts[0], parts[1]
    facing = parts[2] if len(parts) > 2 else 1
    f = root / "workplace/dosbox/game/GROUPS.DAT"
    if not f.exists():
        raise SystemExit(f"找不到 {f} —— 先跑一次 tools/dosbox_run.sh 建立複本")
    d = bytearray(f.read_bytes())
    r = memoryview(d)[PARTY5 * GL:(PARTY5 + 1) * GL]
    put16(r, OFF_WX, x)
    put16(r, OFF_WY, y)
    put16(r, OFF_FACING, facing)
    # ⚠ **一併清掉迷宮編號** —— 放到世界地圖上就表示不在迷宮裡。
    # 少了這一步,上一次實驗留下的編號會讓遊戲以為隊伍還在地城,
    # 開機之後直接轉交 MAZEMOVE 然後噴
    # `File not found in line 61440 of module MAZEMOVE` ——
    # 那個訊息看起來像**檔案缺了**,實際上是存檔的狀態不一致。
    put16(r, OFF_MAZENUM, 99)
    f.write_bytes(bytes(d))
    print(f"已把 PARTY #5 放到 ({x},{y}) 面向 {facing}:{f}")


def main(argv):
    root = pathlib.Path(__file__).resolve().parent.parent
    for i, a in enumerate(argv):
        if a == "place":
            place(root, argv[i + 1])
            argv = argv[:i] + argv[i + 2:]
            break
    if not argv:
        return
    f = root / "workplace/dosbox/game/CHARS.DAT"
    if not f.exists():
        raise SystemExit(f"找不到 {f} —— 先跑一次 tools/dosbox_run.sh 建立複本")
    d = bytearray(f.read_bytes())
    for i in range(5):
        r = memoryview(d)[i * L:(i + 1) * L]
        name = bytes(r[1:11]).decode("latin-1").strip()
        if "hunt" in argv and i == 0:
            r[OFF_SKILLS - 1 + HUNTING - 1] = ord("1")
            print(f"  {name}:技能 {HUNTING}(Hunting)打開")
        if "strong" in argv:
            put16(r, OFF_MAXHP, 200)
            put16(r, OFF_HP, 200)
            put16(r, OFF_TOHIT, 90)
            print(f"  {name}:生命值 200/200、命中 90")
    f.write_bytes(bytes(d))
    print(f"已改寫 {f}")


if __name__ == "__main__":
    main(sys.argv[1:])
