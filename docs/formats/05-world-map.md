# 世界地圖 — `WRLDMAP.BIN` / `FASTWRLD.BIN` / `TOWNDATA.BIN` — **READY**

## 1. BSAVE 容器(所有 `.BIN` 共用)

```
0xFD  segment(2)  offset(2)  length(2)   ← 7 bytes 標頭
資料 …
0x1A                                     ← EOF
```

⚠ **標頭裡的 offset 是作者端的位址,不是載入位址** ——
BASIC 的 `BLOAD file$, offset` 自己指定載入點。**不要拿它當結構資訊。**

⚠ **7 是奇數**:不扣標頭就切 2-byte 交錯資料,奇偶性會整個翻掉。

## 2. `WRLDMAP.BIN` — 24,934 bytes

扣掉 7 + 1 之後 24,926 bytes = **12,463 格 × 2 bytes**。

```
寬 103、高 121        (103 × 121 = 12,463)
索引 = y × 103 + x
每格 2 bytes,little-endian;**高位元組 100% 為 0,地形值在低位元組**
```

⚠ **本節第一版把兩個位元組寫反了**(寫成「低位元組為 0 → 實際用高位元組」)。
實測:第一個 byte(LE 低位)有 31 種值、僅 1.80% 為 0;
第二個 byte(高位)**12,463 格全部是 0**。

> **判準**:「哪個位元組帶資料」這種敘述,寫進規格前要**兩邊都數一次**。
> 只數一邊會得到「這個 byte 有值」,而那與「另一個 byte 沒值」不是同一件事。

## 3. 地形圖塊

`FASTWRLD.BIN` 存 **9 張**地形圖塊。
`WRLDITEM.PIC` 的**第 k 行 = 圖塊編號 k + 10**。

## 4. `TOWNDATA.BIN` — 13 個城鎮

單精度 **MBF 浮點**(指數在第二個 word 的高位元組)。
13 筆,三重驗證通過(筆數、座標落在地圖範圍內、與 `TOWN.EXE` 的城鎮字串數一致)。

## 未解

`WRLDMAP` 高位元組的**地形值 → 地形種類**對照只解到「哪些值會觸發遭遇檢查」
(12、13、20–32,見 [`spec/01`](../spec/01-combat.md));完整對照表未建。

出處:[`re/17`](../re/17-wrldmap-cell-size.md)、[`19`](../re/19-wrldmap-dimensions.md)、
[`21`](../re/21-bsave-container.md)、[`51`](../re/51-fastwrld.md)–[`54`](../re/54-f-closure.md)
