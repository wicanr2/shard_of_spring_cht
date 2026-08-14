# `MONSTERS.DAT` — 怪物表 — **READY**

74 筆 × **36 bytes**,定長隨機存取(`OPEN … mov cx, 24h`;`2664 ÷ 36 = 74`)。
照怪物型別**隨機存取**,不整批載入。

```
位移 1–16   名稱(16 bytes,空白補齊)
位移 17–36  十個 2-byte 整數(CVI)
```

| 欄 | 位移 | 語意 | → 戰鬥屬性 | 信心 |
|---:|---:|---|---:|---|
| 1 | 17 | 速度(乘亂數後存入)| 2 | 證據充分 |
| 2 | 19 | 力量 | 6 | 證據充分 |
| 3 | 21 | 命中能力 | 9 | 已確認 |
| 4 | 23 | **生命值的骰基**(乘亂數後存入)| 3 | 已確認 |
| 5 | 25 | 武器編號(`0` → 60 = 赤手)| 4 | 已確認 |
| 6 | 27 | 類別 / 圖組 | 11 | 證據充分 |
| 7 | 29 | 防具編號 | 5 | 已確認 |
| 8 | 31 | **經驗值**(戰鬥中只寫不讀)| 19 | 證據充分 |
| 9 | 33 | 難度階級(1–10, 13;**不是等級**)| 13 | 證據充分 |
| 10 | 35 | 法力點數 | 7 | 已確認 |

⚠ **欄9 不是等級**:`Lv5 Wizard` 與 `Lv6 Fighter` 同為 5、`Lv12`/`Lv13` 同為 8、
`Lv15`/`Lv16` 同為 9 —— 三組碰撞。它把戰力相當的一組歸成同一階。

⚠ **怪物的速度與生命值每次遭遇現骰**(乘亂數),角色的存在存檔裡。

## 法術系別 = 狀態編號

`SPELLS.DAT` 欄2(系別)命中時直接寫進目標的狀態欄:

| 系別 | 名稱 | 狀態表(0 起算)|
|---:|---|---|
| 0 | — | `OK` |
| 1 | Fire runes | `Poisoned` |
| 2 | Metal runes | `Bound` |
| 3 | Wind runes | `Still Air` |
| 4 | Ice runes | `Frozen` |
| 5 | Spirit runes | `D E A D` |

驗證:`CHAINS`(Metal)→ `Bound`、`STILL AIR`(Wind)→ `Still Air`、
`FREEZE`(Ice)→ `Frozen`,三筆全中。

出處:[`re/72`](../re/72-e-file-formats-from-readers.md)、[`73`](../re/73-monster-columns-to-combat-array.md)、
[`82`](../re/82-monster-columns-semantics.md)、[`83`](../re/83-hp-is-attribute-3.md)、
[`90`](../re/90-monster-column-9-is-a-tier.md)、[`111`](../re/111-status-code-equals-school.md)
