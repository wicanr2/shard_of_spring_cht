# 166 — 營地的 `H)unt` / `I)dentify` / `P)rint`:三道閘門共用一個「今天用過了」旗標

日期:2026-08-15
接續:[`165-dgroup-string-map.md`](165-dgroup-string-map.md)、[`150-experience-is-offset-90.md`](150-experience-is-offset-90.md) §4.2
子系統:**K. 輸入語意** / **D. 角色與存檔**
輸入:`CAMP.EXE`、`ITEMS.DAT`(SHA-256 見 [`00-inputs.md`](00-inputs.md))、手冊 p.39

## 結論

| # | 結果 | 信心 |
|---|---|---|
| 1 | `CHARS.DAT` **位移 86 = 今天已經用過技能**(`'1'` = 用過)。`H)unt` 與 `I)dentify` 共用 | **已確認** |
| 2 | `H)unt`:要在**野外**(`ds:3534 ≥ 99`)、要是**戰士**、要有**技能旗標第 9 格**(位移 50 = `Hunting`)| **已確認** |
| 3 | 打獵成功會加**補給品**(`GROUPS.DAT` 位移 23),**夾在上限** `ds:6F10` | **已確認** |
| 4 | `I)dentify`:要是**法師**、狀態 ≤ 1、依道具編號選三個 lore 技能之一(位移 47 / 48 / 49)| **已確認** |
| 5 | 三個 lore 的分界是 **`≤ 20` / `21–56` / 其餘**,`99` = 空格 | **已確認**(讀到的常數)|
| 6 | 背包存的是 **0-based** 編號,所以第一段(`≤ 20`)= 全部 21 件武器與護甲 | **已確認**([`167`](167-record-field-accessor-and-identified-flags.md) §3)|
| 7 | 服務呼叫**操作碼 38 = 取角色狀態**;`> 1` → `That character is incapacitated.` | 證據充分 |

> ✅ **這兩道閘門後來用實跑驗過**([`174`](174-oracle-confirms-the-camp-gates.md)):
> 四句訊息逐字對上,順序也完全吻合。證據等級因此升到第 1 級。

## 1. 位移 86:每天一次的旗標

`H)unt`(`0x11299`)與 `I)dentify`(`0x10FDE`)都做同一件事:

```
MID$(角色記錄, 86, 1) == '1'  →  call 0x12DB7  →  'You have used that skill today.'
```

`0x12DB7` 那一段就是印那句話的常式([`165`](165-dgroup-string-map.md) 的字串表:
描述子 `ds:7AA6`)。四句共用的技能訊息各是一段:

| 位址 | 訊息 |
|---|---|
| `0x12D67` | `You don't have that skill.` |
| `0x12D8F` | `That character is incapacitated.` |
| `0x12DB7` | `You have used that skill today.` |
| `0x12DDF` | `That character is not a wizard.` |

手冊 p.39 兩條都寫著「每天一次」:
「有 HUNT 技能的隊員**每天可以打獵一次**」、「巫師…**每天辨別一件物品**」。
**旗標與手冊互相印證** —— 這是 §2.1 條件 3 要的那份獨立資料。

⚠ **誰把位移 86 清回 `'0'` 沒有讀到**。最可能是 `S)leep`(睡一覺 = 過一天),
但那是推測。

## 2. `H)unt`

`CAMP.EXE` `0x111F0`–`0x11375`:

```
0111F0  cmp ds:3534h, 63h            ; 99
0111F5  jl  → 'You're inside!'       ; ★ 只能在野外打獵
011222  提示 'Character # to hunt ? (ESC exits)' → ds:7072 = 角色編號(0 = 離開)
011242  記錄 = ds:34E0[編號]         ; 每個角色 4 bytes 的字串描述子
01125E  MID$(記錄, 15, 1) == '2'     ; 職業:法師?
011278  MID$(記錄, 50, 1) == '0'     ; ★ 技能旗標第 9 格 = Hunting
01128A  or / and → 兩者任一成立 → 'You don't have that skill.'
011299  MID$(記錄, 86, 1) == '1'     → 'You have used that skill today.'
0112B8  操作碼 38(取狀態);> 1 → 'That character is incapacitated.'
0112D8  ds:731C = INT(…) + ds:731E,若 < 0 夾成 0
011328  ds:731C == 0 → 'The hunt was not successful.'
        否則         → 'The hunt was successful!'
01134D  補給品 ← min(補給品 + ds:731C, ds:6F10)    ; 操作碼 23
```

技能編號對得上:`CAMP` 自己的技能表字串寫著 `9)  Hunting  (2)`,
而技能旗標在 `CHARS.DAT` **位移 41 + n**([`formats/01`](../formats/01-chars-dat.md)),
`41 + 9 = 50` ✓。

補給品是 `GROUPS.DAT` **位移 23**([`formats/02`](../formats/02-groups-dat.md)),
手冊 p.39:「如果打獵成功了,你會增加幾份**食糧**」✓。

⚠ **擲骰本身沒讀完**:`ds:731E`(加項)與 `INT 3D:33` 吃的參數都未解,
所以「成功機率多少、加幾份」不知道。讀到的是**形狀**(擲一次、夾在 ≥ 0、
0 就是失敗)與**上限存在**。

`ds:3534` 是**當前迷宮編號**,`99` = 不在任何迷宮
([`169`](169-encounter-zone-selects-the-monster.md) §4:`MAZEMOVE` 把同一個變數
與迷宮編號 1–12 比)。實跑也對上了 —— 在世界地圖進營地按 `H` 不會印
`You're inside!`([`174`](174-oracle-confirms-the-camp-gates.md) §2)。

## 3. `I)dentify`

`0x10F8F`–`0x11130`:

```
010FC9  MID$(記錄, 15, 1) == '1'  → 'That character is not a wizard.'   ; ★ 只有法師
010FE7  MID$(記錄, 86, 1) == '1'  → 'You have used that skill today.'
010FFC  操作碼 38(取狀態);> 1 → incapacitated
01101C  提示 'Item to ID ?' → ds:7294 = 道具編號
01109F  依編號選技能位移:
            編號 ≤ 20        → 位移 47(技能 6 = Weapon lore)
            21 ≤ 編號 ≤ 56   → 位移 48(技能 7 = Potion lore)
            編號 == 99       → 空格,回選單
            其餘             → 位移 49(技能 8 = Item lore)
0110E9  MID$(記錄, 那個位移, 1) == '0' → 'You are not trained in that lore!'
0111B7  之後還有一個 'Failed' 的分支 —— **擲骰未讀**
```

三個位移對得上 `CAMP` 自己的技能表:`6) Weapon lore`、`7) Potion lore`、
`8) Item lore`,而 `41 + 6/7/8 = 47/48/49` ✓。

手冊 p.39 也對得上:「WEAPON LORE 可以辨別武器和護甲,ITEM LORE 可以辨別
各項特殊物品,POTION LORE 可以辨別各種藥劑」。

### 3.1 分界落在哪:背包存的是 0-based 編號

`TOWNDATA` 的販售範圍最小值是 **0**,而「藥水舖」賣 **21–26** =
`ITEMS.DAT` 第 22–27 列,第 22 列正是 `Heal potion`
([`167`](167-record-field-accessor-and-identified-flags.md) §3)。所以:

| 編號 | `ITEMS.DAT` 的列 | 技能 |
|---|---|---|
| 0–20 | 1–21 = **全部 21 件武器與護甲** | `Weapon lore` |
| 21–56 | 22–57 = 藥水 + 任務道具 | `Potion lore` |
| > 56 | **沒有真實道具** | `Item lore` |

第一段與手冊「WEAPON LORE 可以辨別武器和護甲」完全吻合,
⚠ 但 **`Item lore` 對玩家而言是一個永遠用不到的技能**
([`167`](167-record-field-accessor-and-identified-flags.md) §4)。

## 4. `P)rint`:輸出角色表到印表機

`0x12529` 起。字串([`165`](165-dgroup-string-map.md) 的表)把版面講完了:

```
'Enter number of character you wish to print. '
'(ESC to exit, or 9 to print entire party)'
'Make sure that your printer is ready and paper is properly positioned.'

'Party #'   '    Location: '   'Wilderness'
'Level:     '  'Hit Pts.   '  'Spell Pts. '  'Experience'  'Status     '
'Skills:'  'Items:'
```

⚠ **程式碼沒有讀**,本節只是字串盤點。

**這一項對 remake 沒有影響**([`CLAUDE.md`](../../CLAUDE.md) §1.2 的判準:
答案不會改變 remake 的行為就不必解)—— 1986 年的並列埠印表機不在範圍內。
引擎的對應做法是把同一份內容**顯示在畫面上**,不是驅動印表機。

## 5. 順帶:`S)leep` 會死人

字串表裡有 `dies in the night.`(`ds:7B5E`),與 `You have slept !` /
`You are not tired` / `You sleep...` 同一區。

⚠ **條件沒讀** —— 最可能是中毒者過夜,但沒有證據。列在這裡是為了**不要漏掉**:
睡覺在本引擎目前只是「回復」,若原版會死人,那是行為差異。

## 6. 明列剩餘的不確定

| 項目 | 狀態 |
|---|---|
| 誰把位移 86 清回 `'0'` | **未解**(§1)|
| 打獵的擲骰:`INT 3D:33` 的參數、`ds:731E`、上限 `ds:6F10` | **未解**(§2)|
| ~~`ds:3534` 在別處被填什麼~~ | **已解**:當前迷宮編號([`169`](169-encounter-zone-selects-the-monster.md) §4)|
| ~~背包格是 0-based 還是 1-based~~ | **已解**:0-based([`167`](167-record-field-accessor-and-identified-flags.md) §3)|
| `I)dentify` 的 `Failed` 分支(擲骰)| **未讀** |
| `P)rint` 的程式碼 | 未讀(§4,依 §1.2 不影響 remake)|
| `S)leep` 的 `dies in the night.` 條件 | **未解**(§5)|
