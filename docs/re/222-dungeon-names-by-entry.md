# 222 — 地城名:`MENU.EXE` 的兩串 `DATA` 依序對上 `MAZEDATA` 的 13 個入口

> 輸入:`MENU.EXE`、`MAZEDATA.BIN`(SHA-256 見 [`00-inputs.md`](00-inputs.md))、
> 原版實跑五張截圖。信心:**已確認**(五個入口的名字直接量到,涵蓋兩串的頭、中、尾)。

補上 [`spec/14`](../spec/14-remake-worklist.md) §12-C 的「地城名還沒接」:
原版走進地城時,**畫面左上角印的是地城名**,重製版印的是 `地城 DG5`。

## 1. 名字在哪

`MAZEDATA.BIN` 的八欄裡**沒有名稱欄**([`137`](137-maze-coordinate-order.md))。
名字是 `MENU.EXE` 的兩串 `DATA` 敘述([`123`](123-menu-data-place-lists.md) §1):

| 串 | 檔案位移 | 內容 |
|---|---:|---|
| 2 | 10030 | ` Old Man in Cave,Swamp King,Black Fort,Edrin's Keep,Gate Keeper,Ralith` |
| 3 | 10103 | ` Rebel's Hideout,Murthin,Cercion,Lothian,Eldron,Vandiguard,Ralith.` |

6 ＋ 7 = **13**,而 `MAZEDATA` 正好 13 筆。

## 2. 對應:兩串接起來,索引 = 入口編號

拿原版當 oracle 逐一走進地城,讀左上角那一行。做法是把 `PARTY #5` 直接放到
入口旁邊那一格(`tools/oracle_patch.py place x,y,朝向`,改的是
`workplace/dosbox/game/` 的複本),再按一次方向鍵走進去:

```
python3 tools/oracle_patch.py place 53,68,1
tools/dosbox_run.sh "wait:8;key:Return;wait:3;key:Return;wait:3;\
                     type:L;wait:4;type:5;wait:6;key:Up;wait:5;shot:d7-in"
```

> **為什麼可以搬隊伍**:要量的是「站在入口按方向鍵印出哪個名字」,
> 而那個名字由入口編號決定 —— 與隊伍怎麼走到那一格無關。
> ⛔ 反過來,任何**吃路程**的東西(遭遇倒數、時鐘、食糧)不能這樣量。
> 不搬的話,從出發點 (8,8) 走到 (53,67) 要 217 個按鍵,而沿途的隨機遭遇
> 會把整條按鍵時間軸打亂 —— **這不是「省時間」,是「量得到與量不到」的差別**。

| 入口 | 世界座標 | 量到的名字 | 對應 | 截圖 |
|---:|---|---|---|---|
| 0 | (101,7) | `Old Man in Cave` | 串2[0] | `d0-in.png` |
| 2 | (23,18) | `Black Fort` | 串2[2] | [`147`](147-leaving-the-maze.md) |
| 5 | (94,50) | `Ralith` | 串2[5] | `d5-in.png` |
| 7 | (53,67) | `Murthin` | 串3[1] | `d7-in.png` |
| 11 | (45,98) | `Vandiguard` | 串3[5] | `d11-in.png` |

**兩串各自的頭、中、尾都命中** → 對應是「串2 接串3,索引就是入口編號」:

| 入口 | 名字 | 入口 | 名字 |
|---:|---|---:|---|
| 0 | Old Man in Cave | 7 | Murthin |
| 1 | Swamp King | 8 | Cercion |
| 2 | Black Fort | 9 | Lothian |
| 3 | Edrin's Keep | 10 | Eldron |
| 4 | Gate Keeper | 11 | Vandiguard |
| 5 | Ralith | 12 | Ralith(DG51)|
| 6 | Rebel's Hideout | | |

三條旁證與這張表一致:

- **入口 7–11 共用 `DG6MAZE` 與 `DT7TEXT`**,而它們的名字正是月華家族的
  五個人 —— `DT7TEXT` 把那一層寫成「月華家族的墓」([`123`](123-menu-data-place-lists.md) §2)。
- **兩串都以 `Ralith` 結尾**,而入口 5(`DG5`)與入口 12(`DG51`)是同一個
  地方的兩層 —— 這也解釋了 `DG51` 這個看起來像跳號的檔名:它是 `DG5` 的下一層。
- 入口 12 的世界座標是 (0,0):**它不從世界地圖進入**,與「Ralith 的下一層」相符。

## 3. 名字印在哪裡

原版印在**畫面最左上角、地圖框的上方**(`d7-in.png`),不是訊息框。
重製版的版面沒有那條位置(視野從 `Margin` 就開始),所以放進訊息框的第一行。

## 4. 仍未解

| 項目 | 狀態 |
|---|---|
| 讀這兩串的程式碼 | 沒讀。本篇靠位元組內容 ＋ 五次實跑,不靠反組譯 |
| **讀檔回到地城時名字怎麼來** | `GROUPS.DAT` 位移 83 只記**迷宮檔號**,而入口 0/1/4/6 共用 `DG1`、7–11 共用 `DG6` —— 檔號還原不出入口編號。原版怎麼記沒查;引擎取第一個相符的入口,名字可能不是原本那一個(`maze_scene.go` 有註記)|
| 入口 6 (86,35) 的地形值是 8 | 其餘 11 個入口都落在 24/25/27/28。這一個不是入口圖塊,沒查 |
