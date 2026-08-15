# 實作規格索引 — 全部 **READY**

`CLAUDE.md` §2 的 RE 閘門已於 [`re/122`](../re/122-d-closure.md) 解除
(十二個子系統 A–L 全部 RE-DONE)。以下規格標 **READY**,可以據以實作。

## 資料格式

| 規格 | 內容 |
|---|---|
| [`formats/01-chars-dat.md`](../formats/01-chars-dat.md) | 角色記錄 94×25 |
| [`formats/02-groups-dat.md`](../formats/02-groups-dat.md) | 隊伍/存檔記錄 90×5、時鐘、光源 |
| [`formats/03-monsters-dat.md`](../formats/03-monsters-dat.md) | 怪物表 36×74、系別↔狀態 |
| [`formats/04-spells-items-dat.md`](../formats/04-spells-items-dat.md) | 法術 33 列、物品 57 列;⚠ 欄3 是**基準價**不是售價 |
| [`formats/05-world-map.md`](../formats/05-world-map.md) | 世界地圖 103×121、BSAVE 容器 |
| [`formats/06-maze.md`](../formats/06-maze.md) | `.SQZ` 解碼、迷宮 81 列、事件表 |
| [`formats/07-graphics.md`](../formats/07-graphics.md) | 圖塊 17×17、`MONST` 交錯、`DRAW` 巨集、調色盤 |

## 遊戲規則與實作

| 規格 | 內容 |
|---|---|
| [`01-combat.md`](01-combat.md) | 單位陣列、先攻、行動點數、命中、傷害、死亡、逃跑 |
| [`02-magic.md`](02-magic.md) | 施法、系別門檻、狀態類法術、魔法道具 |
| [`03-engine-plan.md`](03-engine-plan.md) | 引擎架構與里程碑(**Go + Ebitengine**)|
| [`04-display-layout.md`](04-display-layout.md) | **顯示層與中文排版**(1024×768、美術 4×、避頭尾)|
| [`05-world-scene.md`](05-world-scene.md) | **世界地圖場景**(9×9 視野、地形值總表、移動、可通行性八條規則)|
| [`06-party-and-save.md`](06-party-and-save.md) | **隊伍、角色與存檔**(兩個檔的關係、成員槽、狀態欄)|
| [`07-combat-scene.md`](07-combat-scene.md) | **戰鬥場景**(單位陣列、先攻、可重現的亂數);⚠ 傷害乘數與命中面數未解 |
| [`08-maze-scene.md`](08-maze-scene.md) | **迷宮與事件**(Major/Minor 座標、視野、事件三類、跨關卡)|
| [`09-magic-items.md`](09-magic-items.md) | **法術與道具**(施法閘門、威力、狀態強度、道具發動);⚠ 效果類別 3/13 未解 |
| [`10-localization.md`](10-localization.md) | **中文化上線**(轉檔期併入、破格的定義與預算)|
| [`11-town-camp-roster.md`](11-town-camp-roster.md) | **城鎮 / 商店 / 旅店 / 酒館 / 訓練所 / 治療所 / 營地 / 名冊 / 角色創造**;⚠ 屬性骰法未解(3d6 已排除)|
| [`12-combat-board.md`](12-combat-board.md) | **戰場**(15×15、行動點數 = 速度、只打面前那一格、走上外圈離場);⚠ 初始佈陣與怪物 AI 是佔位 |

## ⚠ 實作前必讀的五條

1. **時鐘的單位換算未解且不可推**。四級計數器的上限是 10 / 26 / 34 / 21,
   **不是** 24 小時 30 天 12 月。remake 照抄計數器,不要換算
   ([`formats/02`](../formats/02-groups-dat.md))。
2. **`ITEMS.DAT` 欄 4/5/6 是雙重身分**,由呼叫端決定意義。
   讀之前先確定情境是「裝備」還是「魔法道具」。
3. **技能表由職業決定**。位移 42–51 的同一格,`Hero` 與 `Wizard` 是不同技能。
4. **兩套地圖索引順序不同**:世界地圖 `y × 103 + x`、迷宮 `欄 × 81 + 列`
   ([`formats/05`](../formats/05-world-map.md) / [`06`](../formats/06-maze.md))。
5. **`.SQZ` 不是壓縮格式**,是文字 + 跑長;而 `MONST*.BIN` 的八張子圖是**交錯**的。

## ⚠ 標 READY 不等於零疑問

每份規格結尾都有「未解」段。已知的洞:
傷害公式的兩個係數、戰鬥屬性 14/18、法術效果類別 3、
`CHARS.DAT` 位移 1、先攻是否每回合重排。

**實作時遇到這些,不要猜 —— 回 `docs/re/` 或回 IDA。**
