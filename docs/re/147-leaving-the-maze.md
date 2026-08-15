# 147 — 離開迷宮:從入口那一格往外走一步

日期:2026-08-15
接續:[`146-maze-exit-negatives-and-palettes.md`](146-maze-exit-negatives-and-palettes.md)
子系統:**G. 地城與迷宮**
輸入:`MAZEMOVE.EXE`、DOSBox 實跑(`Black Fort`)

## 結論

**站在入口那一格,往迷宮外面的方向走一步,就離開。**
原版在這一刻印 `Leaving maze ..` 並切回 `WRLDMOVE`。

**信心等級:已確認** —— 字串 + 實跑,兩個來源。

這解掉了 [`spec/08`](../spec/08-maze-scene.md) §6 從 M5 掛到現在的
「⚠ 未解 —— 沒讀到『走出入口格』的程式碼」。

## 1. 字串先給了答案

`MAZEMOVE.EXE` `0x0500A` 起是一組模組轉交的訊息:

```
Leaving maze ..
Making Camp..
    C O M B A T !
    B A T T L E !
```

**「離開迷宮」是一個明確的動作**,不是「按 ESC 回上一層」那種介面行為 ——
它與「進戰鬥」「紮營」並排,而那兩個都是切模組。

⚠ [`146`](146-maze-exit-negatives-and-palettes.md) 用**按鍵窮舉**找了半天
(`ESC`/`E`/`Q` 全是陰性),而答案在**字串表**裡 ——
`Leaving maze` 這五個字從第一次 dump 字串就在檔案裡。

> **判準**:要找「某個動作怎麼觸發」時,**先看有沒有對應的訊息字串**。
> 有訊息就表示那個動作存在,而訊息旁邊的鄰居會告訴你它屬於哪一類
> (這裡的鄰居是三個切模組的訊息 → 它也是切模組,不是介面動作)。
> 按鍵窮舉是最後手段,不是第一步。

## 2. 實跑

從 `Black Fort` 的入口格按一次 `↑`:**整個畫面換掉**(調色盤由迷宮的灰/洋紅
變回世界地圖的綠/紅/棕),回到世界地圖 `(23,18)`,`Hour` 不變。

對照:同一格按 `↓` 兩次是往迷宮**裡面**走(看得到走廊與一個青色物件)。
**所以方向是有意義的** —— 出去的是「朝外」那一邊。

## 3. remake 怎麼實作

```
走到迷宮格陣列的**界外** → 離開迷宮
```

⚠ **這個判斷要放在可通行性之前。** `Maze.At()` 對界外回 `0`,而 `0` 是**可通行**
([`formats/06`](../formats/06-maze.md))—— 先判可通行的話界外會被當成空地,
或被別的檢查擋成「撞牆」,兩種結果都是**把玩家關在迷宮裡**,
而畫面上只看得到「走不動」。

`internal/maze` 因此多了一個 `Left` 結果,`internal/original.Maze` 多了 `InBounds`。

## 4. 順帶從字串看到的東西

`MAZEMOVE.EXE` 的字串還透露幾件**先前沒有記錄**的機制:

| 字串 | 意義 |
|---|---|
| `Enter gem to touch (B,G,V,R)` + `MURTHIN` / `CERCION` / `LOTHIAN` / `VANDIGUARD` + `The brothers are:` | 迷宮裡有一道**寶石謎題**,四個名字是「兄弟」|
| `Which party member do you wish to heal? (0 exits)` / `This pool is empty!` / `That character can't be helped here !` | 迷宮裡有**治療池**,而且會用完 |
| `found an item.` / `Your party is full of items, please discard some from Camp.` | 撿道具;**「背包滿了」再次印證 10 格上限**([`144`](144-created-record-exposes-layout.md) §3)|
| `MB T108 O3 L8a` 等九段 | **BASIC `PLAY` 巨集 —— 遊戲有音樂**,而本專案完全沒有記錄過 |
| `0x04400`–`0x047B7` 八段長敘述 | **結局文字**(取得春之石之後的收尾)|

⚠ 這幾項都**還沒解**,列在這裡是為了不再「重新發現」一次。
音樂那一項尤其:九段 `PLAY` 巨集是**可以直接轉成音符的**,
而 [`spec/03`](../spec/03-engine-plan.md) 的里程碑裡完全沒有聲音。

## 5. 尚未解開

| 項目 | 狀態 |
|---|---|
| 寶石謎題的規則 | 未解(§4)|
| 治療池的次數與恢復量 | 未解(§4)|
| 九段 `PLAY` 巨集對應哪些場景 | 未解(§4)|
| 那一次 `Q`→`E` 回到世界地圖 | 仍未重現([`146`](146-maze-exit-negatives-and-palettes.md) §2)|
