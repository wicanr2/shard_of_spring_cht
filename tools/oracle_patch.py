#!/usr/bin/env python3
"""替 oracle 量測改造 `workplace/dosbox/game/` 裡的存檔複本。

    python3 tools/oracle_patch.py hunt      # 給第 1 位隊員 Hunting 技能
    python3 tools/oracle_patch.py strong    # 全隊生命值與命中拉高,好打得贏
    python3 tools/oracle_patch.py hunt strong

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


def put16(b, pos1, v):
    b[pos1 - 1] = v & 0xFF
    b[pos1] = (v >> 8) & 0xFF


def main(argv):
    root = pathlib.Path(__file__).resolve().parent.parent
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
