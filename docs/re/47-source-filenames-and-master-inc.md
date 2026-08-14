# 47 — 原始碼檔名還留在執行檔裡,`MASTER.INC` 解釋了 COMMON 區

日期:2026-08-14
接續:[`46-string-table-partial.md`](46-string-table-partial.md)
子系統:**A. 執行檔架構** / **L. 中文化落點**

## 結論

十一支模組都保留了自己的**原始 BASIC 檔名與 include 檔名**。

| 模組 | 原始碼 | include |
|---|---|---|
| `START.EXE` | `START.BAS` | — |
| `MENU.EXE` | `MENU.BAS` | `MASTER.INC`、**`INSTALL.INC`** |
| `TOWN.EXE` | `TOWN.BAS` | `MASTER.INC`、`TOWNCAMP.INC` |
| `CAMP.EXE` | `CAMP.BAS` | `MASTER.INC`、`TOWNCAMP.INC` |
| `CMBT.EXE` | `CMBT.BAS` | `MASTER.INC` |
| `MAZEMOVE.EXE` | `MAZEMOVE.BAS` | `MASTER.INC` |
| `WRLDMOVE.EXE` | `WRLDMOVE.BAS` | `MASTER.INC` |
| `CHARUTIL.EXE` | `CHARUTIL.BAS` | `MASTER.INC` |
| `MTEST.EXE` | `MTEST.BAS` | — |
| `MIO2.EXE` | `MIO2.BAS` | — |
| **`WSIO.EXE`** | **`BITMAKE.BAS`** | — |

**信心等級:已確認**(位元組直接讀出,`tools/extract_text.py`)。

## 1. `MASTER.INC` 就是 COMMON 區的來源

[`43`](43-common-block-and-array-indexing.md) §4 量到:七支主要模組
**共用完全相同的 `ds:` 位移**(`0x66C8`–`0x681A`),而私有變數完全不重疊。
當時的解釋是「這是 BASIC 的 COMMON 區」,信心等級**證據充分**。

現在有了**原因**:那七支模組(加 `MENU`,共八支)**全部 include 同一份 `MASTER.INC`**。

BASIC 的 `COMMON` 宣告必須在每一支模組裡逐字相同、順序相同,
否則 `CHAIN` 之後位移會對不上 —— 把宣告放進一份 include 檔是唯一可維護的做法。
**同一份 include → 同一組宣告 → 同一組位移。**

兩條證據互相獨立:左邊是 **1,702 + 853 + … 個指令運算元的統計**,
右邊是**檔案裡的一個字串**。沒有共同的錯誤來源。
[`43`](43-common-block-and-array-indexing.md) §4 因此升級為**已確認**。

## 2. `TOWNCAMP.INC`:只有 `TOWN` 與 `CAMP`

[`43`](43-common-block-and-array-indexing.md) §4 的統計裡,
「被 4 支用到」有 8 個位移、「被 3 支」有 11 個 ——
那些次級的共用群,對得上次級的 include 檔。

`TOWN` 與 `CAMP` 共用 `TOWNCAMP.INC`,而這兩支在遊戲裡確實是同一類畫面
(城鎮選單 / 營地選單,都是「選一個建築/動作」的列表)。

## 3. `INSTALL.INC` 只在 `MENU`

`MENU.EXE` 是唯一 include `INSTALL.INC` 的模組,
而手冊查詢題的字串([`33`](33-quiz-validates-columns.md))也在 `MENU.EXE`:

```
What is the Bonus Damage by Strength for a character with a Strength of 20  (Chart, pg 11)
What is the Maximum S.P. gain per level for a Wizard with an Intellect of 19 (Chart, pg 12)
```

**所以開機的手冊查詢屬於 `INSTALL.INC`。** 這與 [`32`](32-dosbox-oracle.md) 的
DOSBox 阻礙是同一件事的兩端。⚠ 依既定界線,**不從程式裡取出答案**;
這裡只記錄它在哪一個編譯單元。

## 4. 解開:`WSIO.EXE` 為什麼叫 `BITMAKE`

[`03`](03-bz-module-header.md) §2 記過:十一支裡只有 `WSIO.EXE` 的
`bz` 模組名(`BITMAKE`)與檔名不符,「意義未知」。

**因為它是 `BITMAKE.BAS` 編出來的。** `bz` 標頭的模組名取自原始碼檔名,
不是輸出檔名 —— 出貨時改了輸出檔名,模組名留著。

這也印證了 [`03`](03-bz-module-header.md) §1 對 `+0x02`(模組名)的判讀。

## 5. 文字總量(中文化的規模)

`tools/extract_text.py` 抽出兩種形式([`46`](46-string-table-partial.md) §1 與 §6):

| 形式 | 段數 | 位元組 |
|---|---:|---:|
| 字串常數(三描述子 + 文字) | 822 | 12,282 |
| `DATA` 敘述(null 結尾) | 130 | 3,456 |
| **合計** | **952** | **15,738** |

清單在 [`generated-text-inventory.json`](generated-text-inventory.json)。

⚠ **這是模組裡的文字,不含資料檔**(`*.DAT` 的文字另計,見 [`18`](18-text-inventory.md))。
⚠ 抽取只認明確的形狀,**寧可漏不要假陽性** ——
已知的一個漏法是 `DATA` 敘述若不以 null 結尾就抓不到。

## 6. 這一輪的方法

找到原始碼檔名不是靠猜,是**把文字抽取器寫對之後順便掉出來的**。
而抽取器之所以寫得出來,是因為 [`46`](46-string-table-partial.md) 先把格式讀懂了。

**順序是:格式 → 工具 → 清單 → 意外發現。**
若一開始就去 `strings | grep BAS`,會得到同樣的檔名但**不知道它們是 include 關係**
—— 那個關係是從「哪些模組各自帶哪些檔名」的分布看出來的。

## 7. 尚未解開

| 項目 | 狀態 |
|---|---|
| `MASTER.INC` 裡宣告了哪些變數(順序 = 位移順序) | 未解 —— **可從位移排序反推宣告順序** |
| 檔案位移 ↔ DS 位址的對應 | 未解([`46`](46-string-table-partial.md) §3)|
| 資料檔裡的文字總量 | 見 [`18`](18-text-inventory.md),未合併計算 |
