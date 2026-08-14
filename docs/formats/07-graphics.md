# 圖形格式 — **READY**

> `CLAUDE.md` §1.2:**只解位元佈局**,不解顯示卡怎麼收這些 bytes。

## 1. 小圖塊(98 bytes,9 個檔)

BSAVE 容器內是 BASIC `GET` 陣列:

```
前 2 個 word = 寬(bit 數)與 高
其後 = 像素資料
```

⚠ **尺寸寫在資料裡** —— 不要對剩餘長度做因數分解。
實際尺寸 **17 × 17**。

## 2. `PICT*.BIN`(5,980 bytes)— 153 × 153

同樣的 `GET` 陣列形式。編號跳號(3/4/5 缺)是原始封裝就少。

## 3. `MONST*.BIN`(742 bytes × 22)— **8 張 17×17 交錯**

⚠ **資料不連續**。八張子圖以 word 為單位交錯:

```python
sub[i] = words[i::8]        # i = 0..7
```

寫入端證據(`MIO2`):`shl si,1` ×3(×8)+ `add si, 圖號` + `shl si,1`
→ 目標是第 `(j×8 + i)` 個 word。

**BASIC 二維陣列是行主序**:`A%(i,j)` 在 word `j*(dim1+1)+i` —— 這就是交錯的來源。

## 4. `*.PIC` — 不是點陣圖

是 BASIC **`DRAW` 巨集語言**的文字:

```
U D L R E F G H     方向     B / N     前綴(不畫 / 不移動)
M±x,±y  相對移動    C n      顏色      S n   縮放
TA n    角度        P f,b    填色
```

`tools/draw_pic.py` 是最小實作。

## 5. 調色盤

`0x3D8 = 0x0E` —— CGA 繪圖模式 + 「黑白」位元 → **第三調色盤(黑 / 青 / 紅 / 白)**。
`0x3D9` **從來沒有被任何模組或 `BRUN30` 寫過**。

## 未解

無(H 的範圍依 §1.2 止於「圖能 dump 成 PNG 且肉眼對得上原版」)。

出處:[`re/19`](../re/19-bsave-container.md)–[`22`](../re/22-pict-and-monst.md)、
[`48`](../re/48-monst-deinterleave.md)、[`49`](../re/49-h-closure.md)
