# 189 — 營地 `I)dentify` 的成功率 = `d100 ≤ 智能 × 4.5`

> 輸入:`CAMP.EXE`(SHA-256 見 [`00-inputs.md`](00-inputs.md))。
> 反組譯前先跑 `tools/ida/unlock_module.py`。
> 信心:**已確認**(位址逐條讀出;常數走 DGROUP 初始資料串)。

補上 [`177`](177-dgroup-init-stream-and-hunt-formula.md) §7 留的「鑑定成功率未讀」。
引擎先前因此讓鑑定**必定成功**並在訊息裡標未解 —— 現在可以拿掉那個佔位。

## 判定(`0x1112D`–`0x11169`)

```
INT 3D:34                ; RND
mov di, 72CAh            ; ds:72CA = 100
INT 3F:91                ; ×
mov bx, 1Ah / INT 3D:03  ; INT()
mov di, 72CEh            ; ds:72CE = 1
INT 3F:81                ; +1              → d100,值域 1…100
mov word ptr ds:66D0h, 14h ; 20
call 0x10079             ; = CVI(MID$(記錄, 20, 2)) = **位移 20 = 智能**
mov bx, ax / INT 3F:57   ; → 浮點
mov di, 72D2h            ; **ds:72D2 = MBF 4.5**
INT 3F:91                ; ×
INT 3F:A5                ; 比較
jbe 0x1116C              ; **≤ → 成功**
jmp 0x111A0              ; 否則 → 'Failed'(`0x111B7` 印 ds:72D6)
```

三個常數由 `python3 tools/dgroup_init.py CAMP.EXE --at 72CA/72CE/72D2` 讀出:
**100 / 1 / 4.5**。

**成功 ⟺ `d100 ≤ 智能 × 4.5`** —— 智能 20 約九成、智能 10 約四成五。

⚠ **與難度無關**:算式裡沒有道具的任何欄位,只有施行者的智能。

## 與 `D)ispell` 同一個成語

[`188`](188-dispell-undead.md) §4 的驅散是
`d100 ≤ (智能 − 難度階級 + 1) × 3.6`,形狀一樣、常數不同,
連取智能的那支常式(`CVI(MID$(記錄, 20, 2))`)都是同一種寫法 ——
兩個模組各有一份(`CAMP 0x10079` / `CMBT 0x10083`)。

> **判準**:讀到一個「`d100` × 某個 DGROUP 浮點」的比較時,先去看
> **同族的另一支**。這一篇能在幾分鐘內收斂,是因為 `188` 已經把成語認出來了。

## 仍未解

| 項目 | 狀態 |
|---|---|
| 成功之後把道具標成「已辨識」寫在哪一格 | 沒追。引擎用自己的旗標 |
| `I)dentify` 是不是也吃「每天一次」 | 呼叫端的閘門另有一段(`town.CanIdentify` 已實作),這一篇只讀擲骰 |
