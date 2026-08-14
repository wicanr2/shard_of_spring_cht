# 97 — 子系統 J(戰鬥規則)= RE-DONE

日期:2026-08-14
接續:[`96-facing-and-action-points.md`](96-facing-and-action-points.md)
子系統:**J. 戰鬥規則**

## 結論

[`95`](95-no-flee-command.md) §4 留下的**兩個未排除的可能都排除了**,
「這個遊戲沒有逃跑指令」升為**證據充分**。

`CLAUDE.md` §2.2 給 J 的四項(命中 / 傷害 / 先攻 / 逃跑)全部有答案,
**J = RE-DONE**。

## 1. `D` 是 Dispell,不是逃跑

指令鏈的 `D`(`0x128F4` → `0x1699B`)拉出它串接的字串:

```
'does not have' + 'the Priesthood' + 'and cannot' + 'dispell.'
'has already'   + 'dispelled'      + 'this combat.'
'is not a'      + 'wizard'         + 'and cannot' + 'dispell.'
'None of these' + 'monsters are'   + 'undead!'
'Attempting'    + 'dispell'
'Ignores priest.'
```

**`D` = 驅散不死生物**。四個前提:

| 檢查 | 位址 |
|---|---|
| 行動點數 ≥ 3 | `0x1699B` `cmp ds:956Ch, 3` |
| 有 `Priesthood` 技能 | `0x169E2` 讀角色記錄位移 **51** 與 `'1'` 比 |
| 是 `Wizard` | 字串 `'is not a wizard and cannot dispell.'` |
| 這場戰鬥還沒用過 | 字串 `'has already dispelled this combat.'` |

位移 51 = 技能 10,而[`94`](94-skill-tables.md)的表 B 第 10 項正是 **`Priesthood`** ——
**兩邊獨立對上**。(表 A 第 10 項是 `Persuasion`,對 `Hero` 而言那一格是別的技能,
所以才要先檢查職業。)

## 2. `sub_1783E` 是戰場地圖更新,不是脫離

```
01783E  mov ax, 1Fh              ; 31
017841  imul word ptr ds:9348h
017847  add di, ds:9346h
01784D  mov bx, [di+6AD4h]       ; A%(i, j),第一維 31
```

**31 欄的二維陣列** —— 那是戰場格子,不是戰鬥流程的出口。
[`95`](95-no-flee-command.md) §4 的「走到邊緣自動脫離」在這條路上**沒有出現**。

## 3. 「沒有逃跑」的證據等級

比照 [`61`](61-i-closure.md) 判定三張法術圖未被載入時的做法
(**窮舉搜尋 + 正對照**):

| 窮舉 | 結果 |
|---|---|
| 三支模組的全部文字 | 無 `flee`/`run`/`escape`/`retreat`/`withdraw` |
| `CMBT` 對模組名字串的全部引用 | **只有兩個叢集**:全滅、戰勝結算 |
| `CMBT` 指令鏈的全部單字元 | `1 2 3 4 D ? / C S A U T Y N`,逐一追過 `D` 與 `Y/N` |
| 移動相關的三支子常式 | `sub_14195` / `sub_143E5` 轉身、`sub_1783E` 地圖 |

| 正對照 | 結果 |
|---|---|
| 同一套字串搜尋 | 找到 `Attacks with:` `cast spells.` `dies.` `Experience:` `Gold:` |
| 同一套引用掃描 | 找到兩張技能表、五個法術系別、Dispell 的六段訊息 |
| 同一套子常式追蹤 | 解出朝向、行動點數、先攻排序 |

**方法在同一批檔案上持續有輸出,而逃跑那一格是空的。**

**信心:證據充分。** ⚠ 不標已確認 —— 要到「已確認」得有
原版實跑的反證(`CLAUDE.md` §6 的 oracle 第 1、2 級),而 DOSBox 環境還沒建。

> **判準:「不存在」的結論永遠比「存在」貴一級。**
> 存在只要一個正例;不存在要窮舉**加上**證明工具沒瞎。
> 這一輪能收,是因為前面十幾輪的正對照剛好都用同一套工具做出來的。

## 4. J 的四個條件

| # | 條件 | 證據 |
|---|---|---|
| 1 | IDA 讀原始指令 | 命中([`76`](76-to-hit-formula.md))、傷害([`79`](79-alignment-resolved-damage-formula.md))、先攻([`93`](93-initiative.md))、朝向與行動點數([`96`](96-facing-and-action-points.md))、Dispell(§1)全部逐條讀出 |
| 2 | 讀寫端點 | 單位陣列 `ds:6822h` 的 289 處存取分類完成([`86`](86-scanner-fixed-conclusion-restored.md));順序表 `ds:6A7Ah` 的排序端與使用端都有 |
| 3 | 獨立資料印證 | 朝向編號對上 [`57`](57-g-closure.md) 的 `MAZEDATA`;速度對上 `MONSTERS.DAT` 欄1 的資料(蝙蝠 10 / 山巨人 4);`Armored skin` 的名稱對上它在傷害公式裡的減項;`Experience:` 結算對上欄8 |
| 4 | 筆記 | [`73`](73-monster-columns-to-combat-array.md)、[`76`](76-to-hit-formula.md)、[`79`](79-alignment-resolved-damage-formula.md)、[`83`](83-hp-is-attribute-3.md)–[`86`](86-scanner-fixed-conclusion-restored.md)、[`92`](92-attribute-15-16.md)–[`96`](96-facing-and-action-points.md)、本篇 |

**J = RE-DONE。** 看板 **11/12**,只剩 D。

## 5. ⚠ 明列剩餘

| 項目 | 性質 |
|---|---|
| 指令鏈的 `?` `/` `C` `S` `A` `U` `T` 語意 | **不在 J 的題目範圍**(那是 K 的輸入語意,而 K 已 RE-DONE 到「哪些鍵有效」的層級)|
| 屬性 8(命中 `+30` 條件)、12、14、18 的語意 | 4 / 19 未解 |
| `ds:9460h` / `ds:9464h`(傷害公式的兩個係數)| 未解 |
| 「沒有逃跑」的 oracle 級驗證 | 要等 DOSBox 環境 |

**這些不阻擋 remake 的核心規則實作**(命中、傷害、先攻、死亡、移動都齊了),
但它們是**已知的洞**,不是「做完了」。
