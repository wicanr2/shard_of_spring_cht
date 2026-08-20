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
| [`formats/08-rndmonst-bin.md`](../formats/08-rndmonst-bin.md) | **隨機遭遇表** 72 列 × 6 欄:區域、隻數上限、四個候選怪 |

## 遊戲規則與實作

| 規格 | 內容 |
|---|---|
| [`01-combat.md`](01-combat.md) | 單位陣列、先攻、行動點數、命中、傷害、死亡、逃跑 |
| [`02-magic.md`](02-magic.md) | 施法、系別門檻、狀態類法術、魔法道具 |
| [`03-engine-plan.md`](03-engine-plan.md) | 引擎架構與里程碑(**Go + Ebitengine**)|
| [`04-display-layout.md`](04-display-layout.md) | **顯示層與中文排版**(1024×768、美術 4×、避頭尾)|
| [`05-world-scene.md`](05-world-scene.md) | **世界地圖場景**(9×9 視野、地形值總表、移動、可通行性八條規則)|
| [`06-party-and-save.md`](06-party-and-save.md) | **隊伍、角色與存檔**(兩個檔的關係、成員槽、狀態欄)|
| [`07-combat-scene.md`](07-combat-scene.md) | **戰鬥場景**(單位陣列、先攻、可重現的亂數);傷害公式整段已讀通([`re/153`](../re/153-damage-formula-closed.md))|
| [`08-maze-scene.md`](08-maze-scene.md) | **迷宮與事件**(Major/Minor 座標、視野、事件三類、跨關卡、隨機遭遇)。⚠ **Major 是南北、Minor 是東西**([`re/224`](../re/224-maze-axes-major-is-north-south.md))|
| [`09-magic-items.md`](09-magic-items.md) | **法術與道具**(施法閘門、威力、狀態強度、道具發動 = `擲骰(100) ≤ 欄6`);⚠ 效果類別 3/13 未解 |
| [`10-localization.md`](10-localization.md) | **中文化上線**(轉檔期併入、破格的定義與預算)|
| [`11-town-camp-roster.md`](11-town-camp-roster.md) | **城鎮 / 商店 / 旅店 / 酒館 / 訓練所 / 治療所 / 營地 / 名冊 / 角色創造**;⚠ 屬性算式的兩個常數未解。升級門檻是公式不是手冊那張表([`re/223`](../re/223-experience-threshold-formula.md))|
| [`13-sound.md`](13-sound.md) | **聲音**(`PLAY` 巨集的解析與方波合成);⚠ 十五段樂譜的**用途**是位置上的推測 |
| [`14-remake-worklist.md`](14-remake-worklist.md) | **Remake worklist —— 狀態的單一真相來源**。RE 階段結束後「規則實作完 → 遊戲做完」之間的差距,含順序與理由 |
| [`15-game-shell.md`](15-game-shell.md) | **遊戲外殼**:標題、主選單、隊伍選擇、全滅與結局。⚠ 原版主選單**沒有「開新遊戲」** |
| [`16-camp-actions.md`](16-camp-actions.md) | 營地的 `C)ast`/`U)se`/`P)rint` 與**施法的投入點數** —— 全部實作完成 |
| [`17-scripted-fights.md`](17-scripted-fights.md) | **腳本戰鬥**:事件指定的怪物清單(`ds:372C + 2i`,哨兵 99)。533 = 2 × Great Dragon + Siriadne |
| [`18-save-format.md`](18-save-format.md) | **自己的存檔格式**(JSON,一檔 = 25 角色 + 5 隊伍 + 進度)。⚠ 順帶修掉「一次性事件會復活」|
| [`19-module-text.md`](19-module-text.md) | **模組內文本的中文化**(F1)。⚠ 主產品是「讓畫面說原版說的話」,**副產品是覆蓋率稽核** |
| [`12-combat-board.md`](12-combat-board.md) | **戰場**(格陣列 31 寬、畫面 15×15 視窗、行動點數 = 速度、只打面前那一格、走上外圈離場);高度 = 31([`re/218`](../re/218-four-named-assumptions-audited.md));怪物**施法**已接線(投入 = 單價 ×2、目標格 = 鎖定的那個人,[`re/226`](../re/226-monster-cast-invest-and-target.md))|

## ⚠ 實作前必讀的六條

1. **時鐘不是地球曆法**:月 1–21、日 1–34、時 **4–26**(重設值是 4)。
   ⚠ 手冊寫「22 個月」是**錯的**,而「26 小時」對上的是**上界不是長度**
   ([`formats/02`](../formats/02-groups-dat.md)、[`re/140`](../re/140-manual-stat-tables.md) §4)。
2. **`ITEMS.DAT` 欄 4/5/6 是雙重身分**,由呼叫端決定意義。
   讀之前先確定情境是「裝備」還是「魔法道具」。
3. **技能表由職業決定**。位移 42–51 的同一格,`Hero` 與 `Wizard` 是不同技能。
4. **兩套地圖的索引都是「第一個座標乘跨距」**:世界地圖 `x × 103 + y`
   (東西 121 × 南北 103)、迷宮 `欄 × 81 + 列`
   ([`formats/05`](../formats/05-world-map.md) / [`06`](../formats/06-maze.md))。
   ⚠ 世界地圖的兩軸**接反過一次**且**沒有症狀**——轉置的地圖照樣畫得出來
   ([`re/141`](../re/141-world-map-axes-were-transposed.md))。
5. **`.SQZ` 不是壓縮格式**,是文字 + 跑長;而 `MONST*.BIN` 的八張子圖是**交錯**的。
6. **背包是 10 格,空格的哨兵是 `99`**(不是 15 格、不是 0)。
   用 0 當空格會讓「找第一個空位」永遠找不到,**買東西一律回「背包已滿」**
   ([`re/144`](../re/144-created-record-exposes-layout.md) §3)。

## ⚠ 標 READY 不等於零疑問

每份規格結尾都有「未解」段。**目前還開著的洞**(2026-08-18,QA 三輪之後):

| 洞 | 出處 | 現在怎麼辦 |
|---|---|---|
| 法術效果**類別 13** | [`spec/09`](09-magic-items.md) | 不套用效果,訊息含「未解」 |
| `ds:1Fh` 的執行期間接寫入 | [`re/218`](../re/218-four-named-assumptions-audited.md) §4 | **靜態關不掉**(結構性)。要實跑一個取整看得出差別的地方才能定案 |
| 十五段樂譜的**用途** | [`spec/13`](13-sound.md) | 位置上的推測,不影響規則 |

**已經填掉的**(不要再照舊引用):一場遭遇的隻數與組成([`re/225`](../re/225-encounter-monster-count-anchor.md):`RNDMONST.BIN` 就是遭遇表)、怪物施法的投入與目標格([`re/226`](../re/226-monster-cast-invest-and-target.md))、傷害公式的兩個係數、擲骰面數(= 100)、
`CHARS.DAT` 位移 1、經驗值的位移與結算算式、魔法道具的發動判定、
屬性算式的形狀**與兩個常數**(`A` = 6、`B` = 2)、先攻每回合重排、
戰鬥屬性 14(= 怪物的法術系別)、法術效果類別 3(= 命中能力)、
迷宮的寶石謎題與治療池**觸發點**、戰後金幣的四個常數、
營地的 `H)unt`/`I)dentify`/`P)rint`、戰場高度(= 31)、
**升級門檻**(是公式不是手冊那張表)、**迷宮的兩軸**。

**實作時遇到這些,不要猜 —— 回 `docs/re/` 或回 IDA。**
