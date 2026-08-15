# 138 — `TOWNDATA.DAT` 的位移 34/36:建築類別與商店庫存

日期:2026-08-15
接續:[`126-shop-price-multiplier.md`](126-shop-price-multiplier.md)
子系統:**F. 世界地圖**(`TOWNDATA.DAT`)
輸入:`TOWNDATA.DAT`、`ITEMS.DAT`、`TOWN.EXE`

## 結論

`TOWNDATA.DAT` 每筆 45 bytes,位移 34 與 36 是兩個 16-bit 整數:

| 位移 34 | 意義 | 位移 36 |
|---:|---|---|
| **≥ 0** | **販售物品編號的起點** | **終點**(含)|
| **−1** | **治療所**(8 間)| 0 |
| **−2** | **酒館**(11 間)| **傳聞編號 1–11** |
| **−3** | **旅店**(10 間)| 0 |
| **−4** | **訓練所**(6 間)| 0 = 武術、1 = 魔法 |

**信心等級:已確認**(§1 的語意對照 26/26,零例外)。

這解掉了 [`spec/11`](../spec/11-town-camp-roster.md) §7 列為「規則未解」的三項
(治療、傳聞、旅店的識別),以及一個先前**根本沒發現的**問題:
每間商店賣什麼。

## 1. 正值 = 物品編號範圍,而店名逐間對得上

| 商店 | 34–36 | 對應的道具 |
|---|---|---|
| The Iron Blade(鐵刃劍舖)| 0–3 | 匕首 … 釘錘 |
| Winsome Weapons(精選兵器)| 0–6 | 匕首 … 戰斧 |
| Flashing Steel | 2–7 | 短劍 … 雙手劍 |
| Kor's Smithy(鐵匠舖)| 12–14 | 布衣 … 鎖甲 |
| **General Goods(雜貨店)** | **47–48** | **火把 … 油燈** |
| **The Tinder Box(火種盒)** | **47–48** | **火把 … 油燈** |

**每一間店的名字都對得上它賣的東西。** 最強的是最後兩筆:
兩間不同城鎮、名字都指向「照明用品」的店,拿到同一組編號。

> **判準**:一個數值欄位的語意,拿它**旁邊那個字串欄位**去對是很強的檢定 ——
> 名字與內容都由作者填寫,而作者不會讓「劍舖賣藥水」。
> 這一步只花了一次列印,而位移 34/36 在專案裡躺了很久沒人問過。

⚠ **先前的實作把 57 件道具全列進每一間店**,而那看起來完全正常 ——
玩家不會知道劍舖不該賣板甲。**「多顯示一些」的錯誤沒有症狀。**

## 2. 負值 = 四種特殊建築

命名逐間吻合:

| 碼 | 名稱樣本 |
|---:|---|
| −1 | `Hamlet Hospital`、`Zor's Healing`、`Church of Ra`、`Healers of Jor` |
| −2 | `The Purple Rat`、`Hero's Pub`、`Red's Tavern`、`The Mermaid` |
| −3 | `Rolo's Resthouse`、`Adventurer's Inn`、`Salty's Slumber`、`Seaside Inn` |
| −4 | `BladeMaster`、`College of Magic`、`Volir's Academy`、`Balik's Arena` |

⚠ 訓練所的位移 36:`College of Magic` 與 `Wizard's Library` 是 **1**,
其餘四間(`BladeMaster` / `Volir's Academy` / `The Rising Star` / `Balik's Arena`)是 **0**。
對上 `TOWN.EXE` 的兩張技能清單(戰士十項 / 法師十項)與 `'Not enough IQ !'` ——
**0 = 武術、1 = 魔法**。

## 3. 價格倍率同樣適用於特殊建築

位移 38 的 MBF 倍率([`126`](126-shop-price-multiplier.md))對旅店、治療所、酒館**都是合法值**,
而且**每間不同**:

```
旅店   0.9 / 1.0 / 1.1 / 1.2 / 1.3 / 1.7 / 2.0
治療   0.8 / 1.0 / 1.2 / 1.25 / 1.4
```

所以住宿費與治療費**也是基準價 × 該店倍率** —— 與商品同一條公式。
⚠ **基準價本身仍未解**(那個常數在 `TOWN.EXE` 裡,還沒定位)。

### ⚠ 三間訓練所的位移 38 不是合法 MBF

```
BladeMaster       03 00 40 00
Volir's Academy   07 00 40 00
Balik's Arena     09 00 40 00
```

指數位元組是 `00` = 數值 0。當成兩個 word 讀是 `(3, 64)` / `(7, 64)` / `(9, 64)`。
⚠ **語意未解**,而且另外三間訓練所是正常的 MBF `1.0` —— 六間裡只有三間如此。
⛔ 不要據此推論。

## 4. 酒館的傳聞:10 段找到,11 個索引

位移 36 在 11 間酒館是 **1–11,兩兩不同**。
`TOWN.EXE` `0x032C9`–`0x03A40` 有 **10 段**長敘述(82–224 bytes),
內容是傳聞的口吻:

```
'You hear that a priest has been kidnapped …'
'A woman from east of the mountains says …'
"The gate leading into Siriadne's fortress is magically bound …"
'The Arena is fine if you are a fighter, but a wizard need go to Terynor …'
```

⚠ **10 ≠ 11 已經解決**([`142`](142-town-service-prices.md) §5):
第 11 段是**續作預告**,不是傳聞 —— 它在描述子鏈上緊接第 10 段,
而再往後就換成檔名與錯誤訊息。

⛔ 仍然不要把 10 段硬塞進 11 個索引 —— 現在是 11 段對 11 個索引。

## 5. 尚未解開

| 項目 | 狀態 |
|---|---|
| 住宿 / 治療 / 食糧的**基準價** | 未解(常數在 `TOWN.EXE`)|
| 第 11 段傳聞 | 未定位(§4)|
| 三間訓練所的位移 38 為何不是 MBF | 未解(§3)|
| 升級規則(`' experience before gaining a level. '`)| 未解 |
| 賣出價 | 未解(沒有讀到賣出的程式碼)|
