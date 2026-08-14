# Remake 引擎規劃 — **待定案**

> 前置條件已滿足:十二個子系統 RE-DONE([`re/122`](../re/122-d-closure.md))、
> 格式與規則規格標 READY([`00-index.md`](00-index.md))。
> `CLAUDE.md` §2 的閘門解除。
>
> ⚠ 本檔的 **§1 技術選型需要使用者定案**,其餘各節不依賴選型。

## 1. 技術選型(建議 + 需定案)

### 建議:**Go + Ebitengine**

| 判準 | 為什麼 |
|---|---|
| 跨平台 | 單一 `go build` 出 Windows / macOS / Linux;無執行期相依 |
| 中文顯示 | `text/v2` 直接吃 TTF/OTF,CJK 不需要額外處理 |
| 2D 圖塊 | 原生就是 2D immediate-mode,17×17 圖塊與 103×121 地圖是小 case |
| 團隊既有技能 | 本機開發環境與工作習慣已是 Go(見全域規則的開發環境節) |
| 資產管線 | `embed` 直接把翻譯 JSON / 字型打進執行檔 |

### 替代方案(若上面不合)

| 方案 | 換來的 | 代價 |
|---|---|---|
| **Godot 4** | 現成編輯器、場景系統、存檔 UI 好做 | 多一層執行期;專案體積大;版控友善度低 |
| **Rust + macroquad** | 體積最小、最快 | 團隊沒有 Rust 累積;CJK 字型處理要自己接 |
| **TypeScript + Canvas** | 瀏覽器直接玩,散布最容易 | 原版素材要走 HTTP;離線與存檔要另外設計 |

**⛔ 這一節沒定案之前不要開始寫引擎程式碼。**

## 2. 架構:九個場景,一份狀態

原版是九支獨立 EXE 靠 `retf` 互相轉交([`re/67`](../re/67-a-closure.md)),
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

原版文字分兩層([`re/62`](../re/62-l-closure.md)):`.DAT` 純文字、EXE 內字串常數。
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

**顯示層的已知問題**(daemon_winter 全部踩過)留到有畫面再決定:
中英混排寬度、點陣字型 vs TTF、原版畫布容不容得下中文。
⚠ **現在不要先決定畫布尺寸**(`CLAUDE.md` §7)。

## 6. 里程碑

| # | 目標 | 驗收 |
|---:|---|---|
| **M0** | 技術選型定案 + 空專案跑起來 | 開得起視窗 |
| **M1** | 資產轉換器:`.DAT` → JSON、圖塊 → PNG | 74 隻怪物、57 件物品、33 個法術的數值與圖能對上原版 |
| **M2** | 世界地圖場景:走得動、看得到地形 | 103×121 地圖畫得出來,座標與原版一致 |
| **M3** | 角色與存檔 | 五個預設角色讀得進來,狀態列顯示正確 |
| **M4** | 戰鬥:命中 / 傷害 / 先攻 / 死亡 | 用固定亂數種子跑,結果可重現 |
| **M5** | 迷宮場景 + 事件 | `.SQZ` 走得動,隱形觸發格與跨檔樓梯正確 |
| **M6** | 法術與道具 | 33 個法術的效果分類都有行為 |
| **M7** | 中文化上線 | 全部介面與地城文字顯示中文,無破格 |

**M1 之前先建 DOSBox oracle**(`CLAUDE.md` §6 的優先序 2)——
沒有 oracle 就沒有「跟原版一樣」的判準,M4 之後每一步都會變成猜。

## 7. 現在該做的三件事

1. **§1 的選型定案**(需要使用者)
2. **建 DOSBox 環境**(`tools/dosbox_run.sh`;⚠ 開機的手冊查詢題需要持有者提供答案)
3. **翻譯繼續**(已派工進行中)

前兩件互相獨立,第 3 件已經在跑。
