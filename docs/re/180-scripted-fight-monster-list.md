# 180 — 腳本戰鬥的怪物清單:`ds:372C` 起 8 槽,哨兵 99;`CMBT` 讀到就不隨機挑

日期:2026-08-15
接續:[`169-encounter-zone-selects-the-monster.md`](169-encounter-zone-selects-the-monster.md)、[`161-maze-event-dispatch.md`](161-maze-event-dispatch.md) §4、[`179-final-battle-composition-from-playthrough.md`](179-final-battle-composition-from-playthrough.md)
子系統:**J. 戰鬥規則** / **G. 地城與迷宮**
輸入:`CMBT.EXE`、`MAZEMOVE.EXE`、`MONSTERS.DAT`(SHA-256 見 [`00-inputs.md`](00-inputs.md))

## 結論

| # | 結果 | 信心 |
|---|---|---|
| 1 | `CMBT 0x110CB` 是**旁路分支**:`ds:372C < 99` → 用腳本清單;`≥ 99` → 走 [`169`](169-encounter-zone-selects-the-monster.md) 的隨機挑怪迴圈 | **已確認** |
| 2 | 腳本清單 = `ds:372C + 2i`(i = 0…7),**哨兵 99**;隻數寫進 `ds:943A`,**遇到第一個 ≥ 99 就停**,不是掃完八格 | **已確認** |
| 3 | 目標 **204** = 1 × `Hill Giant`(`MONSTERS.DAT` 0-based 第 10 列)| **已確認** |
| 4 | 目標 **533** = 2 × `Great Dragon`(0-based 53)+ 1 × `Siriadne !`(0-based 71)| **已確認** |
| 5 | `MAZEMOVE` 對這個陣列有 **15 處寫入點**(13 處直接立即數 + `sub_13112` 一支共用子程式,24 個呼叫端)| 證據充分 |

第 3/4 項的證據等級是**兩條獨立鏈**合起來的結果 —— 見 §4。

## 1. `CMBT` 的旁路分支

```
0110CB  cmp word ptr ds:372Ch, 63h     ; 99
0110D0  jl  → 0110D5                   ; < 99 → 腳本清單
0110D2  jmp → 011109                   ; ≥ 99 → re/169 的 RND 挑怪迴圈
0110D5  mov word ptr ds:943Ah, 1       ; 第 0 槽先算一隻
0110E2…011106  i = 1…7:[372Ch + 2i] < 99 → ds:943A++
               遇到第一個 ≥ 99 **立刻跳出**
011302  兩條路徑的收斂點
```

**`ds:943A` 是「這場有幾隻怪」** —— 這一格不是本輪新解的:
[`152`](152-experience-settlement-formula.md) §2.1 讀金幣迴圈時就寫著
`ds:96AE = ds:943A − 1 ; 上界 = 怪物數 − 1`。

> **這是本輪最強的一條印證,而且是白撿的**:旁路分支算完隻數之後寫的那一格,
> 正好是另一輪從**完全不同的功能**(戰後金幣結算)獨立認出來的「怪物數」。
> 兩處對得上,表示「這個分支在決定這場戰鬥有幾隻怪」不是猜的。

⚠ **哨兵是 99,與背包空格、`LightPick`「沒有光源」同一個值** ——
這款遊戲到處用 99 當「沒有」。⛔ 不要在任何一處把它當成 0。

## 2. `MAZEMOVE` 的寫入端

十五處。兩種形狀:

**直接寫立即數**(13 處,`0x11A02` … `0x12BE4`):

```
012BE4  mov word ptr ds:372Ch, 35h     ; 53
012BEA  mov word ptr ds:372Eh, 35h     ; 53
012BF0  mov word ptr ds:3730h, 47h     ; 71
```

**經共用子程式 `sub_13112`**(24 個呼叫端),呼叫端先把值放進一個暫存格,
再 `push` 那一格的位址與哨兵:

```
0121B2  mov word ptr ds:9038h, 0Ah     ; 10
0121C7  call sub_13112                 ; push &9038 / push &903A(=99)/ push cs
013153  mov di, [bp+8] / mov ax, [di] / mov ds:372Ch, ax
```

## 3. 兩個目標的組成

| 目標 | 寫入 | `MONSTERS.DAT`(0-based)| 難度階級 |
|---|---|---|---:|
| **204** | `372C = 10` | 第 10 列 `Hill Giant` | 3 |
| **533** | `372C = 53`、`372E = 53`、`3730 = 71` | 第 53 列 `Great Dragon` ×2、第 71 列 `Siriadne !` | 10 / 10 / 13 |

## 4. 兩條獨立證據鏈得到同一個答案

[`179`](179-final-battle-composition-from-playthrough.md) 是先寫的,來源是
**DOS 版通關紀錄 + `MONSTERS.DAT` 的階級表**(第 3 級證據),結論是
「Siriadne + 兩隻 Great Dragon」。本篇是反組譯(第 2 級),結論相同。

**兩條鏈沒有共用任何前提**:一條看的是玩家在螢幕上遇到什麼,
一條看的是 `MAZEMOVE` 往 `ds:372C` 寫了哪三個數字。
所以 [`CLAUDE.md`](../../CLAUDE.md) §2.1 條件 3 要的「一份獨立資料互相印證」成立,
第 3/4 項因此標**已確認**。

⚠ **那一輪刻意沒有把 [`179`](179-final-battle-composition-from-playthrough.md) 的答案
告訴做反組譯的人。** 先告訴他就會變成「去找我要的答案」,
而兩條鏈的獨立性一旦破壞就補不回來 —— 那是這個結論能標「已確認」的唯一理由。

### 4.1 第三條:階級對得上區域規則

`Great Dragon` 階級 10、`Siriadne !` 階級 13,而 [`169`](169-encounter-zone-selects-the-monster.md)
的規則是 `|階級 − 區域| ≤ 1`:通關紀錄說**門外會隨機遇到六隻 greater dragons**
→ 那一帶的區域必然是 9–11 → **同一片地城裡 `Great Dragon` 遇得到、`Siriadne !` 遇不到**。
這正是「她只能由腳本放上場」的直接後果,而現在腳本被讀出來了。

## 5. ⚠ 目標編號與處理常式的連結是**推的**

`0x12BE4` 那三行**沒有任何 `cmp 533`** 可以指。理由是結構性的:
BASIC 的執行期行號派工不會留下靜態 xref([`42`](42-module-ds-and-the-66dc-boundary.md) §3、
[`161`](161-maze-event-dispatch.md) §3.2 —— 「找不到呼叫端」在資料驅動的派工裡不是有效問題)。

連結靠三件事:

1. 數值精確對上 `MONSTERS.DAT`,而 `Siriadne !` 與 `Hill Giant` 在全表**各只出現一次**
2. 位置**緊接在目標 532**(寶石謎題,已知錨點 `0x129EA`)之後
3. 組成與 [`179`](179-final-battle-composition-from-playthrough.md) 的通關紀錄一致

⚠ **這一條單獨看只到「證據充分」。** 標成已確認的是**組成**(§4 兩條鏈),
不是「`0x12BE4` 這個位址就是 533 的處理常式」。⛔ 之後若發現某個別的目標
也召兩龍一 Siriadne,要回頭看這一節。

## 6. 明列剩餘的不確定

| 項目 | 狀態 |
|---|---|
| 打完之後誰把 `ds:372C` 重置回 99 | **未解** —— 沒找到重置點。理論上殘留的舊值會讓下一場多出怪物,**但沒有反例也沒有實跑驗證** |
| 三隻的**站位**與初始朝向 | **未解**([`164`](164-board-stride-31-and-base-13.md) §3:怪物是擲座標找空格,兩個範圍未解)|
| `Siriadne !` 的屬性有沒有被腳本加成 | **未解** —— `MONSTERS.DAT` 第 71 列是基準值 |
| 其餘 13 處寫入點各對應哪個事件 | **未盤** —— 本輪只解了 204 與 533 |
| 打贏 533 之後除了播結局還做了什麼 | **未解**([`161`](161-maze-event-dispatch.md) §4 只讀到字串)|

⚠ `sub_13AC9` 寫的 `ds:352C` **與怪物陣列無關**(204/533 都以哨兵 99 呼叫它,
`CMBT` 在 `0x1328C` 重置回 99)—— 已排除「那是第四隻怪」的可能。
