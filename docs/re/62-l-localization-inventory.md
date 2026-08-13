# 62 — 子系統 L:中文化落點總表

日期:2026-08-14
接續:[`61-i-closure-unused-tiles.md`](61-i-closure-unused-tiles.md)
子系統:**L. 中文化落點盤點**

## 結論

全遊戲的文字共 **1,476 段 / 34,499 bytes**,分成兩類:

> ⚠ 本文第一版寫 1,416 段 / 33,795 bytes,**漏了 `USERLIB.EXE`**(60 段 / 704 bytes)。
> 成因與判準見 [`63`](63-userlib-strings-and-l-correction.md) §4。下表已補。

| | bytes | 佔比 |
|---|---:|---:|
| **可變長**(資料檔) | 15,439 | 45% |
| **只能等長**(模組映像內 + 固定欄位) | 19,060 | 55% |

**中文化的技術路線因此是兩段的**:資料檔可以自由重寫,模組內的字串只能等長替換
(中文佔 2 bytes,所以字數是原文的一半)。

## 1. 總表

| 落點 | 段/筆 | bytes | 長度 | 依據 |
|---|---:|---:|---|---|
| 模組:字串常數(11 支 EXE)| 822 | 12,282 | ❌ 等長 | [`46`](46-string-table-partial.md) §1 |
| 模組:`DATA` 敘述 | 130 | 3,456 | ❌ 等長 | [`46`](46-string-table-partial.md) §6 |
| `DT*TEXT.DAT`(9 檔)| 87 | 11,044 | ✅ 可變 | [`51`](51-mazedata-and-world-entrances.md) §6 |
| **`TITLES.DAT`** | 113 | 1,451 | ✅ 可變 | §2 |
| `ITEMS.DAT` | 57 | 1,756 | ✅ 可變 | [`16`](16-rule-tables.md) §3 |
| `SPELLS.DAT` | 34 | 1,188 | ✅ 可變 | [`16`](16-rule-tables.md) §2 |
| `MONSTERS.DAT` 名稱 | 74 | 1,184 | ❌ 固定 16 | [`16`](16-rule-tables.md) §1 |
| `TOWNDATA.DAT` 名稱 | 74 | 1,184 | ❌ 固定 16 | [`53`](53-world-tiles-towns-and-draw-renderer.md) §2 |
| `CHARS.DAT` 名稱 | 25 | 250 | ❌ 固定 10 | [`27`](27-chars-dat.md) §2 |
| **`USERLIB.EXE`** | 60 | 704 | ❌ 等長 | [`63`](63-userlib-strings-and-l-correction.md) §2 |
| **合計** | **1,476** | **34,499** | | |

清單檔:[`generated-text-inventory.json`](generated-text-inventory.json)(模組部分,含每段的檔案位移)。

## 2. `TITLES.DAT` 是外置的 UI 字串表

113 行帶引號的字串:

```
"Character"
"Utilities"
"* Characters *"
"C)reate"
"R)emove"
"N)ew Name"
"* Parties *"
"D)isband"
"J)oin"
…
```

**這解釋了 [`58`](58-key-dispatch-mechanism.md) §4 的一個現象**:
`CHARUTIL` 的按鍵鏈抽出 `JFDIR`,而模組裡找不到對應的提示 ——
**提示不在 EXE 裡,在 `TITLES.DAT`。**

⚠ 這也表示 [`28`](28-input-semantics.md) 盤點畫面提示時**漏了這個檔**。

## 3. 兩種長度限制的成因

### 模組內:描述子存長度

[`46`](46-string-table-partial.md) §1 已確認格式是 `<長度:2><DS 指標:2>` × 3 + 文字。
改長度要同時改長度欄與後面所有指標,而**檔案位移 ↔ DS 位址的對應未解**
([`46`](46-string-table-partial.md) §3 否證了線性對應)。

**所以等長替換是目前唯一安全的做法** ——
長度欄不動、指標不動,只換文字位元組。

### 固定欄位:記錄長寫死在讀取端

`MONSTERS.DAT` 是 74 × 36 bytes、名稱佔前 16;
`CHARS.DAT` 是 25 × 94、名稱在 `+0x01` 佔 10。
記錄長是程式算出來的,改欄寬要改程式。

⚠ **`CHARS.DAT` 是存檔**,玩家的檔案會覆蓋出貨版 —— 中文化要處理**兩份**:
出貨的範例角色,以及執行時新建角色的輸入長度限制。

## 4. 字數的實際影響

等長替換下,`Level:␣␣␣␣␣`(11 bytes)只能放 **5 個中文字**。
全遊戲 18,356 bytes 的固定長度區,換成中文後只剩 **約 9,178 字**的容量。

| 落點 | 原文 bytes | 中文字數上限 |
|---|---:|---:|
| 模組字串常數 | 12,282 | 6,141 |
| 模組 `DATA` | 3,456 | 1,728 |
| `MONSTERS` 名稱 | 1,184 | 8 字/筆 |
| `TOWNDATA` 名稱 | 1,184 | 8 字/筆 |
| `CHARS` 名稱 | 250 | 5 字/筆 |

**怪物名 8 個中文字、城鎮名 8 個字、角色名 5 個字** —— 這些都夠用。
真正吃緊的是選單與訊息(模組字串常數),那 6,141 字要塞下原本的 12,282 個英文字元。

## 5. L 的四項條件

| # | 條件 | 狀態 |
|---|---|---|
| 1 | IDA 讀原始指令 | ✅ 字串描述子的取用([`58`](58-key-dispatch-mechanism.md) §2:`mov bx, <描述子>`)、`DT` 事件目標的查表([`60`](60-event-lookup-and-tile-19.md) §1)|
| 2 | 讀寫端點 | ✅ 同上 |
| 3 | 獨立資料印證 | ✅ 段數/位元組數由抽取器產出並經正對照([`47`](47-source-filenames-and-master-inc.md) §5:七個已知字串 6/7,漏的那個帶出第二種儲存形式)|
| 4 | 筆記 | ✅ 本文 + `generated-text-inventory.json` |

**L(中文化落點盤點)判定為 RE-DONE。**

⚠ 明確**不含**、屬於後續實作而非 RE 的:
- **改長度的方法**(需要解 [`46`](46-string-table-partial.md) §3 的位移對應)——
  本文的結論是「目前只能等長」,那是**盤點的結果**,不是待解的 RE 項目
- 中文字型與顯示(CGA 320×200 下怎麼畫中文)—— 屬 remake 實作
- 翻譯本身

## 6. 方法

這一列能收,是因為前面幾輪**順便**解掉了它的前置條件:

- [`46`](46-string-table-partial.md) 解字串格式,是為了找 `CHARS.DAT` 的讀取端
- [`47`](47-source-filenames-and-master-inc.md) 寫抽取器,是為了驗證格式
- [`51`](51-mazedata-and-world-entrances.md) 解 `DT*TEXT`,是為了 `MAZEDATA` 欄 5
- [`58`](58-key-dispatch-mechanism.md) 找按鍵,才發現 `TITLES.DAT` 這個漏掉的落點

**沒有一輪是為了 L 而做的。** 盤點型的工作往往不需要專門攻,
把別的東西解對了它自己就齊了 —— 但**要有人回頭把它寫下來**,
否則散在十份筆記裡等於沒有。
