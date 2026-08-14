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
| 手冊刻意留英文的**專有名詞**(人名、地名)| **跟著留英文大寫**(§6 硬規則 4)|
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
2. **定長欄位的位元組上限**:`MONSTERS.DAT` 與 `TOWNDATA.DAT` 的名稱欄各 16 bytes
   (中文 2 bytes → 最多 8 字);`CHARS.DAT` 的角色名 10 bytes(玩家自取,不譯)。
3. 時間單位不做換算(見 [`docs/formats/02`](../docs/formats/02-groups-dat.md))。
4. **劇情專有名詞保留英文大寫,不音譯。** 這是精訊的做法(手冊全篇如此)。
   適用清單 —— **人名**:`SIRIADNE`、`DEVIR`、`GALIN`、`ELDRON`、`MURTHIN`、
   `CERCION`、`LOTHIAN`、`VANDIGUARD`、`EDRIN`、`BUGEM`;
   **劇情地名與家族**:`YMROS`、`RALITH`、`ISLANDA`、`EDRIN'S KEEP`、
   `BLACKFORT`、`MOONGLOW`。

   ⚠ 原版自己有 `Black Fort`(`MENU.EXE`)與 `Blackfort`(`DT51TEXT`/`TOWN.EXE`)
   兩種拼法([`docs/re/123`](../docs/re/123-menu-data-place-lists.md) §4),
   譯文採**該處原文的拼法**,不統一 —— 保留英文就沒有統一的必要。

   ### ⚠ 這條的邊界是我劃的,可推翻

   **本規則不含城鎮名與商店名**(§5 那 13 個城鎮與 61 家店維持中文)。

   理由:精訊手冊**從來沒提過任何一個城鎮或商店**,所以「精訊會怎麼處理它們」
   沒有直接證據;而手冊確實把 `YMROS`/`SIRIADNE`/`RALITH` 這類**敘述文裡的
   劇情名詞**留成英文,那才是有證據的部分。把規則擴到城鎮會讓
   世界地圖與 61 家店全部變回英文,等於這一塊不做中文化。

   例外:**店主名若在別處也以角色身分出現**,跟著本規則走 ——
   目前只有 `Vandiguard's` 一家(該名同時是地城名、墓主名、怪物名)。
   只出現在店名裡的店主(`Kor`、`Rolo`、`Zor`、`Corbin`、`Loven`、`Jorlor`、
   `Volir`、`Erlock`、`Balik`、`Red`)視為店名的一部分,維持中文。

   > 這是**判斷,不是精訊的明示**。若專案負責人認為城鎮名也該回英文,
   > 改這一段即可,`source/towndata.tsv` 有原文欄可直接回退。
5. ⛔ **玩家要打回去的字串不譯。** `CAMP.EXE` 檔案位移 17638 有字面字串
   `DAZA REVELI`,前後是 `Cast what spell?` 與失敗訊息
   `Mumble, mumble, what spell did you say ?`、成功訊息 `The gate opens` ——
   那是大門的通關咒語,`DT6TEXT.DAT` 的提示必須原樣保留。
   > remake 若要改成中文咒語,**提示與比對字串必須同一次改**。
6. **`fits` 欄量的是顯示欄寬,不是檔案位元組數。** 只有規則 2 的定長欄位受檔案限制;
   `TITLES` / `SPELLS` / `ITEMS` / `DT*TEXT` 都是變長文字檔,而 remake 由外部資源
   載入字串,檔內長度不構成約束。畫面欄寬預算取決於畫布尺寸,尚未定案。
7. **手冊的誤植不照抄**:「七首」→ 匕首、「隊份」→ 隊員、「睦過覺」→ 睡過覺、
   `WIND WAIK` → `WIND WALK`、`MAGIC TORCH` 表格的 `3` → `2`。
   抄錄檔 [`docs/manual/raw/`](../docs/manual/README.md) 保留原樣,**譯文用正確的**。
