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

---

## 5. 為什麼條件 1、2 在這個遊戲特別難(2026-08-13 補)

`~/.claude/knowledge-base/retro/ida-pro-9.4.md` 記載的經典技法是
**掃「記錄大小當乘數」** —— 索引一張定長記錄表必然要乘上 stride,
而那個乘法在程式碼裡是很強的指紋(該 kb 的案例:`mul` 出現 14 次、
乘的是將領記錄大小 `0x21`,一次就定案了欄位語意)。

**套在 `CMBT.EXE` 上零命中。**

`MONSTERS.DAT` 的 stride 是 36(`0x24`,[`16`](16-rule-tables.md) §1),
在已解鎖的 16,553 條指令裡掃 `0x24`:命中 10 處,**全部是 `mov` 與 `add`,
一個 `mul` 都沒有**。

### 結構性原因

模組是 BASIC 編譯的,**陣列索引由執行期代勞**
([`06`](06-runtime-int-abi.md):模組透過 `INT 3Eh`/`3Fh` 呼叫執行期)。
所以「乘以記錄大小」這個動作發生在 `BRUN30` 裡,不在模組裡 ——
**模組程式碼中不會出現那個指紋。**

這與 [`24`](24-tooling-conflict.md) §2 的發現同源:
`BLOAD` 的載入位址寫在檔案標頭裡,模組也不需要那個常數。
**兩次都是「由執行期決定的東西不會出現在模組程式碼裡」。**

### 對達成 RE-DONE 的意義

`CLAUDE.md` §2.1 的條件 2(用 xref 確認讀寫端)在這個遊戲上
**不能靠掃模組程式碼達成**,要嘛:

1. 先解出執行期的派工表語意([`07`](07-dispatch-tables.md)),
   知道哪個 thunk 索引是「讀陣列元素」,再從呼叫端的參數回推欄位
2. 或走 DOSBox 動態觀察(oracle 優先序裡高於靜態推論)

**這不是努力不夠,是這個技術棧的結構決定的。**
把它寫下來,是為了讓後續不要重複投入在「掃模組找欄位存取」這條路上。
