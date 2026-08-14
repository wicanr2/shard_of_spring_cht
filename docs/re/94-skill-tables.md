# 94 — 兩張技能表共用同一段位移,由職業決定用哪一張

日期:2026-08-14
接續:[`93-initiative.md`](93-initiative.md)
子系統:**D. 角色資料** × **J. 戰鬥規則**

## 結論

`CAMP` 的 `DATA` 敘述裡有**兩張各十項的技能表**,
而 `CHARS.DAT` **位移 42–51 就是那十個旗標** ——
**同一段位移,職業 `1` 讀表 A、職業 `2` 讀表 B。**

| # | 表 A(`Hero`)| 表 B(`Wizard`)|
|---:|---|---|
| 1 | Sword | Fire runes |
| 2 | Axe | Metal runes |
| 3 | Mace | Wind runes |
| 4 | Karate | Ice runes |
| 5 | Darkvision | Spirit runes |
| 6 | Tactics | Weapon lore |
| **7** | **Armored skin** | Potion lore |
| **8** | **Berserking** | Item lore |
| 9 | Hunting | Monster lore |
| 10 | Persuasion | Priesthood |

技能 `n` → **位移 `41 + n`**([`74`](74-spell-and-item-columns.md) §1 的
`MID$(角色字串, 系別 + 41, 1)` 就是表 B 的 1–5)。

## 1. 五個預設角色,名字就是驗證

| 角色 | 職業 | 位移 42–51 | 技能 |
|---|---|---|---|
| `Segrono` | Hero | `1000001100` | Sword, Armored skin, Berserking |
| **`Hard Axe`** | Hero | `0100011100` | **Axe**, Tactics, Armored skin, Berserking |
| `Grod` | Hero | `1000001100` | Sword, Armored skin, Berserking |
| **`Fire Hawk`** | Wizard | `1100110100` | **Fire runes**, Metal runes, Spirit runes, Weapon lore, Item lore |
| `Richtatha` | Wizard | `1001100001` | Fire runes, Ice runes, Spirit runes, Priesthood |

**`Hard Axe` 的技能是 Axe、`Fire Hawk` 的技能是 Fire runes。**

設計者給預設角色取的名字反映他們的技能 —— 兩筆各自命中,
而且用的是**兩張不同的表**。這比任何單一測試都強。

## 2. 屬性 16 / 17 的語意

[`92`](92-attribute-15-16.md) 只知道它們是「`Hero` 專屬的 0/1 旗標」。現在:

| 屬性 | 位移 | 技能 | 作用 |
|---:|---:|---|---|
| **17** | 48 | **Armored skin** | 傷害公式的**減項**([`79`](79-alignment-resolved-damage-formula.md) §3)|
| **16** | 49 | **Berserking** | 未解 |

**`Armored skin` 直接從傷害裡扣** —— 名稱與作用完全吻合。

而職業閘門也解釋清楚了:對 `Wizard` 而言位移 48/49 是
`Potion lore` / `Item lore`,**與戰鬥無關,所以填 0**。
不是「Wizard 沒有這個能力」,是**同一格在兩張表裡是不同的技能**。

> **判準:同一塊資料在不同情境下是不同的東西 —— 這已經是本專案第三次。**
> [`76`](76-to-hit-formula.md) §2 的 `ITEMS` 欄4/欄5、[`91`](91-e-closure.md) §1 的欄6,
> 現在是技能旗標。**問「這一格是什麼」之前先問「誰在讀、用哪張表」。**

## 3. 五個法術系別的名稱定案

表 B 的 1–5 就是 [`91`](91-e-closure.md) §2 一直沒指認的法術系別:

| 系別 | 名稱 | `SPELLS.DAT` 裡的成員(抽樣)|
|---:|---|---|
| 1 | **Fire runes** | `COLUMN OF FIRE` `FLAME STRIKE` `FIRE STORM` `MELT` `MAGIC TORCH` `FLAME SHIELD` |
| 2 | **Metal runes** | `SWORD` `CHAINS` `DEATH BLADE` `STRENGTH` `BREAK BONDS` `ARMOR` |
| 3 | **Wind runes** | `TEMPEST` `STILL AIR` `WINGS` `WIND WALK` `FREEDOM` `BREATH OF LIFE` |
| 4 | **Ice runes** | `HAIL STORM` `CHILL` `SLOW` `FREEZE` `ICE SHIELD` `CRYSTALIGHT` |
| 5 | **Spirit runes** | `SPIRIT WRACK` `WEAKEN` `HEAL` `RESURRECT` `CURE POISON` `SANCTUARY` |

**33 個法術逐一對得上。** `SWORD` / `CHAINS` / `ARMOR` 歸「金屬」、
`BREATH OF LIFE` 歸「風」—— 這種分法只有拿到表才想得到,猜不出來。

順帶:[`91`](91-e-closure.md) 起草時懷疑過「解除法術與所解法術同系別」,
被 `MELT`(火)解 `FREEZE`(冰)否證 —— 現在看更清楚:**火融冰**,
跨系別是設計意圖,不是例外。

## 4. `CHARS.DAT` 的語意覆蓋率跳升

把 [`81`](81-chars-record-to-combat-attributes.md) 的「位移 → 屬性」
與各屬性的語意接起來:

| 位移 | 語意 |
|---:|---|
| 2–11 | 名稱 |
| 14 | 種族碼 |
| 15 | 職業碼 |
| **16** | **速度**(→ 屬性 2,[`93`](93-initiative.md))|
| **18** | **力量**(→ 屬性 6)|
| **24** | **命中能力**(→ 屬性 9)|
| **28** | **當前生命值**(→ 屬性 3)|
| **32** | **法力點數**(→ 屬性 7)|
| 34 / 36 | 武器 / 防具的背包格號 |
| **42–51** | **十個技能旗標**(表由職業決定)|
| 54+2i | 背包第 i 格 |

**有語意的位置從 9 個增為 23 個。**

## 5. 還缺

- `CHARS.DAT` 位移 12, 17, 19–23, 25–27, 29–31, 33, 35, 37, 38, 40, 41, 45, 59…89
- 屬性 16(`Berserking`)在戰鬥裡做什麼
- **逃跑判定** —— J 只剩這一項;
  `CMBT`/`WRLDMOVE`/`MAZEMOVE` **都沒有 `flee`/`run`/`escape` 字串**
