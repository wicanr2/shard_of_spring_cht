# 155 — 迷宮的寶石謎題與治療池,以及 `GROUPS.DAT` 位移 63

日期:2026-08-15
接續:[`147-leaving-the-maze.md`](147-leaving-the-maze.md)、[`71-k-closure.md`](71-k-closure.md)
子系統:**G. 地城與迷宮**、**D. 存檔**
輸入:`MAZEMOVE.EXE`(SHA-256 見 [`00-inputs.md`](00-inputs.md))

## 結論

| # | 結果 | 信心 |
|---|---|---|
| 1 | 寶石謎題的答案是 **`BBRG`**,答對 **傳送到迷宮座標(欄 46、列 37)、面向西** | **已確認** |
| 2 | 治療池:**`GROUPS.DAT` 位移 63 是已使用次數**,`< 11` 才能用 | **已確認** |
| 3 | 治療池**治不了狀態 > 2 的人**(凝滯 / 冰封 / 死亡)| 已確認 |
| 4 | 治療量 = `INT(RND × N) + M`,**夾在最大生命值** | 已確認(N/M 未解)|

[`147`](147-leaving-the-maze.md) §4 把這兩項列為「新注意到的機制,規則未解」——
現在有規則了。

## 1. 寶石謎題

`MAZEMOVE.EXE` `0x12AD6`:

```
012AD6  mov bx, 8F48h / INT 3F:B8      ; 收玩家輸入的四個字母
012AE0  mov bx, 91A4h                  ; 'BBRG'
012AE3  INT 3F:62                      ; 字串比較
012AE8  jz 012AED                      ; 相符
012AEA  jmp 012B6B                     ; 不符
```

**收滿四個字元才比一次**([`71`](71-k-closure.md) §4),所以
`B`/`G`/`V`/`R` 不會出現在單字元的按鍵鏈裡 —— 那不是缺口。

### 1.1 答對的效果是傳送

```
012AED  mov word ptr ds:8DD0h, 220h    ; 訊息編號 544
012AFE  … 印一段文字 …
012B29  mov word ptr ds:351Eh, 25h     ; 37
012B2F  mov word ptr ds:351Ch, 2Eh     ; 46
012B35  mov word ptr ds:3520h, 04h     ; 朝向 4 = 西
```

`ds:351C` / `ds:351E` / `ds:3520` 就是存檔的**位移 79 / 81 / 41**
([`80`](80-save-write-end.md) §2 的欄位表),也就是**迷宮座標與朝向**。

> **答對寶石謎題 = 被送到迷宮的另一個地方。** 不是拿道具、不是開門。

答錯走 `0x12B6B`,訊息編號 **545**(`0x221`),座標不動。

⚠ 兩個訊息編號 544 / 545 指向的文字**沒有讀出來** ——
它們走 `DT*TEXT.DAT` 的編號查表([`formats/06`](../formats/06-maze.md)),
而是哪一支地城的文字檔取決於當前迷宮。

## 2. 治療池

### 2.1 用完的判斷在存檔裡

```
01368F  mov bx, 34DCh                  ; 隊伍記錄字串
01369A  mov word ptr ds:66D0h, 3Fh     ; 位移 63
0136A0  call …
0136A3  mov ds:948Eh, ax
0136A6  cmp word ptr ds:948Eh, 0Bh     ; 11
0136AB  jl  0136B0                     ; < 11 → 可以用
0136AD  jmp 013900                     ; 否則 'This pool is empty!'
```

**`GROUPS.DAT` 位移 63 = 治療池的已使用次數,上限 11。**
它是奇數位移,與 [`99`](99-parity-separates-the-two-records.md) 的
「`GROUPS.DAT` 的整數欄位全在奇數位移」一致。

⚠ 這是**跨迷宮共用的一個計數器**,不是每個池各自一個 ——
記錄裡只有這一個欄位,而它存在隊伍記錄而非迷宮資料裡。

`013900` 那段印完 `This pool is empty!` 之後 `mov word ptr ds:3532h, 0`。

### 2.2 選人與資格

```
0136B0  印 'Which party member do you wish to heal? (0 exits) '
0136DD  讀數字 → ds:94CE
0136EF  or ax, ax / jnz → 繼續;0 → 離開
0136F6  1 ≤ 編號 ≤ ds:34F8(隊伍人數),否則重問
013714  記錄 = ds:34E0 + 4 × 編號       ; 角色記錄字串陣列
013728  mov word ptr ds:66D0h, 26h     ; 位移 38 = 狀態
013734  cmp ax, 2 / jg → 印 "That character can't be helped here !"
```

**狀態 > 2 治不了。** 對照狀態表(`OK / Poisoned / Bound / Still Air / Frozen / D E A D`):

| 狀態 | 可否 |
|---|---|
| 0 正常、1 中毒、2 束縛 | ✅ 可以 |
| 3 凝滯、4 冰封、5 死亡 | ❌ `That character can't be helped here !` |

⚠ **它治的是生命值,不是狀態** —— 中毒的人治完仍然中毒(§2.3 只寫位移 28)。

### 2.3 治療量

```
013773  INT 3D:34                      ; RND
013778  mov di, 950Ah / INT 3F:91      ; × N
013780  mov di, 8FC0h / INT 3F:81      ; + M
013788  INT 3F:77 → ds:9504            ; 治療量
0137A6  mov word ptr ds:66D0h, 1Ch     ; 位移 28 = 當前生命值 → ds:9506
0137C3  mov word ptr ds:66D0h, 1Ah     ; 位移 26 = 最大生命值 → ds:9508
0137D5  if 當前 + 治療量 ≥ 最大:
0137DF      治療量 = 最大 − 當前         ; 夾住
0137FD  位移 28 ← 當前 + 治療量
```

**`INT(RND × N) + M` 是那個標準成語**([`152`](152-experience-settlement-formula.md) §3.1),
所以 `M` 幾乎確定是 1、治療量是 `1…N`。
⚠ `N`(`ds:950A`)**未解** —— 與 [`154`](154-die-is-d100.md) 的面數不同,
這一個沒有觀測後果可以反推(手冊沒寫治療池)。

## 3. 對 `formats/02` 的補充

| 位移 | 語意 | 信心 |
|---:|---|---|
| **63** | **治療池已使用次數**(上限 11)| 已確認 |
| 65 | 未解 —— `0x13663` 讀它並與 `ds:3534 == 5` 做 `and` | — |

## 4. 明列剩餘的不確定

| 項目 | 狀態 |
|---|---|
| `ds:950A`(治療量的面數)| **未解** |
| 訊息編號 544 / 545 的文字 | 未解 —— 要先定出當前迷宮的 `DT*TEXT.DAT` |
| `GROUPS.DAT` 位移 65 | 未解(§3)|
| 位移 63 由誰遞增 | **未讀** —— 本篇只讀到「讀取與比較」那一端 |
| 寶石謎題在哪一座迷宮、哪一格觸發 | 未解 —— 事件表在 `DE*EFF.BIN`([`60`](60-event-lookup-and-tile-19.md))|
