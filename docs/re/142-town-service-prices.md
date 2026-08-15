# 142 — 住宿 / 食糧 / 治療的基準價,以及商店管線的實跑印證

日期:2026-08-15
接續:[`141-world-map-axes-were-transposed.md`](141-world-map-axes-were-transposed.md)、[`138-towndata-building-types.md`](138-towndata-building-types.md)
子系統:**F. 世界地圖**(`TOWNDATA.DAT`)
輸入:`TOWNDATA.DAT`、`ITEMS.DAT`、DOSBox 實跑(Green Hamlet)

## 結論

Green Hamlet 的旅店、酒館、治療所、兩間商店**價格倍率全部是 `1.0`**
(`TOWNDATA.DAT` 位移 38 的 MBF),所以畫面上的數字**就是基準價**:

| 項目 | 基準價 | 來源畫面 |
|---|---:|---|
| 住宿(每晚)| **25** | `Your rooms will cost 25 gold each night.` |
| 食糧(每份)| **2** | `One day's food for the party costs 2 gold.` |
| 治療傷勢 | **2** | 治療所價目表 `Healing 2` |
| 解毒 | **40** | `Unpoison 40` |
| 解除束縛 | **94 / 等級** | `Unbind 94 /lvl` |
| 復活 | **100 / 等級** | `Ressurect 100 /lvl` |

**信心等級:已確認**(倍率 1.0 的店讀到的就是基準價;四個項目同一張畫面)。

這解掉了 [`138`](138-towndata-building-types.md) §5 與
[`spec/11`](../spec/11-town-camp-roster.md) 標了好幾輪的「基準價未解」。

⚠ **`Healing 2` 的單位是推的**:手冊 p.37 寫「治療創傷的費用**依傷害的程度**而定」,
而畫面只印一個 `2` —— 最省的解釋是**每點生命 2 金幣**。
沒有實際治療過(出貨隊伍滿血),所以這一項是**證據充分**,不是已確認。

## 1. 為什麼一間城鎮就夠

[`126`](126-shop-price-multiplier.md) 已經確認售價 = `INT(基準價 × 該店倍率)`。
Green Hamlet 七間建築的倍率是:

```
The Iron Blade 1.0   Rolo's Resthouse 1.0   Hamlet Hospital 1.0
Kor's Smithy   1.1   Winsome Weapons  1.1   The Purple Rat   1.0   General Goods 1.0
```

要問基準價,**挑倍率恰好 1.0 的那幾間**就不必反推 ——
反推會踩到 MBF 的截斷誤差(`1.1` 實際是 `1.100000023841858`)。

> **判準**:量一個被係數乘過的量時,先找**係數為 1 的樣本**。
> 這比量一個 1.1 倍的樣本再除回去省事,也少一個誤差來源。

## 2. 商店管線的三項印證

同一次實跑順帶把靜態解出的三件事全部對上了:

| 靜態結論 | 畫面 |
|---|---|
| `The Iron Blade` 賣道具 0–3([`138`](138-towndata-building-types.md) §1)| `A Dagger 2 / B Small axe 6 / C Short sword 15 / D Mace 13` —— **正好四件** |
| `General Goods` 賣道具 47–48 | `A Torch 2 / B Oil lantern 10` —— **正好兩件** |
| `ITEMS.DAT` 欄 3 是基準價([`126`](126-shop-price-multiplier.md))| 六個價格與檔案裡的欄 3 **逐項相同** |

⚠ 這同時是 [`138`](138-towndata-building-types.md) §1 那句
「先前把 57 件道具全列進每一間店」的正面否證:**原版真的只列範圍內的**。

## 3. 建築畫面的指令

| 建築 | 畫面 |
|---|---|
| 酒館 | `Do you wish to: T)alk with other adventurers, B)uy food? (ESC to exit)` |
| 旅店 | 直接問晚數 `(1-9, 0 exits)` |
| 治療所 | 先列價目表,再問 `Enter character # to be healed, (ESC exits)` |
| 商店 | `Item to buy, ESC to leave.` |

⚠ **食糧確實買在酒館** —— [`140`](140-manual-stat-tables.md) §6 從手冊推的那一條,
這裡由畫面直接證實(`B)uy food` 就在酒館的選單裡)。

⚠ 治療所是**先選人再結帳**,所以「治療傷勢 2」乘的是那個人的缺血量,
與 §結論 的推測一致。

## 4. 「賣出價未解」其實是問錯了問題

商店畫面的指令列只有 `Item to buy, ESC to leave.` —— **沒有賣出選項**。
手冊 p.36 描述 `SHOPPES` 時也只寫買,營地的指令是
`TRADE`(隊員之間傳遞)與 `DROP`(丟掉,拿不回來),同樣沒有賣。

**兩個獨立來源都沒有賣出 ⇒ 這款遊戲沒有賣出功能。**
[`spec/11`](../spec/11-town-camp-roster.md) §3 的「賣出價未解、M8 不做賣出」
結論(不做)是對的,理由要改:**不是解不出公式,是原版沒有這件事。**

> **判準**:一個「未解的公式」找很久找不到時,先問**這個功能存不存在** ——
> 「找不到賣出公式」與「沒有賣出」在靜態分析上長得一模一樣,
> 而只有後者能結案。

**信心等級:證據充分**(畫面 + 手冊兩個獨立來源都沒有;沒有窮舉過所有按鍵)。

## 5. 尚未解開

| 項目 | 狀態 |
|---|---|
| `Healing 2` 的乘數究竟是「每點生命」還是別的 | 證據充分,未實測 |
| `PERSUASIVENESS` 技能怎麼影響價格 | 未解(手冊 p.36 說會降價,沒給公式)|
| 其餘 12 個城鎮的座標對應 | 未解([`141`](141-world-map-axes-were-transposed.md) §2)|
