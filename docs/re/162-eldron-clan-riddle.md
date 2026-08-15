# 162 — Eldron 的氏族謎題:輸入四個名字,判定只數「對得上幾個」

日期:2026-08-15
接續:[`161-maze-event-dispatch.md`](161-maze-event-dispatch.md)
子系統:**G. 地城與迷宮**
輸入:`MAZEMOVE.EXE`、`DE7EFF.BIN`、`DT7TEXT.DAT`(SHA-256 見 [`00-inputs.md`](00-inputs.md))

## 結論

| # | 結果 | 信心 |
|---|---|---|
| 1 | DG7 有第三個互動機關:**Eldron Greyhair 的氏族謎題**,事件目標 **705**、格 24/33、朝向 3 | **已確認** |
| 2 | 答案是**四個名字**:`MURTHIN` / `CERCION` / `LOTHIAN` / `VANDIGUARD` | **已確認**(字串常數 + 比對迴圈)|
| 3 | 判定是 **4×4 全對全**,只數對得上幾組,湊滿 4 就算過 —— **順序不拘,而且同一個名字打四次也會過** | **已確認** |
| 4 | 進度由一個三態旗標 `ds:91E4` 記著(0 / 1 / 2),透過操作碼 **71** 的服務呼叫存取 | **已確認** |
| 5 | `MAZEMOVE` 的字串常數在檔案裡是 `<長度><DGROUP 位址>` 開頭的記錄串 —— **可以把程式碼裡的 `ds:9xxx` 運算元對回實際字串** | 證據充分 |

## 1. 謎題的內容

`DT7TEXT.DAT` 把整個謎題講完了:

| 編號 | 文字 |
|---|---|
| 701 | The Tomb of **Murthin** (The Mad). … |
| 702 | The Tomb of **Lothian**. … |
| 703 | The Tomb of **Cercion**. … |
| 704 | The Tomb of **Vandiguard**. … |
| 707 | The tomb of Eldron Greyhair. … 'Search the tombs of the Moonglow clan and return here when you know their names.' |
| 706 | 'I hope you are prepared to name the clan, lest you suffer needlessly!' |
| 708 | 'You have done well. … To aid you I give you this ring. With it your enemies will be struck by a **Tempest**!' |
| 709 | 'You must try again, for without passing this test, I can not assist you any more.' |
| 705 | 'Why have you returned? … You can complete your quest without any further aid from me.' |

四座墓在 `DE7EFF.BIN` 裡各是一筆事件(701–704),Eldron 自己是第五筆,**目標 705**。

## 2. 控制流

`MAZEMOVE` `0x12D28`–`0x1308D`:

```
012D28  bx = 34DCh / dx = 66CCh / INT 3F:61   ; 服務呼叫的字串引數
012D33  ds:66D0 ← 71                          ; 操作碼 71 = 取具名狀態
012D39  call …
012D3C  mov ds:91E4h, ax                      ; ★ 進度旗標
012D3F  cmp ax, 2  → 訊息 705(已經拿過,結束)
012D86  cmp ds:91E4h, 0 → 訊息 707(去搜墓,結束)
012DF3  否則(= 1)     → 訊息 706,出題
…
012FF3  ds:66CA ← 71 / call …                 ; ★ 旗標寫回
013006  訊息 708(給戒指)
013087  訊息 709(答錯)
```

**旗標是三態的**:0 = 還沒見過 Eldron、1 = 見過了可以作答、2 = 已經拿到戒指。
⚠ **0 → 1 的轉換在哪沒讀到** —— 最合理的位置是 701–704 那四段程式(踩過墓就 +1
或設 1),但**沒有讀到**,不要當成已知。

## 3. 出題與判定

```
012E30  四個名字 ← 字串常數 ds:91FE / 920A / 9216 / 9222
        存進 ds:8EDA / 8EDE / 8EE2 / 8EE6      ; 間距 4(字串描述子)
…       印 'The brothers are:',再問 '#1 ' '#2 ' '#3 ' '#4 '
        玩家的四個回答存進 ds:8EEE 起,間距 4

012F82  ds:9246 ← 0                            ; 對上的組數
012F94  bx = 8EEE + 回答索引 × 4                ; 外層 i = 0…3
012FA4  ax = 8EDA + 名字索引 × 4                ; 內層 j = 0…3
012FA8  INT 3F:62  字串比較
012FB2  相等 → inc ds:9246
012FC7  內層 cmp 3 / jle
012FD3  外層 cmp 3 / jle
012FD8  cmp ds:9246, 4 → 相等才算過
```

**判定只是「數對得上幾組」**,沒有記哪個名字用過。所以:

- **順序不拘** —— 四個名字隨便排都會過;
- ⚠ **同一個名字打四次也會過** —— 它會對上同一個名字四次,計數一樣是 4。

第二點是原版的漏洞,不是我讀錯:迴圈裡沒有任何「已用過」的標記。
**實作要照抄**(專案原則是重現原版行為),但要具名讓它顯眼。

## 4. 字串常數的記錄格式(順帶解出來的)

四個名字在 `MAZEMOVE` 檔案裡長這樣(線性 `0x14990` 起):

```
07 00 02 92   0C 00 02 92   "MURTHIN"
07 00 0E 92   0C 00 0E 92   "CERCION"
07 00 1A 92   0C 00 1A 92   "LOTHIAN"
0A 00 26 92   0E 00 26 92   "VANDIGUARD"
11 00 34 92   16 00 34 92   "The brothers are:"
```

- 第一個 word 是**字串長度**;
- 第二個 word 是這段文字在執行期 **DGROUP 的位址**;
- 第三個 word 是**到下一筆的間距** = 長度湊成偶數再 + 4
  (7→12、10→14、17→22,三筆都對);
- 程式碼拿到的是**描述子位址**,比文字位址小 4
  (`mov bx, 920Ah` ↔ 文字在 `ds:920E`)—— 執行期的 DGROUP 是
  `[長度][位址][文字]` 連續排列。

### 為什麼這件事重要

[`42`](42-module-ds-and-the-66dc-boundary.md) §3 的結論是「模組的變數大半不在檔案裡,
靜態分析看得到存取、看不到內容」,而那擋住了一整類問題
(例如 [`136`](136-damage-coefficients-still-unresolved.md) 的傷害係數)。

**字串常數是例外**:它們有初始值,而且初始值連同 DGROUP 位址一起寫在檔案裡。
沿著這串記錄走一遍,就能建一張 **DGROUP 位址 → 字串內容**的表 ——
之後看到 `mov ax, 3798h` 這種運算元(例如
[`161`](161-maze-event-dispatch.md) §1 那個擋住作廢迴圈的字串比較)就有機會查得出來。

⚠ **本輪沒有把那支工具寫出來,也沒有驗證這個格式在別的模組成立。**
第 5 項標**證據充分**:五筆記錄對得上,但沒有跑過整份檔案驗證沒有例外。

## 5. 明列剩餘的不確定

| 項目 | 狀態 |
|---|---|
| 旗標 0 → 1 由誰設 | **未解**(§2)|
| 戒指怎麼發給隊伍、Tempest 是哪一個法術編號 | 未讀 |
| 名字比對是否區分大小寫、有沒有去空白 | **未讀** —— `INT 3F:62` 的語意沒有查到這一層 |
| 服務呼叫操作碼 71 存的是哪一個具名狀態、存在哪 | 未解([`161`](161-maze-event-dispatch.md) §5)|
| 這個格式在別的模組是否成立 | 未驗證(§4)|
