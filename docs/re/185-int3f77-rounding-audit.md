# 185 — 全專案盤點:哪些算式因為「`3F:77` 是四捨五入」而要改

日期:2026-08-16
接續:[`184-levelup-hp-sp-growth.md`](184-levelup-hp-sp-growth.md) §5/§7
子系統:**跨全部**(這是一次稽核,不是新解一個子系統)
輸入:13 支執行檔(SHA-256 見 [`00-inputs.md`](00-inputs.md))

## 為什麼要做這一輪

[`184`](184-levelup-hp-sp-growth.md) §5 定案 `INT 3F:77` 是**四捨五入**。
`docs/re/` 有一批算式寫成 `INT(…)`,而那個 `INT` 是**當時假設截尾**才寫上去的。
兩種要分清楚:

- **A 類**:原版真的有 `INT 3D:03` → 算式不變
- **B 類**:只有 `3F:77` → **算式錯了**,`INT(x)` 要改成 `四捨五入(x)`

## 分母與結果

**72 處 `3F:77`,8 支模組**(CMBT 38、TOWN 13、CAMP 7、CHARUTIL 4、WSIO 4、
MAZEMOVE 3、WRLDMOVE 2、USERLIB 1);`CD 3D 03` 共 61 處。

| 類別 | 數量 |
|---|---|
| **A(有 `INT()`,算式不變)** | 51 |
| **B(沒有,算式要改)** | 21 |

## 1. ⭐ 方法:兩件會讓整份掃描讀錯的事

### 1.1 模組有兩種排版,CMBT 那一族每個 `CD 3F xx` 後面接 `90 90`

| 排版 | 模組 | 內嵌運算元在 |
|---|---|---|
| **5-byte 槽**(`CD 3F xx 90 90`)| CMBT / MENU / MAZEMOVE / WSIO / MIO2 / MTEST | `+5` |
| **無填充** | TOWN / CAMP / WRLDMOVE / CHARUTIL / USERLIB | `+3` |

證據:CMBT **1016/1016** 個 `CD 3F` 後面有填充,TOWN **0/533**;
CMBT 內嵌索引的 `+3` 位元組 **77/77** 都是 `0x90`,`+5` 全落在變數編號區。

⚠ **沒有這一條,整個 CMBT 都會讀錯** —— 稽核的第一版把 CMBT 判成 A=29,
修正後 A=32。

我自己在 `CMBT 0x136C5`–`0x136DE` 逐條看過,`90 90` 確實每一個 `CD 3F xx` 後面都有。

### 1.2 至少 18 個 `3F` 索引是 4 bytes

`3F: 71 72 85 86 8D 8E 95 96 9D 9E A5 A6 AB AD AE B5 CA CB` ——
判定法是從 `BRUN30` 派工表逐支常式看它有沒有 `call sub_11EC3`
(取內嵌運算元,[`184`](184-levelup-hp-sp-growth.md) §7),再用
「模組裡該位置的位元組是否全落在 `0x80–0x86`」回驗。

⚠ **`INT 3Dh` / `3Eh` 一個都沒有 4-byte 的** → `CD 3D 03` 固定 3 bytes。
⚠ `3F:8F` / `3F:93` 的 `lodsw` 是**假陽性**:那是在 `call sub_11F6B` 之後、`si` 來自 `bx`。

### 1.3 正對照

| 對照 | 預期 | 結果 |
|---|---|---|
| [`183`](183-levelup-attribute-growth.md) §3 屬性成長(`TOWN 0x11343 → 0x11346`)| A | ✓ |
| [`152`](152-experience-settlement-formula.md) §1.3 經驗除法(`CMBT 0x12D03 → 0x12D08`)| A | ✓ |
| [`184`](184-levelup-hp-sp-growth.md) 自己那三處 | B | ✓ |

## 2. B 類:算式要改的,按對玩家的影響排序

通式:`round(RND × N + C)` 的值域是 `C … C+N`(兩端各半權重)、均值 `C + N/2` ——
比 `INT(RND × N) + C` **平均多 0.5、上限多 1**。

| # | 位址 | 原本記成 | 改正後 | 玩家看得到的差別 |
|---|---|---|---|---|
| 1 | `CHARUTIL 0x10D18` | [`156`](156-attribute-roll-shape.md) §1:重擲「指令序列逐條相同」| **不同** —— 重擲**沒有** `3D:03` | 首擲 2–13 均值 7.5、**重擲 2–14 均值 8.0**;⭐ **按 ESC 重擲數學上嚴格有利**,而且只有重擲擲得出 14 |
| 2 | `CMBT 0x136DA` | [`153`](153-damage-formula-closed.md) §1 / [`154`](154-die-is-d100.md):`INT(RND×100)+1` | `round(RND×100+1)` → **1–101** | **命中率 = (命中值 − 0.5) ÷ 100**,比原記載低 0.5 個百分點 |
| 3 | `CMBT 0x150AF` | [`173`](173-spell-pipeline-and-the-13-way-dispatch.md) §1:`ds:9A48 ← INT(商)` | `round(欄4 ÷ 欄5)` | 法術等級的商小數 ≥ .5 時**多一整級**([`172`](172-spells-column-arrays.md) §3 沒寫 `INT`,那一版是對的)|
| 4 | `CMBT 0x1373D` | [`153`](153-damage-formula-closed.md) §2 第二次擲骰 | `round(RND×100+1)` | 狂暴率 **25.5%** 而非 25% |
| 5 | `MAZEMOVE 0x13788` | [`155`](155-gem-puzzle-and-healing-pool.md) / [`178`](178-settings-from-three-sources.md):`INT(RND×5)+1` = 1–5 | `round(RND×5+1)` → **1–6**,均值 3.5 | 治療池每次多回 0.5,上限 6 |
| 6 | `CMBT 0x13070` | [`152`](152-experience-settlement-formula.md) §2.3:`INT(總額 × 0.575)` | `round(總額 × 0.575)` | 戰後金幣 +0.5 |
| 7 | `MAZEMOVE 0x11DC1` | 無([`169`](169-encounter-zone-selects-the-monster.md) 列為未讀)| `round(RND×6+1)` → 1–7 | 疑為遭遇隻數,**語意未定案** |
| 8 | `WRLDMOVE 0x11530` / `0x11545` | 無 | `round(RND×6+2)`、`round(RND×4+4)` | 疑為 `DRAW` 巨集參數,**用途未知** |
| 9 | `CMBT 0x1665E`、`0x112CA` | 無 | `round(ds:93A2 ÷ 欄5)` / 一段未解式 | ⛔ **未知,卡住** |
| 10 | `TOWN 0x10360`、`CAMP 0x12AD4`、`USERLIB 0x00F83` | 無 | `round((18 − 字串長) × 0.5)` 之類 | **文字置中**,奇數空白時多 1 欄 —— 非玩法 |
| 11 | `WSIO` ×4 | 無 | —— | **未追**,非玩法 |

### ⚠ 「d100」在兩處不是同一個分佈

| 用途 | 位址 | 有 `INT()` | 值域 |
|---|---|---|---|
| 命中 / 狂暴 | `CMBT 0x136DA`、`0x1373D` | ✗ | **1–101** |
| 魔法道具發動率 | `CMBT 0x17C84` | ✓ | **1–100** |

所以 [`157`](157-magic-item-rate-is-a-percentage.md)「欄 6 就是百分比」**成立**,
而 [`154`](154-die-is-d100.md) 的命中率要減 0.5。**同一個成語不能整批套。**

### 2.1 第 1 條:我自己重讀過

[`156`](156-attribute-roll-shape.md) §1 寫「另一處在 `0x10CF7`,指令序列逐條相同」。
逐位元組重切(IDA 在這一段一樣切錯位):

```
首擲 0x10ABA …  mov bx, 1Ah / INT 3D:03 / INT 3F:77      ← 有 INT()
重擲 0x10CF7 …  INT 3F:81 / mov dx, bx / INT 3F:77       ← 沒有
```

`3F:81` 不在 §1.2 那張 4-byte 清單裡,所以 `0x10D16` 的 `8B D3` 是獨立的
`mov dx, bx`,不是內嵌運算元 —— 中間**確實沒有**塞得下 `mov bx,1Ah` + `CD 3D 03`。

> **判準**:「兩處形狀完全一致」這種話要逐位元組比,不能看助憶碼像不像。
> 這裡差的是 6 個位元組,而兩段在反組譯輸出上看起來幾乎一樣。

## 3. A 類(算式不變)

[`183`](183-levelup-attribute-growth.md) 屬性成長、[`152`](152-experience-settlement-formula.md) 經驗與每隻怪的金幣、
[`153`](153-damage-formula-closed.md) 傷害收尾與力量加值、[`157`](157-magic-item-rate-is-a-percentage.md) 道具發動率、
[`169`](169-encounter-zone-selects-the-monster.md) 遭遇挑選、[`160`](160-party-deploys-three-per-row.md) 三人一列、
[`163`](163-attribute-14-is-action-type.md) 屬性 14、[`156`](156-attribute-roll-shape.md) 創角**首擲**、
[`177`](177-dgroup-init-stream-and-hunt-formula.md) 打獵 —— 全部有讀到的 `3D:03`。

⚠ **A 類裡有 2 處仍要小心**:`CMBT 0x10C5B`(`INT()` 之後又乘一個執行期值)、
`CHARUTIL 0x11433`(尾端加 `ds:66E4`)—— 若那兩個量非整數,末端仍在四捨五入。**未知**。

## 4. 順帶解掉 / 順帶抓到

- [`126`](126-shop-price-multiplier.md) §7 的「讀倍率的程式碼位置:未讀」→ 四個使用點都在 TOWN
  (`0x106D6` / `0x10789` / `0x10D4E` / `0x11779`),形狀 `[di+6B60h] × ds:6E2A → INT() → 3F:77`,
  **售價算式是 A 類,不變**。
- [`152`](152-experience-settlement-formula.md) §2.3 列為未解的 `ds:96E8` / `ds:96B8` / `ds:93C0`,
  在 DGROUP 初始串裡讀得到:**0.575 / 1.7 / 2.1**。
- ⚠ `docs/spec/07-combat-scene.md` 把 `INT 3D:03` 當成亂數產生器 —— 那是
  [`143`](143-character-creation.md) §5 踩過、[`152`](152-experience-settlement-formula.md) §3 已推翻的坑,
  又在 spec 留了一份。
- ⚠ [`150`](150-experience-is-offset-90.md) 把 `INT 3D:03` 註成「相減」。

## 5. 明列剩餘的不確定

| 項目 | 狀態 |
|---|---|
| `CMBT 0x1665E`、`0x112CA` 的算式語意 | ⛔ **未知,卡住** |
| `MAZEMOVE 0x11DC1`、`WRLDMOVE 0x11530/0x11545` 的用途 | **未定案** |
| `WSIO` 的 4 處 | **未追**(非玩法)|
| A 類那 2 處的末端取整 | **未知** |
