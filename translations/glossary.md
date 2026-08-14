# 統一譯名表

> **所有翻譯以本表為準。** 表上沒有的新詞,先加進來再用。
> 原文一律取自遊戲字串([`docs/re/88`](../docs/re/88-races-and-classes.md) §3:
> 專有名詞不從語意猜,只從遊戲自己的字串取)。

## 職業(`CHARS.DAT` 位移 15)

| 原文 | 譯名 | 備註 |
|---|---|---|
| `Hero` | **勇者** | ⚠ **不是**「戰士」——`Fighter` 在本作是**怪物名** |
| `Wizard` | **法師** | 玩家職業 |

## 種族(`CHARS.DAT` 位移 14)

| 碼 | 原文 | 譯名 |
|:-:|---|---|
| `H` | Human | 人類 |
| `T` | Troll | 巨魔 |
| `D` | Dwarf | 矮人 |
| `E` | Elf | 精靈 |
| `G` | Gnome | 地精 |

## 法術系別(`SPELLS.DAT` 欄2 = 狀態編號)

| 編號 | 原文 | 譯名 |
|---:|---|---|
| 1 | Fire runes | 火之符文 |
| 2 | Metal runes | 金之符文 |
| 3 | Wind runes | 風之符文 |
| 4 | Ice runes | 冰之符文 |
| 5 | Spirit runes | 靈之符文 |

## 狀態(`CHARS.DAT` 位移 38,0 起算)

| 編號 | 原文 | 譯名 | 顯示寬度上限 |
|---:|---|---|---|
| 0 | `OK` | 正常 | 原文 2 → 中文 2 字 |
| 1 | `Poisoned` | 中毒 | 8 → 2 字 |
| 2 | `Bound` | 束縛 | 5 → 2 字 |
| 3 | `Still Air` | 凝滯 | 9 → 2 字 |
| 4 | `Frozen` | 冰封 | 6 → 2 字 |
| 5 | `D E A D` | 死亡 | 7 → 2 字 |

## 技能 — `Hero` 表

| 原文 | 譯名 | | 原文 | 譯名 |
|---|---|---|---|---|
| Sword | 劍術 | | Tactics | 戰術 |
| Axe | 斧術 | | Armored skin | 硬皮 |
| Mace | 錘術 | | Berserking | 狂暴 |
| Karate | 徒手 | | Hunting | 狩獵 |
| Darkvision | 夜視 | | Persuasion | 說服 |

## 技能 — `Wizard` 表

| 原文 | 譯名 | | 原文 | 譯名 |
|---|---|---|---|---|
| Fire/Metal/Wind/Ice/Spirit runes | 見上「法術系別」 | | Potion lore | 藥劑學 |
| Weapon lore | 兵器學 | | Item lore | 器物學 |
| Monster lore | 怪物學 | | Priesthood | 聖職 |

## 介面詞

| 原文 | 譯名 | 顯示限制 |
|---|---|---|
| `Hit Pts.` / `H.P.` | 生命 / HP | 角色卡欄位 |
| `Spell Pts.` / `S.P.` | 法力 / MP | 同上 |
| `Level:` | 等級 | |
| `Experience` | 經驗 | |
| `Status` | 狀態 | |
| `Skills:` | 技能 | |
| `Provisions:` | 補給 | 隊伍層級 |
| `Gold:` | 金幣 | |
| `Visibility =` | 視野 | |
| `Hour:` / `Day:` / `In the Month` | 時 / 日 / 月 | ⚠ **單位換算未解**,不要寫「24 小時制」之類 |

## 人名(角色 / 頭目 / 墓主)

音譯統一,見下方硬規則 4。同一個名字在 `MONSTERS.DAT`(戰鬥畫面)、
地城敘述、`MENU.EXE` 的地城名單三處都會出現,**三處必須一致**。

| 原文 | 譯名 | 身分 |
|---|---|---|
| `Siriadne` | 希瑞雅妮 | 最終頭目 |
| `Devir (the Destroyer)` | (毀滅者)迪維爾 | 頭目,`Siriadne` 的盟友 |
| `Galin the Good` | 善者蓋林 | 被 `Devir` 奪權的前任統治者 |
| `Eldron Greyhair` | 灰髮艾爾德隆 | 終結 Moonglow 家族統治者,地城名兼墓主 |
| `Murthin (The Mad)` | (瘋子)莫辛 | Moonglow 四墓之一 |
| `Cercion` | 瑟西恩 | 同上 |
| `Lothian` | 洛西安 | 同上 |
| `Vandiguard` | 范迪加德 | 同上 |
| `Edrin` | 艾德林 | 地名 `Edrin's Keep` 的來源 |
| `Bugem` | 布金姆 | 怪物 |

## 地名

⚠ **`Ralith` 與 `Ymros` 這兩個詞出現次數最多**,譯名一旦動到要全檔重掃。

| 原文 | 譯名 | 出處 |
|---|---|---|
| `Ymros` | 伊姆羅斯 | 起始大陸(`DT0TEXT`)|
| `Ralith` | 拉利斯 | 最終地城;`MONSTERS.DAT` 有 `R A L I T H` → **拉 利 斯**(逐字間隔要保留)|
| `Old Man in Cave` | 洞中老者 | 地城名單(`MENU.EXE`,[`docs/re/123`](../docs/re/123-menu-data-place-lists.md))|
| `Swamp King` | 沼澤之王 | 同上,普通名詞照譯 |
| `Black Fort` / `Blackfort` | 黑堡 | ⚠ **原版兩種拼法**,中文統一(見 [`123`](../docs/re/123-menu-data-place-lists.md) §4)|
| `Edrin's Keep` | 艾德林要塞 | 地城名單 |
| `Gate Keeper` | 守門者 | 地城名單;也是該地城的頭目稱謂 |
| `Rebel's Hideout` | 反抗軍藏身處 | 地城名單 |
| `Moonglow` | 月華 | 家族名(`Moonglow clan` → 月華家族)|
| `Islanda` | 伊斯蘭達 | `TOWN.EXE` 的背景敘述 |

### 城鎮(`TOWNDATA.DAT`,13 個,定長 16 bytes)

城鎮名在**每一筆商店記錄裡都重複一次**(61 筆),還會出現在世界地圖上,
所以這 13 個譯名一動就要全檔重掃。61 間商店的譯名見
[`translations/source/towndata.tsv`](source/towndata.tsv)。

| 原文 | 譯名 | | 原文 | 譯名 |
|---|---|---|---|---|
| `Green Hamlet` | 翠綠村 | | `Janthrin` | 詹斯林 |
| `Precious Plains` | 珍寶平原 | | `Athe` | 阿瑟 |
| `Gleon` | 格里昂 | | `Spider Bay` | 蜘蛛灣 |
| `Arcania` | 阿卡尼亞 | | `Atlantis` | 亞特蘭提斯 |
| `Woodhaven` | 林安鎮 | | `Oceana` | 歐西安娜 |
| `Terynor` | 特里諾 | | `Triton` | 崔頓 |
| `Myrquacid` | 密夸希德 | | | |

### 商店店主的人名(音譯,**尚未跨檔核對**)

目前只在 `TOWNDATA.DAT` 出現過一次,若之後在對話或劇情文字裡再出現,以本表為準:

`Kor` 柯爾、`Rolo` 羅洛、`Zor` 佐爾、`Corbin` 柯賓、`Loven` 洛文、
`Jorlor` 卓洛、`Volir` 沃利爾、`Erlock` 厄洛克、`Balik` 巴利克、`Red` 瑞德。

## ⚠ 硬規則

1. **`Hero` ≠ 戰士。** 遊戲裡的 `Fighter` 是怪物(`Lvl 1 Fighter`),兩者必須用不同的詞。
   ⚠ 連帶陷阱:`TITLES.DAT` 的職業選單寫 `A) Warrior`,但它**對應的是 `Hero` 職業**,
   要譯「勇者」。地城敘述裡小寫的 `warrior(s)`(雕像、先王)是普通名詞,譯「戰士」正確。
2. **`MONSTERS.DAT` 的名稱欄是定長 16 bytes** —— 中文佔 2 bytes,所以**最多 8 個中文字**。
   `TOWNDATA.DAT` 的城鎮名與商店名同樣是定長 16(每筆 45 = 16 + 16 + 13)。
   `CHARS.DAT` 的角色名是定長 10(玩家自取,不翻)。
3. 時間單位不做換算(見 [`docs/formats/02`](../docs/formats/02-groups-dat.md))。
4. **專有名詞一律音譯,不保留英文。** 同一個名字在戰鬥畫面顯示中文、
   在地城敘述卻是英文,對玩家是同一個角色分裂成兩個。**唯一例外見規則 5。**
5. ⛔ **玩家要打回去的字串不譯。** `CAMP.EXE` 檔案位移 17638 有字面字串
   `DAZA REVELI`,前後是 `Cast what spell?` 與失敗訊息
   `Mumble, mumble, what spell did you say ?`、成功訊息 `The gate opens` ——
   那是**大門的通關咒語**,`DT6TEXT.DAT` 的提示文字必須原樣保留 `DAZA` / `REVELI`,
   否則玩家照譯文打進去會開不了門。
   > remake 若要改成中文咒語,**提示與比對字串必須同一次改**,不可只改一邊。
6. **`fits` 欄量的是顯示欄寬,不是檔案位元組數。** 只有規則 2 列出的定長欄位
   才受檔案位元組限制;`TITLES.DAT` / `SPELLS.DAT` / `ITEMS.DAT` / `DT*TEXT.DAT`
   都是變長文字檔,而 remake 由外部資源載入字串,**檔內長度完全不構成約束**。
   ⚠ 畫面欄寬預算取決於畫布尺寸,那一項尚未定案
   ([`docs/spec/03`](../docs/spec/03-engine-plan.md) §1),所以目前只記錄超出量,不強行縮短。
