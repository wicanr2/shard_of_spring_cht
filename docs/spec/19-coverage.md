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

> 這一節列的是**引擎沒有那個畫面或那一步**的原版訊息 —— 不是措辭不同。
> 措辭的接線是 [`19-module-text.md`](19-module-text.md) §6.1 的 `wired` 欄,
> 進度由 `tools/check_module_text.py` 印出來。
>
> ⛔ **列出來不等於要補。** 補不補是另一個決定;其中幾項還卡在
> [`CLAUDE.md`](../../CLAUDE.md) §2 的閘門 —— 規則沒讀出來之前不能實作。

### 2.1 規則層寫好了、按鍵沒接:**改名**

`town.Rename()`(`engine/internal/town/roster.go:67`)已經實作並有測試,
但 `engine/` 底下**沒有任何呼叫端** —— 名冊畫面收不到改名這個指令。
對應原文 `Which character to rename ?` / `Please enter the new name (9 char max): `
(CHARUTIL:7/8)。

⚠ 這是 `H)unt` / `I)dentify` 那一類的第三例:**規則做完了、接線斷了,
而畫面上看不出來**(選單根本沒列出這個指令,所以「按不到」不像壞掉)。
原版的字串表是現成的檢查表,這一項就是對字串時掉出來的。

⚠ 名字長度上限**兩份原版資料自己打架**(10 vs 9),
見 [`19-module-text.md`](19-module-text.md) §4 —— 接改名之前要先裁決。

### 2.2 戰鬥缺三塊

| 缺什麼 | 原文(模組:索引)| 說明 |
|---|---|---|
| **單位檢視面板** | `Status: ` / `Speed:` / `Skill:` / `Strength:` / `Magical:` / `Armor rating:` / `Attacks with:` / `YES` / `no` / `(ESC, ` / ` scrolls)`(CMBT:179–192)| 原版可以在戰場上翻看每個單位的屬性。引擎的單位列只有名字/生命/速度 |
| **驅散不死生物 D)ispell** | CMBT:142–159(18 段)| 祭司職能:`the Priesthood` 的角色可以驅散,一場一次,對非不死生物回 `None of these monsters are undead!` |
| **攻擊附帶中毒** | `and is poisoned!`(CMBT:79)| 引擎的 `Attack()` 只算傷害,不會讓目標中毒 —— 而中毒狀態本身是有的(睡覺扣血、治療所解毒都吃它)|

⚠ 驅散那一塊**規則完全沒讀過**,不只是接線 —— 成功率、對哪些怪物有效、
「一場一次」的旗標存在哪,全部未知。照 §2 的閘門,補之前要先回 RE。

### 2.3 戰後與道具

| 缺什麼 | 原文 | 說明 |
|---|---|---|
| 拾取確認 | `Gold: N found` / `Do you take it?` / `(Y/N)`(CMBT:58–63)| 引擎的金幣直接進隊伍,沒有問要不要撿 |
| 道具損壞 | `Item Breaks !`(CMBT:22 / CAMP:127)| 魔法道具發動失敗時原版有機率弄壞它;引擎只回「法術失效!」 |
| 迷宮撿道具 | `found an item.` / `Your party is full of items, please discard some from Camp.`(MAZEMOVE:84/85)| 引擎的迷宮不會掉道具,也就沒有背包滿的分支 |
| 照明剩餘回合 | `You now have about N turns of light.`(CAMP:134/135)| 引擎有能見度,但沒有「照明會燒完」這件事 |

### 2.4 城鎮與營地的小分支

| 缺什麼 | 原文 |
|---|---|
| 旅店住**幾晚** | `Your rooms will cost N gold each night.  How many nights…(1-9, 0 exits) ?`(TOWN:30/31)—— 引擎一次只住一晚 |
| 一次買**幾份**口糧 | `…How many rations (1-9, 0 exits) ?`(TOWN:59)—— 引擎一次買一份 |
| 付款確認 | `That will cost N gold, Pay (Y/N)?`(TOWN:27/28)—— 引擎直接扣 |
| 智能不足 | `Not enough IQ !`(TOWN:54) |
| 睡覺時死亡 | `dies in the night.`(TOWN:81 / CAMP:137)—— 引擎會扣到 0,但不報這件事 |
| 不累就不睡 | `You are not tired`(CAMP:55) |
| 裝備的武器技能閘門 | `NO SKILL !`(CAMP:43)—— 引擎不檢查會不會用這件武器 |
| `DAZA REVELI` 大門 | CAMP:78–80(`The gate opens` / `Mumble, mumble, what spell did you say ?`)—— 對著大門唸咒語的機關 |

### 2.5 外殼與世界地圖

| 缺什麼 | 原文 |
|---|---|
| 全滅的隊伍不能選 | `Parties of dead characters are not allowed !!!`(MENU:99)—— `selectParty()` 只擋空隊伍 |
| 解散隊伍 | `No party # to disband !`(CHARUTIL:58/59) |
| 隊伍資訊頁 | `Saved in the month of the ` / `Currently in the ` / `maze: `(CHARUTIL:42/44/45)—— 存檔時間與所在地城 |
| 音效開關 | `Sound is now on.` / `Sound is now off.`(CMBT:52/53) |
| 拱門敘述與離場謝詞 | WRLDMOVE:18/19 —— 世界地圖邊界的兩段文字 |

## 3. 已補起來的(先前列在這裡,現在有了)

| 原本的缺口 | 現況 |
|---|---|
| 藥劑「用給自己還是給別人」 | 兩處都做了:營地 `G)ive`、戰鬥 `T)oss`(`engine/use_item.go`)|
| 「那是戰鬥用道具!」閘門 | `engine/camp_actions.go` 有了。⚠ `isCombatOnlyItem` 的判斷式是**猜的**,不是讀到的 |
| 升級的屬性成長與技能點 | 規則解出來了([`re/183`](../re/183-levelup-attribute-growth.md)):擲三次、每次五選一 +1、夾 20;技能點每級無條件 +1。畫面接上 `skill_alloc_scene.go` |
| `P)rogram Notes` 的製作群謝辭 | 移到標題畫面(MENU:93)|
| 離開遊戲的道別文字 | 接在結局畫面(MENU:148)|
| 施法游標只收 I/J/K/M | 方向鍵也收了 —— DOS 版自己的字串是 `Use arrow keys to position cursor.`(CMBT:110/111),手冊 p.34 講的是 Apple II 版 |
| `docs/spec/11` 說「P/C/U 未實作」 | 三項都已實作,見 [`16-camp-actions.md`](16-camp-actions.md) |

## 4. 未逐條查核的部分(誠實列出,不是查過沒寫)

- **`KEYPAD TEMPLATE`(CMBT:31)與周邊的方塊字元**(索引 32–40):原版是小鍵盤
  方向對照圖。引擎的按鍵表(`shell_scene.go`)有列戰鬥鍵位,但不是九宮格對照圖 ——
  沒有去讀 CMBT 那張圖的實際佈局,要重建視覺對照圖得回頭補這一步。
- **`(0 exits)` 與 `(ESC exits)` 的收單鍵不一致**:原版有些地方用數字 `0`、
  有些用 `ESC`,譯文如實分別記錄。引擎目前多半兩個都收,但**沒有逐畫面核對**。
- **原版把一句話拆成多段**的那些(例如 `is not a` + `wizard` + `and cannot` +
  `cast spells.`),引擎併成一句時用的是**另一個模組的同義句**(CAMP 版),
  所以 CMBT 那幾段留著沒接。這不是缺功能,是同一句話在原版有兩份。
