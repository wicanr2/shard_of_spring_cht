# Remake 引擎規劃 — **READY**

> 前置條件全部滿足:十二個子系統 RE-DONE([`re/122`](../re/122-d-closure.md))、
> 格式與規則規格標 READY([`00-index.md`](00-index.md))、技術選型已定案(§1)。
> **`CLAUDE.md` §2 的閘門解除,可以開始寫引擎程式碼。**

## 1. 技術選型:**Go + Ebitengine**(已定案)

專案負責人裁定,2026-08-14。

| 判準 | 為什麼 |
|---|---|
| 跨平台 | 單一 `go build` 出 Windows / macOS / Linux;無執行期相依 |
| 中文顯示 | `text/v2` 直接吃 TTF/OTF,CJK 不需要額外處理 |
| 2D 圖塊 | 原生就是 2D immediate-mode,17×17 圖塊與 103×121 地圖是小 case |
| 團隊既有技能 | 本機開發環境與工作習慣已是 Go |
| 資產管線 | `embed` 直接把翻譯 JSON / 字型打進執行檔 |

曾評估但未採用:Godot 4(多一層執行期)、Rust + macroquad(團隊無累積)、
TypeScript + Canvas(離線與存檔要另外設計)。

⚠ **`go build` 一律走 docker**(`CLAUDE.md` §8),不裝系統 Go。

## 2. 架構:九個場景,一份狀態

原版是九支獨立 EXE 靠 `retf` 互相轉交([`re/67`](../re/67-a-closure-module-handoff.md)),
狀態存在 COMMON 區與兩個 `.DAT`。

**remake 不要模仿這個結構** —— 那是 1986 年 640K 的產物。改成:

```
GameState              ← 單一狀態物件(對應 CHARS.DAT + GROUPS.DAT 的欄位)
  ├ Party [5]Character
  └ World  WorldState  (時鐘、座標、光源、遭遇倒數 …)

Scene(介面)           ← 對應原版的九支 EXE
  ├ TitleScene         (START/MENU)
  ├ TownScene          (TOWN)
  ├ WorldMoveScene     (WRLDMOVE)
  ├ MazeMoveScene      (MAZEMOVE)
  ├ CombatScene        (CMBT)
  ├ CampScene          (CAMP)
  └ CharUtilScene      (CHARUTIL)

SceneManager           ← 取代 retf 轉交;場景切換不序列化,直接傳 *GameState
```

**存檔**只在玩家按 `S` 時發生(對應 `USERLIB` 槽 34),
寫**自己的格式**,不寫回原版 `.DAT`(`CLAUDE.md` §8:不覆蓋 `game/` 底下任何檔)。

## 3. 資料層:規格直翻成型別

| 規格 | 型別 |
|---|---|
| [`formats/01`](../formats/01-chars-dat.md) | `Character`(速度/力量/智力/體質/命中/HP/MP/裝備/狀態/等級/技能旗標/背包)|
| [`formats/02`](../formats/02-groups-dat.md) | `WorldState`(補給/遭遇倒數/四級時鐘/座標×2/朝向/光源/能見度)|
| [`formats/03`](../formats/03-monsters-dat.md) | `MonsterDef` × 74 |
| [`formats/04`](../formats/04-spells-items-dat.md) | `SpellDef` × 33、`ItemDef` × 57 |

**原版 `.DAT` 只在「資產轉換」階段讀一次**,轉成 JSON/TOML 進 repo
(不含原版素材,只含結構化的數值 —— 但**數值本身是原版資料**,
依 `CLAUDE.md` §1「不散布原版資料檔」,轉出的 JSON **一樣 gitignore**,
由玩家用自備的原版跑轉換器產生)。

⚠ **這一條要注意**:remake 的公開產出只有**引擎程式碼與翻譯文本**。
數值表要留在玩家本機。

## 4. 規則層:照 spec 實作,不即興

- 命中 / 傷害 / 先攻 / 死亡 / 逃跑 → [`spec/01`](01-combat.md)
- 施法 / 狀態 → [`spec/02`](02-magic.md)

**每一條公式在程式碼裡註明出處**(`// docs/spec/01 §4`),
規格改了才改程式碼。⚠ 遇到規格標「未解」的地方(傷害的兩個係數、
戰鬥屬性 14/18)**不要猜一個值填進去** —— 用具名常數 + `TODO` 指回規格。

## 5. 文字層:全部外部化

原版文字分兩層([`re/62`](../re/62-l-localization-inventory.md)):`.DAT` 純文字、EXE 內字串常數。
**remake 兩層都變成同一種東西** —— 一份 `strings/zh-Hant.json`,
由 `translations/` 底下的 TSV 產生。

```
translations/source/*.tsv     ← 原文清冊(agent 產出)
translations/names/*.tsv      ← 名稱類譯文
translations/dungeon-text/*.tsv ← 地城文字譯文
translations/glossary.md      ← 統一譯名(唯一真相)
        ↓ tools/build_strings.py
assets/strings/zh-Hant.json   ← 引擎讀這個
assets/strings/en.json        ← 原文,當 fallback 與對照
```

**顯示層已定案**,見 [`04-display-layout.md`](04-display-layout.md):
1024×768、美術 4× 整數放大、文字層 TTF 走原生解析度、避頭尾與中英混排規則。

## 6. 里程碑

| # | 目標 | 驗收 |
|---:|---|---|
| **M0** | 技術選型定案 + 空專案跑起來 | ✅ **完成**:視窗開得起來、版面五區塊座標經測試與截圖雙重驗證 |
| **M1** | 資產轉換器:`.DAT` → JSON、圖塊 → PNG | ✅ **完成**:74/33/57/61 數值經手冊交叉驗證(13 個測試),9 圖塊 + 5 大圖 + 22 怪物全數 dump 並肉眼比對 |
| **M2** | 世界地圖場景:走得動、看得到地形 | ✅ **完成**:座標經畫面驗證;圖塊來源 **100%**(`re/132`);可通行性八條規則已實作(`re/131`)|
| **M3** | 角色與存檔 | ✅ **完成**:出貨的 PARTY #5 讀得進來(座標/時鐘/金幣/補給全部來自存檔),狀態欄以中文繪出,存檔寫回**只動已解欄位**(逐位元組驗證)|
| **M4** | 戰鬥:命中 / 傷害 / 先攻 / 死亡 | ✅ **完成**:同種子逐回合傷害序列相同;三個結束條件分別用生命值與朝向;⚠ 傷害乘數 `k₁` 與命中面數未解,做成具名常數並顯示在畫面上 |
| **M5** | 迷宮場景 + 事件 | `.SQZ` 走得動,隱形觸發格與跨檔樓梯正確 |
| **M6** | 法術與道具 | 33 個法術的效果分類都有行為 |
| **M7** | 中文化上線 | 全部介面與地城文字顯示中文,無破格 |

⚠ **M1 的驗收判準是「對得上原版資料檔」,不是「對得上 DOSBox」。**
資料側的數值有手冊當獨立來源([`re/125`](../re/125-manual-confirms-spells.md):
法術 33/33、武器 8/8),不需要實跑。DOSBox 是輔助驗證不是卡控
(`CLAUDE.md` §6)。真正需要實跑裁決的是 M4 之後的行為問題
(先攻是否每回合重排、戰鬥是捲動還是縮放)。

## 7. M0 的範圍

只做三件,**不碰遊戲邏輯**:

1. Go module + Ebitengine 視窗 1024×768
2. 把 [`04-display-layout.md`](04-display-layout.md) §2 的五個區塊畫成框,
   驗證座標算術正確、沒有超出畫布
3. docker 建置包裝(`tools/go.sh`)—— `CLAUDE.md` §8:編譯一律走 docker

**不做**:讀原版資料、字型、輸入處理、任何規則。那些是 M1 之後。
