# 129 — 世界地圖的圖塊派工:1–9 走點陣,11 不畫,其餘走向量

日期:2026-08-15
接續:[`52-world-map-reader-and-shared-grid.md`](52-world-map-reader-and-shared-grid.md)、
[`128-wrlditem-tile-bias.md`](128-wrlditem-tile-bias.md)
子系統:**F. 世界地圖**
輸入:`WRLDMOVE.EXE`(SHA-256 見 [`00-inputs.md`](00-inputs.md))

## 結論

| # | 結果 | 信心 |
|---|---|---|
| 1 | 繪製派工鏈在 `0x1116C`–`0x112EB` | **已確認**(逐條讀出)|
| 2 | 1–9 的圖形是 `0xCA1C` 起、**間距 92** 的連續記錄 = `fastwrld.bin` 的 9 張 | **已確認** |
| 3 | **值 11 什麼都不畫**,其餘走 `0xC980 + 4×值` 的向量表([`132`](132-world-tile-dispatch-corrected.md))| **已確認** |
| 4 | **值 5、6、11 在程式碼裡被歸為同一組(水)** | **已確認**(§3)|
| 5 | 海岸線判定會讀**四個鄰格** | **已確認**(§3)|
| 6 | 遭遇觸發集合 **12、13、20–32** 有了程式碼側證據 | **已確認**(§4)|
| 7 | `WRLDMOVE` 自己載入 `wrldmap.bin` / `wrlditem.pic` / `fastwrld.bin` | **已確認**(§5)|

全部 39 個地形值都有來源,見 [`132`](132-world-tile-dispatch-corrected.md) §3。

## 1. 派工鏈

`0x1116C` 起是十段相同形狀的比對:

```
cmp  word ptr ds:0CF96h, N
jz   →  取畫布參數 → mov bx, <圖形位址> → INT 3Eh(繪製)→ jmp 0x112EB
```

| 地形值 | 圖形位址 | 與前一筆間距 |
|---:|---|---:|
| 1 | `0CA1Ch` | — |
| 2–9 | `0CA78h`…`0CCFCh` | 各 **92** |

92 的間距與 `fastwrld.bin` 的 828 = 9 × 92 完全吻合
([`53`](53-world-tiles-towns-and-draw-renderer.md) §1),
所以 **1–9 就是 `fastwrld.bin` 的九張**。
[`132`](132-world-tile-dispatch-corrected.md) §2 補上了載入端:
`fastwrld.bin` 確實 BLOAD 到 `0xCA1C`。

鏈的第十筆**形狀不同**:`cmp 0Bh / jnz` —— 值 11 跳過所有繪製,
其餘的值落到 `0xC980 + 4×值` 的向量表
([`132`](132-world-tile-dispatch-corrected.md) §1)。

## 2. `FASTWRLD` 這個名字的意思

`WSIO.EXE`(即 `BITMAKE.BAS`,[`47`](47-source-filenames-and-master-inc.md) §4)裡有
`fastwrld.bin` 與字串 `Capture or Load image now` ——
**那是一支把畫面擷取成點陣的開發工具**。

所以 `fastwrld.bin` 是**預先渲染好的點陣圖**,對照 `wrlditem.pic` 的向量巨集:
出現頻率高的地形走「快」路徑(直接 PUT 點陣),罕見的物件才即時 `DRAW`。

而海洋(值 11,佔全圖 55.6%)走的是**更快的第三條**:什麼都不畫,
底色本身就是海。

## 3. 值 5、6、11 是同一組:水

`0x112EB` 之後不是函式結尾,是**海岸線判定**:

```
cmp ds:0CF92h, 9     → bx = −1 若 值 ≤ 9
cmp ds:0CF92h, 5     → dx = −1 若 值 == 5
cmp ds:0CF92h, 6     → cx = −1 若 值 == 6
or  cx, dx           ; 值是 5 或 6
not cx               ; 值不是 5 也不是 6
and cx, bx           ; 「≤ 9 且不是 5/6」= 陸地
jnz → 繼續;否則跳過
```

接著重取**西邊那一格**(`(Y×103 + X) − 1`,注意 `dec di` 在 `shl di,1` 之前):

```
cmp ax, 5   → 是水
cmp ax, 6   → 是水
cmp ax, 0Bh → 是水
or … → 三者任一成立就畫海岸
```

同樣的形狀在 `0x113B0` / `0x11430` / `0x114B2` 各出現一次 ——
合計**四個鄰格**(西、東、北、南)。

### ⚠ 這推翻了 [`53`](53-world-tiles-towns-and-draw-renderer.md) 對 5/6 的猜測

`53` §1 把值 5、6 寫成「疏林 / 沼澤?**假設**」,依據是渲染出來
「少量小方塊」。**程式碼把它們與海洋(11)放進同一個判斷式** ——
它們是**水**(淺水 / 湖泊之類),不是林地。

> **判準**:圖像看起來像什麼是**最弱的一級證據**。
> 同一批值,「渲染後像小方塊」推出林地,而程式碼的分組直接說是水。
> **有程式碼可讀的時候不要用看圖來分類。**

## 4. 遭遇觸發集合有了程式碼證據

`0x10D5A`–`0x10D78`:

```
cmp ds:0CF1Ah, 0Ch    ; 12
cmp ds:0CF1Ah, 0Dh    ; 13
cmp ds:0CF1Ah, 14h    ; 20   (jge)
cmp ds:0CF1Ah, 20h    ; 32   (jle)
```

**12、13、以及 20–32 的區間** —— 與 [`60`](60-event-lookup-and-tile-19.md) 從別處得到的
集合逐字相同,現在有了 `WRLDMOVE` 這一側的直接證據。

## 5. `WRLDMOVE` 載入的檔案

檔案位移 7018 起連續三個字串:

```
wrldmap.bin → wrlditem.pic → fastwrld.bin
```

⚠ **檔名在資料裡是小寫。** 我第一次掃的時候用大小寫敏感的樣式搜
`FASTWRLD|WRLDITEM|WATER`,得到**零命中**,差點寫成「`WRLDMOVE` 不載入圖形檔」。
`~/diagnosis-notes/docs/02-query-returned-empty` 的第一種形狀:自己的過濾器有洞。

## 6. 尚未解開

| 項目 | 狀態 |
|---|---|
| 值 10、12、13 的語意 | 未知 |
| 值 0 的語意(224 格,只在上緣與左緣)| 地圖邊界,**已確認**([`132`](132-world-tile-dispatch-corrected.md) §3)|

⚠ 本篇第一版把「值 ≥ 12 怎麼畫」與「`0xCFA6` 由誰填」列為未解,
**兩項都不成立** —— 前者已解,後者的前提是錯的。
經過見 [`132`](132-world-tile-dispatch-corrected.md) §5。
