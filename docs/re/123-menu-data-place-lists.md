# 123 — `MENU.EXE` 的三串 `DATA` 敘述:地城名單與兩條路線

日期:2026-08-14
接續:[`62-l-localization-inventory.md`](62-l-localization-inventory.md)
子系統:**L. 中文化落點**(追加證據)/ **G. 地城**(旁證)
輸入:`MENU.EXE`(SHA-256 見 [`00-inputs.md`](00-inputs.md))

## 結論

`MENU.EXE` 檔案位移 `9968`–`10169` 有**三串連續的 null 結尾 `DATA` 敘述**,
合計 195 bytes。第二、三串是**地城名單,各以 `Ralith` 結尾** ——
對上 `DT0TEXT.DAT` 老者那段「抵達 Ralith 就無法回頭……明智選擇路徑」,
兩張清單是**兩條通往同一終點的路線**。

第一串 10 個代號的語意**未知**,不臆測(§3)。

## 1. 三串的內容與位移

| # | 檔案位移 | bytes | 內容 |
|---|---:|---:|---|
| 1 | 9968 | 59 | ` Onyx,Phoenix,Wind,Jade,Fire,Hyacinth,Ruby,Ebony,Gold,Comet` |
| 2 | 10030 | 70 | ` Old Man in Cave,Swamp King,Black Fort,Edrin's Keep,Gate Keeper,Ralith` |
| 3 | 10103 | 66 | ` Rebel's Hideout,Murthin,Cercion,Lothian,Eldron,Vandiguard,Ralith.` |

三串各自以 `0x00` 結尾(null 分別在 10027 / 10100 / 10169),
中間夾 2 byte 的間隔 —— 與
[`46`](46-string-table-partial.md) §6 認定的 `DATA` 敘述形式相同
(null 結尾的純文字,不帶描述子)。**信心等級:已確認**(直接讀位元組)。

每串開頭都有一個前導空白,這是 BASIC `DATA` 敘述在逗號後保留空白的慣例,
不是名稱的一部分。

## 2. 兩條路線,同一終點

第 2 串 6 站、第 3 串 7 站,**最後一站都是 `Ralith`**。

`DT0TEXT.DAT` 第一段(老者)寫:

> 'Once you have reached Ralith there will be no turning back.
> Only a teleport spell will return you to Ymros. Choose a path wisely.'

「**choose a path**」與「兩張都終止於 `Ralith` 的清單」互相印證。
**信心等級:證據充分**(兩份獨立來源:模組內的 `DATA` 與資料檔的敘述文字)。
⚠ 尚未讀到讀這兩串的程式碼,所以「玩家在選單上二選一」仍是**假設** ——
清單相鄰也可能只是同一個 `READ` 迴圈的連續資料。

### 第 3 串的四個名字同時是怪物

`Murthin` / `Cercion` / `Lothian` / `Vandiguard` 在
`MONSTERS.DAT` 裡也是怪物名([`16`](16-rule-tables.md) §1),
而 `DT7TEXT.DAT` 把它們寫成 **Moonglow 家族的四座墓**:

> The tomb of Eldron Greyhair. The spirit of Eldron says,
> 'Search the tombs of the Moonglow clan and return here when you know their names.'

**地城名 = 墓主名 = 該地城的頭目名**,三者同名。

## 3. 第一串 10 個代號:語意未知

`Onyx / Phoenix / Wind / Jade / Fire / Hyacinth / Ruby / Ebony / Gold / Comet`

這十個詞**只出現在 `MENU.EXE`**(全 100 個原版檔案的位元組掃描,單一命中),
沒有任何其他檔案引用,也還沒找到讀取端。

⚠ 數量 10 與 `DE*EFF.BIN` 的 10 對成對檔案相同,
**但這只是數字巧合,不構成證據** —— 十這個數在本作出現多次。
在讀到讀取端之前不寫任何解釋。**信心等級:未知。**

## 4. ⚠ 原版自己的拼法不一致:`Black Fort` vs `Blackfort`

| 拼法 | 落點 |
|---|---|
| `Black Fort`(兩字) | `MENU.EXE` 的第 2 串 |
| `Blackfort`(一字) | `DT51TEXT.DAT`、`TOWN.EXE` |

**這是原版的不一致,不是誤讀**(兩邊都是直接讀位元組)。
中文化時兩處要統一成同一個譯名,並在
[`translations/glossary.md`](../../translations/glossary.md) 記下這個分歧,
否則之後有人拿其中一種拼法去 grep 會漏掉另一半。

> **判準**:原文的專有名詞出現多種拼法時,**以位元組為準逐一列出全部拼法**,
> 不要挑一個當「正確的」。譯名統一是譯文那一側的事,
> 來源那一側要保留分歧的事實,否則盤點的分母會少。

## 5. 對 L 項的影響

這三串屬於 [`62`](62-l-localization-inventory.md) 總表的
「模組:`DATA` 敘述 130 段 / 3,456 bytes」那一列,**總數不變**。
但它們是**專有名詞的權威來源**(地城名單),
所以列進 [`translations/glossary.md`](../../translations/glossary.md) 的地名表。

## 6. 尚未解開

| 項目 | 狀態 |
|---|---|
| 第 1 串 10 個代號的語意 | **未知**,卡在找不到讀取端 |
| 讀這三串的 `READ` 迴圈 | 未解 |
| 「兩條路線」是玩家選的還是劇情固定 | 假設,要讀取用端或 DOSBox 實跑裁決 |
