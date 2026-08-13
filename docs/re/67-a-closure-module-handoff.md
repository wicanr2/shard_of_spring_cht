# 67 — 子系統 A 收尾:執行期怎麼把控制交進模組

日期:2026-08-14
接續:[`66-userlib-slot-semantics.md`](66-userlib-slot-semantics.md)
子系統:**A. 執行檔架構與模組轉交**

## 結論

**執行期用 `retf`(遠返回)進入模組**,目標是 `模組節區 : 0x30`。

```
0115E5  cmp  word ptr ds:0A18h, 7A62h   ; ★ 簽章:'bz'?
0115F6  mov  es, word ptr ds:0A24h      ; ★ 模組節區
0115FA  push es                         ;    推入節區
0115FB  mov  ax, 30h                    ; ★ bz → 位移 0x30
011604  jz   → 11609
011606  mov  ax, 3Ah                    ; ★ 非 bz(即 bm)→ 位移 0x3A
011609  push ax                         ;    推入位移
   …
011623  retf                            ; ★ 遠返回 = 跳進模組
```

同時解掉 [`15`](15-chain-filename-and-misalignment.md) 留的
「`ds:0B06h` 檔名緩衝區由誰填」:

```
01164D  mov  word ptr ds:0A28h, 0B06h   ; ★ ds:0A28h ← 指向檔名緩衝區
011655  mov  bx, ds:0A28h
01165B  call near ptr sub_14CB8         ; 載入該名稱的模組
```

`sub_14CB8` 緊接在重定位常式 `sub_14BDD`(`0x14BDD`–`0x14CB8`,[`03`](03-bz-module-header.md) §3)之後 ——
**那是模組載入器**,而 `ds:0A28h` 告訴它去哪拿檔名。

## 1. 兩個入口位移的交叉驗證

| 格式 | 執行期推入的位移 | 該位移的位元組 | 判讀 |
|---|---|---|---|
| `bz` | **`0x30`** | `MENU`: `e9 46 00` / `CMBT`: `e9 3d 00` / `START`: `bb ff ff` | `JMP` 或 `MOV` —— 合法指令起點 |
| `bm` | **`0x3A`** | `USERLIB`: `e9 33 00` | `JMP near` |

[`05`](05-module-code-start.md) §1 是**從資料側**判出 `bz+0x30`(十一支全部是合法指令起點)。
現在**從執行期側**拿到同一個 `0x30` —— 一個是位元組判讀,一個是程式碼裡的立即數。

**而 `bm` 的 `0x3A` 是這一輪才知道的** ——
[`05`](05-module-code-start.md) 當時只看 `bz`,沒有 `bm` 的樣本。
`USERLIB` 的 `bm+0x3A` 是 `e9 33 00`(`JMP near`),**合法**。

## 2. 相關的執行期全域

| 位址 | 內容 |
|---|---|
| `ds:0A18h` | 目前模組的**簽章**(`0x7A62` = `bz`,否則當 `bm`)|
| `ds:0A24h` | 目前模組的**節區** |
| `ds:0A28h` | 指向**下一個要載入的模組檔名**(值 = `0x0B06`)|
| `ds:0B06h` | 檔名緩衝區本體([`15`](15-chain-filename-and-misalignment.md))|

**信心等級:已確認**(逐條指令)。

## 3. A 的四項條件

| # | 條件 | 證據 |
|---|---|---|
| 1 | IDA 讀原始指令 | loader stub 全流程([`02`](02-loader-stub.md))、重定位([`03`](03-bz-module-header.md) §3)、INT ABI([`06`](06-runtime-int-abi.md))、**模組轉交(本文)** |
| 2 | 讀寫端點 | `bz` 標頭欄位([`03`](03-bz-module-header.md)/[`09`](09-bz-segment-map.md))、`ds:0A18h`/`0A24h`/`0A28h`(本文)|
| 3 | 獨立資料印證 | `+0x16` 模組大小 11/11([`03`](03-bz-module-header.md) §2)、節區對照 66/66([`09`](09-bz-segment-map.md))、入口位移資料側 ↔ 程式碼側(§1)、原始碼檔名與 include 關係([`47`](47-source-filenames-and-master-inc.md))|
| 4 | 筆記 | ✅ |

### 涵蓋範圍

| 項目 | 狀態 |
|---|---|
| loader stub 做什麼 | ✅ [`02`](02-loader-stub.md) |
| `bz` 標頭與重定位 | ✅ [`03`](03-bz-module-header.md)、[`09`](09-bz-segment-map.md) |
| 模組程式碼起點 | ✅ [`05`](05-module-code-start.md) + §1 |
| 執行期呼叫介面 | ✅ [`06`](06-runtime-int-abi.md)(細節屬 **B**)|
| **模組轉交** | ✅ 本文 |
| 模組 ↔ 原始碼對應 | ✅ [`47`](47-source-filenames-and-master-inc.md) |
| COMMON 區與私有變數 | ✅ [`43`](43-common-block-and-array-indexing.md)、[`52`](52-world-map-reader-and-shared-grid.md) |

**A(執行檔架構與模組轉交)判定為 RE-DONE。**

⚠ 明確**不含**、因此不影響此判定的:
- **`bz` 標頭剩下的 7 個未知欄位**([`09`](09-bz-segment-map.md) §4)——
  已知欄位足以定位模組本體、節區、入口與重定位;
  未知欄位**沒有任何一個被本專案需要的流程用到**。
  ⚠ 這是「範圍內但不需要」,不是「解不開」。
- 三張派工表的語意 —— 屬 **B**
- `INT 21h` 各功能號的行為 —— `CLAUDE.md` §1.2 的界線

## 4. 方法

這一輪只讀了約 60 條指令,但那 60 條同時關掉三個掛了很久的問題:
模組轉交([`04`](04-module-layout-entry.md) §5)、`ds:0B06h` 由誰填([`15`](15-chain-filename-and-misalignment.md))、
`bm` 的入口位移(從來沒問過)。

原因是**位置對**:`sub_14BDD`(重定位)的呼叫端一路往下讀,
本來就會經過「載入完 → 設定 → 交棒」這條線。

判準:**卡住的問題如果彼此相關,不要各自去找;
找那條會依序經過它們全部的程式碼路徑。**
[`03`](03-bz-module-header.md) §3 早就把 `sub_14BDD` 讀完了,
但**沒有往下讀它的呼叫端**——那 60 條指令等了很久。
