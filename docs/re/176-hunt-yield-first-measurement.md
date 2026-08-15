# 176 — 打獵量到一次 **+3 份**;順帶兩個做 oracle 量測要知道的事實

日期:2026-08-15
接續:[`166-camp-hunt-identify-print.md`](166-camp-hunt-identify-print.md) §2、[`174-oracle-confirms-the-camp-gates.md`](174-oracle-confirms-the-camp-gates.md)
子系統:**K. 輸入語意** / **D. 角色與存檔**
輸入:`game/sharspri/`(SHA-256 見 [`00-inputs.md`](00-inputs.md))、`tools/dosbox_run.sh`、`tools/oracle_patch.py`

## 結論

| # | 結果 | 信心 |
|---|---|---|
| 1 | 一次**成功**的打獵讓補給品 **20 → 23**(+3) | **已確認(實跑,一次觀察)** |
| 2 | 打獵**會失敗** —— 多數次數補給品不變 | **已確認(實跑,多次)** |
| 3 | 營地選單是 **11 個指令 + ESC**,逐字對上 [`150`](150-experience-is-offset-90.md) §5.2 | **已確認(實跑)** |
| 4 | **原版在營地動作之後不寫檔** —— 每次啟動都從同一份存檔開始 | **已確認(實跑)** |
| 5 | 打獵的**分佈沒有量到** —— 一個成功樣本推不出面數 | 已由 [`177`](177-dgroup-init-stream-and-hunt-formula.md) §4 解掉(公式讀出來了)|

## 1. 怎麼讓它量得到

出貨五人**沒有人會 `Hunting`**([`174`](174-oracle-confirms-the-camp-gates.md) §4),
所以先用 [`tools/oracle_patch.py`](../../tools/oracle_patch.py) 把
`workplace/dosbox/game/CHARS.DAT` 裡第一位隊員的技能旗標第 9 格打開。

⚠ **只動 `workplace/` 的複本**,`game/sharspri/` 唯讀([`CLAUDE.md`](../../CLAUDE.md) §8)。

**為什麼改這個欄位不會污染量測**:程式先過技能閘門
([`166`](166-camp-hunt-identify-print.md) §2)**才**擲骰,而擲骰那一段不讀技能旗標。
所以「把旗標打開」只是讓流程走得下去,不影響收穫的分佈。

> **判準**:用改存檔的方式做量測時,要說得出**被改的欄位有沒有進入被量的公式**。
> 說不出來就不要改 —— 那會得到一個看起來乾淨、其實測到自己的數字。

## 2. 量到什麼

```
C → H → 1        Provisions: 20 → 23
```

**一次成功 = +3 份。** 其餘幾次執行補給品不變(打獵失敗)。

⚠ **一個成功樣本推不出擲骰的面數。** [`166`](166-camp-hunt-identify-print.md) §2
讀到的形狀是 `ds:731C = INT(…) + ds:731E`,夾在 ≥ 0,`0` 就算失敗 ——
「+3」只告訴我們那個分佈**取得到 3**,取不取得到 1、2、4 不知道。

完整公式是 `max(0, INT(RND × 16) − 6)`,**兩個常數都是從檔案讀出來的**,
不是從樣本推的([`177`](177-dgroup-init-stream-and-hunt-formula.md) §4)。
「+3」落在 1–9 的範圍裡,與公式相容。

## 3. 兩個做 oracle 量測要知道的事實

### 3.1 原版在營地動作之後不寫檔

打完獵之後檢查 `workplace/dosbox/game/`:

```
CHARS.DAT  Segrono 位移 86 仍是 ' '(不是 '1')
GROUPS.DAT 補給品仍是 20
```

**所以每次啟動都從同一份存檔開始。** 這有兩個後果:

- 好處:量測**可重現** —— 不必每次重建複本;
- 壞處:**不能用 save-diff 量營地動作**,得從畫面讀數字,
  或先在遊戲裡存檔([`149`](149-every-action-costs-one-tick.md) 的 `S`)。

### 3.2 亂數要靠**不同的步數**解相關,不是靠不同的 `wait:`

序列執行時同一條 timeline 跑兩次結果會不同,`cycles=fixed 4000` 讓速度可重現
但**亂數不可重現**。

⚠ **平行執行不是這樣。** 同一秒啟動的執行**共用種子**:12 個只差 `wait:` 長度的執行
分成三批同時啟動,每一批內部的結果完全相同(全 20 / 全 25)——
12 個樣本其實只有 3 個有效觀察。

有效的做法是**讓每個執行走不同的步數**(`key:Down` 重複 n 次,n 各不相同),
細節與那批 28 個樣本見 [`177`](177-dgroup-init-stream-and-hunt-formula.md) §5。

> **判準**:平行取樣之前先跑一個**全同的批次**看結果會不會全部一樣。
> 若只跑一批,12 個相同的數字看起來會像「分佈很集中」。

## 4. 營地選單逐字對上

```
Camp:
P)rint char(s)   S)leep      #)inspect char   C)ast spell
R)eorder         T)rade      D)rop            E)quip
H)unt            I)dentify   U)se an item     ESC to leave
```

**11 個指令 + ESC**,與 [`150`](150-experience-is-offset-90.md) §5.2 從反組譯字串
盤出來的完全一致。⚠ 排版不同(實跑是單欄,`150` 記的是三欄)——
**指令集合對得上就夠了**,排版不是規格的一部分。

## 5. 明列剩餘的不確定

| 項目 | 狀態 |
|---|---|
| ~~打獵的擲骰面數與加項~~ | **已解**:`max(0, INT(RND × 16) − 6)`([`177`](177-dgroup-init-stream-and-hunt-formula.md) §4)|
| 補給品的上限 `ds:6F10` | **未量** —— 沒有編譯期初值([`177`](177-dgroup-init-stream-and-hunt-formula.md) §6),只能實跑堆到上限 |
| ~~打獵的成功率~~ | **已解**:失敗 7/16 = 43.75%(同上)|
| 戰後金幣 | **未量**([`175`](175-battle-view-is-nine-tiles.md) §5:隊伍打不贏)|
