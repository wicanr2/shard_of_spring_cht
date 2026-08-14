# 137 — 迷宮座標:三個來源**順序一致**,衝突的是名字不是算式

日期:2026-08-15
接續:[`56-maze-tile-classes-and-mazedata-columns.md`](56-maze-tile-classes-and-mazedata-columns.md)、
[`59-de-eff-event-table.md`](59-de-eff-event-table.md)、[`114-maze-coordinates.md`](114-maze-coordinates.md)
子系統:**G. 地城與迷宮**
輸入:`MAZEDATA.BIN`、`DG*MAZE.SQZ`、`DE*EFF.BIN`

## 結論

迷宮的平面索引是 `Major × 81 + Minor`。三個來源的座標對**全部是 (Major, Minor) 順序**:

| 來源 | Major(乘 81)| Minor | 檢定 |
|---|---|---|---|
| `MAZEDATA.BIN` | 欄 2 | 欄 3 | 起點可通行 **12/12**(反過來 10/12)|
| `DE*EFF.BIN` | 欄 0 | 欄 1 | 觸發格是圖塊 19 的 **63/75**(反過來 **0/75**)|
| `GROUPS.DAT` | 位移 79 | 位移 81 | `81 × 位移79 + (位移81 − 1)`([`114`](114-maze-coordinates.md))|

**沒有「三個來源三種順序」這回事。**

## 1. 衝突在命名,不在算式

| 筆記 | 把乘 81 的那個索引叫做 |
|---|---|
| [`56`](56-maze-tile-classes-and-mazedata-columns.md) §3 | 「**列**」(欄 2 = 起始列)|
| [`59`](59-de-eff-event-table.md) §1 | 「**列**」(欄 0 = 觸發位置的列)|
| [`114`](114-maze-coordinates.md) | 「**欄**」(位移 79 = 迷宮內的欄)|

三篇的**算式都是對的** —— `56` 檢定的是 `grid[欄2][欄3]`、`114` 讀的是
`81 × 位移79 + …`,兩者一致。錯的只有中文名。

> **判準**:跨筆記引用座標時,**引用算式不要引用名字**。
> 「列」與「欄」在不同人(不同輪)的心智模型裡會對調,而**兩種讀法都畫得出圖**,
> 只是轉了 90 度 —— 沒有任何錯誤訊息。
>
> 這一輪就是照 `56` 的中文名寫進 [`spec/08`](../spec/08-maze-scene.md),
> 結果起點可通行率掉到 10/12。**是測試抓到的,不是眼睛。**

## 2. 兩個檢定

### `MAZEDATA` 的起點

```
rows[欄2][欄3]  →  可通行 12 / 阻擋 0
rows[欄3][欄2]  →  可通行 10 / 阻擋 2   (DG2 格值 7、DG6 格值 10)
```

12 是全部(第 13 筆是 `(0,0)` 佔位)。

### 事件表的觸發格

拿「目標 ≥ 100(有文字)的事件應該踩在圖塊 19」這條
([`60`](60-event-lookup-and-tile-19.md) §2)去檢定:

```
(欄0, 欄1)  →  63 / 75 踩在圖塊 19
(欄1, 欄0)  →   0 / 75
```

**0/75 是最乾淨的否證形式** —— 不是「比較差」,是「一個都不對」。

## 3. 實作端的處理:不用「欄/列」這兩個字

`engine/internal/original/maze.go` 一律用 `Major` / `Minor`:

```go
func (m *Maze) At(major, minor int) int  // major × 81 + minor
type MazeEntry struct { StartMajor, StartMinor int }
type Event struct { Major, Minor int; ... }
```

> **判準**:一個名字如果在專案裡已經被兩種相反的意思用過,
> **就不要再用那個名字**。改一個沒有歷史包袱的詞,比在每個呼叫端加註解便宜。

## 4. 尚未解開

| 項目 | 狀態 |
|---|---|
| 事件表欄 2 的 `0` 是什麼方向 | 未解([`60`](60-event-lookup-and-tile-19.md) §5)|
| 為什麼 63/75 而不是 75/75 | 其餘 12 筆踩在圖塊 25/26/27 等([`60`](60-event-lookup-and-tile-19.md) §2 的分佈),**不是誤差** |
| `MAZEDATA` 欄 4 只有 `{1, 3}` | 未解 |
