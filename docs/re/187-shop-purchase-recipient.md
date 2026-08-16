# 187 — 商店買到的東西進誰的背包(E4,已關)

> 輸入:`game/sharspri/`(SHA-256 見 [`00-inputs.md`](00-inputs.md))、`TOWN.EXE`、
> `tools/dosbox_run.sh`。
> 信心:**已確認(第 1 級:實跑原版)**。

## 1. 答案:原版會問「交給誰」

```
Cost is  2
Give to char #
(ESC to exit)
```

選好道具之後,原版問 **`Give to char #`** —— 輸入角色編號(1–5),
**輸入之後才扣錢**,東西進**那個人**的背包。

實跑驗證(`tools/dosbox_run.sh`,PARTY #5 從起點 (8,8) 往東一步進 Green Hamlet):

| 步驟 | 畫面 |
|---|---|
| `A` 進 The Iron Blade | `A Dagger 2 / B Small axe 6 / C Short sword 15 / D Mace 13` |
| `A` 選匕首 | `Cost is  2` / `Give to char #` / `(ESC to exit)`,金幣仍是 **75** |
| `3` 交給第 3 位 | 金幣 **73**,回到品項清單 |
| 營地 `#` → `3` → 翻頁 | Grod 的 `Items:` 出現 **Dagger** |

金幣、收貨者、背包三項一次對上。

## 2. 程式碼對得上(`TOWN.EXE 0x10850`–`0x108B3`)

```
bx = ds:6EE8            ; ★ 玩家輸入的角色編號
bx = bx × 4 + 34E0h     ; COMMON 區的記錄字串陣列(docs/re/43)
for k = 0 …:
    位移 = 54 + 2k       ; 背包十格(docs/re/167 §2)
    if 欄位值 == 99:     ; 空格哨兵(docs/re/144 §3)
        放進這一格,結束
印 'No more room!'
```

`ds:6EE8` 之所以是**變數**,就是因為它是玩家輸入的 —— 反過來說,
**「它是變數」本身就是「原版有問人」的證據**,只是當時沒認出來。

## 3. 為什麼靜態先前沒找到

字串搜尋找的是 `Which character` / `who` / `for whom` ——
而原版的字面是 **`Give to char #`**。那一段一直在清冊裡
(`translations/source/module-TOWN.tsv` 第 15 列,`ui`,而且早就譯成
「交給角色 #」),**是搜尋詞錯了,不是資料缺**。

> **判準**:`grep` 找不到時,先問「原版會怎麼講這件事」再換一組詞,
> 不要直接跳到「這個功能不存在」。
> 更快的一條:**先看譯文清冊**——那是一份現成的、按模組分好的字串索引。

## 4. 對 remake 的影響

引擎先前**固定給第一位**,而且是靜態讀不到那一步之後的具名佔位。
現在照原版:選道具 → 問「交給角色 #」→ 1–5 選人 / ESC 取消。
背包滿了的 `No more room!`(`internal/town` 的 `BuyPackFull`)本來就在。
