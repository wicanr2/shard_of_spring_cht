# 181 — `DE*EFF.MST` 是母片:原版把「事件已觸發」寫回 `.BIN`,`R)estore Mazes` 就是還原它

日期:2026-08-15
接續:[`161-maze-event-dispatch.md`](161-maze-event-dispatch.md) §2、[`15-game-shell.md`](../spec/15-game-shell.md) §1.1
子系統:**G. 地城與迷宮** / **D. 存檔**
輸入:`DE0EFF.BIN`–`DE7EFF.BIN` 與同名 `.MST`(9 對)、`MENU.EXE`(SHA-256 見 [`00-inputs.md`](00-inputs.md))

## 結論

| # | 結果 | 信心 |
|---|---|---|
| 1 | 9 對 `.BIN` / `.MST` **事件資料完全相同**,只差 BSAVE 標頭的 4 bytes(載入位址)| **已確認**(逐 byte 比對)|
| 2 | 出貨的 `.BIN` 是**乾淨狀態**,不是玩過的存檔 | **已確認**(第 1 項的直接後果)|
| 3 | `.MST` 是**母片**,`R)estore Mazes` 把它複製成 `.BIN` | **已確認**(`MENU.EXE` 的字串)|
| 4 | 原版把「一次性事件已觸發」**寫回 `.BIN` 檔案本身** | 證據充分 |

## 1. 逐 byte 比對

```
DE0EFF.BIN 對 DE0EFF.MST:長度同 1068,差 4 bytes
  位移 1  185 vs 84      位移 2  15 vs 37
  位移 3   34 vs 10      位移 4  104 vs 25
```

九對**全部只差位移 1–4**,而那正是 BSAVE 容器標頭裡的 `segment:offset`
([`19`](19-bsave-container.md))。而且 `.BIN` 那一側**九個檔的值完全一樣**
(`0FB9:6822`),`.MST` 那一側則逐檔不同。

**位移 7 之後的 1,061 bytes 九對全同** —— 事件資料本身沒有任何差異。

## 2. `R)estore Mazes` 在做什麼

`MENU.EXE` 的字串([`15`](../spec/15-game-shell.md) §1.1):

```
'Please have your GAME and BOOT disks ready, you will be asked to swap disks several times.'
'Insert BOOT disk for file #'  ' of 9'   'de'  'eff.mst'
'Insert GAME disk for file #'  ' of 9'   'de'  'eff.bin'
'Restoration complete!  Please insert the BOOT disk.'
```

**從 BOOT 讀 `de*eff.mst`,寫到 GAME 的 `de*eff.bin`,九個檔。**
方向明確:`.MST` → `.BIN`。

## 3. 為什麼這表示原版會寫回

一個「把母片複製回去」的選單項,**只有在工作檔會被改動時才有存在的理由**
(Chesterton's Fence:先問它擋住了什麼)。而已知原版確實會改那份資料:
[`161`](161-maze-event-dispatch.md) §2 讀到一次性事件的作廢方式是
**把座標欄寫成 99**,讓它再也掃不到。

所以完整的圖是:

```
玩的時候   觸發一次性事件 → 座標欄 ← 99 → 寫回 DE*EFF.BIN
要重玩     R)estore Mazes → DE*EFF.MST 複製成 DE*EFF.BIN
```

⚠ **信心是「證據充分」不是「已確認」** —— 我沒有看到寫檔那一段程式碼,
也沒有實跑觀察到 `.BIN` 被改。⛔ 不要把它寫成已確認。
要升級的話:在 DOSBox 裡觸發一個一次性事件,再比對 `.BIN`。

## 4. 這對 remake 的意義

**remake 不能照抄這個做法。** [`CLAUDE.md`](../../CLAUDE.md) §8:`game/` 唯讀。
把遊戲進度寫進原版資料檔還有一個更根本的問題 —— **那樣「存檔」與「遊戲資料」
是同一份檔案**,多個存檔就不可能存在。

引擎要把「哪些事件已作廢」記進**自己的存檔**,見
[`spec/18`](../spec/18-save-format.md)。

⚠ **現行引擎在這一點上是壞的**:`maze.DisableTarget` 只改記憶體裡那份
`level.events`,而那份是每次進迷宮從 `.SQZ`/`.BIN` 重讀的 ——
**走出迷宮再走回去,一次性事件就復活了**。這個缺口在
[`spec/18`](../spec/18-save-format.md) 修。

## 5. 明列剩餘的不確定

| 項目 | 狀態 |
|---|---|
| 寫回 `.BIN` 的那段程式碼 | **未讀**(§3 的信心因此是證據充分)|
| `.BIN` 的 BSAVE 載入位址九檔相同(`0FB9:6822`)代表什麼 | **未解** —— `0x6822` 是 `CMBT` 的怪物屬性陣列基底([`152`](152-experience-settlement-formula.md) §2.1),但 `DE*EFF` 是 `MAZEMOVE` 讀的,兩者對不上 |
| `.MST` 的載入位址為什麼逐檔不同 | **未解** |
