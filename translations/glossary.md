# 統一譯名表

> **所有翻譯以本表為準。** 表上沒有的新詞,先加進來再用。
> 原文一律取自遊戲字串([`docs/re/88`](../docs/re/88-races-and-classes.md) §3:
> 專有名詞不從語意猜,只從遊戲自己的字串取)。

## 職業(`CHARS.DAT` 位移 15)

| 原文 | 譯名 | 備註 |
|---|---|---|
| `Hero` | **勇者** | ⚠ **不是**「戰士」——`Fighter` 在本作是**怪物名** |
| `Wizard` | **法師** | 玩家職業 |

## 種族(`CHARS.DAT` 位移 14)

| 碼 | 原文 | 譯名 |
|:-:|---|---|
| `H` | Human | 人類 |
| `T` | Troll | 巨魔 |
| `D` | Dwarf | 矮人 |
| `E` | Elf | 精靈 |
| `G` | Gnome | 地精 |

## 法術系別(`SPELLS.DAT` 欄2 = 狀態編號)

| 編號 | 原文 | 譯名 |
|---:|---|---|
| 1 | Fire runes | 火之符文 |
| 2 | Metal runes | 金之符文 |
| 3 | Wind runes | 風之符文 |
| 4 | Ice runes | 冰之符文 |
| 5 | Spirit runes | 靈之符文 |

## 狀態(`CHARS.DAT` 位移 38,0 起算)

| 編號 | 原文 | 譯名 | 顯示寬度上限 |
|---:|---|---|---|
| 0 | `OK` | 正常 | 原文 2 → 中文 2 字 |
| 1 | `Poisoned` | 中毒 | 8 → 2 字 |
| 2 | `Bound` | 束縛 | 5 → 2 字 |
| 3 | `Still Air` | 凝滯 | 9 → 2 字 |
| 4 | `Frozen` | 冰封 | 6 → 2 字 |
| 5 | `D E A D` | 死亡 | 7 → 2 字 |

## 技能 — `Hero` 表

| 原文 | 譯名 | | 原文 | 譯名 |
|---|---|---|---|---|
| Sword | 劍術 | | Tactics | 戰術 |
| Axe | 斧術 | | Armored skin | 硬皮 |
| Mace | 錘術 | | Berserking | 狂暴 |
| Karate | 徒手 | | Hunting | 狩獵 |
| Darkvision | 夜視 | | Persuasion | 說服 |

## 技能 — `Wizard` 表

| 原文 | 譯名 | | 原文 | 譯名 |
|---|---|---|---|---|
| Fire/Metal/Wind/Ice/Spirit runes | 見上「法術系別」 | | Potion lore | 藥劑學 |
| Weapon lore | 兵器學 | | Item lore | 器物學 |
| Monster lore | 怪物學 | | Priesthood | 聖職 |

## 介面詞

| 原文 | 譯名 | 顯示限制 |
|---|---|---|
| `Hit Pts.` / `H.P.` | 生命 / HP | 角色卡欄位 |
| `Spell Pts.` / `S.P.` | 法力 / MP | 同上 |
| `Level:` | 等級 | |
| `Experience` | 經驗 | |
| `Status` | 狀態 | |
| `Skills:` | 技能 | |
| `Provisions:` | 補給 | 隊伍層級 |
| `Gold:` | 金幣 | |
| `Visibility =` | 視野 | |
| `Hour:` / `Day:` / `In the Month` | 時 / 日 / 月 | ⚠ **單位換算未解**,不要寫「24 小時制」之類 |

## ⚠ 硬規則

1. **`Hero` ≠ 戰士。** 遊戲裡的 `Fighter` 是怪物(`Lvl 1 Fighter`),兩者必須用不同的詞。
2. **`MONSTERS.DAT` 的名稱欄是定長 16 bytes** —— 中文佔 2 bytes,所以**最多 8 個中文字**。
3. 時間單位不做換算(見 [`docs/formats/02`](../docs/formats/02-groups-dat.md))。
