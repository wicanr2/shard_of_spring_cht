# 地城與迷宮 — `DG*MAZE.SQZ` / `MAZEDATA.BIN` / `DT*TEXT.DAT` / `DE*EFF.BIN` — **READY**

## 1. `.SQZ` — **不是壓縮格式**,是文字 + 跑長編碼

⚠ 副檔名誤導。解碼規則從 `MAZEMOVE` 的解碼器讀出:

```python
CBASE, THRESH, ROWS = 42, 53, 81      # ds:964Eh, ds:964Ch, mov di,51h
v = raw[i] - CBASE
if v <= -29:            # CR → 換列
elif v >= THRESH:       # 跑長:重複 (raw[i+1] - CBASE) 次
    rows[-1] += [v - THRESH] * (raw[i+1] - CBASE); i += 2
else:
    rows[-1].append(v); i += 1
```

**格線 81 列**,索引 `欄 × 81 + 列`(與世界地圖的 `y × 103 + x` 不同,注意順序)。

⚠ **解碼器沒有欄數檢查** —— 有些列不足 81 格是合法的,不是殘差。

## 2. 圖塊值的特例

| 值 | 意義 |
|---:|---|
| 19 | **隱形觸發格**(畫成地板,踩到觸發事件)|
| 負值 | **跨檔案樓梯**(通往另一個 `.SQZ`)|
| 0 / 18 | 用得到但不繪製 |

## 3. `MAZEDATA.BIN` — 八欄

每個迷宮一筆,八個欄位(含入口座標、朝向、關聯的文字檔與事件檔編號)。
欄 4 的朝向用 `1`北 `2`東 `3`南 `4`西 —— **與戰鬥的朝向同一套編號**。

## 4. `DE*EFF.BIN` — 事件表

**106 × 5** 的表。查表迴圈:`cmp bx, [si-7710h]`(= `ds:0x88F0`)、`add ax, 6Ah`(106)。
每個迷宮一個檔(`DE5EFF.BIN`、`DE51EFF.BIN` …),由 `GROUPS.DAT` 的迷宮編號選檔。

## 5. `DT*TEXT.DAT` — 房間文字

純文字:**3 位數編號 + ASCII 敘述**,沒有加密沒有壓縮。
中文化的主要落點之一(見 [`re/62`](../re/62-l-localization-inventory.md))。

## 未解

`MAZEDATA` 八欄裡兩欄的語意;事件表 5 個欄位的完整對照。

出處:[`re/50`](../re/50-sqz-maze-format.md)、[`55`](../re/55-sqz-decoder-from-code.md)–[`57`](../re/57-g-closure.md)、
[`59`](../re/59-de-eff-event-table.md)、[`60`](../re/60-event-lookup-and-tile-19.md)
