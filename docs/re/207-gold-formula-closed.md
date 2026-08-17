# 207 — 戰後金幣定案:`INT(1.7^階級 + RND × 2.1^階級 + 1)`,逐隻累加

> 輸入:`CMBT.EXE`、`BRUN30.EXE`(SHA-256 見 [`00-inputs.md`](00-inputs.md))。
> 反組譯前先跑 `tools/ida/unlock_module.py`。
> 信心:**已確認**(三個常數從 DGROUP 初值讀出;`3F:23` 的呼叫約定從
> `BRUN30` 的實作逐位元組讀出;累加器的消費端獨立印證)。

關掉 [`152`](152-experience-settlement-formula.md) §2.3 的
「四個未解的常數」與 [`200`](200-loot-drops-mechanism.md) §4 的
「每隻怪的金幣怎麼算」—— 這是 worklist §8.1 C 組的第 1 項。

## 1. 三個常數都在檔案裡

[`152`](152-experience-settlement-formula.md) §2.3 寫「那些是執行期變數,
不在檔案映像裡……**剩下唯一的路是實測**」。**那個判斷是錯的。**
BASIC 把運算式裡的數字字面值放成 DGROUP 常數,而常數有編譯期初值
([`177`](177-dgroup-init-stream-and-hunt-formula.md)):

```
$ python3 tools/dgroup_init.py CMBT.EXE --at 96B8   →  9a 99 59 81  MBF 1.7
$ python3 tools/dgroup_init.py CMBT.EXE --at 93C0   →  66 66 06 82  MBF 2.1
$ python3 tools/dgroup_init.py CMBT.EXE --at 9464   →  00 00 00 81  MBF 1
```

`ds:96A8`/`96AC`/`96B4` 三個**累加器與暫存**沒有初值 —— 它們確實是執行期變數,
但公式需要的是常數,而常數都在。

> **判準:「靜態讀不到」要指名是哪一種變數。**
> 同一句話涵蓋了「有初值的常數」與「執行期才有值的累加器」,
> 於是把一個一行指令就能查的東西寫成了「只能實測」。

## 2. ⚠ 1.7 與 2.1 是**底數**,不是指數

先前([`152`](152-experience-settlement-formula.md) §2.3、`spoils.go` 的註解)
把管線讀成 `階級 ^ 1.7`。**反了。** 決定性的證據在 `BRUN30` 的次方常式:

```
01B8A6  mov di, 1Ah                ; 進入點之一:di ← FAC
01B8A9  call 0x11F6B               ; ★ 3Fh 表第 0x23 筆指到這裡
01B8AC  xor ax, ax …
01B8B4  mov ax, [di+2] / or ah,ah / jz 1B8F8   ; **[di] 為 0 → 結果 1.0**
01B8BB  mov dx, [si+2] / or dh,dh / jz 1B905   ; **[si] 為 0 → 結果 0**
01B8D4  cmp ah, 90h …
01B8E8  sub ah, 91h / xchg cl,ah / shl ah,cl   ; 從 **[di]** 取出整數指數
```

`x ^ 0 = 1` 而 `0 ^ y = 0` —— 所以 **`di` 是指數、`si` 是底數**,
而整數快速路徑取的也是 `di`。

派工表第 0x23 筆指到 **`0x1B8A9`**,也就是 `mov si,1Ah` 與 `mov di,1Ah`
**兩個進入點都跳過**(第 0x25 / 0x27 筆才分別指到它們)——
**si 與 di 全部由呼叫端給**。而呼叫端:

```
012D8B  mov bx, [di+6822h] / INT 3F:57   ; FAC ← 難度階級 T(欄9)
012D94  mov di, 96B4h / INT 3F:7D        ; ds:96B4 ← T,**di 留在 96B4**
012D9C  mov si, 96B8h / INT 3F:23        ; FAC = [si] ^ [di] = **1.7 ^ T**
012DA4  mov si, 93C0h
012DA7  INT 3F:71 內嵌 81                ; 暫存81 ← FAC
012DAD  INT 3F:23                        ; FAC = [si] ^ [di] = **2.1 ^ T**
012DB2  INT 3F:71 內嵌 82                ; 暫存82 ← FAC
012DB8  INT 3D:34                        ; FAC ← RND
012DBF  INT 3F:95 內嵌 82                ; FAC ×= 暫存82
012DC5  INT 3F:85 內嵌 81                ; FAC = 暫存81 + FAC
012DCB  mov di, 9464h / INT 3F:81        ; FAC += 1
012DD8  mov bx, 1Ah / INT 3D:03          ; INT() 截尾
012DDD  mov di, 96A8h / INT 3F:81 / 3F:7D ; ds:96A8 += 這一隻
```

**`di` 從 `0x12D94` 一路活到第二次次方**,因為中間三個常式都不動它:

| 常式 | 為什麼 di 沒變 |
|---|---|
| `3F:7D`(`0x11E2D`)| `movsw / movsw / sub di,4 / sub si,4` —— 搬完自己減回去 |
| `3F:71`(`0x11E5B`)| 進出各存/取 `ds:0A5C`、`ds:0A5E` |
| `3F:23`(經 `0x11F6B`)| 那是 `3Fh` 的**通用序幕**:`push bp/si/di/ax/cx/dx/bx` → 呼叫本體 → 全部 `pop` 回來 |

⚠ **`ds:96B4` 在整支 `CMBT` 只被用這一次**(位元組層級掃過)。
它存在的唯一理由就是「`3F:23` 要指數放在記憶體裡」——
**一個只出現一次的變數,問它為什麼需要存在,往往就問出了呼叫約定。**

## 3. 公式

```
每一隻在場的怪物,T = MONSTERS.DAT 欄9(難度階級,值域 1–10 與 13):

    這一隻的金幣 = INT( 1.7^T  +  RND × 2.1^T  +  1 )      ← INT 為截尾

這一場的金幣 = Σ 每一隻
```

| T | `1.7^T` | `2.1^T` | 金幣範圍 |
|---:|---:|---:|---|
| 1 | 1.7 | 2.1 | 2 – 4 |
| 3 | 4.9 | 9.3 | 5 – 15 |
| 5 | 14.2 | 40.8 | 15 – 56 |
| 10 | 201.6 | 1 667 | 202 – 1 869 |
| 13 | 990 | 15 437 | 991 – 16 427 |

**兩個底數不同(1.7 < 2.1)讓「保底」隨階級線性感增長、「浮動」隨階級爆炸性增長**
—— 低階怪的金幣幾乎固定,高階怪一隻可以抵一整趟。

## 4. 獨立印證:累加器直接進隊伍的金幣欄

```
012E9E  mov di, 96A8h / mov si, 34C0h / INT 3F:7F
012EB0  INT 3F:A1                  ; 與 ds:930E 比(上限,未解)
012EC7  INT 3F:7D
```

`ds:34C0` 是**隊伍金幣**(`GROUPS.DAT` 位移 19,
[`134`](134-groups-load-routine-and-gold.md) §1)。
累加器 `ds:96A8` 在迴圈結束後直接與它合併 —— **那條路徑證明這個迴圈算的就是金幣**,
不必靠「結算畫面同時印 `Gold:`」這種位置證據。

## 5. 對 remake 的影響

`combat.TotalGold` 從「每隻擲 1…難度階級」的具名佔位換成上面的公式;
`combat.GoldAssumption` 那個具名假設可以拿掉。

⚠ **掉落的道具編號跟著改變**:[`200`](200-loot-drops-mechanism.md) §3.2 的
`G = round(總金幣 × 0.575)` 現在有真的總金幣可以餵。⚠ 但 `ds:96AC`
(供掉落用的**整數**累加器)的更新那三行沒有完全讀通,見 §6。

⚠ **這條規則放大了怪物階級的重要性。** 引擎的 `Unit.Tier` 對隊員固定 99,
所以 `TotalGold` 必須只算怪物 —— 99 代進去會溢位。

## 6. 仍未解

| 項目 | 狀態 |
|---|---|
| `ds:96AC`(整數累加器)那三行 | `FAC ← (float)ds:96AC` → `di = cx` → `FAC += [di]` → 取整 → 寫回。`cx` 追到 `mov bx,di`(`0x12DBD`),而那時 di = 96B4 → 看起來是加了**階級**而不是金幣。⚠ `INT 3D:34`(RND)會不會改 di 沒有查(`3Dh` 是另一張表,未必有通用序幕)。引擎暫時把整數總額當成金幣總額 |
| `ds:930E` 的上限比較 | 只讀到形狀(`3F:A1` 浮點比較 + 兩條分支),沒有讀那個常數 |
| 手冊有沒有講金幣 | 沒有([`152`](152-experience-settlement-formula.md) §2.3 查過)。所以這條規則**沒有第 1 級證據**,靠的是程式碼 + 消費端 |
