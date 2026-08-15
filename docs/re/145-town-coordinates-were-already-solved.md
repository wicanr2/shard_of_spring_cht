# 145 — 城鎮座標對應:答案在 `re/53` 躺了很久,而實作端一直在猜

日期:2026-08-15
接續:[`141-world-map-axes-were-transposed.md`](141-world-map-axes-were-transposed.md)、[`53-world-tiles-towns-and-draw-renderer.md`](53-world-tiles-towns-and-draw-renderer.md)
子系統:**F. 世界地圖**
輸入:`TOWNDATA.BIN`、`TOWNDATA.DAT`、`WRLDMAP.BIN`、DOSBox 實跑

## 結論

`TOWNDATA.BIN` 是 **13 個城鎮的座標表**,而且 [`53`](53-world-tiles-towns-and-draw-renderer.md) §2
早就用三重交叉驗證解出來了。**引擎端一直沒用它**,而是用
「把地圖上的城鎮格按座標排序、取第 n 個」的佔位。

**那個佔位是錯的。** 實跑走到 `(24, 12)` 進到的是 **Arcania**(表的第 4 列),
而排序給出的是 Gleon。

| | |
|---|---|
| 表的形狀 | 56 個 MBF 單精度,**逐欄**存:索引 = 欄 × 14 + 列,第 0 列全零 |
| 欄 1 / 2 / 3 | **東西座標 / 南北座標 / 建築數** |
| 驗證 | 13/13 的座標落在圖塊 30–32 上;13/13 的建築數等於 `TOWNDATA.DAT` 的記錄數 |
| 實跑印證 | 第 1 列 `(9,8)` = Green Hamlet、第 4 列 `(24,12)` = Arcania |

**信心等級:已確認。**

## 1. 表

```
列   東西  南北  建築數   城鎮(TOWNDATA.DAT 的出現順序)
 1     9     8     7      Green Hamlet     ← 實跑印證
 2    26    26     4      Precious Plains
 3    91    12     5      Gleon
 4    24    12     6      Arcania          ← 實跑印證
 5    51    37     5      Woodhaven
 6    97    26     7      Terynor
 7   114     9     6      Myrquacid
 8    67    52     4      Athe
 9    35    57     5      Spider Bay
10   114    23     4      Janthrin
11    44    81     3      Atlantis
12    69    85     3      Oceana
13    88    72     2      Triton
```

⚠ [`53`](53-world-tiles-towns-and-draw-renderer.md) §2 把欄 1/2 標成 `Y`/`X` ——
那是**兩軸接反時的命名**([`141`](141-world-map-axes-were-transposed.md))。
數值沒有錯,名字反了,該篇已改。

⚠ 排法是**逐欄**(BASIC 二維陣列的常見配置)。寫成「列 × 4 + 欄」會取到別欄的值,
而那些值同樣落在合理範圍內 —— **索引寫反不會越界,只會安靜地讀錯**。

## 2. 為什麼會漏掉

[`52`](52-world-map-reader-and-shared-grid.md) §結尾的表把 `TOWNDATA.BIN` 標成**未解**,
而下一篇 [`53`](53-world-tiles-towns-and-draw-renderer.md) 就解掉了。
實作端引用的是前者。

> **判準**:這是「答案早就在自己的 docs 裡」的**第六次**。
> 前幾次的形狀是「沒查」或「查了沒讀完」,這一次不同 ——
> **我引用的那份筆記本身是對的,只是它比答案早一篇。**
>
> 對策:一份筆記把某項標成「未解」時,那句話的有效期只到**下一篇**。
> 要引用「未解」這個狀態,得往後掃到最新的一篇 ——
> 而 `CONTEXT.md` §2 的現況表就是為了不必每次都掃。**先看現況表。**

## 3. 這一輪的實跑

從出貨存檔起點走 28 步到 `(24,12)`:路徑用可通行性八條規則
([`131`](131-world-passability.md))在專案外部跑 BFS 算出來,再轉成按鍵序列
(每一步「先轉再走」,[`139`](139-oracle-reaches-gameplay.md) §4)。

畫面顯示 `Arcania` 與六間建築 `Potion Shoppe / Zor's Healing / Cloud Nine /
Foaming Brew / Magic Shoppe / Best Healing`,與 `TOWNDATA.DAT` 記錄 16–21 逐間吻合。

> **判準**:要證偽一個排序假設,**挑一個兩種排法給出不同答案的點**去測。
> `(24,12)` 在「依南北再東西」是第 3 個、在座標表是第 4 個 —— 一次就分開了。

## 4. 尚未解開

| 項目 | 狀態 |
|---|---|
| 世界地圖上 13 個城鎮**圖塊值 30/31/32 的差別** | 未解 —— 與城鎮大小、建築數都對不上 |
