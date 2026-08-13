# Shard of Spring（SSI 1986）— remake ＋ 繁體中文化

> ⚡ **動手前先讀 [`CONTEXT.md`](./CONTEXT.md)** —— 那是全專案的單一入口:
> 現況一覽、RE 知識庫索引、術語表、已被推翻的斷言。本檔只放目標、邊界、閘門與硬規則。
>
> 建立日期：2026-08-13。工作方式沿用 `~/cht/daemon_winter`（Demon's Winter）的做法，
> 但**逆向工具一律用 IDA Pro**（見 §3），且**RE 閘門比 daemon_winter 更嚴**（見 §2）。

## 1. 專案目標

把 SSI《Shard of Spring》(1986/1987, MS-DOS 版由 Digital Illusions 移植) 的**遊戲機制**
完整逆向，在此基礎上做兩件事：

1. **Remake** — 重寫一套可維護、可跨平台執行的引擎（語言與框架待 §6 的規格階段結束再定案）。
2. **繁體中文化** — 介面、選單、道具／法術／怪物名、地城敘述、劇情文字全部中文化，
   讓華人玩家能完整玩完並看懂。

定位是**文化資產保存**，不是「能跑就好」。

### 兩條硬性原則

1. **完整性 > 投報。** 在 §1.2 劃定的範圍內，不得以「成本高、效益低」為由跳過任何格式、
   任何檔案。卡關就換方法，記錄「卡在哪、試過什麼」，不寫「暫緩／低投報」當結論。
2. **RE 齊了才實作。** 見 §2。這是本專案最重要的一條。

### 1.2 ⛔ RE 的深度邊界：止於遊戲機制，不挖 DOS／BIOS

**要還原的是「這款遊戲的規則」，不是「1986 年的 PC 怎麼畫一個像素」。**

| 挖 | 不挖 |
|---|---|
| 命中判定、傷害公式、經驗值曲線 | `INT 21h` 檔案 I/O 的參數怎麼填 |
| 法術效果的資料結構與套用順序 | `INT 10h` 顯示模式切換、CGA/EGA 暫存器時序 |
| 迷宮格式、可通行性、遭遇觸發條件 | 鍵盤掃描碼怎麼從 `INT 09h` 進緩衝區 |
| 圖檔的**像素佈局**（才能轉出素材） | 原版怎麼 blit 到 `0xB800`、怎麼等 retrace |
| 存檔欄位語意、模組間傳的是什麼狀態 | DOS 記憶體配置、`EXEC`／覆蓋載入的實作細節 |

判準一句話：**如果答案不會改變 remake 的行為，就不必解。**
碰到 BIOS／DOS 層的呼叫，只要問到「它造成什麼效果」（切到哪個顯示模式、寫了哪個檔、
讀了哪個鍵）就停，不要往下追那個中斷本身怎麼運作。

⚠ 一個例外：**圖形資料的位元佈局要挖到底**，因為素材轉不出來就沒有 remake。
但那是解「檔案裡的 bytes 怎麼排」，不是解「顯示卡怎麼收這些 bytes」——
一旦能把圖 dump 成 PNG 且肉眼對得上原版，這一項就結束。

這條邊界同時是**進度保護**：daemon_winter 花在顯示層細節上的輪次不成比例，
而那些結論裡有相當一部分對 remake 沒有影響。

### 不做的事

不散布原版執行檔、資料檔、美術。公開產出只有引擎程式碼與翻譯文本，玩家自備合法原版。
`game/`、`original/` 一律 gitignore。不協助破解 DRM 或修改付費驗證。

---

## 2. ⛔ 動工閘門：RE 沒確認完成，不准寫任何 remake 程式碼

**這一節不是建議，是閘門。** daemon_winter 的教訓是：邊猜邊實作，會產生一批
「通過全部測試、看起來也對、但其實是錯的」實作，而錯誤結論一旦頂著「已驗證」標籤活下來，
往後每一輪都在它上面疊東西，拆的成本是當初查證的十倍以上。
那個專案的「已被推翻的斷言」表最後累積到七十幾條，**其中相當一部分是因為沒等 RE 做完就先寫程式**。

### 2.1 「RE 確認完成」的定義

一個子系統要標成 **RE-DONE**，必須同時滿足四項：

| # | 條件 | 不算數的做法 |
|---|---|---|
| 1 | **在 IDA 裡讀過原始指令**，不是只讀匯出的文字，也不是只 grep `.asm` | 「工具這樣寫」；「grep 找不到所以不存在」 |
| 2 | **用 IDA 的 xref 圖確認過讀寫端**（`XrefType()`，不要自己猜指令語意） | 從呼叫端參數順序反推語意 |
| 3 | **有一份獨立資料能互相印證**（另一個檔案、手冊、攻略、或 DOSBox 實跑） | 再加一條「自己驗自己」的單元測試 |
| 4 | 筆記寫進 `docs/re/`，標明**輸入檔 + SHA-256 + IDA linear address + 信心等級** | 憑印象寫「唯一」「只有一處」而沒有全檔掃描佐證 |

信心等級只有四種寫法：**已確認 / 證據充分 / 假設 / 未知**。不准有第五種模糊說法。

⚠ **條件 2 對 `ds:xxxx` 這類全域不成立。** 16-bit 執行檔的資料段基底 IDA 並不知道，
`mov si, ds:0A3Ah` 的運算元不會被解析成位址，**xref 一律是空的或假的**
（`docs/re/39` §1：對 `ds:0A3Ah` 下 xref 得到 1 筆假命中，掃運算元文字得到 62 筆真的）。
這種目標改用 `tools/ida/find_dsref.py` 掃運算元文字，並在筆記裡**附上掃描分母**
（掃了幾條指令）。**xref 為空要先問「這種存取形式會不會產生 xref」**，
不要讀成「沒人用」。

⚠ 判定 RE-DONE 時套 §1.2 的邊界：**該子系統裡屬於 DOS／BIOS 層的部分不需要解**，
但要在筆記裡明寫「這一段是 BIOS 呼叫，效果是 X，未往下追」，不要留白讓後人以為漏了。

### 2.2 完成度看板（狀態的單一真相來源）

> **H 已 RE-DONE**（`docs/re/49` §4 逐條核對）；其餘各列未達門檻。
> 任何一列變成 RE-DONE 之前，對應的 remake 程式碼一行都不能寫。

| 子系統 | 主要輸入 | 狀態 |
|---|---|---|
| A. 執行檔架構與模組轉交 | `START/MENU/TOWN/CMBT/CAMP/MAZEMOVE/WRLDMOVE/CHARUTIL.EXE` | 進行中(`docs/re/01`–`06`:模組本體已解鎖至 40.8%;`43`:COMMON 區與私有變數已分開;模組轉交仍未解)|
| B. 執行期模組的呼叫介面 | `BRUN30.EXE` | 進行中(`06` INT ABI 已確認、`40` 認出 MS BASIC 5.60、`41` **DS 基底已解**、`44` **`3Dh` 909 處**、`45` `3Dh` 族;三張表的語意仍多半未解)|
| C. 原生輔助程式庫 | **`USERLIB.EXE`**(另三支已改歸 A) | 進行中(`docs/re/30`:`bm` 族、114 個重定位項;內容未解)|
| D. 角色／隊伍資料與存檔 | `CHARS.DAT`、`GROUPS.DAT` | 進行中(`27`:`CHARS.DAT` 25 槽×94B 定案;`43`:**記憶體側已定位** —— 主陣列 `ds:6822`(15×≥20 word)、隊伍人數 `ds:34F8`(上限 5)、隊員名字 `ds:34E0`;欄位語意與檔案對應未解)|
| E. 規則資料表 | `ITEMS.DAT`、`SPELLS.DAT`、`MONSTERS.DAT`、`TITLES.DAT`、`TOWNDATA.DAT` | 進行中(`docs/re/16`:三張表格式定案,兩個欄位語意有交叉印證;讀取端未解)|
| F. 世界地圖 | `WRLDMAP.BIN`、`TOWNDATA.BIN` | 進行中(`19`/`51`/`52`:103×121、格子是 word、索引 `y×103+x`、**讀取端已讀(條件 1 ✅)**、圖塊 24/25/27/28 = 地城入口、圖塊 1–9 的圖形位址;缺其餘圖塊語意與 `TOWNDATA.BIN`)|
| G. 地城與迷宮 | `DG*MAZE.SQZ`、`MAZEDATA.BIN`、`DT*TEXT.DAT` | 進行中(`50`:`.SQZ` 是文字+跑長 81×51;`51`:**`MAZEDATA` 是 13×8 關卡表(3 欄定案)、`DT*TEXT` 已確認**;缺符號語意、欄 2/3/4/7、讀取端)|
| H. 圖形格式（**只解位元佈局**） | `PICT*.BIN`、`MONST*.BIN`、`*.PIC`、98-byte 圖塊群 | ✅ **RE-DONE**(`19`–`22`、`48`、`49`:圖塊 17×17、`PICT` 153×153、`MONST` 8×17×17 交錯、`WRLDMAP` 103×121、`.PIC` 是 `DRAW` 巨集、調色盤 `0x3D8=0x0E`。四項條件逐條核對見 `49` §4)|
| I. 法術效果表 | `DE*EFF.BIN` / `DE*EFF.MST`、`FIRESTRM/HAILSTRM/WINDSTRM.BIN` | 進行中(`docs/re/29`:`BIN`/`MST` 資料相同、只差標頭;內容未解,可能該歸到 G)|
| J. 戰鬥規則 | `CMBT.EXE`、`FASTCMBT.BIN`、`RNDMONST.BIN` | 進行中(`31`:`w9` 解為魔法相關;`43`/`52`:**戰鬥格 = 15 欄的「當前地圖」陣列**,隊伍在第 9–13 欄;公式未解)|
| K. 輸入語意（**不含 BIOS 層**） | 哪些鍵在哪個畫面有效、對應什麼動作 | 進行中(`docs/re/28`:`X)字` 慣例、五個畫面的鍵表;提示與實作是否一致未驗)|
| L. 中文化落點盤點 | 全部（見 §7） | 未開始 |

`CONFIG.SOS`、`STARTUP.BIN`、`BORDER.BIN` 這類純顯示設定，依 §1.2 **只需知道效果**，
不單獨列為 RE 項目。

### 2.3 唯一允許在閘門前寫的程式碼

只有兩類，而且都不進 remake 本體：

- **`tools/` 底下的分析工具**（IDAPython／IDC 腳本、dump 器、格式驗證器）
- **一次性的格式探測腳本**（放 `/tmp` 或 `tools/scratch/`）

「先寫個 prototype 試試看」不在允許範圍內。要試就在 DOSBox 上試原版。

---

## 3. IDA Pro 是本專案唯一的反組譯工具

完整環境與踩過的坑在 `~/.claude/knowledge-base/retro/ida-pro-9.4.md`，**動手前先讀那份**。
這裡只列本專案一定會用到的、以及與那份 kb 不同的部分。

**一律用 IDA 反組譯。** 不因為某支執行檔「看起來是某某編譯器產的」就改用專用解析工具、
或去找該編譯器的格式文件來套。編譯器來源只是背景資訊，不是 RE 路線的分岔點 ——
所有結論都要能在 IDA 的反組譯與 xref 圖上指出證據。

### 3.1 環境

```
本專案用：ida-pro-9.4-idapython:py312-v1（見 §3.2，Dockerfile 在 kb）
基底 image：ida-pro-9.4-ver3:latest（本機 dist 在 /home/anr2/ida_94_official/dist）
headless： idat（ida 是 GUI）
```

包成 `tools/ida.sh`，樣板抄 `~/cht/大時代的故事/tools/ida.sh`
（含 `--network none`、記憶體與 pid 上限、`-u $(id -u)`、query 用完即拆的暫存 DB）。

```bash
tools/ida.sh analyze TOWN.EXE                        # 產 .i64 + .asm
tools/ida.sh query tools/ida_xref.py TOWN.EXE.i64 <符號>
```

### 3.2 ⭐ IDAPython 可用 —— 但要用修正過的 image

```
ida-pro-9.4-idapython:py312-v1
```

Dockerfile 保存在 `~/.claude/knowledge-base/retro/assets/ida-pro-9.4-idapython.Dockerfile`，
完整原理與實測矩陣見 `~/.claude/knowledge-base/retro/ida-pro-9.4.md`。
**本專案沿用這顆,不要臨時掛載主機 Python,也不要另建功能重複的 image。**

要點三條：

1. **基底 image 的 IDAPython 是靜默失敗** —— 零輸出、零錯誤訊息，
   而且 exit code 在不同 image 上分別回 0 與 1。**唯一可信的訊號是輸出檔本身。**
2. 修法有**兩個獨立根因**：缺 `libpython3.12t64`、以及 `idapyswitch` 的選擇寫進
   `$HOME/.idapro`（所以要以最終執行身分跑，不能用 root）。只修一個等於沒修。
3. 上面那顆在**加不加 `-u $(id -u)` 都能動**，且不留 root-owned 檔案。

| 能力 | 狀態 |
|---|---|
| IDAPython（上面那顆 image） | ✅ 可用，**優先用這個** |
| IDC 腳本 | ✅ 可用，當退路 |
| Hex-Rays 反編譯 | ❌ 16-bit real mode 不支援 → **只有組語，沒有 C** |
| 產 `.asm` / `.i64` | ✅ |

**「只有組語」對本專案是好消息**：daemon_winter 有三次錯誤斷言來自
「反編譯器對含跳表的函式靜默捏造控制流」。這裡沒有反編譯器可信，也就沒有這個坑。

匯出腳本的形狀抄 `~/cht/civ1/tools/ida/export_*.py`（三十幾支現成的）。

### 3.3 四條會反覆咬人的規則

1. **headless 的 `print` 不進 stdout。** 腳本一律把結果寫成明確的輸出檔，
   而且**要驗檔案非空、schema 對、輸入 SHA-256 對、exit 狀態對** ——
   `idat -A` 的 exit code 為 0 不代表腳本真的產出了證據。
2. **IDC 少了 `#include <idc.idc>` 會安靜 exit 1**，沒有任何錯誤訊息。崩掉那次還會把
   `.i64` 留在不可開啟狀態，之後所有指令都回
   `Failed to initialize IDA as library (error code 4)` ——
   **那訊息看起來像授權失效，實際上刪掉 `.i64` 重跑 `analyze` 就好**。
   判別法：拿另一個 `.i64` 跑同一支已知可用的腳本。
3. **不要 grep `.asm` 找位址。** 16-bit 專案的反組譯文字顯示 `segment:offset`，
   而符號名來自 IDA 資料庫的**線性位址** —— 五位十六進位常數在 `.asm` 裡是零命中，
   而零命中與「真的沒人碰」長得一模一樣。要查記憶體用途就逐 byte 呼叫
   `idautils.DataRefsTo` / `get_first_dref_to`。
4. **讀寫判定用 `XrefType()`**，不要 `strstr("mov ...")`（IDA 補多個空格會漏），
   也不要 `print_operand(x,0) == name`（`push` 的第 0 個運算元是來源不是目的）。
   xref 圖抓不到間接寫入（`ptr=&x` → `es:[di]=v`）；看到「讀很多處、寫只有一處」，
   先去看「取址」那幾筆。

### 3.4 容器衛生（沿用 civ1 的做法）

- 原版與 `.i64` **唯讀掛載**，DB 先複製到容器暫存層再分析
- 只把指定的輸出目錄掛成可寫，退出前 `chown` 回目前使用者，不留 root-owned 檔案
- 一律 `--rm --network none`，帶記憶體／CPU／pid 上限
- **不碰共用的 docker 資源**（見 §8）

### 3.5 Ghidra 的定位

**不使用。** 本專案不引入第二套反組譯器 —— 兩份反組譯不一致時，正解是回 IDA 讀原始指令，
不是在兩者之間投票，那只會多一個要維護的位址體系。

---

## 4. 目前已知的一手事實（2026-08-13，開檔第一輪）

> 以下全部是**直接讀檔看到的位元組**，不是推論。信心等級：已確認。
> 推論與待驗證項另見 §5，兩者不要混。

### 4.1 檔案來源

| 項目 | 值 |
|---|---|
| 來源 | archive.org `msdos_Shard_of_Spring_1986` |
| 封裝 | `Shard_of_Spring_1986.zip`，199,245 bytes，TorrentZip（`TORRENTZIPPED-AEA6754A`）|
| SHA-256 | `4982ea6c05134e0afd763f090e68393ba2d070624700b4c1aa7eed25631f262c` |
| MD5 | `994eccfa6477ab53e9f3a6799eebf2e6`（與 archive.org 記載一致）|
| 內容 | `sharspri/` 底下 100 檔、421,839 bytes 未壓縮，時間戳 1996-12-24 |
| 啟動點 | `START.EXE`（archive.org 的 DOSBox 設定也指向這支）|

執行檔 SHA-256 記在 `docs/re/00-inputs.md`，筆記引用位址時必須同時引用該表。

### 4.2 執行檔群的組成

九支遊戲流程 EXE（`START`/`MENU`/`TOWN`/`CMBT`/`CAMP`/`MAZEMOVE`/`WRLDMOVE`/`CHARUTIL`/`MTEST`）
＋ 四支輔助（`USERLIB`/`MIO2`/`WSIO`/`BRUN30`）。全部是 MZ 格式的 16-bit DOS 執行檔。

`BRUN30.EXE`（70,680 bytes）是**執行期模組**（runtime module）：內含
`String Space Corrupt`、`during G.C.`、`No Line Number in`、`module ` 等執行期錯誤訊息。
`START.EXE` 內含 `Wrong version of runtime module`、`Program too large`。

**對 RE 的意義**：九支遊戲 EXE 的邏輯有一部分是靠呼叫 `BRUN30` 完成的，
所以 §2.2 的 B 項（執行期模組的呼叫介面）要先解 —— 把那些 far call 對應到功能之後，
其他 EXE 的組語才讀得懂。**這是 IDA 上的一般化工作（建 entry point 表），
不需要去找任何編譯器的格式文件。**

### 4.3 原始碼模組名洩漏在執行檔裡

編譯器把來源模組名留在 EXE 開頭，這是本專案最有價值的線索之一：

| 執行檔 | 內含來源名 |
|---|---|
| `START.EXE` | `START.BAS` |
| `MENU.EXE` | `MENU.BAS`、`MASTER.INC`、`INSTALL.INC` |
| `TOWN.EXE` | `TOWN.BAS`、`MASTER.INC`、`TOWNCAMP.INC` |
| `CMBT.EXE` | `CMBT.BAS`、`MASTER.INC` |

`MASTER.INC` 出現在三支以上 → 那是共用的常數與變數宣告。
**在 IDA 裡找出三支 EXE 共有的那段結構，等於直接拿到全域狀態的佈局**，
比逐支硬讀省一個量級。

### 4.4 遊戲文字有兩個落點

**落點一：純文字資料檔**（可直接讀，沒有加密沒有壓縮）

| 檔案 | 格式 |
|---|---|
| `DT0-DT7TEXT.DAT` | 3 位數編號 + ASCII 敘述，地城房間文字 |
| `TITLES.DAT` | 雙引號包住的選單標題，CRLF 分隔 |
| `ITEMS.DAT` | CSV：`名稱,小寫名,價格,?,?,?` |
| `SPELLS.DAT` | CSV：`法術名,系別,?,?,?,命中訊息` |
| `MONSTERS.DAT` | 定長 16-byte 名稱 + 一串 16-bit little-endian 數值 |
| `TOWNDATA.DAT`、`GROUPS.DAT`、`CHARS.DAT` | 待解 |
| `CONFIG.SOS` | 17 bytes：`C,C, 1000 , 0 ` + CRLF + EOF(0x1A)。顯示模式設定 |

**落點二：EXE 內的字串常數**（明碼可見）

例如 `TOWN.EXE` 裡的技能清單、`MENU.EXE` 裡的狀態名（`OK,Poisoned,Bound,Still Air,Frozen,D E A D`）
與 24 個道具形容詞、`CMBT.EXE` 裡的陣型座標。
**這一批是中文化的難點**：改長度會動到 EXE 佈局。處理方式見 §7。

### 4.5 二進位資料檔的規律尺寸

| 尺寸 | 檔案 | 初步意義 |
|---|---|---|
| 98 bytes | `BORDER/DOOR/EXITSPOT/FIRESTRM/HAILSTRM/LAVA/MAZEWALL/WATER/WINDSTRM.BIN` | 同一種小圖塊，9 個 |
| 742 bytes | `MONST1.BIN` – `MONST22.BIN` | 怪物圖，22 隻 |
| 1,068 bytes | `DE*EFF.BIN` / `DE*EFF.MST` 各 10 對 | 法術效果，`.BIN`/`.MST` 成對 |
| 5,980 bytes | `PICT1/2/6/7/8.BIN` | 大圖，編號跳號（3/4/5 缺）|
| 836 bytes | `FASTCMBT.BIN`、`FASTWRLD.BIN` | 成對，可能是加速版繪製常式 |
| 24,934 bytes | `WRLDMAP.BIN` | 世界地圖 |
| 16,392 bytes | `STARTUP.BIN` | 16384 + 8，疑似一整頁顯示緩衝 + 標頭 |
| `.SQZ` | `DG1/2/3/5/51/6MAZE.SQZ` | 壓縮迷宮，壓縮法未知；編號跳號（4 缺、多一個 51）|

**編號跳號要當成線索追**，不要當成「缺檔」略過。

---

## 5. 推論與待驗證（不是事實，動工前必須用 IDA 裁決）

### 5.1 RE-01：先建全域盤點（最高優先）

第一件事不是讀任何一支 EXE 的細節，而是**把十三支執行檔全部丟進 IDA 建 DB，
產出一份可比較的初始清冊**（函式數、segment 佈局、entry point、字串表、imports、
跨模組共同結構）。做法直接參考 `civ1/tools/ida/export_inventory.py`。

先有清冊才知道：哪一支最大最值得先讀、`MASTER.INC` 的共用結構落在哪、
`BRUN30` 的呼叫入口有幾個。**沒有清冊就開始讀組語，等於在黑暗裡挑函式。**

### 5.2 RE-02：多支 EXE 之間怎麼傳遞狀態

九支獨立 EXE 共用一份隊伍狀態。`START.EXE` 裡有 `OMENU` 字串與
`Cannot find $  Enter new path: $`，暗示它會去找並啟動別的模組。

**要解的是「傳了什麼」，不是「怎麼傳」**（後者屬於 §1.2 的 DOS 層，不挖）。
也就是說：確認狀態的**佈局與欄位語意**即可，載入機制只要知道
「切到某模組時整份狀態會延續」就夠了。

這一項決定 remake 的整體架構（九個場景共用一個狀態物件，還是真的有序列化邊界），
所以排在 RE-01 之後、其他之前。

### 5.3 記著的疑點（有答案之前不要繞過去）

- `PICT3/4/5.BIN` 與 `DG4MAZE.SQZ` 缺；`DG51MAZE.SQZ` 多出來。是原始封裝就少，
  還是命名規則不是我想的那樣？
- `MTEST.EXE`（3,904 bytes）看起來像開發期的測試程式，不是遊戲流程的一部分 ——
  但要驗過才能這樣寫。
- `USERLIB.EXE`（30,813 bytes）標頭前綴是 `bm` 而非其他 EXE 的 `bz`，且含
  `[Choice ?]`、`[Press ENTER]`、`[Press a key]`、`1WRLD` 等字串 —— 疑似原生組語寫的
  I/O 與繪圖程式庫。**如果是，那 §2.2 的 K 項（輸入語意）主要落在這裡。**

---

## 6. 工作方式：SDD（規格驅動）

```
IDA 反組譯 → docs/re/（證據）→ docs/formats/ + docs/spec/（收攏）→ 標 READY → 才實作
```

**只有標 READY 的規格可以動手。** 規格沒 READY 就寫程式，等同繞過 §2 的閘門。

### 每一輪都要做

0. **先查手上已經有的**（見下）
1. 更新受影響的 markdown
2. **重新檢視既有斷言，清掉被推翻的** —— 不是加註解了事。
   正文改寫成正確答案，推翻紀錄集中到 `CONTEXT.md` 的「已被推翻的斷言」表，
   正文最多留一個指標。單獨讀到那一節的人只會看到那一節。
3. commit + push
4. 更新 `CONTEXT.md` 的「現況一覽」

### ⚠ 下手前的三十秒檢查

反組譯的慣性是「不知道 → 去讀組合語言」，那條反射太強，會跳過「不知道 → 先查」。
**查不到的成本是三十秒，挖錯方向的成本是好幾輪。**

1. `grep docs/` —— 這個結論以前寫過嗎？
2. `grep tools/` 與程式碼 —— 這個腳本已經有了嗎？
3. 手冊／攻略有沒有直接寫答案
4. `translations/glossary.md` —— 要造新譯名之前，表上有沒有

**如果一個問題「感覺以前碰過」，那就是碰過。** 相信那個感覺，先去查。

範圍也適用**跨專案**：`~/cht/` 底下有二十幾個同類專案，工具鏈問題（IDA、DOSBox、
字型、打包）多半有人解過。本檔 §3.2 的 IDAPython 修法就是這樣從 `civ1` 撿回來的 ——
而在撿到之前，本檔第一版已經照 kb 寫了「一律用 IDC」。

更難防的一種：查了，但查到過期的那份。看到「推測／未解」時，**再 grep 一次那個關鍵字**
確認沒有別處標「已驗證」。**「標推測」不等於「還沒解」，只等於「寫那一行的時候還沒解」。**

### oracle 優先序（由高到低）

1. **人親自實測原版**
2. **DOSBox 原版實跑**（`tools/dosbox_run.sh` 待建；設定參考
   `~/.claude/knowledge-base/retro/dosbox-game-configs.md`，`cycles=auto` 是可重現性的敵人）
3. 官方手冊 / 社群攻略
4. IDA 反組譯推論

**低層級不可推翻高層級。** 反組譯看起來再合理，被原版實跑打臉就是反組譯錯了。

⚠ 但**前提是同一個平台的建置**。Shard of Spring 有 Apple II / C64 / DOS 三個版本，
手冊描述的規則不保證等於 DOS 版行為。套手冊之前先問「這一頁講的是哪個平台」。

### 卡在靜態層時，先問「能不能直接跑一次」

daemon_winter 有一條斷言在靜態掃描上卡了兩輪，**DOSBox 十分鐘就推翻了**
（改三個 byte 再跑一次）。裁決成本比繼續擴大靜態掃描便宜一個量級。

---

## 7. 中文化的特殊挑戰（預先記著，不是現在要解）

這款遊戲的文字分成兩層，兩層的難度差很多：

| 層 | 難度 | 說明 |
|---|---|---|
| 純文字 `.DAT` | 低 | 直接改。但 `MONSTERS.DAT` 是**定長 16-byte 名稱**，中文的視覺寬度要重新算 |
| EXE 內字串常數 | 高 | 改長度會動到 EXE 佈局 |

因為要做的是 **remake 而非 patch 原版 EXE**，第二層的正解是
**把字串抽出來變成外部資源，由新引擎載入** —— 而不是想辦法在原 EXE 裡塞中文。
但**抽取的前提是先知道每一條字串的用途與呼叫端**，這又回到 §2 的閘門。

顯示層另一組已知問題（daemon_winter 全部踩過）：
中英混排寬度、點陣字型選擇、原版畫布尺寸容不容得下中文、選單對齊。
這些等 §2.2 的 H 項有答案再談，**現在不要先決定畫布尺寸**。

---

## 8. 環境硬規則

- **編譯一律走 docker**，不污染系統環境
- **Python 一律 docker uv.venv**，不在系統 `pip install`
- 手打 `docker run` 一律帶 `--rm --log-opt max-size=10m --log-opt max-file=3`
- **`game/` 與 `original/` 唯讀**。要分析先複製到 `workplace/`，不得就地修改
- `.i64`、`.asm`、解包後的 binary、原版素材、截圖 **全部 gitignore**
- **不碰共用的 docker 資源**：禁止 `docker image prune` / `system prune` / `volume prune`
  / `builder prune` / `rmi` / `container prune`。這台機器同時放著多個客戶專案的 image，
  誤刪過一次事故
- 測試用存檔一律寫 `/tmp`，不覆蓋 `game/sharspri/` 底下任何檔案

### 派 subagent 時

- 邊界寫進 prompt，不能只靠 agent 自律：明文列出不准改的目錄、不准做的收尾動作
  （commit、push、清理、重編）。**沒寫的就等於允許**
- **agent 還在跑時量到的中間產物不能當結論**
- **`git add` 要指名檔案，不要加目錄**
- 每個 agent 的結論，協調者要**獨立複核一條證據鏈**才收

---

## 9. 目錄結構

```
shard_of_spring/
├── CLAUDE.md              ← 本檔：目標、邊界、閘門、工具、硬規則
├── CONTEXT.md             ← 待建：全專案單一入口（現況、文件索引、術語表、已被推翻的斷言）
├── original/              ← archive.org 原始封裝（gitignore）
├── game/sharspri/         ← 解壓後的原版（唯讀，gitignore）
├── workplace/             ← 分析用的複本、IDA 工作目錄（gitignore）
├── tools/
│   ├── ida.sh             ← headless 包裝
│   └── ida/               ← IDAPython 匯出腳本
├── docs/
│   ├── re/                ← 反組譯筆記（含 00-inputs.md 的 SHA-256 表）
│   ├── formats/           ← 資料格式規格
│   └── spec/              ← 收攏後的實作規格（標 READY 才能動工）
└── translations/
    └── glossary.md        ← 統一譯名表
```

`CONTEXT.md` 建立之後，**它取代本檔成為新 session 的第一份必讀**；
本檔只保留目標、邊界、閘門與硬規則，不重複現況。

## 10. GitHub repo

```
https://github.com/wicanr2/shard_of_spring_cht
```

目前是 **private**。原版素材全部 gitignore，要轉 public 只需一行
`gh repo edit --visibility public`（先確認 git 歷史裡沒有原版檔案）。

---

## 11. 下一步

按順序，不要跳：

1. ~~建 git repo、`.gitignore`、`docs/re/00-inputs.md`~~ ✅ 完成
2. ~~建 IDAPython 可用的 image~~ ✅ 完成：`ida-pro-9.4-idapython:py312-v1`，已用探針驗過
3. 從 `~/cht/大時代的故事/tools/` 抄 `ida.sh`、從 `~/cht/civ1/tools/ida/` 抄
   `export_inventory.py` 過來改
4. ~~**RE-01**：十三支執行檔全部建 DB，產出初始清冊~~ ✅ 見 `docs/re/01-inventory.md`
   → **接續**:讀 3,047-byte loader stub 的 `start_0`，讓 IDA 分析得到 `seg000`
5. 建 DOSBox 環境，把原版跑起來當 oracle
6. **RE-02**（§5.2）：模組間傳遞的狀態佈局
7. 之後才輪到 §2.2 看板上其他子系統
