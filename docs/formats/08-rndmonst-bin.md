# 08 — `RNDMONST.BIN`:隨機遭遇表

> 來源:[`re/225`](../re/225-encounter-monster-count-anchor.md)。
> 狀態:**READY**(格式與三個欄位的語意都已確認,十二場實測相符)。

## 容器

BSAVE(`0xFD` + segment/offset/length + 本體 + `0x1A`),與
`MAZEDATA.BIN`、`FASTWRLD.BIN` 同一種([`formats/07`](07-graphics.md))。

```
872 bytes = 7(標頭)+ 864(本體 = 432 word)+ 1(EOF)
432 word = 72 列 × 6 欄
```

## 排法:**欄主序**

與 `MAZEDATA.BIN` 相同 —— 每一欄是一段連續的 72 個 word:

```
第 c 欄第 i 列 = word[c × 72 + i]
```

原版的取值就是這個式子(`CMBT 0x11185`:`(列號 + 72) × 2` 取第 1 欄)。

## 六個欄位

| 欄 | 語意 | 值域 | 出處 |
|---:|---|---|---|
| 0 | **區域** —— 與當前區域比,`|差| ≤ 1` 才是合格的遭遇 | 1–10 | [`re/169`](../re/169-encounter-zone-selects-the-monster.md) §1 |
| 1 | **隻數的係數,同時是隻數的上限** | 2–7 | [`re/225`](../re/225-encounter-monster-count-anchor.md) §6 |
| 2–5 | **四個候選怪物的編號**(`MONSTERS.DAT` 的 0-based 列號,**允許重複**)| 0–62 | 同上 |

⚠ **區域比對用的是欄 0,不是那隻怪自己的難度階級。** 兩者值域一樣,
所以「拿怪物表的欄 9 去比」在統計上也說得通 —— 分得開的是**組成**:
一場遭遇是一群,而怪物表沒有「一組」這種東西。

## 怎麼變成一場戰鬥

```
挑一列:隨機列號,|欄0 − 區域| > 1 就重挑(原版沒有次數上限)
隻數  :INT(欄1 × RND × 0.5) + 欄1 × 0.5 + 1,四捨五入,**上限 = 欄1**
組成  :while 隻數 > 已放:
           k = INT(RND × 4) + 1              ; 欄 2…5
           n = round(隻數 × RND − 已放 + 1)   ; 這一種連放幾隻
           放 n 隻欄(1+k) 的怪
```

⚠ **擲的是「這一種放幾隻」**,所以同一種會成串出現;
實測十二場有七場清一色、三場兩種混編。

## 引擎

`original.DecodeEncounters` → `assets/data/encounters.json`;
規則在 `rules.EncounterCount` / `rules.EncounterRun`,
組裝在 `engine/combat_scene.go` 的 `pickEncounter` / `composeEncounter`。
