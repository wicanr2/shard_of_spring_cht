# 19-coverage — 模組字串覆蓋率稽核

對應 [`19-module-text.md`](19-module-text.md) 階段三。逐句問「引擎有沒有對應的東西」,
清冊來源是 `translations/source/module-*.tsv`(11 支模組,828 段,分類見該檔)。

⚠ **這份文件只列缺口,不補洞。** 補不補是另一個決定([`19-module-text.md`](19-module-text.md) 末段)。

⚠ **對應不是逐字比對。** 引擎的畫面文字是實作時自己寫的中文(`tools/check_ui_language.py`
回 0 條),F1 的目的是把措辭換成原版說的話——所以下面只要「功能/訊息類別存在」就算
**已覆蓋(措辭另計)**,不會因為中文寫法不同就列成缺口。真正列成缺口的是
**功能不存在**或**分支缺一段**。

## 0. 清冊範圍與一個已知的計數落差

`tools/dgroup_strings.py` 掃 11 支模組共 **828 段**,以此為準。

⚠ 先前出現過另外兩個數字,原因都查清楚了:
[`19-module-text.md`](19-module-text.md) 初稿的 **801** 是**解析器有洞** ——
用 regex 讀工具輸出時漏掉 27 行(含引號或跨行的字串);
`translations/README.md` 的 **822** 出自 [`re/62`](../re/62-l-localization-inventory.md)
的另一套盤點方法,不是同一支工具。

> **判準**:同一件事出現三個數字時,不要挑一個看起來合理的 ——
> 回去用**最直接的方法**再數一次(`grep -c "^ds:"`)。
> 錯的那個是自己的解析器,而那個洞在輸出裡完全看不出來。

分佈:

| 模組 | 段數 | `ui` | `data` | `na-disk` | `na-printer` | `na-debug` |
|---|---:|---:|---:|---:|---:|---:|
| START | 3 | 2 | 1 | | | |
| MENU | 149 | 15 | 54 | 80 | | |
| TOWN | 101 | 73 | 27 | | | 1 |
| CMBT | 193 | 124 | 69 | | | |
| CAMP | 152 | 101 | 36 | | 14 | 1 |
| MAZEMOVE | 92 | 28 | 62 | | | 2 |
| WRLDMOVE | 46 | 8 | 38 | | | |
| CHARUTIL | 71 | 30 | 38 | 2 | | 1 |
| MTEST | 5 | | | | | 5 |
| MIO2 | 13 | | | | | 13 |
| WSIO | 3 | | 2 | | | 1 |
| **合計** | **828** | **381** | **327** | **82** | **14** | **24** |

`ui` 381 段裡,11 段(TOWN 酒館傳聞,索引 62–72)已在既有的
`translations/module-text/TOWN-rumors.tsv` 譯完,本輪只新譯**其餘 370 段**
(見 `translations/module-text/*.tsv`)。

## 1. 已覆蓋(功能存在,措辭由 F1 譯文批次接手)

逐條核對引擎原始碼(`grep` 中文字串 + 讀呼叫端邏輯),下列原版訊息類別**都有
對應的引擎功能**,列出來是為了說明「稽核真的查過,不是漏查」:

| 原版訊息(模組:索引) | 引擎對應 | 檔案:行 |
|---|---|---|
| `Which party member do you wish to heal? (0 exits)`(MAZEMOVE:78) | 「要治療哪一位隊員?(0 離開)」 | `engine/maze_prompt.go:49` |
| `This pool is empty!`(MAZEMOVE:83) | 「這座池子已經乾了。」 | `engine/maze_prompt.go:45` |
| `Healing`/`Unpoison`/`Unbind`/`Ressurect`(TOWN:21-24)| `HealKind.String()`:「治療傷勢」/「解毒」/「解除束縛」/「復活」(3/4 用詞已與本輪譯文一致,`Healing`↔`醫療`一項用詞不同,純措辭)| `engine/internal/town/heal.go:19-29` |
| `No more room!`(TOWN:19,買道具背包滿)| `BuyResult.String()`:「背包已滿」 | `engine/internal/town/town.go:57-64` |
| `Character # to hunt ?` / `The hunt was (not) successful.`(CAMP:64-67)| `H)` 打獵,「XX 打獵成功!補給 +N」/「這趟沒有收穫。」 | `engine/town_scene.go:748-772` |
| `I)dentify`(CAMP:16 附近)| `I)` 鑑定(必定成功,已標未解見 `docs/spec/11`)| `engine/town_scene.go:622` |
| `Human`/`Troll`/`Dwarf`/`Elf`/`Gnome`(多模組)| `RaceName()`:「人類」「巨魔」「矮人」「精靈」「地精」 | `engine/internal/original/chars.go:159` |
| `You are the wrong class!` / `experience before gaining a level.`(TOWN:38,40-41,升級閘門)| `TrainResult.String()`:「這間訓練所不收這個職業」/「經驗還不夠」/「已經是最高等級」 | `engine/internal/town/train.go:24-32` |
| `Which spell do you wish to cast?`(CMBT:88-90,戰鬥施法)| `C)` 施法選單(`openCast`)| `engine/cast_scene.go:45` |
| `Character # to cast spell ?`(CAMP:75,營地施法)| `C)ast spell`(`campCastKey`)| `engine/camp_actions.go:115` |
| `Character # to use item ?`(CAMP:88,營地用道具;分支見 §2-1)| `U)se an item`(`campUseKey`)| `engine/camp_actions.go:282` |
| `Enter number of character you wish to print.`(CAMP:111,列印角色卡)| `P)rint`,改成畫面顯示(`campPrintKey`)| `engine/camp_actions.go:407` |
| A6 按鍵表整體 | 主選單 `P` 鍵開按鍵表覆蓋層 | `engine/shell_scene.go:364` |

⚠ **這條連帶訂正一份過期文件**:`docs/spec/11-town-camp-roster.md`(§「仍未實作的
三個」)寫「`P)rint char(s)`、`C)ast spell`、`U)se an item` —— 選單上列出來並標
「未實作」」。稽核當下讀原始碼,三項**都已經實作**(`docs/spec/16-camp-actions.md`
E1/E2/E3 記錄了接線過程,晚於 `11`)。`11` 那一段的敘述已經過期,建議專案負責人
排一輪把它改成指向 `16`,**本輪不動 `docs/` 其他檔**,先在此標記。

## 2. 缺口:功能不存在,不是措辭問題

### 2-1 ⚠ 藥劑「用給自己還是給別人」的分支,兩處都沒接

原版有**兩支不同的字串**,對應兩個不同情境:

| 情境 | 原文 | 模組:索引 |
|---|---|---|
| 營地用道具 | `Do you wish to use the potion on Y)ourself or G)ive it to another character?` | CAMP:92 |
| 戰鬥用道具 | `Do you wish to use the potion on Y)ourself or T)oss it to another character?` | CMBT:165 |

**營地**:`engine/camp_actions.go:333`(`useItem`)套用效果時**固定 `campUnit(*c)` 打自己
`&cUnit`**(見同檔 421 行呼叫處 `magic.Apply(s, invest, &cUnit, []*combat.Unit{&cUnit})`),
沒有選「自己/別人」這一步。這不是實作漏掉——`docs/spec/16-camp-actions.md` §3(E2)
描述的流程本身就是「選人 → 列出背包 → 選一件 → 套用」,**規格層級就沒有納入這個分支**。
相對原版,仍然是一個功能缺口,一併回報。

**戰鬥**:`engine/combat_scene.go` 的按鍵處理(`boardKey`)與 `engine/main.go` 的戰鬥
分派(216-260 行)只認 `方向鍵`/`A`(攻擊)/`C`(施法)/`Enter`/`ESC`——**沒有 `U`
或任何「使用道具」的入口**。CMBT 索引 162-192(`This character has no items to use!`
到 `(ESC, scrolls)`,共 31 段)整段功能在引擎裡**完全不存在**,不是措辭問題。

### 2-2 「那是戰鬥用道具!」的閘門沒有實作

原文 `That is a Combat Item!`(CAMP:100,一字不差對應 `docs/spec/19-module-text.md`
§1.1 表列的範例句)—— 全域搜尋 `engine/` 沒有任何「戰鬥」+「道具」的閘門字串,
`useItem()`(`engine/camp_actions.go:333`)對編號 ≤ `magic.MagicItemMin` 的道具
只回「不是魔法道具,什麼事也沒發生」,**不會擋下「這是戰鬥限定道具,不能在營地用」**
這一類。原版會先判斷道具類別再擋,引擎目前沒有這個判斷式。

### 2-3 升級只長生命/法力,原版的「屬性成長」與「新技能點」都沒實作

原版 TOWN 升級流程完整字串鏈(索引 40-52):

```
The Guild decides you need N experience before gaining a level.
You made a level!  You gain N hit points.
You also gain N Spell Points!
Stats are up by:
1 pt of *  ⟵ 屬性名重複套用,列出這次隨機漲的屬性
You have N points left.
Enter skill, (0 exits)
```

`docs/re/`、`docs/spec/` 全文搜尋 `Stats are up by` / `屬性提升` 只在原始字串
清冊(`generated-string-refs.json`/`generated-text-inventory.json`)裡出現,
**沒有任何規格文件討論過這個機制,也沒有標成已知未解項**——這是本輪稽核新發現的
缺口,先前沒人記錄過。

對照引擎 `Train()`(`engine/internal/town/train.go:44-71`):只動
`c.Level`、`c.MaxHP`/`c.HP`、`c.MaxSP`/`c.SP` 三組欄位,**不動任何 Trait
(速度/力量/智能/體能/技巧)**,也**不發新的技能點**——`SkillPts` 只在
`engine/internal/town/create.go:52`(角色創建)寫入,升級流程完全不碰它。
換句話說:「升級後可以再分配技能點」這件事,在角色創建**之後就再也碰不到**。

⚠ 這一項底層規則(隨骰決定漲哪個屬性、漲多少;技能點的發放量)完全沒讀過,
屬於**新的 RE 缺口**,不是單純接線問題——照 `CLAUDE.md` §2 的閘門,補這一段
之前要先回 RE 階段解出公式,不能直接猜一個數字實作。

### 2-4 兩則「已被規格明確裁定不做」的文字(非遺漏,列出以求「一句不漏報」)

| 原文 | 模組:索引 | 狀態 |
|---|---|---|
| `P)rogram Notes.` 的原內容(`Dedicated to Lori Proudfoot...` 製作人員名單)| MENU:93 | 按鍵位置被 `docs/spec/15-game-shell.md` §1.1 明確裁定改成「遊戲內按鍵表」,原文不會被搬進 remake |
| `Don, Leslie, and Martin thank you for playing the Shard of Spring.`(離開遊戲的道別文字)| MENU:148 | `docs/spec/15-game-shell.md` §9 驗收 2 只要求「`Q` 真的關掉程式」;`engine/shell_scene.go:255` 的 `Q` 直接 `ebiten.Termination`,沒有道別畫面。規格沒有明講要不要留這段道別,**先列出來,是否補一個「感謝遊玩」畫面待裁決** |

## 3. 未直接查核的部分(誠實列出,不是查過沒寫)

- **CMBT 的 `KEYPAD TEMPLATE`(索引31)與周邊 `[ESC]`/`[ATK]`/`[D][` 等方塊字元
  (索引32-40)**:原版是小鍵盤方向對照圖。引擎按鍵表(`shell_scene.go:364`)有列出
  戰鬥鍵位,但不是同一種「九宮格對照圖」版面——判定成「措辭/版面不同,功能覆蓋」,
  沒有另外去讀 CMBT 那張圖實際佈局,若未來要重建視覺對照圖需要回頭補這一步。
- **CAMP 的 `T)rade`/`R)eorder` 訊息措辭**細節(NO ROOM/OK 等)只核對了功能存在
  (`docs/spec/11-town-camp-roster.md` 已記錄兩者接線),沒有逐句比對引擎現在顯示
  的中文是否與本輪譯文一致——留給日後把 `translations/module-text/*.tsv` 接回
  引擎那一輪(§驗收5)處理。
- **`(0 exits)` 系列提示的離開鍵語意**(部分模組寫 `(ESC exits)`,部分寫
  `(0 exits)`)沒有逐一核對引擎目前用的是哪一種收單鍵——兩者在原版是不同輸入
  習慣(數字 0 vs ESC),translation TSV 已如實照原文分別記錄,是否要在引擎裡
  統一由後續實作決定。

## 4. 結論摘要

| # | 缺口 | 類型 |
|---|---|---|
| 1 | 藥劑使用的「自己/給別人」分支——營地固定打自己,戰鬥整支功能不存在 | 功能缺失(戰鬥) + 規格層級簡化(營地)|
| 2 | 「那是戰鬥用道具!」閘門沒實作 | 功能缺失 |
| 3 | 升級只長 HP/SP,屬性成長與新技能點沒有規則也沒有實作 | 新發現的 RE 缺口 + 功能缺失 |
| 4 | `docs/spec/11` 的「P/C/U 未實作」敘述已過期 | 文件落後於程式碼 |
| 5 | 離開遊戲的道別文字沒有畫面(是否要補,待裁決)| 規格未決 |

第 1–3 項在 `CLAUDE.md` §2 的閘門下,**要補之前都得先確認規則**(尤其第 3 項的
屬性成長骰法與技能點發放量,目前完全沒讀過,不能用猜的數字實作)。
