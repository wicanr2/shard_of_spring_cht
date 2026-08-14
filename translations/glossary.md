# 統一譯名表

> **權威來源:精訊資訊 1987 年官方中文說明書**([`docs/manual/`](../docs/manual/README.md))。
> 專案負責人裁定 2026-08-14:**以精訊翻譯為主。**
> 手冊有的詞一律照用,不另造;手冊沒有的才由本表補,並比照它的風格。

## ⚠ 先理解精訊當年做了什麼

**精訊賣的是英文版遊戲 + 中文說明書,遊戲畫面本身沒有中文化。**
所以手冊只譯**解說用的詞彙**(特性、技能、法術、職業),
凡是**遊戲畫面上會出現的名稱**(道具、怪物、城鎮、人名地名)一律留英文。

「以精訊為主」因此分三層落實:

| 情況 | 做法 |
|---|---|
| 手冊有中文 | **照用**,不得改寫(§1–§4)|
| 手冊刻意留英文的**專有名詞**(人名、地名)| 譯成中文,但**附原文對照**:`中文(English)`(§6 硬規則 4)|
| 手冊**完全沒碰**的(怪物名、道具名、城鎮名、商店名、地城敘述)| 由本專案譯,比照精訊風格(§5)|

第三層是本專案的主要工作 —— 精訊沒做,不代表不用做。

---

## 1. 職業與特性

### 職業(`CHARS.DAT` 位移 15)

| 原文 | 譯名 | 出處 |
|---|---|---|
| `WARRIOR` / `Hero` / `Fighter` | **戰士** | 手冊 p.10「一種是戰士(WARRIOR)」;p.49 的 H.P. 表同一個職業寫 `MAX-FIGHTER` |
| `WIZARD` | **巫師** | 手冊 p.10「另一種是巫師(WIZARD)」|

⚠ **DOS 版的資料檔用 `Hero`,`TITLES.DAT` 的選單寫 `A) Warrior`,
手冊寫 `WARRIOR`,H.P. 表寫 `FIGHTER` —— 四個詞是同一個職業,全部譯「戰士」。**
`MONSTERS.DAT` 的 `Lvl 1 Fighter` 是這個職業的 NPC,同樣譯「戰士」。

### 特性 TRAITS(手冊 p.11–13)

| 原文 | 譯名 |
|---|---|
| `SPEED` | 速度 |
| `STRENGTH` | 力量 |
| `INTELLECT` | 智能 |
| `ENDURANCE` | 體能 |
| `SKILL` | 技巧 |

⚠ **`SKILL` 有兩個意思**:當特性時是「技巧」,當技能清單時是「技能」。
手冊兩處都用,依上下文選。

### 其他數值(手冊 p.14–16)

| 原文 | 譯名 |
|---|---|
| `LEVEL` | 等級 |
| `HIT POINTS` / `H.P.` | 生命點數 |
| `SPELL POINTS` / `S.P.` | 法力點數 |
| `EXPERIENCE` | 經驗 |
| `PROVISIONS` | 食糧 |

## 2. 戰士技能(手冊 p.17,①戰士技能 WARRIOR SKILLS)

| 原文 | 譯名 | | 原文 | 譯名 |
|---|---|---|---|---|
| `SWORD` | 劍 | | `TACTICS` | 策略 |
| `AXE` | 斧 | | `ARMORED SKIN` | 護甲 |
| `MACE` | 釘頭鎚 | | `BERSERKING` | 技擊術 |
| `KARATE` | 空手道 | | `HUNTING` | 打獵 |
| `DARK VISION` | 夜視 | | `PERSUASIVENESS` | 說服能力 |

⚠ `MORNING STAR`(流星鎚)是 `MACE` 技能涵蓋的武器,不是獨立技能。

## 3. 巫師技能與法術系別(手冊 p.18–20)

| 原文 | 譯名 | | 原文 | 譯名 |
|---|---|---|---|---|
| `FIRE RUNES` | **火誌** | | `WEAPON LORE` | 武器知識 |
| `METAL RUNES` | **金誌** | | `POTION LORE` | 藥劑知識 |
| `WIND RUNES` | **風誌** | | `ITEM LORE` | 物品知識 |
| `ICE RUNES` | **冰誌** | | `MONSTER LORE` | 怪物知識 |
| `SPIRIT RUNES` | **精神之誌** | | `PRIESTHOOD` | 降魔術 |

系別編號(`SPELLS.DAT` 欄 2)= 火 1、金 2、風 3、冰 4、精神 5。

## 4. 法術 33 個(手冊 p.26–29)

括號內是**每一級的法力單價**,與 `SPELLS.DAT` 欄 5 **33/33 相符**
([`docs/re/125`](../docs/re/125-manual-confirms-spells.md))。

### 火誌 FIRE RUNES

| 原文 | 譯名 | 單價 |
|---|---|---:|
| `COLUMN OF FIRE` | 火柱 | 1 |
| `FLAME STRIKE` | 烈焰猛擊術 | 16 |
| `FIRE STORM` | 狂焰暴風術 | 10 |
| `MELT` | 溶解術 | 11 |
| `FLAME SHIELD` | 火焰盾牌術 | 4 |
| `MAGIC TORCH` | 魔火焰術 | 2 |

### 金誌 METAL RUNES

| 原文 | 譯名 | 單價 |
|---|---|---:|
| `SWORD` | 劍術 | 2 |
| `CHAINS` | 鐵鍊術 | 10 |
| `DEATH BLADE` | 死亡劍刃術 | 15 |
| `STRENGTH` | 強壯術 | 1 |
| `BREAK BONDS` | 破鐐銬術 | 11 |
| `ARMOR` | 護甲術 | 2 |

### 風誌 WIND RUNES

| 原文 | 譯名 | 單價 |
|---|---|---:|
| `TEMPEST` | 暴風雨術 | 6 |
| `STILL AIR` | 空氣凝結術 | 11 |
| `WINGS OF VICTORY` | 勝利之翼 | 1 |
| `WINGS` | 魔翼術 | 4 |
| `FREEDOM` | 自由術 | 13 |
| `WIND WALK` | 風行術 | 10 |
| `BREATH OF LIFE` | 生命之氣 | 5 |

### 冰誌 ICE RUNES

| 原文 | 譯名 | 單價 |
|---|---|---:|
| `HAIL STORM` | 冰雹暴風術 | 7 |
| `CHILL` | 寒風術 | 1 |
| `SLOW` | 遲緩術 | 3 |
| `FREEZE` | 冰凍術 | 9 |
| `ICE SHIELD` | 冰盾術 | 3 |
| `CRYSTALIGHT` | 水晶燈術 | 2 |

### 精神之誌 SPIRIT RUNES

| 原文 | 譯名 | 單價 |
|---|---|---:|
| `SPIRIT WRACK` | 滅魂術 | 20 |
| `WEAKEN` | 衰弱術 | 1 |
| `CLUMSINESS` | 愚蠢術 | 2 |
| `HEAL` | 醫療術 | 1 |
| `RESURRECT` | 復活術 | 25 |
| `CURE POISON` | 解毒術 | 9 |
| `TRANSFERENCE` | 移轉術 | 3 |
| `SANCTUARY` | 聖靈庇護術 | 3 |

⚠ 資料檔拼 `RESURRECT`,手冊 p.29 拼 `RESSURECT` —— **以資料檔為準**。

## 5. 手冊沒有的詞(本專案自造,比照精訊風格)

### 種族(`CHARS.DAT` 位移 14)

**手冊全篇沒有中文種族名**(p.9、p.49 兩張種族表都是純英文),所以由本表定:

| 碼 | 原文 | 譯名 |
|:-:|---|---|
| `H` | `HUMAN` | 人類 |
| `T` | `TROLL` | 巨魔 |
| `D` | `DWARF` | 矮人 |
| `E` | `ELF` | 精靈 |
| `G` | `GNOME` | 地精 |

### 狀態(`CHARS.DAT` 位移 38,0 起算)

| 編號 | 原文 | 譯名 |
|---:|---|---|
| 0 | `OK` | 正常 |
| 1 | `Poisoned` | 中毒 |
| 2 | `Bound` | 束縛 |
| 3 | `Still Air` | 凝滯 |
| 4 | `Frozen` | 冰封 |
| 5 | `D E A D` | 死亡 |

⚠ 3 對應法術 `STILL AIR`(精訊譯「空氣凝結術」),狀態欄位寬只有 2 字,故用「凝滯」。

### 物品

手冊解說文裡出現過的:`DAGGER` 匕首、`LEATHER ARMOR` 皮甲、`PLATE ARMOR` 板甲、
`TORCH` 火把、`LANTERN` 油燈、`POTION` 藥劑、`RING` 指環、`VIAL` 瓶。

⚠ 手冊 p.11 把 `DAGGER` 印成「**七首**」,那是「匕首」的誤植,不照抄。

其餘 57 個道具名手冊未譯,見 [`names/items.tsv`](names/items.tsv)。

### 怪物、城鎮、商店

手冊完全未譯。見 [`names/monsters.tsv`](names/monsters.tsv)、
[`source/towndata.tsv`](source/towndata.tsv)。

城鎮 13 個:翠綠村 `Green Hamlet`、珍寶平原 `Precious Plains`、格里昂 `Gleon`、
阿卡尼亞 `Arcania`、林安鎮 `Woodhaven`、特里諾 `Terynor`、密夸希德 `Myrquacid`、
阿瑟 `Athe`、蜘蛛灣 `Spider Bay`、詹斯林 `Janthrin`、亞特蘭提斯 `Atlantis`、
歐西安娜 `Oceana`、崔頓 `Triton`。

## 6. ⚠ 硬規則

1. **精訊譯名優先。** §1–§4 的詞一律照手冊,不得改寫成「更好聽」的版本。
   要改必須先確認手冊真的沒有那個詞。
2. **⚠ 定長欄位的 16-byte 上限對 remake 不成立。**
   `MONSTERS.DAT` / `TOWNDATA.DAT` 的名稱欄各 16 bytes、`CHARS.DAT` 的角色名 10 bytes
   —— 那是**改寫原版 `.DAT` 檔**才有的限制。本專案做的是 remake:
   原版資料檔只在資產轉換階段讀一次,轉成 JSON 進引擎
   ([`docs/spec/03`](../docs/spec/03-engine-plan.md) §3),
   **中文字串住在 JSON 裡,不住在 16-byte 欄位裡**。

   TSV 的 `trans_bytes` 欄保留,但只當**參考值**,不是閘門。
   真正的上限是**畫面欄寬**,預算見規則 6。

   > **判準**:同一個毛病犯第二次了(第一次是 `fits` 欄,見 `translations/README.md` §3)。
   > **每一條「上限」都要問「是誰規定的、對哪一條路線成立」** ——
   > patch 原版與 remake 的約束完全不同,而 RE 筆記是照 patch 路線寫的。
3. 時間單位不做換算(見 [`docs/formats/02`](../docs/formats/02-groups-dat.md))。
4. **專有名詞用「中文(英文)」對照格式。** 專案負責人裁定 2026-08-14:
   精訊沒譯的部分由本專案譯,專有名詞附原文方便玩家對照。

   ```
   希瑞雅妮(Siriadne)   拉利斯(Ralith)   艾德林要塞(Edrin's Keep)
   ```

   ### 適用範圍與兩條細則

   **適用於所有專有名詞**:人名、地名、**城鎮名、商店名**、地城名。
   ⚠ **普通名詞不加註** —— `Orc` 半獸人、`Spider` 蜘蛛、`Dagger` 匕首
   這類是種類不是名字,加註只會變吵。

   **① 同一段第一次出現才加註,之後只用中文。**
   一段房間敘述裡 `Siriadne` 可能出現三次,每次都加註會蓋掉正文。
   段落邊界 = TSV 的一列。

   **② 位元組長度不是限制**(規則 2)。
   `翠綠村(Green Hamlet)` 是 20 bytes、`希瑞雅妮(Siriadne)` 是 18 bytes,
   都超過原版的 16-byte 欄寬 —— **那個欄寬對 remake 不成立**。

   ### 適用清單

   **人名**:希瑞雅妮 `Siriadne`、迪維爾 `Devir`、蓋林 `Galin`、艾爾德隆 `Eldron`、
   莫辛 `Murthin`、瑟西恩 `Cercion`、洛西安 `Lothian`、范迪加德 `Vandiguard`、
   艾德林 `Edrin`、布金姆 `Bugem`

   **地名與家族**:伊姆羅斯 `Ymros`、拉利斯 `Ralith`、伊斯蘭達 `Islanda`、
   艾德林要塞 `Edrin's Keep`、黑堡 `Blackfort`、月華 `Moonglow`

   **帶稱號的複合形**:毀滅者迪維爾 `Devir the Destroyer`、
   善者蓋林 `Galin the Good`、灰髮艾爾德隆 `Eldron Greyhair`、瘋子莫辛 `Murthin the Mad`

   **城鎮 13 個 + 商店 61 家**:同樣用對照格式(專案負責人裁定)。
   例:`翠綠村(Green Hamlet)`、`奧德修斯酒館(Odysseus Pub)`。
   ⚠ 清單放**主視野**(61 欄)不放側欄(30 欄),所以名稱加註放得下
   —— 見 [`spec/04`](../docs/spec/04-display-layout.md) §5(原本的「未決」前提是錯的)。

   ⚠ 原版自己有 `Black Fort`(`MENU.EXE`)與 `Blackfort`(`DT51TEXT`/`TOWN.EXE`)
   兩種拼法([`docs/re/123`](../docs/re/123-menu-data-place-lists.md) §4)。
   加註時採**該處原文的拼法**,中文一律「黑堡」。

   ⚠ 原版把 `Destroyer` 拼成 `Destoyer`(`DT2TEXT` 201 段)。
   加註時照該處原文,**不要替原版訂正拼字** —— 玩家在畫面上看到的是錯的那個,
   加註的目的就是讓他對得上。
5. ⛔ **玩家要打回去的字串不譯。** `CAMP.EXE` 檔案位移 17638 有字面字串
   `DAZA REVELI`,前後是 `Cast what spell?` 與失敗訊息
   `Mumble, mumble, what spell did you say ?`、成功訊息 `The gate opens` ——
   那是大門的通關咒語,`DT6TEXT.DAT` 的提示必須原樣保留。
   > remake 若要改成中文咒語,**提示與比對字串必須同一次改**。
6. **`fits` 欄量的是顯示欄寬,不是檔案位元組數。**
   畫布已定案 1024×768([`docs/spec/04`](../docs/spec/04-display-layout.md)),
   欄寬預算:**敘述覆蓋層 60 欄 × 5 行、側欄清單 30 欄**(欄 = 半形單位,全形算 2)。

   | 用途 | 上限 | 現況 |
   |---|---:|---|
   | 地城敘述 | 300 欄(60×5)| 最長 207 欄 ✅ |
   | 商店 / 城鎮清單 | 30 欄 | 名稱最長 27 欄 ✅,但整列(編號+名稱+價格)要 36 欄 ⚠ |
   | 怪物名 | 30 欄 | 最長 22 欄 ✅ |

   ⚠ 商店清單的整列超出,解法在 [`spec/04`](../docs/spec/04-display-layout.md) §5,
   **不是縮短譯名** —— 名稱本身放得下。
7. **手冊的誤植不照抄**:「七首」→ 匕首、「隊份」→ 隊員、「睦過覺」→ 睡過覺、
   `WIND WAIK` → `WIND WALK`、`MAGIC TORCH` 表格的 `3` → `2`。
   抄錄檔 [`docs/manual/raw/`](../docs/manual/README.md) 保留原樣,**譯文用正確的**。
