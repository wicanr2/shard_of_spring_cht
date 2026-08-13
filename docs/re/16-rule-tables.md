# 16 — 規則資料表:`MONSTERS` / `SPELLS` / `ITEMS`

日期:2026-08-13
接續:[`15-chain-filename-and-misalignment.md`](15-chain-filename-and-misalignment.md)
子系統:`CLAUDE.md` §2.2 的 **E. 規則資料表**

## 結論

三張表的**記錄結構已定案**,其中兩個欄位的**語意**取得了獨立來源的印證:
`MONSTERS.DAT` 的 `w5` 是怪物圖索引,`SPELLS.DAT` 的欄 1 是符文系別。

⚠ 依 `CLAUDE.md` §2.1,以下**尚未達到 RE-DONE** ——
缺條件 1 與 2(在 IDA 裡讀讀取端、用 xref 確認)。
本文的證據是**資料側 + 交叉印證**,足以支撐格式,不足以宣告語意定案。

## 1. `MONSTERS.DAT`:74 筆 × 36 bytes

| 位移 | 大小 | 內容 |
|---|---|---|
| `+0x00` | 16 | 名稱,空白補齊 |
| `+0x10` | 20 | 10 個 16-bit little-endian 欄位(`w0`–`w9`)|

### 記錄長的驗證

2,664 bytes 同時被 24 與 36 整除,**算術分不出來**
(這正是 kb 記的「算術對不代表切分單位對」)。
改用「每筆前 16 bytes 必須全是可列印 ASCII」當判準:

| 記錄長 | 筆數 | 名稱欄合格 | |
|---|---:|---|:-:|
| 24 | 111 | 37/111 | ❌ |
| **36** | **74** | **74/74** | ✅ |

### 欄位值域

| 欄 | 範圍 | 相異 | 語意 |
|---|---|---:|---|
| `w0` | 4–26 | 17 | 未解 |
| `w1` | 2–35 | 26 | 未解 |
| `w2` | 5–25 | 18 | 未解 |
| `w3` | 2–90 | 38 | 未解 |
| `w4` | 0–62 | 13 | 未解 |
| **`w5`** | **1–22** | **20** | **怪物圖索引**(見 §1.1)|
| `w6` | 0–9 | 9 | 未解 |
| `w7` | 15–5000 | 55 | 未解(值域像經驗值或金幣)|
| `w8` | 1–13 | 11 | 未解(值域像等級)|
| `w9` | 0–88 | 20 | 未解 |

### 1.1 `w5` = 怪物圖索引(兩層獨立印證)

**第一層 —— 數值範圍**:`w5` 全部落在 1–22,
而磁碟上剛好有 `MONST1.BIN`–`MONST22.BIN` 共 22 個檔。
兩個來源是「資料檔的位元組」與「目錄裡的檔名」,無共同錯誤來源。

**第二層 —— 語意分組**:`w5` 相同的怪物在主題上完全一致。

| `w5` | 共用該圖的怪物 |
|---:|---|
| 3 | 各級 Wizard |
| 5 | Skeleton、Zombie、Ghoul… |
| 10 | Hill Giant、Titan、Ogre、Mountain Giant |
| 13 | Spider、Giant Spider |
| 16 | Cobra、Death Adder、Pit Viper、Rattlesnake、Giant Snake |
| 22 | Baby / Small / Large / Great Dragon |

**範圍對得上只證明「可能是索引」,分組對得上才證明「是圖的索引」。**

## 2. `SPELLS.DAT`:34 筆 × 6 欄 CSV

`CR LF` 分隔,**34 行全部剛好 6 欄**(零例外)。

| 欄 | 內容 | 狀態 |
|---|---|---|
| 0 | 法術名 | 已確認 |
| **1** | **符文系別** | 見 §2.1 |
| 2 | 0–13 | 未解 |
| 3 | −5–60(**有負值**) | 未解 |
| 4 | 0–25 | 未解 |
| 5 | 命中訊息 | 已確認 |

### 2.1 欄 1 = 符文系別(5/5 對上,連順序都對)

依欄 1 分組後每一組主題完全一致,
而 `TOWN.EXE` 的技能字串裡剛好有五個符文系,順序也相同:

| 欄 1 | 該組法術 | `TOWN.EXE` 的技能 |
|---:|---|---|
| 1 | COLUMN OF FIRE、FLAME STRIKE、FIRE STORM、MELT、FLAME SHIELD、MAGIC TORCH | **Fire runes** |
| 2 | SWORD、CHAINS、DEATH BLADE、STRENGTH、BREAK BONDS、ARMOR | **Metal runes** |
| 3 | TEMPEST、STILL AIR、WINGS、FREEDOM、WIND WALK、BREATH OF LIFE | **Wind runes** |
| 4 | HAIL STORM、CHILL、SLOW、FREEZE、ICE SHIELD、CRYSTALIGHT | **Ice runes** |
| 5 | SPIRIT WRACK、WEAKEN、HEAL、RESURRECT、CURE POISON、SANCTUARY | **Spirit runes** |
| 0 | ET CETERA(僅 1 個) | 無對應,疑似雜項 |

兩個來源:`SPELLS.DAT` 的位元組、`TOWN.EXE` 的字串常數。無共同錯誤來源。

**旁證**:磁碟上有 `FIRESTRM.BIN` 與 `HAILSTRM.BIN`,
恰好對應 FIRE STORM(系 1)與 HAIL STORM(系 4)。
`WINDSTRM.BIN` 沒有同名法術,系 3 最接近的是 TEMPEST ——
**列為待查,不當佐證**。

## 3. `ITEMS.DAT`:57 筆 × 6 欄 CSV

**57 行全部剛好 6 欄。**

| 欄 | 內容 | 值域 | 狀態 |
|---|---|---|---|
| 0 | 正式名稱 | — | 已確認 |
| 1 | **未鑑定時的名稱** | — | **已確認**(`Dagger`→`knife`、`Mace`→`spiked club`)|
| 2 | 疑似價格 | 0–8331 | 遞增:匕首 2、短劍 15、闊劍 30、雙手劍 100 |
| 3 | 疑似傷害 | 0–99 | 同樣遞增 |
| 4 | 未解 | 0–26 | |
| 5 | 疑似類別或子索引 | 0–100 | 前 8 筆是 1,2,3,4,5,6,7,8 |

欄 1 對中文化很關鍵:**同一件道具有兩個名字**,
而 `docs/re/01` §4 盤點文字落點時沒有把這一層分出來。

## 4. 其他純文字檔

| 檔案 | 結構 |
|---|---|
| `TITLES.DAT` | 113 行,每行一個雙引號字串(選單標題)|
| `GROUPS.DAT`、`TOWNDATA.DAT`、`CHARS.DAT` | **不是 CRLF 分隔** —— 需另解 |

## 5. 尚未解開

| 項目 | 狀態 |
|---|---|
| `MONSTERS.DAT` 其餘 9 個欄位 | 未解 —— 要找讀取端 |
| `SPELLS.DAT` 欄 2/3/4(欄 3 有負值)| 未解 |
| `ITEMS.DAT` 欄 2–5 的確切語意 | 疑似,未證實 |
| `GROUPS`/`TOWNDATA`/`CHARS.DAT` 的結構 | 未解 |
| 以上全部的**讀取端** | 未解 —— 達到 RE-DONE 的必要條件 |
