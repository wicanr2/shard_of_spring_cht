# CONTEXT — 專案脈絡與 RE 知識庫索引

> **這份是全專案的單一入口。** 對話被壓縮、或換一個新 session 接手時,先讀這份,
> 就能重建完整全局,再依索引跳到需要的文件。
> 規則與閘門在 [`CLAUDE.md`](CLAUDE.md),本檔不重複,只放**現況、索引、術語、已被推翻的斷言**。
>
> 最後更新:2026-08-13

---

## 1. 這個專案在做什麼

把 SSI《Shard of Spring》(1986/1987, MS-DOS 版由 Digital Illusions 移植) 的遊戲機制
完整逆向,建立一份**可查、可驗證的 RE 知識庫**,並在此基礎上做 remake 與繁體中文化。

三條硬性原則(細節見 `CLAUDE.md`):

1. **RE 沒確認完成,不准寫任何 remake 程式碼**(`CLAUDE.md` §2 閘門)
2. **RE 的深度止於遊戲機制,不挖 DOS/BIOS**(`CLAUDE.md` §1.2)
3. **一律用 IDA Pro 反組譯**,不引入第二套位址體系

---

## 2. 現況一覽

### 已完成

| 領域 | 狀態 |
|---|---|
| 原版取得與雜湊清冊 | 13 支執行檔 + 封裝,`docs/re/00-inputs.md` |
| IDA 工具鏈 | `tools/ida.sh` + 三支 IDAPython(`export_inventory` / `dump_func` / `find_imm`)|
| 十三支執行檔清冊 | `docs/re/01`,含函式/段/字串/立即數統計 |
| loader stub 全解 | `docs/re/02`,3,047 bytes,十一支共用 |
| `bz` 模組標頭與重定位 | `docs/re/03`,`+0x16` 經 11/11 獨立印證 |
| EXE 佈局與進入點算式 | `docs/re/04`,`+0x16` 三個獨立來源 11/11 一致 |

### 進行中

| 項目 | 卡在哪 |
|---|---|
| **讓 IDA 分析模組本體** | 模組區的位置與長度已三重驗證(`docs/re/04` §2)。MZ 進入點指向 stub 而非模組本體,**執行期怎麼跳進模組本體仍未解** |

### 一句話現況

**十一支遊戲 EXE 有 70–93% 的內容還沒被反組譯過**;
瓶頸是單一的(IDA 不知道 `seg000` 是程式碼),解一次十一支一起解鎖。
子系統看板在 `CLAUDE.md` §2.2,目前 A 項進行中,其餘未開始。

---

## 3. 文件索引

| 編號 | 主題 | 一句話 |
|---|---|---|
| [`00-inputs.md`](docs/re/00-inputs.md) | 輸入檔清冊 | 13 支執行檔的 SHA-256;**所有結論只對這些雜湊成立** |
| [`01-inventory.md`](docs/re/01-inventory.md) | 初始清冊 | IDA 只分析到 loader stub;UNK 段是明碼 8086 不是壓縮 |
| [`02-loader-stub.md`](docs/re/02-loader-stub.md) | loader stub | 依 `PATH=` 載 `BRUN30.EXE`、依 `LIB=` 載 `USERLIB.EXE` |
| [`03-bz-module-header.md`](docs/re/03-bz-module-header.md) | 模組標頭 | `+0x16` = 模組大小(paragraphs);重定位分兩類修補 |
| [`04-module-layout-entry.md`](docs/re/04-module-layout-entry.md) | 佈局與進入點 | `[模組區][stub]`;MZ CS:IP 指向 stub;`sub_14CB8` 是通用 MZ 載入器 |

工具:`tools/ida.sh`(headless 包裝)、`tools/ida/*.py`(匯出腳本)。
原始 JSON 在 `workplace/ida/out/`(gitignore,可用 `docs/re/01` §6 的指令重跑)。

---

## 4. 術語表

| 術語 | 定義 |
|---|---|
| **loader stub** | 十一支遊戲 EXE 共用的 3,047-byte 前導碼,負責載入執行期。不含遊戲邏輯 |
| **模組本體** | `seg000` 起的 `bz<NAME>` 區塊,遊戲的實際程式碼與資料所在 |
| **執行期模組** | `BRUN30.EXE`,由 stub 依 `PATH=` 搜尋載入 |
| **使用者程式庫** | `USERLIB.EXE`,依 `LIB=` 搜尋載入,可省略 |
| **`bz` 標頭** | 模組本體開頭的結構,簽章 `0x7A62` |
| **分類門檻** | `bz` 標頭 `+0x12`,重定位時用來判斷「這個位址屬於模組還是執行期」 |

新術語一律先進本表再用(`rulebook/50`)。

---

## 5. 新 session 必須知道的關鍵事實

### 位址換算(踩過一次)

`seg005`(loader stub 的 CODE 段)基底是 linear **`0x10180`**,不是 `0x10000`:

```
linear   = 0x10180 + 段內位移
檔案位移 = 512 + (linear − 0x10000)
```

用錯基底解出來的資料**看起來像合理的程式碼位元組**,不會報錯。
**判別法:拿一個已知符號回推基底**(`mov word_10AC5, dx` + 立即數 `0x945` → 基底 `0x10180`)。

### 一支函式裡 `es` 會換基底

`sub_14BDD` 前半的 `es:xx` 指 `BRUN30` 控制區塊,`mov es, di` 之後才是模組節區。
**不追蹤 `es` 何時被重設就抄位移,會把兩個結構混成一個**(`docs/re/03` §1 的但書)。

### 「零命中」與「不存在」長得一樣

`far_call_targets` 在十二支上都是 0 —— 那是「沒被分析」的後果,不是「沒有跨段呼叫」。
下任何「不存在」的結論前先做正對照(`~/diagnosis-notes/docs/02-query-returned-empty/`)。

### 橫跨不同輸入卻不動的數字 = 查法壞了

`func=10` 出現在十二支大小差十倍的 EXE 上,是這一輪最重要的線索。
**不要把它讀成結果。**

---

## 6. 已被推翻的斷言

| 曾經寫過 | 真相 |
|---|---|
| 十一支的 loader stub「逐位元組相同」 | **只有結構相同**(函式大小序列與段大小一致);位元組不同,差異來自重定位與嵌入的模組名。`docs/re/01` §2 |
| `bz` 標頭有 `+0x32` / `+0x34` / `+0x54` 三個欄位 | 那三個位移屬於 **`BRUN30` 的控制區塊**(`ds:0CACh`),不是模組標頭。錯在沒注意 `sub_14BDD` 中途換過 `es` 的基底。`docs/re/03` §1 |
| `ds:0A28h` 存的 `0x0B06` 是「模組本體的進入點位移」 | 是**檔名字串的指標**。`sub_14CB8` 拿它去 `INT 21h AH=3Dh` 開檔。**常數是位址還是指標,要看使用端怎麼用,不能從值的樣子猜**。`docs/re/04` §4 |
| kb 寫的「IDAPython 實測無輸出,一律寫 IDC」 | 可用,但要修正過的 image(`ida-pro-9.4-idapython:py312-v1`)。兩個獨立根因見 `~/.claude/knowledge-base/retro/ida-pro-9.4.md` |

> 這張表只增不減。**推翻一條斷言時,要同時把正文改寫成正確答案**,
> 不是在正文加註解(`rulebook/63`)。

---

## 7. 每一輪都要做

0. **先查手上已經有的**(`grep docs/`、`grep tools/`、看術語表)
1. 更新受影響的 markdown
2. **清掉被推翻的斷言**,正文改寫成正確答案,推翻紀錄補進 §6
3. commit + push
4. 更新本檔 §2 現況一覽
