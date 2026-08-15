# 179 — 最終戰的組成:通關紀錄說是 **Siriadne + 兩隻 Great Dragon**(第 3 級證據)

日期:2026-08-15
接續:[`161-maze-event-dispatch.md`](161-maze-event-dispatch.md) §4、[`169-encounter-zone-selects-the-monster.md`](169-encounter-zone-selects-the-monster.md)
子系統:**G. 地城與迷宮** / **J. 戰鬥規則**
輸入:`MONSTERS.DAT`、`DT51TEXT.DAT`(SHA-256 見 [`00-inputs.md`](00-inputs.md))、社群通關紀錄(§1)

## 結論

| # | 結果 | 信心 |
|---|---|---|
| 1 | 目標 **533** = `Siriadne !` + **2 隻 `Great Dragon`** | **證據充分**(第 3 級 + 三項資料側交叉檢查)|
| 2 | 目標 **204** = 1 隻 `Hill Giant` | **證據充分**(遊戲自己的文字寫著)|
| 3 | `Great Dragon`(階級 10)在 Siriadne 門外**隨機遭遇得到**,`Siriadne !`(階級 13)**遭遇不到** | **已確認**(與 [`169`](169-encounter-zone-selects-the-monster.md) 的規則自洽)|

⚠ **這一篇不是反組譯結論。** 通關紀錄是 [`CLAUDE.md`](../../CLAUDE.md) §6 的**第 3 級**證據,
而「腳本怎麼指定怪物」要的是第 2 級(IDA)。本篇的用途是**當那一輪的獨立對照**,
不是替代它。⛔ 反組譯讀出別的答案時**以反組譯為準**。

## 1. 來源

CRPG Addict 的通關紀錄([2010-09](http://crpgaddict.blogspot.com/2010/09/shard-of-spring-won-you-bastards.html)),
作者明寫玩的是 **DOS 版**(他提到「攻略說有一座叫 The Tunnels 的地城,我找不到,
也許是 DOS 版的差異」)—— **平台對得上**,而 [`CLAUDE.md`](../../CLAUDE.md) §6
警告過手冊那類來源要先問「這一頁講的是哪個平台」。

> the final battle against her only featured two such dragons
>
> random encounters with a party of **6 greater dragons** appear right outside her door

`DT51TEXT.DAT` 的 533 段自己也寫著:

> With a sweeping motion of her huge wings, Siriadne summons **two ancient dragons**
> from the open sky.

## 2. `MONSTERS.DAT` 裡對應到哪幾列

**`MONSTERS.DAT` 沒有 `Ancient Dragon` 這一列。** 「ancient dragons」是敘述用語,
通關紀錄講的是「greater dragons」,而資料檔裡是 `Great Dragon`:

| 列(1-based)| 0-based | 名稱 | 難度階級(欄 9)|
|---:|---:|---|---:|
| 11 | 10 | `Hill Giant` | 3 |
| 51 | 50 | `Baby Dragon` | 4 |
| 52 | 51 | `Small Dragon` | 4 |
| 53 | 52 | `Large Dragon` | 7 |
| **54** | **53** | **`Great Dragon`** | **10** |
| **72** | **71** | **`Siriadne !`** | **13** |

## 3. 三項交叉檢查

### 3.1 204 對得上遊戲自己的文字

`DT2TEXT.DAT` 的 204 段:「…says **a hill giant** with a toothless grin」——
**遊戲自己講了是什麼怪**,而 `MONSTERS.DAT` 第 11 列正是 `Hill Giant`。
這一項不需要外部來源。

### 3.2 「門外遭遇得到、她本人遭遇不到」與 [`169`](169-encounter-zone-selects-the-monster.md) 自洽

[`169`](169-encounter-zone-selects-the-monster.md) 讀出的規則是
`|難度階級 − 區域| ≤ 1`,不合就重擲:

- `Great Dragon` 階級 **10** → 區域 9–11 挑得到。通關紀錄說門外會遇到**六隻**,
  所以 DG51 那一帶的區域編號**必然落在 9–11** ✓
- `Siriadne !` 階級 **13** → 要區域 12–14,而 [`169`](169-encounter-zone-selects-the-monster.md) §1.1
  盤過七個區域**沒有一個到得了** → **她只能由腳本放上場** ✓

**兩件事互相印證**:同一片地城裡,一種龍遇得到、首領遇不到,
而那正是「階級 10 在區域範圍內、階級 13 不在」的直接後果。

### 3.3 為什麼是 `Great Dragon` 而不是另外三種

`Baby`(4)/ `Small`(4)/ `Large`(7)三種的階級都太低 ——
若門外隨機遇到的是它們,DG51 的區域就得是 3–8,那樣 `Great Dragon` 反而遇不到,
與通關紀錄的「六隻 greater dragons」矛盾。**四選一在階級表下只剩一個解。**

## 4. ⚠ 這一篇**不能**回答的事

| 問題 | 為什麼答不了 |
|---|---|
| 腳本**怎麼**把這幾隻塞給 `CMBT` | 通關紀錄只看得到結果,看不到機制 |
| 三隻的**站位** | 同上 |
| Siriadne 的屬性有沒有被腳本改過 | 同上 —— `MONSTERS.DAT` 第 72 列是她的**基準**值,腳本有沒有加成未知 |
| 打贏之後**確切**做了什麼(除了播結局) | [`161`](161-maze-event-dispatch.md) §4 只讀到字串,沒讀到效果 |

**上面四項要靠反組譯**,而那是另一輪的事。

## 5. 明列剩餘的不確定

| 項目 | 狀態 |
|---|---|
| 腳本指定怪物的機制 | **未解** —— 反組譯那一輪的主題 |
| 三隻的站位與初始朝向 | **未解** |
| Siriadne 的屬性是否被腳本覆寫 | **未解** |
| 「6 隻 greater dragons」是不是固定隊形 | **未解** —— 那是隨機遭遇,`CMBT` 怎麼決定隻數沒讀到 |
