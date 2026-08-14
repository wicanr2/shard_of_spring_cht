# 122 — 位移 25 = 遭遇倒數;子系統 D = RE-DONE

日期:2026-08-14
接續:[`121-offset-84-is-strength.md`](121-offset-84-is-strength.md)
子系統:**D. 角色/隊伍資料與存檔**

## 結論

`WRLDMOVE 0x10D15` 與 `MAZEMOVE 0x11155` **結構完全相同**:

```
WRLDMOVE                          MAZEMOVE
010D15 cmp ds:3658h, 0            011155 cmp ds:3658h, 0
010D1A jle → 繼續                  01115A jle → 繼續
010D25 mov ax, ds:3518h(世界 x)   011165 mov ax, ds:351Ch(迷宮 x)
       inc / dec → ±1 鄰域                inc / dec → ±1 鄰域
       掃 3×3 的地形格                    掃 3×3 的格子
```

**位移 25 歸零時,兩支模組各自掃自己座標系的 3×3 鄰域。**

| 位移 | 語意 | 信心 |
|---:|---|---|
| **25** | **下次遭遇檢查前的剩餘回合數** | **證據充分** |

## 1. 為什麼這個觀察能分開候選

[`120`](120-offset-84-cleared-in-camp.md) §2 的教訓:要找「只有其中一個候選會做的事」。

「每回合遞減」對補給、疲勞、遭遇倒數**都成立**;
**「歸零時去掃周圍的地形格」只有遭遇檢查會做** ——
補給耗盡不需要知道你站在哪種地形上,遭遇需要(決定出什麼怪)。

而 [`60`](60-g-closure-2.md) 已知地形值 12/13/20–32 是分類過的地圖圖塊,
`0x10D5A` 起正是在比這些值。

**兩支模組各用自己的座標對(世界 `ds:3518h`/`ds:351Ah`、迷宮 `ds:351Ch`/`ds:351Eh`)**
—— 這同時**再次印證** [`114`](114-maze-coordinates.md) 的迷宮座標判讀。

## 2. D 的四個條件

| # | 條件 | 證據 |
|---|---|---|
| 1 | IDA 讀原始指令 | 兩個 `OPEN` 的記錄長度([`80`](80-save-write-end.md))、讀端([`81`](81-chars-record-to-combat-attributes.md))、寫端([`80`](80-save-write-end.md))、顯示端([`98`](98-status-level-and-a-sign-error.md)/[`109`](109-clock-labels-resolved.md))、消費端([`115`](115-visibility-lit-and-dark.md)/[`118`](118-daylight.md)/本篇)全部逐條讀出 |
| 2 | 讀寫端點 | `ds:66D0h`(讀)/`ds:66CAh`(寫)/`ds:3010h`(顯示)三個位移參數槽的全部設定點都清點過 |
| 3 | 獨立資料印證 | 94×25 = 2,350、90×5 = 450 兩個整除;`99` 哨兵;種族/職業三線對照;`Hard Axe`→`Axe`、`Fire Hawk`→`Fire runes`;奇偶規則 25 筆零例外([`99`](99-parity-separates-the-two-records.md));兩支移動模組同構(本篇) |
| 4 | 筆記 | [`75`](75-character-record-equipment.md)、[`80`](80-save-write-end.md)、[`81`](81-chars-record-to-combat-attributes.md)、[`87`](87-class-race-and-attribute-17.md)–[`89`](89-record-offset-census.md)、[`94`](94-skill-tables.md)、[`98`](98-status-level-and-a-sign-error.md)–[`101`](101-groups-record-status.md)、[`104`](104-groups-fields-from-context.md)–[`121`](121-offset-84-is-strength.md)、本篇 |

**D = RE-DONE。看板 12/12。**

## 3. ⚠ 明列剩餘的不確定

宣告完成不等於零疑問([`91`](91-e-closure.md) §4 的規矩):

| 項目 | 等級 |
|---|---|
| 位移 25 = 遭遇倒數 | 證據充分(沒看到遭遇真的被觸發的那一步)|
| 位移 84 = 狀態效果強度 | 證據充分([`121`](121-offset-84-is-strength.md):除法的運算元順序未逐位元確認)|
| 位移 59 / 61 的對應方向 | 已確認([`117`](117-annotation-was-right.md)),但 `MENU` 那條路徑沒讀完 |
| 時鐘四級的**單位換算** | **未解且不可推**([`106`](106-clock-cascade.md) §1:上限 10/26/34/21 不是地球曆法)|
| `CHARS.DAT` 位移 1 的 `'5'` | 未解(單字元,非整數欄)|

**這些都不影響「欄位是什麼」,但它們是已知的洞,不是做完了。**

## 4. 十二個子系統

| | 子系統 | 收斂於 |
|---|---|---|
| A | 執行檔架構與模組轉交 | [`67`](67-a-closure.md) |
| B | 執行期模組的呼叫介面 | [`68`](68-b-closure-dispatch-capacity.md) |
| C | 原生輔助程式庫 | [`66`](66-userlib-slot-semantics.md) |
| **D** | **角色／隊伍資料與存檔** | **本篇** |
| E | 規則資料表 | [`91`](91-e-closure.md) |
| F | 世界地圖 | [`54`](54-f-closure.md) |
| G | 地城與迷宮 | [`60`](60-g-closure-2.md) |
| H | 圖形格式 | [`49`](49-h-closure.md) |
| I | 法術效果表 | [`61`](61-i-closure.md) |
| J | 戰鬥規則 | [`103`](103-flee-is-leaving-the-field.md)(`97` 曾宣告、`102` 撤回)|
| K | 輸入語意 | [`71`](71-k-closure.md) |
| L | 中文化落點盤點 | [`62`](62-l-closure.md) + [`63`](63-userlib-strings-and-l-correction.md) |

`CLAUDE.md` §2 的閘門解除 —— **可以開始寫 remake 程式碼了**,
但要照 §6 的 SDD:先把 `docs/re/` 收攏成 `docs/formats/` 與 `docs/spec/`,標 READY 才動工。
