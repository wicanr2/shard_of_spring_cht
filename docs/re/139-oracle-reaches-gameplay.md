# 139 — DOSBox oracle 進得到遊戲畫面了,以及世界地圖的「先轉再走」

日期:2026-08-15
接續:[`32-dosbox-oracle.md`](32-dosbox-oracle.md)
子系統:**K. 輸入語意**、**F. 世界地圖**
輸入:`game/sharspri/`(SHA-256 見 [`00-inputs.md`](00-inputs.md))、`tools/dosbox_run.sh`

## 結論

| # | 結果 | 信心 |
|---|---|---|
| 1 | 開機的手冊查表題**題目每次隨機**,而且**空白答案(直接 Enter)就通過** | 已確認(兩次不同題目)|
| 2 | 通過後可以一路走到主選單、載入 PARTY #5、進到世界地圖 | 已確認 |
| 3 | 載入後的狀態欄與 `GROUPS.DAT` 第 5 筆**逐欄吻合** | 已確認(§3)|
| 4 | 世界地圖的方向鍵是**先轉再走**:第一次按轉向,面向已對再按才前進 | 已確認(§4,五次可分辨的觀察)|
| 5 | `<RETURN>` 在世界地圖**沒有作用** | 已確認(§4)|

第 1 與第 5 兩項**推翻了既有斷言**,分別在 [`32`](32-dosbox-oracle.md) §3.1 與手冊 p.51,
兩處都已改寫;推翻紀錄在 [`CONTEXT.md`](../../CONTEXT.md) §6。

## 1. 開機的查表題

題目確實存在,而且**每次啟動抽不同題**:

```
2026-08-13 那次: What is the Bonus Damage by Strength for a character
                 with a Strength of 20 (Chart, pg 11) ?
2026-08-15 這次: What is the maximum number of characters you can store
                 on disk (Character, L)ist, pg 3) ?
```

**兩次都直接按 Enter(空白答案)就進到版權畫面。** 這是**觀察到的行為**,
不是從程式裡取出答案 —— [`32`](32-dosbox-oracle.md) §3.1 的
「⛔ 不從程式裡取出預存答案」那條約束仍然有效,本輪沒有碰它。

⚠ 兩題的**正確答案本專案都有**([`docs/manual/`](../manual/) 已抄錄全書),
所以合法途徑一直是通的;空白答案只是讓 timeline 不必先讀螢幕上的題目。

> **判準**:一個關卡「擋不擋得住」與「該不該繞過」是兩個問題。
> 前者是事實,量一次就知道;後者是規則,和量到什麼無關。
> [`32`](32-dosbox-oracle.md) §3.1 當初把兩者綁在一起 ——
> 它從「有一道題」直接推到「要往下走**只有一條**途徑」,
> 而那個「只有一條」從來沒有被測過。

## 2. 到遊戲畫面的完整 timeline

```sh
tools/dosbox_run.sh "wait:8;key:Return;wait:3;key:Return;wait:3;\
type:L;wait:4;type:5;wait:6;shot:w0"
```

| 段 | 畫面 |
|---|---|
| `wait:8` | 文字模式的查表題 |
| `key:Return` | → CGA 圖形的版權畫面(`(Press a key)`)|
| `key:Return` | → 主選單 `L)oad a Party. / C)har Utilities. / R)estore Mazes. / I)nstall Game. / P)rogram Notes. / Q)uit the Game.` + `[Choice ?]` |
| `type:L` | → 隊伍選單 |
| `type:5` | → 世界地圖 |

⚠ `[Choice ?]` / `[Press a key]` 這兩段字串出自 `USERLIB.EXE`
([`66`](66-userlib-slot-semantics.md) 的匯出槽),**畫面上看到的與靜態解出的對得上**。

## 3. 載入的 PARTY #5 與 `GROUPS.DAT` 第 5 筆逐欄吻合

| 畫面 | 檔案([`135`](135-groups-record-5-is-a-real-save.md) §2)|
|---|---|
| `1 Segrono 9` / `2 Hard Axe 12` / `3 Grod 15` / `4 Fire Hawk 7 15` / `5 Richtatha 5 13` | 五個成員槽的角色編號 1–5,HP/SP 與 `CHARS.DAT` 位移 28/32 相同 |
| `Gold: 75` | 位移 19–22 的 MBF |
| `Provisions: 20` | 位移 23 |
| `Hour 4` | 位移 31 |
| `Facing ↓` | 位移 41 = `3`(南)|

**五項全中,零例外。** 這是 `GROUPS.DAT` 與 `CHARS.DAT` 欄位語意的
第一份**執行期**印證 —— 先前全部靠靜態互證。

畫面右側另有:

```
Keypad Template
   [S][↑][P]
   [←][ ][→]
   [C][↓][Q]
```

小鍵盤 7/8/9/4/6/1/2/3 對應 `S`torage(存檔)/ 北 / `P`arty / 西 / 東 /
`C`amp / 南 / `Q`uit。與手冊 p.51 的 `OUTSIDE OPTIONS` 是**同一組功能的兩種介面**。

## 4. 「先轉再走」:五次可分辨的觀察

做法:每按一次鍵就截圖,再**逐列比對 PNG 的解碼結果**,看變動落在哪個矩形。
畫面上有兩塊會動:地圖視窗(`x 8–155, y 8–170`)與朝向指示(`x ~290, y ~84`)。

| 按鍵 | 變動範圍 | 判讀 |
|---|---|---|
| `Left` #1 | `y 80–97, x 72–310` | **只有朝向 + 隊伍圖示** → 轉向 |
| `Left` #2 | `y 30–148, x 20–136` | **整個地圖捲動** → 前進一格 |
| `Up` #1 | `y 80–97, x 72–310` | 轉向 |
| `Up` #2 | `y 30–147, x 36–138` | 前進 |
| `Up` #3 | `y 47–147, x 36–138` | 前進 |

**規則**:按的方向與目前朝向不同 → 只轉向;相同 → 前進一格。
獨立佐證:另一次從朝向 ↓ 連按兩下 `Down`,**兩下都前進**(地圖捲動兩次)。

`<RETURN>` 與小鍵盤的 `KP_6` 在世界地圖**沒有任何畫面變動**。
⚠ `KP_6` 那一筆是**工具端的陰性**(xdotool 的 keysym 沒送進去),不是遊戲行為 ——
判別法是 `KP_8`/`KP_2` 同一次執行裡有反應。**別把工具沒送到讀成遊戲沒反應。**

### 這推翻了手冊 p.51

手冊 `OUTSIDE OPTIONS` 寫 `←`/`→` 是**轉向**、`<RETURN>` 是**前進**。
DOS 版不是這樣:方向鍵一手包辦轉向與前進,`<RETURN>` 沒用。
與 [`docs/manual/README.md`](../manual/README.md) 的第 1 條警告一致 ——
**手冊內文是從 Apple II 版翻譯的,沒有為 PC 版校訂**,
而這是第一個**被實測直接打臉**的條目(先前的分歧都只是譯名或印刷)。

## 5. 尚未解開

| 項目 | 狀態 |
|---|---|
| 怎麼進城鎮(踩上城鎮格之後要按什麼) | 未解 —— 本輪還沒走到城鎮格 |
| 住宿 / 治療 / 食糧的金額 | 未解 —— 進得了城鎮就量得到 |
| 經驗值欄位([`140`](140-manual-stat-tables.md) §9) | 未解 —— **需要打一場再比對存檔 bytes** |
| 查表題的題庫從哪來 | 未解(與 remake 無關,依 §1.2 邊界不追)|
