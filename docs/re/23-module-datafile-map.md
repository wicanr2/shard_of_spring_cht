# 23 — 模組 ↔ 資料檔對照表

日期:2026-08-13
接續:[`22-pict-and-monst.md`](22-pict-and-monst.md)

## 結論

從十一支模組本體掃出檔名字串,得到**哪一支模組讀哪些資料檔**。
這張表的用途:`CLAUDE.md` §2.1 的條件 1、2 要求「在 IDA 裡讀讀取端」,
而本表直接指出**每個子系統該去哪一支模組找**。

## 1. 對照表

| 模組 | 引用的資料檔 |
|---|---|
| `CMBT.EXE` | `MONSTERS.DAT`、`ITEMS.DAT`、`SPELLS.DAT`、`fastcmbt.bin`、`rndmonst.bin`、**`monst2.bin`**、**`monst4.bin`** |
| `MAZEMOVE.EXE` | `MAZE.SQZ`、`TEXT.DAT`、`mazewall.bin`、`mazeitem.PIC`、`pict6.BIN`、`DE5EFF.BIN`、`DE51EFF.BIN` |
| `MENU.EXE` | `CHARS.DAT`、`GROUPS.DAT`、`mazedata.bin`、`startup.bin`、`walkdraw.pic`、`*eff.bin` |
| `TOWN.EXE` | `ITEMS.DAT`、`towndata.bin`、`towndata.dat` |
| `CAMP.EXE` | `spells.dat`、`ITEMS.DAT` |
| `CHARUTIL.EXE` | `CHARS.DAT`、`GROUPS.DAT`、`titles.DAT` |
| `WRLDMOVE.EXE` | `wrldmap.bin`、`wrlditem.pic`、`fastwrld.bin`、`pict6.BIN` |
| `WSIO.EXE`、`MIO2.EXE`、`MTEST.EXE` | `fastwrld.bin` 等片段 |

⚠ 掃描抓的是**字串常數**,所以只涵蓋「檔名寫死在程式裡」的情況。
動態組出來的檔名(見 §2)抓不到,**這張表是下界不是全集**。

## 2. `MONST*.BIN` 的檔名是組出來的

`CMBT.EXE` 裡只出現 `monst2.bin` 與 `monst4.bin` 兩個字面值,
但磁碟上有 22 個 `MONST1`–`MONST22.BIN`。

所以其餘 20 個是**執行期組出來的**(BASIC 的字串串接,類似
`"MONST" + 編號 + ".BIN"`)。這與 [`16`](16-rule-tables.md) §1.1 解出的
`MONSTERS.DAT` 欄位 `w5`(怪物圖索引,值域 1–22)完全吻合 ——
**`w5` 就是拿去組檔名的那個數字。**

兩份分析互相印證:一邊是資料檔的欄位值域,一邊是程式裡的檔名字面值不足。

`monst2` / `monst4` 之所以寫死,可能是某些場合固定用這兩張圖(待查)。

## 3. 各子系統的讀取端在哪

| 子系統 | 待解項目 | 去哪一支找 |
|---|---|---|
| E. 規則資料表 | `MONSTERS.DAT` 的 `w0`–`w9` | `CMBT.EXE` |
| E | `ITEMS.DAT` 欄 2–5 | `CMBT` / `TOWN` / `CAMP`(三支都讀)|
| E | `SPELLS.DAT` 欄 2–4 | `CMBT` / `CAMP` |
| F. 世界地圖 | 圖塊語意、座標系 | `WRLDMOVE.EXE` |
| G. 地城 | `.SQZ` 解碼器 | `MAZEMOVE.EXE` |
| H. 圖形 | `MONST*` 的尺寸 | `CMBT.EXE` |
| C. 角色資料 | `CHARS.DAT`、`GROUPS.DAT` 結構 | `CHARUTIL.EXE`(專職)、`MENU.EXE` |

**`CHARUTIL.EXE` 是角色資料的專職模組** —— 子系統 C 目前標「未開始」,
但它的讀取端已經定位,而該模組已解鎖到 37.2%、19 個函式。

## 4. 尚未解開

| 項目 | 狀態 |
|---|---|
| 動態組出的檔名(至少 20 個 `MONST*`)| 已推論,未在程式碼裡確認 |
| `monst2` / `monst4` 為何寫死 | 未解 |
| 各讀取端的實際位址 | 未解 —— 本表只縮小到模組層級 |
