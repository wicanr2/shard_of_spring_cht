# 161 — 地城事件的派工:機關的觸發點就是事件目標編號本身

日期:2026-08-15
接續:[`60-event-lookup-and-tile-19.md`](60-event-lookup-and-tile-19.md)、[`155-gem-puzzle-and-healing-pool.md`](155-gem-puzzle-and-healing-pool.md)
子系統:**G. 地城與迷宮**
輸入:`MAZEMOVE.EXE`、`DE*EFF.BIN` ×10、`DT*TEXT.DAT` ×9(SHA-256 見 [`00-inputs.md`](00-inputs.md))

## 結論

| # | 結果 | 信心 |
|---|---|---|
| 1 | **每個事件目標編號各有一段自己的程式**;絕大多數只印文字,五個編號另外做事 | **已確認** |
| 2 | **治療池 = 目標 518**(`DE5EFF` 格 54/42、朝向 4)、**寶石謎題 = 目標 532**(`DE51EFF` 格 30/42、朝向 3)| **已確認** |
| 3 | **Eldron 的氏族謎題 = 目標 705**(`DE7EFF` 格 24/33、朝向 3)—— 見 [`162`](162-eldron-clan-riddle.md) | **已確認** |
| 4 | 派工器另外特判**兩個**編號:**204**(山丘巨人挾持祭司)與 **533**(最終首領 Siriadne)| **已確認** |
| 5 | 命中的事件**會被就地作廢**(欄 0/1/2 覆寫成 99),但前提是一個未解的字串比較 | 已確認(寫入)／**未解**(前提)|
| 6 | `ds:66C8`/`66CA` 是**跨模組服務呼叫的引數槽**(字串 + 操作碼),不是顯示參數 | 證據充分 |
| 7 | ⚠ **訂正**:`DE51EFF` 對到的是 **`DT51TEXT.DAT`**,不是 `DT5TEXT.DAT` | **已確認** |

第 2、3 項把 [`spec/08`](../spec/08-maze-scene.md) §5.5 的「觸發點未解」關掉。

## 1. 派工器

`ds:3532` 是**待處理事件的目標編號**。派工器在 `MAZEMOVE` `0x11563`–`0x115F4`:

```
011563  mov ds:3532h, ax             ; ★ 事件目標 ← 前一支常式的回傳值
011566  or ax, ax / bx = (ax > 0)    ; 條件 A:目標為正
011570  bx = 8E7Ch / ax = 3798h
011576  INT 3F:62  字串比較          ; 條件 B:某字串相等(**未解**)
011581  and cx, dx / and cx, cx
011585  jnz 01158A                   ;  A ∧ B → 進作廢迴圈
011587  jmp 0115EE                   ;  否則直接跳到收尾

01158A  ax = 0                       ; ── 作廢迴圈 i = 0…105 ──
011590  ax = i + 013Eh               ; 318 = 3 × 106 → **欄 3(目標)**
011597  bx = [di + 88F0h]            ; 事件表在 ds:88F0,欄主序
01159B  cmp bx, ds:3532h
01159F  jz  0115A4                   ;   目標相同 → 作廢這一列
0115A4  push ax(=&i) / INT 3E:83 / call 013EC7
0115B8  i = i + 1 / cmp i, 105 / jle 011590

0115C4  cmp ds:3532h, 204  → jz  → call 01316F   ; ★ 特例一
0115D9  cmp ds:3532h, 533  → jz  → call 0131D0   ; ★ 特例二
0115EE  ds:3532 ← 0                  ; 清掉待處理事件
```

## 2. 作廢:把座標覆寫成 99

`0x13EC7`:

```
013ED4  mov word ptr [di+88F0h], 99      ; 欄 0 = Major
013EDC  bx = 列 + 106 → [di+88F0h] = 99  ; 欄 1 = Minor
013EE9  dx = 列 + 212 → [di+88F0h] = 99  ; 欄 2 = Dir
```

迷宮的 Major 值域是 0…80([`57`](57-g-closure.md) 的 81 寬),**99 落在值域外** ——
之後掃表永遠不會再命中這一列。

⚠ 迴圈**不提早跳出**:它把**所有**目標相同的列一起作廢。
⚠ **不要照著實作**:它的前提是 `0x11576` 那個未解的字串比較(§1 條件 B)。
不知道那個條件何時成立,就不知道哪些事件是一次性的。

## 3. 每個目標編號各有一段程式

`MAZEMOVE` 裡有近五十處形狀完全相同的三條指令:

```
mov ax, ds:3532h / mov ds:8DD0h, ax / call <顯示>
```

**它們是各自獨立的常式**(每一段都以 `ret` 結束),不是一條共用路徑 ——
也就是「目標編號 N」對應「第 N 段程式」,預設行為是把 N 當文字編號印出來。
五個編號的那一段多做了事:

| 目標 | 位置 | 多做的事 |
|---|---|---|
| **518** | `DE5EFF` 格 54/42、朝向 4 | 治療池([`155`](155-gem-puzzle-and-healing-pool.md) §2)|
| **532** | `DE51EFF` 格 30/42、朝向 3 | 寶石謎題([`155`](155-gem-puzzle-and-healing-pool.md) §1)|
| **705** | `DE7EFF` 格 24/33、朝向 3 | Eldron 的氏族謎題([`162`](162-eldron-clan-riddle.md))|
| **204** | `DE2EFF` 格 57/14、朝向 1 | 山丘巨人挾持祭司(§4)|
| **533** | `DE51EFF` 格 36/38、朝向 3 | 最終首領 Siriadne(§4)|

### 3.1 怎麼確定的:兩條互相獨立的證據

**一、程式裡寫死的訊息編號。** 掃 `MAZEMOVE` 全檔的 `mov word ptr ds:8DD0h, imm`,
只有八處:

```
0x129EA → 532      0x12AED → 544      0x12B6B → 545        ← 寶石謎題(提示／答對／答錯)
0x12D47 → 705      0x12DF3 → 706      0x12D90 → 707
0x13006 → 708      0x13087 → 709                            ← Eldron 謎題
```

**二、DT 文字的內容。** 那些編號的文字直接說明了機關是什麼:

| 編號 | 檔案 | 文字 |
|---|---|---|
| 518 | `DT5TEXT.DAT` | A pool of water that eminates power is here. |
| 532 | `DT51TEXT.DAT` | There are four many-faceted gems implanted in the wall here. … |
| 705 | `DT7TEXT.DAT` | 'Why have you returned? …' |

治療池是這五個裡**唯一沒有寫死訊息編號**的([`155`](155-gem-puzzle-and-healing-pool.md)
讀到的訊息是 `MAZEMOVE` 內的字串常數 `This pool is empty!` 之類,不走 DT),
所以它的信心來自「規則吻合 + 文字語意 + 事件表裡有 518」三者對得起來。

⚠ `DT5TEXT.DAT` 的 **503** 也寫到池子(「池中央有一尊銀色雕像」)。
兩者的差別是 518 明寫 *eminates power*,而 [`155`](155-gem-puzzle-and-healing-pool.md)
讀到的常式是治療 —— **518 是比 503 好的解釋,但這一步是語意判斷,不是讀程式讀到的。**

### 3.2 為什麼靜態找不到「誰呼叫這些常式」

掃過整份 `MAZEMOVE` 的 `E8`(near call)、`E9`/`EB`(jmp)、以及**全檔的 word**,
`0x129EA`(寶石)、`0x1368F`(治療池)、`0x12AD6`、`0x12729` 都是**零命中** ——
沒有任何直接轉移,也沒有任何跳表存著它們的段內位移。

**這個零不是「沒人用」**:同一支掃描在 `0x1316F` 上找得到呼叫端(`0x115D5`),
所以掃描本身是有效的(正對照)。差別在於這些區段是靠 BASIC 執行期的
行號機制進入的,而那張表是載入時才建的([`42`](42-module-ds-and-the-66dc-boundary.md) §3)。

**所以不必再找呼叫端** —— 對應關係由**目標編號**給,不由控制流給。

## 4. 兩個特例:204 與 533

這兩個編號在全部十份 `DE*EFF.BIN` 裡**各只出現一次**,而 DT 文字說明了是什麼:

> **204**(`DT2TEXT.DAT`)—— 'Thank the Gods you have come to rescue me!', cries a
> ragged old man wearing a holy symbol around his neck. 'Too bad he'll never get
> out alive!', says a hill giant with a toothless grin.
>
> **533**(`DT51TEXT.DAT`)—— With a sweeping motion of her huge wings, Siriadne
> summons two ancient dragons from the open sky. …

`MAZEMOVE` 的字串常數區裡有對應的**後續**:

```
0x14A48  'The priest thanks you for freeing him from his giant captor and blesses the party.'
0x14200–0x145B7  結局文字八段(「You have won! …」到「Evil is personified in the form of dragons…」)
```

所以 204 是「打贏巨人 → 祭司祝福」、533 是「打贏 Siriadne → 結局」。
兩支處理常式(`0x1316F` / `0x131D0`)的形狀也對得上:一連串
`INT 3F:61 字串指派`,中間經過 `INT 3D:00`(進 `USERLIB`,見
[`64`](64-userlib-call-mechanism.md)),而字串常數區裡就放著模組名
`CMBT` / `DEAD` / `WRLDMOVE` / `CAMP`。

⚠ **沒有讀到「填了哪幾隻怪物」,也沒有讀到祝福的實際效果。**
第 4 項的信心來自文字,不是來自那幾條指派。

## 5. `ds:66C8` / `66CA` 是服務呼叫的引數槽

[`42`](42-module-ds-and-the-66dc-boundary.md) §4 把 `0x66C8`–`0x681A`
(七支模組共用)列為「高價值但未解」。本輪讀到它的用法:**成對出現的
(字串, 操作碼)**,後面接一個 call:

| 位址 | 字串槽 | 操作碼槽 | 值 |
|---|---|---|---|
| `0x1155A` | `66CC` | `66D0` | 85 |
| `0x115F4` | `66C8` | `66CA` | 85 |
| `0x131AC` | `66C4` | `66CA` | 69 |
| `0x12D33` | `66CC` | `66D0` | **71** |
| `0x12FF3` | — | `66CA` | **71** |

操作碼 71 的那兩處一讀一寫**同一個旗標** `ds:91E4`(見
[`162`](162-eldron-clan-riddle.md) §2),所以 71 是「取／存一個具名的持久狀態」。
85 與 69 的語意未解。

⚠ 這推翻了本文初稿寫的「`ds:66C8 ← 目標` 是顯示參數」。它是**引數槽**。

## 6. ⚠ 訂正:`DE51EFF` 對到 `DT51TEXT.DAT`

分析途中我把 `DE51EFF.BIN` 對到 `DT5TEXT.DAT`(以為 DG5 / DG51 是同一座地城的兩半,
共用一份文字),於是得到一份「查不到文字的目標」清單長達 14 筆,
還差點據此推論「5xx 有一整批是硬寫的腳本」。

`ls game/sharspri/` 一眼就看得到 **`DT51TEXT.DAT` 是存在的**。
用正確對應重掃之後,查不到文字的只剩三處:

| 檔案 | 查不到文字的目標 |
|---|---|
| `DE4EFF.BIN` | 403、404、405、406 |
| `DE5EFF.BIN` | 501 |
| `DE51EFF.BIN` | 502 |

`501` 在 DG5、`502` 在 DG51 —— **一對互指**,形狀就是兩座地城之間的通道。
`403`–`406` 仍未解。

> **判準**:對應關係要**去檔案系統列出來看**,不要從命名規則推。
> 我推的規則(「51 是 5 的一半 → 共用 5 的文字」)本身講得通,
> 而它讓一個現成的檔案在整輪分析裡不存在。

## 7. 明列剩餘的不確定

| 項目 | 狀態 |
|---|---|
| `0x11576` 的字串比較(作廢迴圈的前提)| **未解** —— 兩個運算元都在執行期 DGROUP |
| 作廢是否跨存檔保留 | **未解** |
| 204 / 533 填了哪幾隻怪物、祝福的效果 | 未讀(§4)|
| 操作碼 85 / 69 的語意 | 未解(§5)|
| `403`–`406` 的目標是什麼 | 未解 |
| 治療池那一段的擲骰面數 `ds:950A` | 未解([`155`](155-gem-puzzle-and-healing-pool.md))|
