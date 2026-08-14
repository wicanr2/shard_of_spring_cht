# `CHARS.DAT` — 角色記錄 — **READY**

25 筆 × **94 bytes**,定長隨機存取。記錄長度從 `USERLIB` 存檔常式的
`OPEN … mov cx, 5Eh` 讀出;`2350 ÷ 94 = 25` 整除。

> 位移一律 **1-based**(BASIC `MID$` 慣例)。2-byte 整數用 `CVI`,
> **全部落在偶數位移**(判別法見 [`re/99`](../re/99-parity-separates-the-two-records.md))。

| 位移 | 大小 | 型別 | 語意 | 信心 |
|---:|---:|---|---|---|
| 1 | 1 | 字元 | **所屬隊伍編號**(`'1'`–`'5'`);`'*'` = 無隊伍／空槽 | 證據充分 |
| 2–11 | 10 | 字元 | 角色名稱(空白補齊)| 已確認 |
| 12 | 2 | 整數 | 角色編號(1–25)| 已確認 |
| 14 | 1 | 字元 | 種族:`H`uman `T`roll `D`warf `E`lf `G`nome | 已確認 |
| 15 | 1 | 字元 | 職業:`1` = **Hero**、`2` = **Wizard** | 已確認 |
| 16 | 2 | 整數 | 速度 | 已確認 |
| 18 | 2 | 整數 | 力量 | 已確認 |
| 20 | 2 | 整數 | 智力(`Wizard` 的法力來源)| 證據充分 |
| 22 | 2 | 整數 | 體質(生命值來源)| 證據充分 |
| 24 | 2 | 整數 | 命中能力 | 已確認 |
| 26 / 28 | 2 | 整數 | 最大 / 當前生命值 | 已確認 |
| 30 / 32 | 2 | 整數 | 最大 / 當前法力 | 已確認 |
| 34 / 36 | 2 | 整數 | 裝備武器 / 防具的**背包格號**(`99` = 未裝備)| 已確認 |
| 38 | 2 | 整數 | 狀態(= 法術系別編號,見 [`03`](03-monsters-dat.md))| 已確認 |
| 40 | 2 | 整數 | 等級 | 已確認 |
| 42–51 | 10 | 字元 | 十個技能旗標(`'0'`/`'1'`),**表由職業決定** | 已確認 |
| 54 + 2i | 2 | 整數 | 背包第 i 格的物品編號 | 已確認 |
| 84 | 2 | 整數 | 狀態效果強度 | 證據充分 |

## 技能表(位移 42–51,技能 n → 位移 41+n)

| n | `Hero` | `Wizard` |
|---:|---|---|
| 1–5 | Sword / Axe / Mace / Karate / Darkvision | **Fire / Metal / Wind / Ice / Spirit runes** |
| 6 | Tactics | Weapon lore |
| 7 | **Armored skin**(進傷害公式的減項)| Potion lore |
| 8 | **Berserking** | Item lore |
| 9 | Hunting | Monster lore |
| 10 | Persuasion | **Priesthood**(Dispell 的前提)|

⚠ 同一格在兩張表裡是**不同的技能** —— 讀之前必須先看職業碼。

## 未解

位移 84 的除法運算元順序未逐位元確認。
位移 1 的 `'*'` 究竟是「無隊伍」還是「空的角色槽」——
出貨資料兩者重合,分不開([`re/133`](../re/133-chars-offset-1-is-party.md) §3)。

出處:[`re/75`](../re/75-character-record-equipment.md)、[`80`](../re/80-save-write-end.md)、
[`81`](../re/81-chars-record-to-combat-attributes.md)、[`87`](../re/87-class-race-and-attribute-17.md)、
[`88`](../re/88-races-and-classes.md)、[`94`](../re/94-skill-tables.md)、
[`98`](../re/98-status-level-and-a-sign-error.md)–[`100`](../re/100-chars-attributes-closed.md)、
[`111`](../re/111-status-code-equals-school.md)、[`121`](../re/121-offset-84-is-strength.md)
