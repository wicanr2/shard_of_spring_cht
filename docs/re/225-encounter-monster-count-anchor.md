# 225 — 一場遭遇有幾隻怪、是哪幾隻:`RNDMONST.BIN` 就是遭遇表

> 輸入:`CMBT.EXE`、`RNDMONST.BIN`(SHA-256 見 [`00-inputs.md`](00-inputs.md);
> 讀執行檔之前先跑 `tools/ida/unlock_module.py`)、原版實跑十二場。
> 信心:**已確認**(算式與組成逐條讀出,十二場實測全部落在值域內)。

## 1. 為什麼要查

原版一場遭遇是**一群**,而且會**混種**:

| 實跑 | 組成 | 截圖 |
|---|---|---|
| 世界地圖 | `Kobold` ×7 | `workplace/dosbox/shots/q3b-c0.png` |
| 拉利斯地城 | `Evil Spirit` ×2 / `Ghoul` / `Spectre` ×2 | `q3b-p3.png` |

引擎先前一場只放一隻,難度、經驗、金幣全部偏低一個量級 ——
現在照這一篇的算式建群(`engine/combat_scene.go` 的 `composeEncounter`)。

[`169`](169-encounter-zone-selects-the-monster.md) §7 早就把「一場遭遇有幾隻怪」
列為未讀。**混種的成因**是每一次擲的是「這一種放幾隻」,
而候選有四個(§6)。

## 2. 位置:`CMBT.EXE 0x11180`–`0x111E2`

`169` §1 讀到的挑怪迴圈結束在 `0x1117D`(`mov ds:945Ah, ax` 收下那一隻)。
**緊接著**就是隻數:

```
011180  INT 3D:34                    ; RND → FAC
011185  dx ← (列號 + ds:9446) × 2    ; 第 1 欄的位元組位移(cx 保持列號不動)
01118F  INT 3F:C4                    ; ax ← A(第1欄, 列號),以下記作 c
011194  xchg ax, bx                  ; bx ← c;**ax ← 943Ch**(描述子位址先存著)
011195  INT 3F:71 內嵌 81            ; 暫存81 ← FAC(= RND)
01119B  INT 3F:57                    ; FAC ← float(c)
0111A0  INT 3F:95 內嵌 81            ; FAC ×= 暫存81           → c × RND
0111A6  mov di, 9460h / INT 3F:91    ; FAC ×= ds:9460 = 0.5(docs/re/153 §6)
0111AE  mov bx, 1Ah / INT 3D:03      ; FAC ← INT(FAC)
0111B6  mov bx, ax                   ; bx ← 943Ch(0x11194 存下來的那個)
0111B8  INT 3F:C4                    ; ★ **dx 沒有動過** → 再取一次**同一欄**
0111BD  xchg ax, bx
0111BE  INT 3F:71 內嵌 82            ; 暫存82 ← FAC(= INT(c × RND × 0.5))
0111C4  INT 3F:57                    ; FAC ← float(c)
0111C9  INT 3F:91                    ; di 還是 9460 → FAC = c × 0.5
0111CE  INT 3F:85 內嵌 82            ; FAC = 暫存82 + FAC
0111D4  mov di, 9464h / INT 3F:81    ; FAC += ds:9464
0111DC  INT 3F:77                    ; → 整數
0111E2  mov ds:943Ah, ax             ; ★ 隻數落在 ds:943A
```

也就是

```
隻數 = INT( c × RND × 0.5 )  +  c × 0.5  +  ds:9464        (最後轉成整數)
```

⚠ **只有一次 `RND`,而且兩次取的是同一欄** —— `0x111B6` 的 `mov bx, ax`
取回的是 `0x11194` 用 `xchg` 藏起來的描述子位址,`dx` 從頭到尾沒動。
與戰後金幣([`207`](207-gold-formula-closed.md))那種「兩個獨立的 `RND` 項」不同族。

## 3. `ds:9446` = 列數(72),欄用乘法選

`0x111E8` 起連續三次用同一個模式取不同欄,取到的四個值存進 `ds:790E`–`ds:7914`:

```
(2 × ds:9446 + 列號) × 2      → 欄 2 → ds:790E
(3 × ds:9446 + 列號) × 2      → 欄 3 → ds:7910
(4 × ds:9446 + 列號) × 2      → 欄 4 → ds:7912
(5 × ds:9446 + 列號) × 2      → 欄 5 → ds:7914
```

**這是「欄各自成一個連續陣列」的排法**(同 [`172`](172-spells-column-arrays.md)
對 `SPELLS.DAT` 讀到的),所以 `ds:9446` 是**列數 = 72**,
而 `add cx, ds:9446` 取的是 **0-based 第 1 欄**。

## 4. `INT 3F:C4` = 陣列取值

派工表(`tools/brun_api.py`)指到 `BRUN30 0x1C324`,逐條讀出來:

```
入口:bx = 陣列描述子、dx = 位元組位移
01C324  test byte ptr [bx+2], 80h      ; 型別旗標的 bit7 分兩族
        bit7 = 0(遠端):ds = [bx](段值)
          [bx+2] == 1 → ax = [dx]            **整數**,結果在 ax
          [bx+2] == 3 → 8 bytes → ds:16      雙精度
          其餘        → 4 bytes → ds:1A      單精度(FAC)
        bit7 = 1(近端):si = [bx] + dx
```

順帶兩個同族的:

| API | 目標 | 語意 |
|---|---|---|
| `INT 3F:C9` | `0x1C443` | **`VARPTR`** —— 把陣列位址算成線性位址,再 `sub_1B43D` 正規化成浮點數 |
| `INT 3F:AD` | `0x1AC66` | **FAC × 2ⁿ** —— `lodsb` 讀內嵌位元組,`add es:1Dh, al` 加在**指數**上。內嵌 01 = ×2、02 = ×4 |

`3F:AD` 那一條同時關掉 [`170`](170-monster-ai-casts-by-spell-family.md) §3 的
「系別 1 與 4 的張數沒有讀到」(見 [`226`](226-monster-cast-invest-and-target.md) §1)。

## 5. 那張陣列是 `RNDMONST.BIN`

進遭遇這一段之前先配置它:

```
011109  push 72 / push 6 / mov bx, 943Ch / INT 3F:45   ; DIM
01111F  mov bx, 943Ch / INT 3F:C9                      ; VARPTR
011126  mov bx, 9448h / mov ax, 37A0h / INT 3F:55      ; ★ 路徑 + 檔名
011131  mov dx, 1Ah   / INT 3E:1A                      ; ★ 載入
```

**`ds:9448` 就是 `rndmonst.bin` 的字串描述子**(`tools/dgroup_strings.py`),
而 `INT 3F:55` 是字串串接([`38`](38-api-are-basic-primitives.md) §1)——
所以這四行是「配置陣列 → 取位址 → 組出 `<路徑>rndmonst.bin` → 載入」。
同一個形狀在 `0x10753` 用在 `fastcmbt.bin` 上,載入位址是 `ds:7B20`
([`227`](227-fastcmbt-is-nine-get-arrays.md) §3)。

整支 `CMBT.EXE` 對 `943C` **只有那三處參照**(位元組掃描,分母 44,695 bytes /
6 個節區)—— 沒有 BASIC 的填表迴圈,資料是直接載進陣列的。

檔案本身對得上:**`RNDMONST.BIN`** 872 bytes = BSAVE 標頭 7 ＋ **432 word** ＋ EOF,
而 432 = **72 × 6**;全套執行檔裡只有 `CMBT` 提到這個檔名。

| 欄 | 內容 | 值域 |
|---:|---|---|
| 0 | **區域** —— 與 `ds:3656` 比,差 ≤ 1 才合格([`169`](169-encounter-zone-selects-the-monster.md))| 1–10 |
| 1 | **隻數的係數,同時是上限** | 2–7 |
| 2–5 | **四個候選怪物的編號**(允許重複)| 0–62 |

⚠ **這推翻了 [`169`](169-encounter-zone-selects-the-monster.md) §1 的「那個值是
`MONSTERS.DAT` 欄 9」** —— 兩者的值域一樣(1–10),而 §1.1 的統計佐證
(七個區域都有候選、難度單調)對這張表**同樣成立**,所以那份佐證分不開兩個讀法。
分得開的是**組成**:一場遭遇會出現「四隻甲 ＋ 一隻乙」,而怪物表沒有「一組」這種東西。

## 6. 隻數與組成:兩段迴圈

隻數(`0x11180`–`0x11252`):

```
c = 欄1
隻數 = INT(c × RND × 0.5) + c × 0.5 + ds:9464    ; ds:9464 = 1(docs/re/136 §3)
011245  cmp ax, 隻數 / jl → ds:943A = c           ; ★ 上限就是同一欄
```

組成(`0x11255`–`0x112FE`):

```
放 = 0
while 隻數 > 放:
    k = INT(RND × 4) + 1                  ; 3F:AD 內嵌 02 = ×4 → 欄 2…5
    n = round(隻數 × RND − 放 + 1)         ; 「這一種放幾隻」
    重複 n 次:清單[放++] = 欄(1+k) 的怪物編號
```

⚠ **擲的是「這一種放幾隻」,不是每一隻各擲一次** —— 所以同一種會**成串**出現。
⚠ `n ≤ 0` 時內層 `FOR` 一次都不跑,外層重挑一欄(不是無窮迴圈,但會空轉一輪)。
⚠ 清單寫進 **`ds:372C`** —— **與腳本戰鬥同一個陣列**(§7),
所以兩條路最後交給戰場的是同一種東西。

## 7. 腳本戰鬥的隻數

另一條路(`0x110CB`–`0x11106`)先看 `ds:372C`:非 99 的個數就是隻數,最多 8 個槽。
引擎的 `startScriptedCombat` 本來就照清單建,行為相同。

## 8. 實測:十二場隨機遭遇

`tools/oracle_patch.py place` ＋ `workplace/qa3b/set_enc.py` 把遭遇倒數設成 4,
一路往東走四步觸發,**拍下開戰清單**(它逐行列出每一隻)。

> ⚠ 兩個會讓取樣安靜失敗的地方:
> 1. **遭遇只在實際位移時檢查** —— 一來一回會被算成「轉身」,永遠不會遭遇;
> 2. **清單只在觸發那一瞬間在畫面上**,`shot` 要緊接在 `key` 後面,中間不能 `wait`。
>    差一秒就變成戰場,而戰場的視窗**看不到全部的怪**(有一場七隻只看得到兩隻)。

| 首隻 | 這一場的組成 | 隻數 | 對得上的表列 |
|---|---|---:|---|
| Goblin | ×7 | 7 | 62(欄1 = 7)|
| Kobold | ×7 | 7 | 64(欄1 = 7,候選 Kobold/Kobold/Rat/Rat)|
| Giant Spider ×2 ＋ Cobra ×3 | 混 | 5 | 37(欄1 = 7,候選 Cobra×3/Giant Spider)|
| Lvl 1 Fighter | ×5 | 5 | 20/21(欄1 = 6)|
| Lvl 1 Fighter ×4 ＋ Lvl 2 Fighter ×2 | 混 | 6 | 23(欄1 = 7)|
| Lvl 2 Fighter | ×5 | 5 | 22/23 |
| Rattlesnake | ×4 | 4 | 58(欄1 = 6)|
| Rattlesnake ×5 ＋ Cobra | 混 | 6 | 58 |
| Bat | ×6 | 6 | 56(欄1 = 7,四欄都是 Bat)|
| Bat | ×5 | 5 | 56 |
| Bugem ×4 ＋ Lvl 1 Fighter | 混 | 5 | 67(欄1 = 6,候選 Bugem/Kobold/Kobold/Lvl1F)|
| Bugem | ×5 | 5 | 36 或 67 |

**十二場全部落在算式的值域內**:欄1 = 6 的列給 4–6、欄1 = 7 的列給 5–7。
⚠ **沒有一場是 8** —— 而算式本身擲得出 8(`3.5 + 3 + 1 = 7.5` → 四捨五入 8),
是 `0x11245` 的上限把它壓回 7。**上限那一行不是保險絲,是規則。**

## 9. 還缺什麼

| 項目 | 狀態 |
|---|---|
| `ds:9464` 的位元組值 | 證據充分是 `1`([`136`](136-damage-coefficients-still-unresolved.md) §3 的 17 個使用點),沒從檔案讀出 |
| `INT 3F:77` 是截斷還是四捨五入 | 依 [`185`](185-int3f77-rounding-audit.md) 取**四捨五入**;`c` 為奇數時兩者差 1,而實測十二場分不開這兩種 |
| 填表的原生常式是哪一個槽 | 未讀 —— 但填進去的內容已經由 `RNDMONST.BIN` 的形狀與實測定案 |
