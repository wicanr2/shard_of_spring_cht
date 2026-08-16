# 模組內文本的中文化 — **READY**

對應 [`14-remake-worklist.md`](14-remake-worklist.md) 的 **F1**。
接續 [`10-localization.md`](10-localization.md)(已上線的 439 段)。

## 1. 這一項到底在做什麼

⚠ **不是「把畫面變中文」** —— 引擎的畫面**早就是中文**了
(`tools/check_ui_language.py` 回 0 條)。那些中文是實作時**自己寫的**,
不是原版的措辭。

```
現在   引擎:「按編號選人」          原版:'Character # to hunt ? (ESC exits)'
之後   引擎:「要哪位隊員去打獵?(ESC 離開)」
```

**F1 是讓畫面說原版說的話。** 這與專案定位一致
([`CLAUDE.md`](../../CLAUDE.md) §1:文化資產保存,不是「能跑就好」)。

### 1.1 副產品比主產品更有價值:**覆蓋率稽核**

把原版每一句玩家看得到的話列出來,逐句問「remake 有沒有對應的東西」——
**沒有對應的就是缺的功能**。初步掃就看到這些:

| 原版的話 | 引擎現況 |
|---|---|
| `Do you wish to use the potion on Y)ourself or G)ive it to another character?` | ⚠ 用道具沒有問「給自己還是給別人」|
| `That is a Combat Item!` | ⚠ 沒有這道閘門 |
| `You need more spell points than that!` | 待查 |
| `The Guild decides you need N experience before gaining a level.` | 待查措辭 |
| `Your party is full of items, please discard some from Camp.` | ⚠ 待查 |
| `Which party member do you wish to heal? (0 exits)` | 治療池 —— 待查 |

⚠ 這正是 `H)unt` / `I)dentify` 按不到那一類的洞:**規則層做完了、接線斷了,
而畫面上看不出來**。原版的字串是現成的檢查表。

## 2. 範圍:801 段裡真正要翻的遠少於此

`tools/dgroup_strings.py` 掃 11 支模組得 **828 段**(`USERLIB` 另計,
它是 `bm` 模組,見 [`re/65`](../re/65-userlib-export-table.md))。分佈:

| 類 | 段 | 處理 |
|---|---:|---|
| `ui`(玩家看得到)| 381 | **要翻** |
| `data`(檔名、`PLAY` 巨集、格式字串)| 327 | ⛔ **不翻** |
| `na-disk`(換磁片 / 安裝)| 82 | ⛔ 不做 |
| `na-debug` | 24 | ⛔ 不做 |
| `na-printer` | 14 | ⛔ 改成畫面顯示 |

⚠ **段數曾經有三個版本**:本規格初稿寫 801、`translations/README.md` 記 822、
實際重數是 **828**。801 是我用 regex 解析工具輸出時**漏掉 27 行**(含引號或跨行的字串);
822 出自 [`re/62`](../re/62-l-localization-inventory.md) 的另一套盤點方法。
**以 `grep -c "^ds:"` 逐模組重數的 828 為準。**

> **判準**:同一件事出現三個數字時,不要挑一個看起來合理的 ——
> 回去用**最直接的方法**再數一次。我的 801 是**自己的解析器有洞**,
> 而那個洞在輸出裡完全看不出來。

124 句長句裡 **`MENU` 佔 47 句,幾乎全是換磁片與安裝** ——
而 [`15`](15-game-shell.md) §1.1 已經裁定 `R)estore Mazes` / `I)nstall Game` **不做**。

**真正玩家看得到的長句約 66 句**:`CAMP` 25、`TOWN` 22、`CHARUTIL` 9、
`MAZEMOVE` 5、`CMBT` 4、`WRLDMOVE` 1。

> **判準**:承諾翻譯之前先看分佈。「1,012 段」聽起來像一季的工作,
> 拆完之後主體是 **66 句 + 一批篩過的短語**。
> ⛔ 不要拿總數當工作量,那會讓人先放棄。

## 3. 分類(每一段都要有一個)

| 標記 | 意思 |
|---|---|
| `ui` | 玩家看得到,**要翻** |
| `na-disk` | 換磁片 / 安裝 / 硬碟設定 —— remake 沒有這些 |
| `na-printer` | 驅動印表機([`11`](11-town-camp-roster.md):改成畫面顯示)|
| `na-debug` | `MTEST` 與開發期訊息 |
| `data` | 檔名、模組名、`PLAY` 巨集、格式字串 —— **不是給人讀的** |

⚠ **`data` 要小心**:`MB T108 O3 L8 E F#GD` 看起來像亂碼,那是**音樂**
([`13`](13-sound.md) 的 `PLAY` 巨集)。⛔ 翻它會把樂譜毀掉。

## 4. ⚠ 一個原版自己打架的地方

| 來源 | 說法 |
|---|---|
| `TITLES.DAT` 第 89 列 | `"name (10 char)"` |
| `CHARUTIL` 字串 | `'Please enter the new name (9 char max):'` |

引擎現在用 **10**([`re/178`](../re/178-settings-from-three-sources.md) §4)。
⚠ **兩份都是原版資料,不能靠「哪個比較可信」裁決** ——
要去讀 `CHARUTIL` 實際擋在幾個字元,或在原版裡打 10 個字看它收不收。
**在裁決之前不要改引擎**,只把矛盾記在這裡。

## 5. 產出

```
translations/source/module-<模組>.tsv      原文清冊(DGROUP 位址 + 原文 + 分類)
translations/module-text/<模組>.tsv        譯文(只有 ui 那一批)
docs/spec/19-coverage.md                   覆蓋率稽核:哪幾句原版的話,引擎沒有對應
```

TSV 欄位沿用 [`translations/README.md`](../../translations/README.md) 的既有形狀
(`row / addr / original / orig_bytes / translation / trans_bytes / fits / note`),
再加一欄 `wired`(F3 的接線落點,見 §6.1)。

## 6. 翻譯規則

- 譯名**一律照** [`glossary.md`](../../translations/glossary.md)。⛔ 不要自己造新譯名;
  表上沒有的先加進 glossary 再用
- **保留原版的語氣**:`Aack!` 這種驚嘆、`Mumble, mumble` 這種戲謔要譯出來,
  ⛔ 不要抹平成公文
- 按鍵提示裡的字母**保留原文字母**:`(ESC exits)` → 「(ESC 離開)」,
  `Y)ourself` → 「Y) 自己」—— **玩家要按的是那個字母**
- 欄寬照 [`04`](04-display-layout.md) §5 的預算;不折行的固定欄要標 `fits`
- **標點是中文的,不是原版的 ASCII**:冒號用全形 `：`
  ([`06`](06-party-and-save.md) §7 驗收 7 訂的),逗號 `,`、驚嘆號 `!`、問號 `?`
  用半形,句號 `。`、頓號 `、` 用全形。⚠ 原版的定寬補位空白同樣不重現 ——
  移植後的版面用像素定位。`tools/check_module_text.py` 比對前會去頭尾空白,
  但**標點必須逐字相同**,所以 TSV 與引擎兩邊要用同一套

## 6.1 `wired` 欄:F3 的接線

`wired` 記的是**這段譯文由哪個檔案組進畫面**(分號分隔可以多個)。
填法見 `tools/set_wired.py`,驗證由 `tools/check_module_text.py` 做。

⚠ **檢查工具只驗「這串字有出現在那個檔案裡」**,驗不了「出現的地方是不是那個
呼叫點」——「是」「否」這種一兩個字的譯文在任何檔案裡都找得到,逐字比對永遠會過。
**填之前要自己讀過呼叫端**,那是 code review 的工作,不是工具的範圍。

原版把同一句話拆成好幾段字串(定寬視窗的排版殘留,例如
`is not a` + `wizard` + `and cannot` + `cast spells.`),而引擎組成一句 ——
這種情況下**那幾段共用同一個落點**。反過來,同一句話在 `TOWN`/`CAMP`/`CMBT`
各有一份(三支獨立 EXE 各自帶字串),引擎只有一份實作,**三段都指向它**。

⛔ **未接不等於沒做完。** 未接的 155 段混著三種:引擎還沒改用原版措辭、
原版拆段而引擎併句、以及**引擎根本沒有那個畫面**。第三種是
[`19-coverage.md`](19-coverage.md) 的工作,不是 F3 的 ——
⛔ 不要為了讓數字好看而在這裡補功能。

## 7. 驗收

| # | 條件 |
|---|---|
| 1 | 801 段**每一段都有分類**,沒有空白 |
| 2 | `data` 那一類逐條看過 —— `PLAY` 巨集**沒有被當成句子翻掉** |
| 3 | `ui` 那一批全部有譯文,且譯名與 glossary **零衝突**(有檢查) |
| 4 | 覆蓋率報告列出**引擎沒有對應的原版訊息**,一句都不漏報 |
| 5 | 引擎的措辭換成譯文之後,`check_ui_language.py` 仍然 0 條 |
| 6 | 破格檢查:不折行的欄位沒有超出 [`04`](04-display-layout.md) §5 的預算 |

⚠ 第 4 條是這一份最有價值的產出。⛔ **不要在稽核時順手把缺的功能補掉** ——
先列出來,補不補是另一個決定。
