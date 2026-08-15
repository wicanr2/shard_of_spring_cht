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
| 6 | ⚠ 哪些道具落在哪一段,取決於背包存的是 0-based 還是 1-based 編號 —— **未解**(§4.1)| **未解** |
| 7 | 服務呼叫**操作碼 38 = 取角色狀態**;`> 1` → `That character is incapacitated.` | 證據充分 |

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

⚠ `ds:3534 ≥ 99` 讀成「在野外」是**從語意推的**(那一支印 `You're inside!`),
沒有讀到 `ds:3534` 在別處被填什麼。

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

### 3.1 ⚠ 分界的兩個讀法都各壞一半

`ITEMS.DAT` 有 **57 列**。背包格存的是 0-based 還是 1-based 編號**沒有讀到**,
而兩種讀法各自撞到一個問題:

| 讀法 | 第一段(Weapon lore)| 第三段(Item lore)|
|---|---|---|
| **0-based**(編號 0–56 = 第 1–57 列)| 第 1–21 列 = **全部武器與護甲** ✓ 與手冊一致 | 空的 —— 沒有任何道具落進來 ✗ |
| **1-based**(編號 1–57)| 第 1–20 列 —— **少了 `Plate +2`**,一件護甲要用 Potion lore ✗ | 第 57 列 `Teleporter, paper dove` = 特殊物品 ✓ |

**兩邊都有一個乾淨的段和一個壞掉的段**,所以不裁決。
要裁決得讀「背包格怎麼填」那一段(`E)quip` 或商店的購買路徑),
或用 DOSBox 拿一件 `Plate +2` 去 ID 一次。

⚠ 這正是 [`159`](159-initiative-resorts-every-round.md) §2「死分支不該存在」
**不適用**的場合:第三段在 0-based 下雖然接不到真道具,
但它接得到保留編號(`59`/`60` = 無防具/無武器,[`formats/04`](../formats/04-spells-items-dat.md)),
所以不是死碼,只是防禦性的。**判準要看它有沒有別的可達路徑。**

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
| `ds:3534` 在別處被填什麼 | 未讀(§2)|
| 背包格是 0-based 還是 1-based | **未解 —— 擋住 §3.1 的分界** |
| `I)dentify` 的 `Failed` 分支(擲骰)| **未讀** |
| `P)rint` 的程式碼 | 未讀(§4,依 §1.2 不影響 remake)|
| `S)leep` 的 `dies in the night.` 條件 | **未解**(§5)|
