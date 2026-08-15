# 174 — 實跑原版驗證營地的兩道閘門,四句訊息逐字對上

日期:2026-08-15
接續:[`166-camp-hunt-identify-print.md`](166-camp-hunt-identify-print.md)、[`139-oracle-reaches-gameplay.md`](139-oracle-reaches-gameplay.md)
子系統:**K. 輸入語意** / **D. 角色與存檔**
輸入:`game/sharspri/`(SHA-256 見 [`00-inputs.md`](00-inputs.md))、`tools/dosbox_run.sh`

## 結論

| # | 結果 | 信心 |
|---|---|---|
| 1 | `H)unt` 在野外**不擋**,選完人才判技能 —— 閘門順序與 [`166`](166-camp-hunt-identify-print.md) §2 一致 | **已確認(實跑)** |
| 2 | `I)dentify` 對戰士回 `That character is not a wizard.`,對法師走到 `Item to ID ?` | **已確認(實跑)** |
| 3 | 世界地圖狀態列直接印 **`Gold:` 與 `Provisions:`** —— 金幣與補給品都是**可觀測的** | **已確認(實跑)** |
| 4 | 出貨隊伍**沒有人會 `Hunting`**,所以打獵的收穫量在這份存檔上量不到 | **已確認** |

這是本專案第一次拿**執行中的原版**去驗營地規則。證據等級因此從第 2 級
(反組譯)升到第 1 級([`CLAUDE.md`](../../CLAUDE.md) §6)。

## 1. 怎麼跑的

```
tools/dosbox_run.sh "wait:8;key:Return;wait:3;key:Return;wait:3;type:L;wait:4;type:5;wait:6;\
                     type:C;wait:4;type:H;wait:3;shot:c1;type:1;wait:3;shot:c2"
```

開機的查表題按 Enter 就過([`139`](139-oracle-reaches-gameplay.md) §1),
`L` 載入名冊、`5` 選 PARTY #5,`C` 進營地(鍵盤模板左下角就是 `[C]`)。

## 2. `H)unt`:三個觀察

| 觀察 | 對應的靜態結論 |
|---|---|
| 在世界地圖進營地按 `H`,**沒有**印 `You're inside!` | `ds:3534 = 99`(不在迷宮)= 野外([`169`](169-encounter-zone-selects-the-monster.md) §4)|
| 直接跳出 `Character # to hunt ? (ESC exits)` | 室內檢查**在選人之前**([`166`](166-camp-hunt-identify-print.md) §2)|
| 選 1(Segrono)之後印 **`You don't have that skill.`** | 技能檢查**在選人之後**;Segrono 是戰士但技能第 9 格是 `0` |

**三個觀察分別對到閘門的三個位置**,順序完全吻合。

⚠ 出貨五人的技能旗標第 9 格**全是 `0`**,所以這份存檔量不到打獵的收穫。
要量 `INT 3D:33` 的參數與上限 `ds:6F10`,得先在遊戲裡練出 `Hunting`
(訓練所)或另建角色 —— **那是下一輪的事,本輪沒做。**

## 3. `I)dentify`:兩個觀察

| 選誰 | 畫面 | 對應 |
|---|---|---|
| 1 Segrono(戰士)| **`That character is not a wizard.`** | 第一道閘門是**職業**([`166`](166-camp-hunt-identify-print.md) §3)|
| 4 Fire Hawk(法師)| **`Item to ID ?`** | 通過職業 → 位移 86 → 狀態三關,走到選道具 |

Fire Hawk 的背包是空的(十格全 `99`),而原版**仍然問「要辨識哪一件」**
—— 與 [`166`](166-camp-hunt-identify-print.md) §3 讀到的「編號 `99` → 回選單」
一致:空格的處理在**選完之後**,不是先擋。

## 4. 狀態列直接印出兩個已解欄位

```
Gold: 75
Provisions: 20
```

- `Gold` = `GROUPS.DAT` 位移 19–22(MBF 單精度)
- `Provisions` = 位移 **23**([`formats/02`](../formats/02-groups-dat.md),打獵加的就是它)

**兩個都在螢幕上** —— 所以 [`152`](152-experience-settlement-formula.md) §2.3
的金幣四個常數、[`166`](166-camp-hunt-identify-print.md) §2 的打獵擲骰,
**都可以用「做一次 → 看數字」量出來**,不必再讀那些執行期變數。

⚠ 本輪**只確認可觀測性,沒有做任何量測**。金幣要先打贏一場,
打獵要先有人會 `Hunting`。

## 5. 明列剩餘的不確定

| 項目 | 狀態 |
|---|---|
| 打獵的收穫量與上限 | **未量** —— 需要一個會 `Hunting` 的角色(§2)|
| 戰後金幣的四個常數 | **未量** —— 需要打贏一場(§4)|
| `I)dentify` 的成功率 | **未量** —— 需要一件未辨識的道具 |
| 營地選單的其餘指令 | 本輪只按了 `H` 與 `I` |
