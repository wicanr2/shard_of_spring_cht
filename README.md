# 春之石 Shard of Spring — 逆向工程 + Go remake + 繁體中文化

SSI《Shard of Spring》(1986 / 1987,MS-DOS 版由 Digital Illusions 移植)的
**完整逆向工程紀錄**,以及在其上重寫的引擎與繁體中文翻譯。

**這個 repo 是 private**,收錄了 `cmd/convert` 轉出的資產(`assets/`)與含原版
美術的引擎截圖 —— 專案負責人裁定([`CLAUDE.md`](CLAUDE.md) §8)。
⛔ **原版磁片的內容本身**(`game/`、`original/`)仍然不進版控。

⚠ **repo 收錄 ≠ 發行附帶。** 公開發行的產物只有引擎程式碼與翻譯文本,
玩家自備合法原版、自己跑轉換器。

---

## 現在長什麼樣

| | |
|---|---|
| ![標題](docs/images/01-title.png) | ![主選單](docs/images/02-menu.png) |
| **標題畫面** | **主選單** —— 逐項對應原版的 `L)oad a Party` / `C)har Utilities` / `P)rogram Notes` / `Q)uit`。⚠ 原版**沒有「開新遊戲」**([`spec/15`](docs/spec/15-game-shell.md) §1)|
| ![世界地圖](docs/images/03-world.png) | ![城鎮](docs/images/04-town.png) |
| **世界地圖** —— 121 × 103 格,9×9 視野,圖塊 4× 整數放大 | **城鎮**(翠綠村)—— 建築清單、商店、旅店、酒館、訓練所、治療所 |
| ![地城](docs/images/05-maze.png) | ![戰鬥](docs/images/06-combat.png) |
| **地城** —— 六座迷宮、能見度裁視野、事件表。左上角的地城名對照[入口表](docs/re/222-dungeon-names-by-entry.md) | **最終戰** —— 巨龍 ×2 + 希瑞雅妮。這個組成是[反組譯](docs/re/180-scripted-fight-monster-list.md)與[通關紀錄](docs/re/179-final-battle-composition-from-playthrough.md)**兩條獨立證據鏈**得到的同一個答案 |
| ![營地](docs/images/07-camp.png) | |
| **野外營地** —— 地圖留著、隊伍那一格換成帳篷、十一個指令開在右下角那個框,照原版的版面 | |

> ⚠ **截圖與 `assets/` 都含原版美術。** 這讓「repo 維持 private」
> ([`CLAUDE.md`](CLAUDE.md) §10)變成整個專案**最吃重的一條** ——
> 一旦轉 public,洩漏的是原版的資料與美術本身。

### 三十秒的樣子

[`docs/promo/shard-of-spring-cht-promo.mp4`](docs/promo/shard-of-spring-cht-promo.mp4)
—— 標題、主選單、世界地圖、城鎮、地城、最終戰,最後接一段**原版 vs 重製版**的並排比較,
41 秒,配的是本專案自己寫的場景配樂。

⚠ 比較段左邊的原版畫面是 [`tools/dosbox_run.sh`](tools/dosbox_run.sh) **實跑抓的**
(路線見 [`re/139`](docs/re/139-oracle-reaches-gameplay.md)),不是掃描或網路上的圖 ——
兩邊都得是自己跑出來的,比較才成立。三組:標題、主選單、世界地圖。標題那一組是**同一張 CGA 美術**
(`STARTUP.BIN` 轉出來的),世界地圖那一組是**同一支隊伍、同一個位置**。

⚠ **畫面是真的按鍵驅動錄出來的**(`engine/promo_test.go` 每格寫一張 PNG),
不是擺好狀態的定格 —— 推廣片要證明的是「玩得動」,而定格不是那個證據。
重錄:`tools/promo.sh`。

⚠ 影片含**原版美術**,與 `docs/images/` 同一個地位:private repo 下沒有問題,
**對外散布是另一件事**,要專案負責人另行決定。

---

## 從哪裡開始讀

| 檔案 | 內容 |
|---|---|
| **[`CONTEXT.md`](CONTEXT.md)** | **單一入口** —— 現況、文件索引、術語表、**已被推翻的斷言** |
| [`docs/PLAYING.md`](docs/PLAYING.md) | **要玩的人看這一份** —— 自備原版怎麼轉檔、存檔在哪、常見問題 |
| [`docs/column-shard-of-spring.md`](docs/column-shard-of-spring.md) | **這款遊戲是什麼、當年在台灣怎麼賣的** —— 含精訊 1987 中文說明書的一手證據 |
| [`docs/spec/14-remake-worklist.md`](docs/spec/14-remake-worklist.md) | **「還剩什麼沒做」的單一真相來源** |
| [`CLAUDE.md`](CLAUDE.md) | 目標、RE 深度邊界、動工閘門、工具鏈、硬規則 |
| [`docs/re/`](docs/re/) | 217 篇分析筆記,編號即閱讀順序 |
| [`docs/spec/`](docs/spec/) | 收攏後的實作規格,標 READY 才能動工 |

新接手的人讀 `CONTEXT.md` 就能重建全局。

## 目前進度

**可以從頭玩到通關,三平台都有發行版。**

| 階段 | 狀態 |
|---|---|
| **逆向工程** | ✅ 結束。[`CLAUDE.md`](CLAUDE.md) §2.2 看板的**十二個子系統全部 RE-DONE**,規格標 READY。`docs/re/` 225 篇筆記 |
| **引擎** | ✅ 世界地圖、地城、戰鬥、戰場、法術、道具、城鎮、商店、營地、名冊、創角、訓練、治療、經驗、技能點、音樂都實作了。18,032 行 Go + 12,983 行測試。⚠ 兩條規則明確地**沒有**實作(見下面「與原版的差異」),而且標在遊戲畫面上 |
| **繁體中文化** | ✅ 資料檔 439 段(怪物 74 / 法術 33 / 道具 57 / 地城 87)＋ 模組內字串 **381 / 381 全部譯完並接回引擎** |
| **打包發行** | ✅ `v0.3.0`(2026-08-20,三輪 QA 之後):Linux **AppImage** / Windows zip / macOS universal,見 [Releases](../../releases)。`tools/release.sh` 一鍵三平台。⚠ 這一版**資產要重轉**(多讀一個原版檔 `RNDMONST.BIN`)|
| **對照原版三輪 QA** | ✅ 用 DOSBox 跑原版逐畫面比對,三輪共約 55 條。規則層只錯四條,而**四條都沒有症狀** —— 升級門檻、迷宮兩軸、地城遭遇、創角重擲上限([worklist](docs/spec/14-remake-worklist.md) §12)|
| **還開著的** | 只剩場景架構重構 C2,而它的[重啟判準量過了、未達成](docs/spec/14-remake-worklist.md) —— 照規則不動 |

進度的單一真相來源是 [`docs/spec/14-remake-worklist.md`](docs/spec/14-remake-worklist.md),
⛔ 不要從這裡複製狀態。

## 與原版的差異

規則層的目標是**一致**;差異集中在載體、呈現與現代環境的必要調整。

| | 原版(1986/1987 MS-DOS) | 這個重製版 |
|---|---|---|
| **遊戲規則** | — | **一致** —— 命中、傷害、經驗曲線、升級成長、法術效力、遭遇、可通行性、事件都照反組譯結果實作,並用 DOSBox 跑原版逐畫面驗過三輪。兩處例外見下面的「缺口」表 |
| **亂數序列** | 自己的 RNG | **不重現**([裁定](docs/spec/14-remake-worklist.md) §9)。同一個種子在本引擎內可重現,但與原版不同 |
| 執行方式 | 九支 EXE 靠 `retf` 互相轉交,共用 `BRUN30` 執行期模組 | 單一執行檔,九個場景共用一份狀態 |
| 畫面 | CGA 320×200,四色 | 1024×768。**美術 4× 整數放大**(不模糊、不重新上色),文字另外用向量字型畫 |
| 語言 | 英文 | 繁體中文。譯名以**精訊**版為主([glossary](translations/glossary.md)) |
| 存檔 | 直接改 `CHARS.DAT` / `GROUPS.DAT` | **自己的 JSON 格式**,多存檔。⛔ 不寫回原版檔案。可匯入原版存檔 |
| 音樂 | 兩首 PC 喇叭方波(通關、死亡),其餘場景安靜 | 同樣兩首 **+ 六首場景配樂**,**F5 切換原版/重製/關閉,預設原版**([spec/13](docs/spec/13-sound.md) §7) |
| 開機防拷 | 手冊查表題 | 保留流程,**按 Enter 就過**。⛔ 不從程式裡取出答案自動通過 |
| 商店賣出 | 沒有這個功能 | 一樣沒有 |
| 按鍵 | 小鍵盤模板(`2/4/6/8` 移動、`0` 攻擊…)| 方向鍵為主,原版的字母鍵保留;少數位置不同會標在提示列 |
| 印表機輸出 | `P)rint` 走印表機 | 改成畫面顯示 |
| 遊戲資料 | 隨磁片附帶 | **不附帶**。玩家自備合法原版,跑一次轉換器 |

還有一類差異是**誠實標記**而非模仿:少數細節沒有讀到的地方,引擎**不猜**,
而是把缺口寫在遊戲畫面上。目前有兩條:

| 缺口 | 現在怎麼做 |
|---|---|
| 一場遭遇有幾隻怪 | 全解:遭遇表是 `RNDMONST.BIN`,72 列 × 6 欄([`re/225`](docs/re/225-encounter-monster-count-anchor.md));隻數由欄 1 決定(它同時是上限),四個候選按「這一種放幾隻」成串放進場 |
| 怪物施法 | 全解:投入 = 法力單價 ×2(不夠就全押)、目標格 = 它鎖定的那個人([`re/226`](docs/re/226-monster-cast-invest-and-target.md))|

⛔ 兩項都不湊一個看起來合理的值 —— 湊了就等於自己發明規則,而玩家看不出來。

[`re/218`](docs/re/218-four-named-assumptions-audited.md) 逐條查證掉四項舊的具名假設,
**其中一項的問題本身不成立** —— 原版根本沒有「法術等級」這個量,是規格多寫了一個中間量。

### 怎麼跑

```bash
# 一律走 docker,不裝系統 Go(CLAUDE.md §8)
# assets/ 已經在 repo 裡;要重新轉(改了轉換器時)才需要這一行
tools/go.sh run ./cmd/convert -in /game/sharspri -out /out
tools/go.sh build && ./build/shard -assets assets
tools/go.sh test -count=1      # ⚠ 一定要帶 -count=1,見下
```

⚠ **測試一定要帶 `-count=1`。** `package main` 匯入 ebiten 會在 `internal/ui`
的 `init()` 呼叫 `glfw.Init()`,沒有 `DISPLAY` 就 panic —— 那發生在任何測試
跑起來之前。`tools/go.sh` 已經內建 Xvfb,但**一旦有過一次成功結果進 build cache,
`go test` 就回 `ok (cached)`,綠燈是快取在說話**。

---

## 這個專案的核心紀律

[`CLAUDE.md`](CLAUDE.md) §2.1 定義 **RE-DONE** 要四項條件同時成立:
在 IDA 讀過原始指令、用 xref 確認讀寫端、**有一份獨立資料互相印證**、
筆記標明輸入檔與信心等級。

信心等級只有四種寫法:**已確認 / 證據充分 / 假設 / 未知**。

**未解的東西不准填一個看起來合理的值。** 用具名常數 + 註明是假設,
並且在**執行時**把它顯示在畫面上 —— 城鎮畫面底部那行
「角色創造的基礎屬性骰法未解,只套種族修正」就是這條紀律的樣子
(`internal/town` 的 `Unresolved`)。

⚠ 這張清單**會縮短**:解出來的就從清單上拿掉,連同那個具名假設一起。
戰後金幣曾經在上面,[`re/207`](docs/re/207-gold-formula-closed.md) 解出算式之後就撤了。

### 已被推翻的斷言

[`CONTEXT.md` §6](CONTEXT.md) 有一張推翻紀錄表,每條附**錯法**。
那張表記的不是「哪裡錯了」,是**「在哪些地方容易推錯」**,
對接手的人比正文更有用。反覆出現的形狀:

- **切分單位錯**(算術全都成立,但真正的格數不同)
- **起點偏移一個 byte**(讓交錯資料整個換半邊)
- **值域對不等於語意對**
- **拿飽和的兩端去驗尺度參數**(端點在任何尺度下都一樣,驗不出東西)
- **把「觀察到的最小值」當成分佈的下界**(沒算那個端點在這個樣本數下看得到的機率)

---

## 授權與邊界

- 引擎程式碼與翻譯文本是本專案的產出
- **原版執行檔、資料檔、美術一律不散布**;`game/`、`original/` 進 `.gitignore`
- 不協助破解 DRM 或修改付費驗證
