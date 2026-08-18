# 225 — 一場遭遇有幾隻怪:算式的位置找到了,還沒解開

> 輸入:`CMBT.EXE`(SHA-256 見 [`00-inputs.md`](00-inputs.md);讀之前先跑
> `tools/ida/unlock_module.py`)、原版實跑兩場。
> 信心:**假設**(位置已確認,算式未解)。

## 1. 為什麼要查

原版一場遭遇是**一群**,而且會**混種**:

| 實跑 | 組成 | 截圖 |
|---|---|---|
| 世界地圖 | `Kobold` ×7 | `workplace/dosbox/shots/q3b-c0.png` |
| 拉利斯地城 | `Evil Spirit` ×2 / `Ghoul` / `Spectre` ×2 | `q3b-p3.png` |

重製版一場只放一隻(`engine/combat_scene.go` 的 `combat.Build(…, []original.Monster{…}, …)`)——
難度、經驗、金幣全部偏低一個量級。

[`169`](169-encounter-zone-selects-the-monster.md) §7 早就把「一場遭遇有幾隻怪」
列為未讀;這一篇把**位置**釘下來。

⚠ **「混種」是新資料**:先前的假設是「同一隻複製 N 份」。
拉利斯那一場有三種怪,所以挑怪的迴圈**不只跑一次**。

## 2. 位置:`CMBT.EXE 0x11180`–`0x111E2`

`169` §1 讀到的挑怪迴圈結束在 `0x1117D`(`mov ds:945Ah, ax` 收下那一隻)。
**緊接著**就是隻數:

```
011180  INT 3D:34                    ; RND
011185  mov dx, cx / add cx, ds:9446h / shl cx, 1 / xchg dx, cx
                                     ; dx ← (列號 + ds:9446) × 2   ← 取某一欄
01118F  INT 3F:C4                    ; 取陣列元素
011195  INT 3F:71 內嵌 81            ; 暫存81 ← FAC(= RND)
01119B  INT 3F:57                    ; FAC ← float(欄位值)
0111A0  INT 3F:95 內嵌 81            ; FAC ×= 暫存81
0111A6  mov di, 9460h / INT 3F:91    ; FAC ×= ds:9460 = 0.5(docs/re/153 §6)
0111AE  mov bx, 1Ah / INT 3D:03      ; INT()
0111B6  INT 3F:C4                    ; 再取一次陣列元素
0111BE  INT 3F:71 內嵌 82            ; 暫存82 ← FAC
0111C4  INT 3F:57 / 3F:91            ; ×
0111CE  INT 3F:85 內嵌 82            ; FAC = 暫存82 + FAC
0111D4  mov di, 9464h / INT 3F:81    ; FAC += ds:9464(傷害公式的 k₂,docs/re/136)
0111DC  INT 3F:77                    ; → 整數
0111E2  mov ds:943Ah, ax             ; ★ 隻數落在 ds:943A
```

形狀與戰後金幣([`207`](207-gold-formula-closed.md))同一族:
**兩個獨立的 `RND` 項相加**,不是一次擲骰。

## 3. `ds:9446` = 列數,欄用乘法選

`0x111E8` 起連續三次用同一個模式取不同欄:

```
(2 × ds:9446 + 列號) × 2      → 另一欄
(3 × ds:9446 + 列號) × 2      → 再一欄
```

**這是「欄各自成一個連續陣列」的排法**(同 [`172`](172-spells-column-arrays.md)
對 `SPELLS.DAT` 讀到的),所以 `ds:9446` 是**列數**(怪物 74 筆),
而 `add cx, ds:9446` 取的是 **0-based 第 1 欄**。

## 4. 還缺什麼

| 項目 | 狀態 |
|---|---|
| `INT 3F:C4` 的運算元約定 | **未解**。基底描述子在 `bx`、位元組位移在 `dx`?兩次呼叫之間 `bx` 沒有重設,要逐條追 |
| 0-based 第 1 欄是哪一欄 | 取決於上一項。若基底就是 `MONSTERS.DAT` 的整表,那是 [`formats/03`](../formats/03-monsters-dat.md) 的**欄 2(力量)** —— 語意上不像隻數,所以**不要照這個假設實作** |
| `ds:9464` 的值 | 未從檔案讀出([`136`](136-damage-coefficients-still-unresolved.md))|
| 挑怪迴圈跑幾次(混種怎麼來的) | **未讀**。實跑證明它不只跑一次 |

⛔ **在這四項解開之前,引擎維持一場一隻,並在畫面上標出來** ——
`docs/spec/14` §10 的具名假設辦法。⛔ 不要「先湊一個看起來合理的隻數」:
那正是 [`CLAUDE.md`](../../CLAUDE.md) §2 擋的那種「通過測試但其實是錯的」實作。
