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

## 4. `INT 3F:C4` 解開了:陣列取值

派工表(`tools/brun_api.py`)指到 `BRUN30 0x1C324`,逐條讀出來:

```
入口:bx = 陣列描述子、dx = 位元組位移
01C324  test byte ptr [bx+2], 80h      ; 型別旗標的 bit7 分兩族
        bit7 = 0(遠端):ds = [bx](段值)
          [bx+2] == 1 → ax = [dx]            **整數**,結果在 ax
          [bx+2] == 3 → 8 bytes → ds:16      雙精度
          其餘        → 4 bytes → ds:1A      單精度(FAC)
        bit7 = 1(近端):si = [bx] + dx
          == 81h → ax = [si]                 **整數**
          == 83h → 8 bytes → ds:16
          其餘   → 4 bytes → ds:1A
```

**`INT 3F:C4` = 「從陣列 `bx` 的位元組位移 `dx` 取一個元素」**,
型別由描述子第 3 個位元組決定。這關掉 §5 原本列在第一條的未解項。

## 5. `ds:943C` 是 `DIM` 出來的陣列,**由原生常式填**

進遭遇這一段之前先配置它:

```
011109  mov ax, 48h (72) / push ax
01110D  mov ax, 6        / push ax
011111  mov bx, 943Ch
011114  INT 3F:45(帶 5 個內嵌位元組)     ; DIM ds:943C(6, 72)
01111F  mov bx, 943Ch / INT 3F:C9         ; ★ 見下
011126  mov bx, 9448h / mov ax, 37A0h / INT 3F:55   ; 兩個整數相加(docs/re/38 §1)
```

`INT 3F:C9`(→ `BRUN30 0x1C443`)把描述子換算成**線性位址**(`[bx] − ds`,
`×16` 進位成 20 位元),再 `call sub_1B43D`;而 `sub_1B43D` 是把 `bx:dx` 的
32 位元整數**正規化成浮點數**(逐位左移 + `0x80` 進位)。合起來就是
**`VARPTR`:把陣列的位址當成一個數值交出去**。

**所以這張表不是用 BASIC 迴圈填的** —— 位址交給原生常式(`USERLIB` 那一族,
[`64`](64-userlib-call-mechanism.md)),由它直接寫進陣列。整支 `CMBT.EXE` 對
`943C` 只有三處參照(`DIM`、`VARPTR`、取值),位元組掃描的分母是 44,695 bytes /
6 個節區,**沒有第四處**。

⚠ 這一條同時說明**為什麼欄位對不上 `MONSTERS.DAT`**:七欄是原生常式挑出來的,
不是檔案的原始欄序。

## 6. 腳本戰鬥的隻數:**已解**

同一段的另一條路(`0x110CB`–`0x11106`)先看 `ds:372C`:

```
0110CB  cmp word ptr ds:372Ch, 63h (99)   ; 腳本清單的哨兵(docs/spec/17)
0110D0  jl → 腳本路   /  否則 → 隨機路
        ds:943A = 1
        迴圈 i = 1…7:[i×2 + 372Ch] < 99 → ds:943A++
```

**腳本戰鬥的隻數 = 清單裡非 99 的個數**,最多 8 個槽。
引擎的 `startScriptedCombat` 本來就照清單建,行為相同。

## 7. 混種的成因:**每個槽各挑一次**

挑怪迴圈(`0x1113A`–`0x1117D`)整段讀完之後,混種不再是謎:

```
01113A  INT 3D:34 RND
01113F  di = 945Ch / INT 3F:91        ; RND × 候選數
011147  bx = 1Ah / INT 3D:03 / 3F:77  ; INT() → 列號
011156  ds:9458 ← 列號
01115F  mov cx, bx                    ; ★ cx = 列號,一路留到 §2 的隻數算式
011161  bx = 943Ch / INT 3F:C4        ; ax ← A(第0欄, 列號) = 難度階級
011169  sub ax, ds:3656h / neg / cmp 1 / jg 01113A   ; |階級 − 區域| > 1 → 重擲
011178  INT 3F:C4 → ds:945A
```

判準是 **`±1`**(`169`),所以同一場裡**相鄰兩級**都合格。實跑十二場,
三場明顯混種:`Lvl 1 Fighter ×4 ＋ Lvl 2 Fighter ×2`、
`Giant Spider ×2 ＋ Cobra ×3`、`Rattlesnake ×5 ＋ Cobra`。

⚠ **`cx` 就是那一隻怪的列號** —— 這是逐條讀出來的(`mov cx, bx`),
所以隻數確實是**從那一隻怪的第 1 欄**算的,不是從區域算的。

## 8. 實測:十二場隨機遭遇

`tools/oracle_patch.py place` ＋ `workplace/qa3b/set_enc.py` 把遭遇倒數設成 4,
一路往東走四步觸發,**拍下開戰清單**(它逐行列出每一隻)。

> ⚠ 兩個會讓取樣安靜地失敗的地方:
> 1. **遭遇只在實際位移時檢查** —— 一來一回會被算成「轉身」,永遠不會遭遇;
> 2. **清單只在觸發那一瞬間在畫面上**,`shot` 要緊接在 `key` 後面,中間不能 `wait`。
>    差一秒就變成戰場,而戰場的視窗**看不到全部的怪**(有一場七隻只看得到兩隻)。
>    「從戰場數」會得到一個穩定的低估值。

| 首隻 | 這一場的組成 | 隻數 |
|---|---|---:|
| Goblin | ×7 | 7 |
| Kobold | ×7 | 7 |
| Giant Spider | ×2 ＋ Cobra ×3 | 5 |
| Lvl 1 Fighter | ×5 | 5 |
| Lvl 1 Fighter | ×4 ＋ Lvl 2 Fighter ×2 | 6 |
| Lvl 2 Fighter | ×5 | 5 |
| Rattlesnake | ×4 | 4 |
| Rattlesnake | ×5 ＋ Cobra | 6 |
| Bat | ×6 | 6 |
| Bat | ×5 | 5 |
| Bugem | ×4 ＋ Lvl 1 Fighter | 5 |
| Bugem | ×5 | 5 |

拿 §2 的算式去反推第 1 欄:`ds:9464` 是擲骰式的 `+1`([`136`](136-damage-coefficients-still-unresolved.md) §3),
代進去之後,某一隻怪的隻數只能落在

```
S(c) = { k + c/2 + 1 : k = 0 … ⌈c/2⌉−1 }
```

`Rattlesnake` 同時出現 4 與 6,只有 `c = 5` 或 `6` 兩種值辦得到
(`c = 7` 起最小值就是 5)。而 `Rattlesnake` 在 `MONSTERS.DAT` 的十個欄位是
`9 9 8 8 62 16 0 45 2 0` —— **沒有一欄是 5 或 6**。

⚠ 這**否證**了「第 1 欄 = `MONSTERS.DAT` 的某一欄」。與 §5 一致:
那張表是原生常式填的,欄位可以是換算過的值。

## 9. 還缺什麼

| 項目 | 狀態 |
|---|---|
| 第 1 欄的值從哪來 | **未讀**。要追 `0x1111F` 的 `VARPTR` 交給誰(`USERLIB` 的哪一個槽) |
| `ds:943C` 七欄各是什麼 | **未讀**。已知第 0 欄 = 難度階級([`169`](169-encounter-zone-selects-the-monster.md));第 2、3 欄在 `0x111E8`/`0x11200` 取,去向 `ds:790E` 等 |
| `ds:9464` 在這裡的值 | 證據充分是 `1`([`136`](136-damage-coefficients-still-unresolved.md) §3),但沒從檔案讀出位元組 |
| `INT 3F:77` 是截斷還是四捨五入 | 未讀。`c` 為奇數時兩者差 1 |

⛔ **在第 1 欄解開之前,引擎維持一場一隻,並在畫面上標出來**
(`docs/spec/14` §10 的具名假設辦法)。⛔ 不要拿「4–7 隨機」湊 ——
那個範圍是**區域 1** 的十二場,而且十二場全部是同一批低階怪。
