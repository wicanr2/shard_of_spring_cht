# 營地的三個未實作指令 + 施法的投入點數 — **READY**

對應 [`14-remake-worklist.md`](14-remake-worklist.md) 的 **E1 / E2 / E3 / E5**。
**規則全部已解**,這一份只講「接到畫面上」。

## 0. 為什麼這四項可以直接做

它們不需要新的 RE 結論 —— 規則層已經實作而且有測試:

| 項目 | 規則在哪 | 缺什麼 |
|---|---|---|
| E1 營地 `C)ast spell` | `internal/magic`([`09`](09-magic-items.md))| 營地的入口與選單 |
| E2 營地 `U)se an item` | `internal/magic` 的魔法道具([`09`](09-magic-items.md) §5)| 同上 |
| E3 `P)rint char(s)` | 版面字串已盤出([`re/166`](../re/166-camp-hunt-identify-print.md) §4)| 改成畫面顯示 |
| E5 施法投入點數 | 公式已解([`09`](09-magic-items.md) §2)| **玩家輸入**那一步 |

⛔ **`E4 商店買給誰`不在這一份裡。** 原版的商店介面**沒有讀到選人的步驟**
([`11`](11-town-camp-roster.md) §3),所以那是 RE 問題不是實作問題。
現在固定給第一位並在畫面上標明,**維持原樣**。

## 1. E5 投入點數:先做這一項

現在施法**固定投一級**。原版讓玩家輸入 —— `CAMP.EXE` 的字串是
`'Spell Pts ?'`([`re/165`](../re/165-dgroup-string-map.md) 的字串表)。

```
選法術 → 提示 'Spell Pts ?' → 玩家輸入數字 → 套 09 §2 的三條公式
```

| 檢查 | 訊息 |
|---|---|
| 投入 > 當前 SP | 法力不足 |
| `INT(投入 ÷ 欄5) < 1` | 投不到一級 → 施法失敗 |

⚠ **三條公式各吃不同的東西**([`09`](09-magic-items.md) §2、§4),別混:

```
等級     = INT(投入 ÷ 欄5)
威力     = 欄4 × 投入          ← 乘的是**投入**,不是等級
狀態強度 = 欄5 ÷ 投入          ← 投得多、值**小**
```

**這一項是 E1 的前置** —— 營地施法沒有投入點數的話,做出來的是半套。

## 2. E1 營地 `C)ast spell`

原版的字串把流程講完了([`re/165`](../re/165-dgroup-string-map.md)):

```
'Character # to cast spell'  → 選施法者
'Cast what spell?'           → 選法術(巫師才有;'(ENTER exits)')
'Spell Pts ?'                → 投入點數(§1)
'That is a combat spell !'   → ⚠ 戰鬥法術不能在營地放
'You don't know that spell'  → 沒學那一系
'Mumble, mumble, what spell' → 輸入不合法
```

**⚠ `That is a combat spell !` 是這一段的關鍵**:營地能放的法術是**子集**。
判斷依據走 [`09`](09-magic-items.md) 的效果類別 —— 需要戰場目標的類別
(傷害類、需要選格的)在營地放不出來。

⚠ **哪些類別算「戰鬥法術」沒有直接讀到。** 實作時用一張**具名的表**,
在程式碼裡註明「這張表是從效果類別推的,不是讀到的判斷式」,
⛔ 不要寫成看起來像 RE 結論的樣子。

目標選擇:營地沒有戰場,所以目標是**隊員**(治療、增益)。
沿用 `internal/magic` 現有的套用函式,不要另寫一套。

## 3. E2 營地 `U)se an item`

```
選人 → 列出他背包裡的道具 → 選一件 → 套用
```

⚠ **未辨識的道具照樣可以用**(原版沒有擋),但顯示成未辨識的名稱
([`11`](11-town-camp-roster.md):`CHARS.DAT` 位移 74–83 的旗標)。

魔法道具的發動走 [`09`](09-magic-items.md) §5:編號 ≤ 26 不走這條路;
`擲骰(100) ≤ 欄6`。

## 4. E3 `P)rint char(s)` → 顯示在畫面上

原版驅動並列埠印表機。`CLAUDE.md` §1.2 的邊界:**不做印表機**,
改成把同一份內容顯示出來。版面照原版的字串([`re/166`](../re/166-camp-hunt-identify-print.md) §4):

```
'Enter number of character you wish to print. '
'(ESC to exit, or 9 to print entire party)'      ← 9 = 整隊

Party #      Location:   Wilderness
Level:       Hit Pts.    Spell Pts.   Experience   Status
Skills:
Items:
```

⚠ **`9` 是「整隊」不是「第 9 個人」。** 隊伍最多 5 人,所以 9 不會撞號。

⛔ **不要印 `Make sure that your printer is ready…`** —— 那句話在 remake 裡是假的。

## 5. 邊界

⛔ **不要動的東西**:

- `engine/main.go` —— 那是外殼的工作([`15`](15-game-shell.md)),同時被改會衝突。
  **需要在 `main.go` 加 hook 的話,不要自己改,在回報裡列出需要的那幾行。**
- `engine/internal/` 底下的**規則**(`magic` / `combat` / `rules`)——
  規則已解且有測試,這一份只接線。⚠ 少數新增的輔助函式可以加在 `internal/town`,
  但**不要改既有函式的簽章**。
- `docs/`、`game/`、`original/`、`workplace/`
- `translations/` —— 新字串先寫成中文常數,譯名照
  [`glossary.md`](../../translations/glossary.md);**不要動 TSV**

## 6. 驗收

| # | 條件 |
|---|---|
| 1 | 施法可以輸入投入點數;投超過 SP、投不到一級,各有一個測試 |
| 2 | 威力乘的是**投入**、強度是**欄5 ÷ 投入** —— 有測試能分辨寫反 |
| 3 | 營地施法:戰士被擋、沒學那一系被擋、戰鬥法術被擋,三句訊息各不同 |
| 4 | 營地用道具:未辨識的也用得出來,名稱顯示成未辨識 |
| 5 | `P)rint` 顯示單人與整隊(`9`),欄位齊全且**不提印表機** |
| 6 | 每一條都有**不開視窗**的測試 |
