# 91 — 子系統 E(規則資料表)= RE-DONE

日期:2026-08-14
接續:[`90-monster-column-9-is-a-tier.md`](90-monster-column-9-is-a-tier.md)
子系統:**E. 規則資料表**

## 結論

`ITEMS` 欄6 與 `SPELLS` 欄3 都解出來了。
三個檔 **22 個欄位全部有語意**,**E = RE-DONE**。

## 1. `ITEMS` 欄6 也是雙重身分

和欄4 / 欄5 一樣([`76`](76-to-hit-formula.md) §2),欄6 對兩類物品是兩件事:

| 物品 | 欄6 | 判讀 |
|---|---|---|
| `Dagger` `Small axe` `Short sword` `Mace` `Morning star` `Broad sword` | 1, 2, 3, 4, 5, 6 | **種類代碼** |
| `Mace +1` / `Chain +1` / `Plate +1` | 4 / 3 / 5 | 與**基礎版同碼** |
| 藥水、火把、油燈 | **100** | |
| 戒指 / 法杖 / 權杖 | 7–15 | |
| 鑰匙、印璽、任務物品 | **0** | |
| `Teleporter` | 50 | |

程式碼(`0x17C92`,只在物品編號 > 26 時走到):

```
017C7F  mov bx, 1Ah / INT 3D:03    ; 1–26 的亂數
017C92  mov bx, [di+7780h]          ; 欄6
017C96  inc bx
017C97  cmp ax, bx
017C99  jl  → ds:930Ch = 1          ; 亂數 < 欄6+1 → 成立
```

**分母是 26**,所以:

- `100` = **必定成立**(藥水、火把 —— 用了一定有效)
- `0` = **永不成立**(鑰匙、任務物品 —— 沒有魔法效果)
- 7–15 = 約 27%–58%(戒指法杖 —— 會失敗)

**三群的判讀與物品性質完全相符**,而閘門 `> 26` 剛好把裝備擋在外面
—— 裝備的 1–6 不走這條路,那裡它是種類代碼。

**信心:證據充分。**

## 2. `SPELLS` 欄3:負值只出現在四個類別

| 類別 | 法術(括號內為欄4)| 判讀 |
|---:|---|---|
| 1 | `FIRE STORM`(+15)`TEMPEST`(+6)`HAIL STORM`(+8)| 群體傷害 |
| 2 | `FLAME STRIKE`(+25)`DEATH BLADE`(+18)`SPIRIT WRACK`(+26)| 單體重擊 |
| **3** | `WINGS OF VICTORY`(+1)`CHILL`(−1)`CLUMSINESS`(−3)| **某個屬性 ±** |
| **4** | `STRENGTH`(+1)`WEAKEN`(−1)| **力量 ±** |
| **5** | `HEAL`(+3)`BREATH OF LIFE`(+10)`COLUMN OF FIRE`(−3)`SWORD`(−5)| **生命值 ±** |
| **6** | `WINGS`(+3)`SLOW`(−3)| **速度 ±** |
| 7 | `ARMOR` `SANCTUARY` `ICE SHIELD` `FLAME SHIELD`(全 +1)| 防護 |
| 8 | `RESURRECT`(+25)| 復活 |
| 9 | `CURE POISON`(+60)| 解毒 |
| 10 | `MELT` `BREAK BONDS` `FREEDOM`(全 +1)| 解除束縛 |
| 11 | `CHAINS` `STILL AIR` `FREEZE`(全 +1)| 束縛 |
| 12 | `WIND WALK` `MAGIC TORCH` `CRYSTALIGHT`| 非戰鬥效用 |
| 13 | `TRANSFERENCE`(+3)| — |

**結構性的證據**:33 個法術裡有負欄4 的只有 5 個,
**而它們全部落在類別 3、4、5、6** —— 其餘九個類別**一個負值都沒有**。

**類別 3–6 是「屬性增減」,欄4 的正負決定增或減**;
其他類別的欄4 一律是正的威力值。這不是從名字看出來的,是從**符號的分佈**。

名字只用來**指認是哪個屬性**:
`STRENGTH`/`WEAKEN` → 力量、`WINGS`/`SLOW` → 速度、`HEAL` → 生命值。
⚠ 類別 3(`WINGS OF VICTORY` / `CHILL` / `CLUMSINESS`)**指不出是哪個屬性**,未解。

## 3. E 的四個條件

| # | 條件 | 證據 |
|---|---|---|
| 1 | IDA 讀原始指令 | 三個 `OPEN` + 讀取迴圈([`72`](72-e-file-formats-from-readers.md));欄位去處([`73`](73-monster-columns-to-combat-array.md));使用端([`74`](74-spell-and-item-columns.md)/[`76`](76-to-hit-formula.md)/[`83`](83-hp-is-attribute-3.md));欄6 的亂數比較(§1)|
| 2 | 讀寫端點 | 三個檔是**唯讀**資料,讀端全部讀出;沒有寫入端 |
| 3 | 獨立資料印證 | 檔案大小零誤差(2350/94、450/90、1256、1870、2664/36);法師/戰士 20/20 分群;三組等級碰撞;跨檔 28+6 交叉驗證 |
| 4 | 筆記 | [`72`](72-e-file-formats-from-readers.md)–[`74`](74-spell-and-item-columns.md)、[`82`](82-monster-columns-semantics.md)、[`83`](83-hp-is-attribute-3.md)、[`90`](90-monster-column-9-is-a-tier.md)、本篇 |

**E = RE-DONE。** 看板 **10/12**,剩 D、J。

## 4. ⚠ 明列剩餘的不確定

宣告 RE-DONE 不等於零疑問。以下三項仍未解,**但都是同一欄之內的細分**,
不影響「這一欄是什麼」:

| 項目 | 性質 |
|---|---|
| `SPELLS` 類別 3 是哪個屬性 | 三個法術名指不出來 |
| 類別 8 / 9 / 13 各只有一個法術 | **群內無對照,結構上無法再分** |
| `ITEMS` 欄6 對裝備的「種類代碼」指向什麼 | 圖示?技能?未對過 |

> **判準:宣告完成時要把剩餘疑問列出來,而不是讓它們消失在「已完成」裡。**
> 尤其「群內只有一個樣本」這種,**不是還沒做,是這條路走不到底** ——
> 要標成結構限制,否則下一輪會有人再花一次力氣。
