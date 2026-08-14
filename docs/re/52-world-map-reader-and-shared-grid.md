# 52 — 世界地圖的讀取端,與 `ds:6822` 其實是「當前地圖」

日期:2026-08-14
接續:[`51-mazedata-and-world-entrances.md`](51-mazedata-and-world-entrances.md)
子系統:**F. 世界地圖** / **J. 戰鬥規則**

## 結論

1. **`WRLDMOVE.EXE` 的取格常式證實了 [`51`](51-mazedata-and-world-entrances.md) 從資料側推出的索引方式**
   —— `(y × 103 + x) × 2 + 0x6822`,逐字相同。§2.1 條件 1 對 F 已滿足。
2. **`ds:6822` 不是某一個固定的陣列,是「當前地圖」** ——
   三支模組用同一個基底、各自的寬度:**世界 103、迷宮 51、戰鬥 15**。
3. 因此 [`43`](43-common-block-and-array-indexing.md) 的「15 列 × ≥20 欄」要改讀成
   **「15 欄的戰鬥格」**,而隊伍站在第 9–13 欄。

## 1. 取格常式(`WRLDMOVE.EXE`,`0x11136`)

```
011136  int  3Fh, 53h
011139  mov  di, [bp+0Ah]      ; 參數 1 的位址
01113C  mov  ax, [di]          ; = Y
01113E  mov  si, 67h           ; ★ 103
011141  imul si                ; Y × 103
011143  mov  di, ax
011145  mov  si, [bp+0Ch]      ; 參數 2 的位址
011148  add  di, [si]          ; + X
01114A  shl  di, 1             ; ×2 → word
01114C  mov  ax, [di+6822h]    ; ★ 取格
011150  mov  ds:0CF92h, ax
011153  cmp  ax, 0Ah           ; 與 10 比較
```

**`(Y × 103 + X) × 2 + 0x6822`,元素是 word。**

[`51`](51-mazedata-and-world-entrances.md) §2 從資料側試了四種索引順序、兩種元素大小,
挑出這一種是因為它讓 12 個入口座標對上 11 個稀有圖塊。
**現在有了程式碼側的直接證據,兩邊逐字相同。**

**信心等級:已確認**(§2.1 條件 1、2、3 齊備)。

## 2. 圖塊 → 圖形的派工

`0x1116C` 起是一條比對鏈:

```
cmp  word ptr ds:0CF96h, 1  →  mov bx, 0CA1Ch
cmp  word ptr ds:0CF96h, 2  →  mov bx, 0CA78h
…
cmp  word ptr ds:0CF96h, 9  →  mov bx, 0CCFCh
cmp  word ptr ds:0CF96h, 0Bh → jnz(值 11 什麼都不畫)
```

| 圖塊 | 圖形位址 | 間距 |
|---:|---|---:|
| 1 | `0xCA1C` | — |
| 2–9 | `0xCA78`…`0xCCFC` | **各 92** |

**圖塊 1–9 的圖形是連續的 92-byte 記錄**,由 `fastwrld.bin` BLOAD 進來
([`132`](132-world-tile-dispatch-corrected.md) §2)。圖塊 10 缺席 ——
因為 `0x11153` 的 `cmp ax, 0Ah / jl` 把 **≥ 10 的值分到另一條路徑**:
值 11 不畫,其餘走 `0xC980 + 4×值` 的 `DRAW` 巨集表
([`132`](132-world-tile-dispatch-corrected.md) §1)。

⚠ 92 bytes 與 [`21`](21-tile-format.md) 的 98-byte 圖塊檔**不同**。
98 = BSAVE 標頭 7 + 資料 90 + EOF 1,而記憶體裡的記錄是 92 ——
差額是陣列的對齊,**未追**(依 §1.2 不影響 remake)。

## 3. `ds:6822` 是「當前地圖」

同一個基底,三支模組三種寬度:

| 模組 | 索引算式 | 寬度 | 出處 |
|---|---|---:|---|
| `WRLDMOVE` | `imul 67h` → `Y×103 + X` | **103** | §1 |
| `MAZEMOVE` | `ax + ds:8DC0h`(預先算好的列位移) | — | `0x10477` |
| `CMBT` | `迴圈變數 + {15,30,45,…,285}` | **15** | [`43`](43-common-block-and-array-indexing.md) §1 |

**裁決性的一點:`MAZEMOVE` 與 `WRLDMOVE` 用同一組門檻值。**

```
MAZEMOVE 0x10477:  cmp word ptr [di+6822h], 5
MAZEMOVE 0x10482:  cmp word ptr [di+6822h], 0Ah    ← 與 WRLDMOVE 0x11153 相同
WRLDMOVE 0x11153:  cmp ax, 0Ah
```

兩支模組對同一個基底做**同樣的值域判斷** ——
所以那是**同一個陣列、同一套圖塊編碼**,只是裝著不同的地圖。

這解釋了為什麼它在 COMMON 區(`MASTER.INC`,[`47`](47-source-filenames-and-master-inc.md)):
**換場景時 `CHAIN` 到別的模組,地圖留在同一個陣列裡。**

## 4. ⚠ 訂正 [`43`](43-common-block-and-array-indexing.md):15 是寬度,不是列數

[`43`](43-common-block-and-array-indexing.md) §1 從 19 個索引常數全是 15 的倍數,
推出「15 列 × 至少 20 欄」。**算術沒錯,標籤反了。**

15 是**每列的格數(寬度)**,那些常數是 `列 × 15`:

```
CMBT 的戰鬥格 = 15 欄 × 至少 20 列
```

而 [`43`](43-common-block-and-array-indexing.md) §2 的戰鬥迴圈
(跑 `9` 到 `隊伍人數 + 8`,取欄位 `+45` 與 `+105`)要改讀成:

> **在第 3 列與第 7 列上,走訪第 9 到第 13 欄** —— 也就是隊伍站在戰鬥格的**右側**。

這比原本的「第 9–13 列是隊伍」更自洽:戰術戰鬥畫面本來就是
**我方一側、敵方一側**,而 15 欄的格子分兩邊剛好。

⚠ **「右側」是從欄號大的一端推的,未從畫面驗證。** 標為假設。

## 5. 這對看板的意義

| 子系統 | 變化 |
|---|---|
| **F** | 條件 1 已滿足(§1);仍缺其餘圖塊語意與 `TOWNDATA.BIN` |
| **J** | 戰鬥格是 15 欄的地圖陣列(§4),與世界/迷宮共用機制 |
| **G** | `MAZEMOVE` 用同一個陣列與同一套圖塊編碼(§3)|

## 6. 尚未解開

| 項目 | 狀態 |
|---|---|
| 圖塊 10 以上走的那條路徑 | 未解(`0x11153` 的分支)|
| 圖塊 1–9 各是什麼地形 | 未解 —— 要對 `ds:0xCA1C` 起的 92-byte 記錄渲染 |
| `TOWNDATA.BIN` | 未解 |
| 戰鬥格的列數 | 未解(常數只到 285 = 列 19)|
| 記憶體記錄 92 vs 檔案 98 | 未追(§2 的但書)|
