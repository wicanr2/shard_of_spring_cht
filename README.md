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
| **地城** —— 六座迷宮、能見度裁視野、事件表 | **最終戰** —— 巨龍 ×2 + 希瑞雅妮。這個組成是[反組譯](docs/re/180-scripted-fight-monster-list.md)與[通關紀錄](docs/re/179-final-battle-composition-from-playthrough.md)**兩條獨立證據鏈**得到的同一個答案 |

> ⚠ **截圖與 `assets/` 都含原版美術。** 這讓「repo 維持 private」
> ([`CLAUDE.md`](CLAUDE.md) §10)變成整個專案**最吃重的一條** ——
> 一旦轉 public,洩漏的是原版的資料與美術本身。

---

## 從哪裡開始讀

| 檔案 | 內容 |
|---|---|
| **[`CONTEXT.md`](CONTEXT.md)** | **單一入口** —— 現況、文件索引、術語表、**已被推翻的斷言** |
| [`docs/PLAYING.md`](docs/PLAYING.md) | **要玩的人看這一份** —— 自備原版怎麼轉檔、存檔在哪、常見問題 |
| [`docs/spec/14-remake-worklist.md`](docs/spec/14-remake-worklist.md) | **「還剩什麼沒做」的單一真相來源** |
| [`CLAUDE.md`](CLAUDE.md) | 目標、RE 深度邊界、動工閘門、工具鏈、硬規則 |
| [`docs/re/`](docs/re/) | 181 篇分析筆記,編號即閱讀順序 |
| [`docs/spec/`](docs/spec/) | 收攏後的實作規格,標 READY 才能動工 |

新接手的人讀 `CONTEXT.md` 就能重建全局。

## 目前狀態

**逆向工程階段已結束** —— [`CLAUDE.md`](CLAUDE.md) §2.2 看板的十二個子系統全部 RE-DONE,
規格標 READY,動工閘門全開。現在是 remake 實作階段。

| | |
|---|---|
| 引擎 | Go + Ebitengine,1024×768,美術 4× 整數放大,文字層 TTF |
| 已實作 | 世界地圖、地城、戰鬥、戰場、法術、道具、城鎮、商店、營地、名冊、創角、訓練、治療、經驗、音樂合成、遊戲外殼、自己的存檔格式 |
| 中文化 | 資料檔 439 段(怪物 74 / 法術 33 / 道具 57 / 地城 87 / UI)＋ 模組內字串 **381 / 381 段全部譯完並接回引擎** |
| 還沒做 | 跨平台打包、玩家的轉檔流程文件、七項⛔**擋在 RE**的規則缺口(戰鬥 AI 逐步選格、戰後金幣算式等)—— 逐項見 [worklist](docs/spec/14-remake-worklist.md) §8.1 |

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
並且在**執行時**把它顯示在畫面上 —— 上面戰鬥截圖底部那行
「金幣公式的組裝順序未解」就是這條紀律的樣子。

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
