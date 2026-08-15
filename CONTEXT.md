# CONTEXT — 專案脈絡與 RE 知識庫索引

> **這份是全專案的單一入口。** 對話被壓縮、或換一個新 session 接手時,先讀這份,
> 就能重建完整全局,再依索引跳到需要的文件。
> 規則與閘門在 [`CLAUDE.md`](CLAUDE.md),本檔不重複,只放**現況、索引、術語、已被推翻的斷言**。
>
> 最後更新:2026-08-15

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
| 模組程式碼起點 | `docs/re/05`,`bz+0x30`,11/11 皆為合法指令起點 |
| **執行期 INT ABI + 模組解鎖** | `docs/re/06`,涵蓋率 0.7% → **40.8%**,160 個函式可讀 |
| 三張派工表 | 位置與內容 `docs/re/07`;**容量 105/165/≤204 已由 `68` 量出**,語意見 `77`(算術 API)|
| 程式碼／資料分離的方法學 | `docs/re/08`,追蹤=下界 5.4%、線性=上界 42.8% |
| **`bz` 標頭是節區表** | `docs/re/09`,六個欄位 66/66 對上 IDA 段起點;程式碼只在 `[0, +0x24)` |
| 執行期不 far call 進模組 | `docs/re/10`,13 處 far 轉移全在 `BRUN30` 內部 |
| 追蹤乾淨終止,下界確定 | `docs/re/11`,靜態可達僅 1,973 bytes;模組區 0 處 far 轉移 |
| **執行期 API 對照表(3/7 高頻)** | `docs/re/12`–`14`:`3F:61` 描述子複製、`3E:42`/`3E:44` 參數傳遞 |
| `BRUN30` 補齊分析 | `docs/re/14`,34.9% → 44.0%,709 → 770 個函式 |
| **模組轉交的檔名來源** | `docs/re/15`,`BRUN30:0x10C25` 把字串描述子抄進 `ds:0B06h` |
| **規則資料表(子系統 E)** | `docs/re/16`,`MONSTERS` 36B×74、`SPELLS`/`ITEMS` CSV;`w5`=怪物圖、欄1=符文系別 |
| **BSAVE 容器 + 地圖尺寸(H/F)** | `docs/re/19`,52/52 是 BSAVE;世界地圖 12,463 格、跨距 103(**兩軸方向由 `141` 訂正為東西 121 × 南北 103**)|
| **CGA 像素佈局(H)** | `docs/re/20`,320×200 2bpp、掃描線兩區交錯 |
| **圖塊格式(H)** | `docs/re/21`,BASIC `GET` 陣列 17×17,**九個檔各自畫出自己的名字** |
| **`PICT*` 定案 / `MONST*` 未解** | `docs/re/22`,`PICT` 153×153 **已渲染驗證**;`MONST` 讀法不成立 |
| **模組↔資料檔對照** | `docs/re/23`,每個子系統的讀取端已縮到模組層級 |
| **工具衝突已記錄** | `docs/re/24`,`trace_module` 會洗掉 `unlock_module`;掃描前先看分母 |
| **字串存放形式(L)** | `docs/re/25`,長度+指標描述子;指標基底未解 |
| **重定位表 11/11 印證節區欄位** | `docs/re/26`,只有 7 項,正好是 `09` 解出的七個欄位 |
| **角色槽(D)** | `docs/re/27`,`CHARS.DAT` 25 槽×94B,兩串 0/1 = 20 技能旗標 |
| **輸入語意(K)** | `docs/re/28`,`X)字` 慣例;戰鬥是唯一用方向鍵的畫面 |
| **`DE*EFF` 成對檔(I)** | `docs/re/29`,`BIN`/`MST` 資料逐位元組相同,只差 BSAVE 標頭 |
| **`USERLIB` 自成一族(C)** | `docs/re/30`,`bm` 簽章 + 114 個重定位項;另三支是一般 `bz` 模組 |
| **戰鬥欄位(J)** | `docs/re/31`,等級序列當對照組;`w9` = 魔法相關,`w8` 訂正 |
| **DOSBox oracle 進得到遊戲** | `docs/re/139`,開機查表題按 Enter 就過;載入 PARTY #5、走進 Green Hamlet、讀得到價目表 |
| **手冊附錄六張統計表** | `docs/re/140`,速度/力量/命中/HP/SP/降魔全部化成規則,三張與反組譯互相印證 |
| **世界地圖兩軸訂正** | `docs/re/141`,**121 東西 × 103 南北、索引 `x×103+y`** —— 實跑推翻靜態結論 |
| **城鎮服務的基準價** | `docs/re/142`,住宿 25、食糧 2、治療 2/點、解毒 40、解束縛 94/級、復活 100/級;**原版沒有賣出功能** |
| **記錄佈局補完** | `docs/re/144`,讓原版自己造一筆記錄再逐位元組比對:背包 10 格哨兵 99、位移 74–83 是第二串旗標、位移 89 = 剩餘技能點數、`'0'`/`'*'` 分家 |
| **角色創造流程** | `docs/re/143`,種族修正確實生效(5/5)、初始生命 = 加完修正的體能、技能點數 = 智能;骰法未解但 **3d6 已排除** |
| **戰場 15×15(M10)** | `docs/spec/12`,行動點數 = 速度、前進 2 / 轉向 1 / 攻擊 3、只打面前那一格、走上外圈離場 —— M4 標的封鎖由手冊 p.34 半行字解除 |
| **曆法定案** | 月 **1–21**(手冊的 22 是錯的)、日 1–34、時 4–26;`docs/re/140` §4 |
| **11 段酒館傳聞全部定位** | `docs/re/142` §5,第 11 段是續作預告不是傳聞;譯文已補 |
| **時鐘的分子** | `docs/re/149`,每一個動作一格(轉身、存檔都算);`GROUPS.DAT` 位移 35 = 東西、37 = 南北 |
| **金幣管線的形狀 + `3F:23` = 次方** | `docs/re/152` §2.2/§2.3,每隻怪物 `階級^p` 兩次、× RND、+1,總額再 `× ds:96E8` 與一次 `× 0.5` 的隨機。⚠ 四個常數是執行期變數(不在檔案裡),而**手冊沒寫金幣** → 剩下唯一的路是 oracle 實測。**引擎已開始發金幣**(具名佔位)—— 不發也是錯的,玩家會買不起東西 |
| **先攻每回合重排** | `docs/re/159`,順序表排序前先重建成 `0…13`;排序後 `inc` 的計數器別處被 `cmp …, 2` —— **只跑一次的話那是死碼**。`ReorderEachRound` 從實作決定變成有依據的選擇 |
| **戰鬥屬性 18 的形狀** | `docs/re/158`,每單位一份的 **±1 偏好**:開場 0、第一次用時擲 50% 硬幣(**值定了就不再重擲**)、遇阻取負,在移動常式裡選兩條對稱分支之一。⚠ 分支內容沒讀完,**不要據此實作 AI**;屬性 14 仍未命名 |
| **魔法道具的成功率是百分比** | `docs/re/157`,`CMBT 0x17C67`:`擲骰(100) ≤ ITEMS 欄6`。⚠ 推翻 `re/91` 的「分母 26 → 戒指法杖 27%–58%」—— **實際是 7%–15%** |
| **屬性算式的形狀** | `docs/re/156`,`屬性 = INT(RND×A + RND×A + B)` —— 兩次亂數同範圍、**只取整一次** → 三角分佈,不是先前假設的 `4d3`。⚠ `A`/`B` 仍未解,15 個樣本約束到 4.9 / 4 |
| **迷宮的兩個機關** | `docs/re/155`,寶石謎題答案 `BBRG` → 傳送到(欄46,列37)面西;治療池由 `GROUPS.DAT` **位移 63** 計數、上限 11、狀態 > 2 治不了、回血夾在滿血。⚠ 規則已實作成純函式,**觸發它們的事件未解**,還沒接上畫面 |
| **擲骰面數 = 100** | `docs/re/154`,手冊 p.13 把命中率印成 `技巧 × 4` **%**,而程式算 `擲骰 ≤ 技巧 × 4` —— 兩者相等只在 100 面時成立。狂暴的 `> 75` 因此是四分之一。`combat.Unresolved` 清空 |
| **傷害公式收口** | `docs/re/153`,三條分支的擲骰面數各不相同;**力量加值 `floor((力量−7)×0.5)` 先前整個漏掉**;`k₁` 由手冊定為 0.5(而且 1.0 被硬排除 —— 同一個變數在別處當機率門檻用);狂暴(屬性16)+ 第二次擲骰 > 75 → **傷害加倍**(`hacks` vs `hits`);傷害 < 1 不扣血 |
| **戰後結算的完整算式** | `docs/re/152`,`每人所得 = INT(九個怪物槽的屬性19總和 ÷ 有資格人數)`;寫回是 `MKS$(CVS(舊值)+所得)`。⚠ 總和**不篩死活**,逃走的怪物照樣給經驗 |
| **三個執行期 API** | `docs/re/152` §3:`3D:34` = `RND`(讀出線性同餘)、`3D:03` = `INT()`、`3D:1C` = `MKS$` —— `INT(RND×N)+1` 這個成語的零件現在全部有名字 |
| **山地的時鐘成本** | `docs/re/151`,起點或終點是山地的那一步花 **2 格**(山→山也是 2 不是 3);手冊 p.32 的「快兩倍」得到形狀。⚠ 同時量到**遭遇不是只靠倒數歸零** —— 倒數還剩 2 就開打 |
| **經驗值的位移(D 收口)** | `docs/re/150`,`CHARS.DAT` **位移 90–93,MBF 單精度**;戰後只有「朝向 > 0(還在場上)**且** 狀態 < 5」的成員分得到 —— **逃走的人拿不到** |
| **戰鬥的鍵盤與行動點** | `docs/re/150` §4,`MP=` 印在畫面上:轉身 1、攻擊 3、被友軍擋住不扣點、點數不足自動換人;**戰鬥中 `S` 是音效開關不是存檔** |
| **營地的 11 個指令** | `docs/re/150` §5.2,`P/S/#/C/R/T/D/E/H/I/U` —— 先前的規格只有三個 |
| **五個屬性的原版標籤** | `docs/re/150` §5.1,`Speed/Strength/Intellect/Endurance/Skill`,在 `TITLES.DAT` 與兩支 EXE 裡;推翻「沒有畫面標籤」|
| **遊戲有音樂(新子系統)** | `docs/re/148` + `spec/13`,15 段 BASIC `PLAY` 樂譜;解析器與方波合成已實作 —— **十二個子系統的清單裡沒有聲音,不是不做是沒發現** |
| **離開迷宮** | `docs/re/147`,走到格陣列界外就離開(原版印 `Leaving maze ..`);答案在字串表裡,而我先窮舉了按鍵 |
| **迷宮離場的陰性結果 + 調色盤** | `docs/re/146`,`ESC`/`E` 已排除;畫面用了不只一組 CGA 調色盤(迷宮是灰/洋紅,世界是棕/紅)|
| **城鎮座標對應** | `docs/re/145`,`TOWNDATA.BIN` 的三欄表(東西 / 南北 / 建築數);實作端已改用它,佔位排序刪除 |
| **題庫驗證資料欄位(E/J)** | `docs/re/33`,`ITEMS` 欄2=價格、欄3=型別相依主數值 |
| **API 索引(B)** | `docs/re/36`,模組實際用 125 個索引/6,429 次;**前十個佔 75%** |
| **高頻 API 是 BASIC 基本操作(B)** | `docs/re/37`–`38`:指派/加法/搬移/堆疊管理。**遊戲功能要往低頻索引找** |
| **xref 對 `ds:xxxx` 全域無效(方法)** | `docs/re/39`:IDA 不建 o_mem 的資料參考 → xref 空。改掃運算元文字(`tools/ida/find_dsref.py`)。**已寫進 `CLAUDE.md` §2.1** |
| **`BRUN30` = MS BASIC Compiler Runtime 5.60(B)** | `docs/re/40`:讀自它自己的字串。可當 §2.1 條件 3 的獨立對照來源 |
| **子系統 B 已 RE-DONE(第八個)** | `docs/re/68` §3:表容量用「表後面接什麼」定出;執行期內部依 §1.2 界線排除 |
| **子系統 A 已 RE-DONE(第七個)** | `docs/re/67` §3:模組轉交、`bm` 入口 `0x3A`、`ds:0B06h` 由誰填三題一次關掉 |
| **子系統 C 已 RE-DONE(第六個)** | `docs/re/66` §4:字串 → 引用點 → 槽號三步對應,呼叫者分布自洽 |
| **子系統 L 已 RE-DONE(第五個)** | `docs/re/62` §5:中文化落點總表,46% 可變長 / 54% 等長 |
| **子系統 I 已 RE-DONE(第四個)** | `docs/re/61` §4:掛錯的檔案重新歸位;八張圖塊遊戲從不載入(帶正對照)|
| **子系統 G 已 RE-DONE(第三個)** | `docs/re/60` §5。⚠ 曾在 `57` 判定、`59` 因範圍擴大收回、`60` 重判 —— 過程留在紀錄裡 |
| **子系統 F 已 RE-DONE(第二個)** | `docs/re/54` §4:世界地圖的五個檔案全解,兩條繪製路徑都讀過 |
| **子系統 H 已 RE-DONE(第一個)** | `docs/re/49` §4:四項條件逐條核對。圖塊/PICT/MONST/WRLDMAP/`.PIC`/調色盤全解 |
| **模組 ↔ 原始碼對應已知(A)** | `docs/re/47`:十一支的 `.BAS` 檔名 + `MASTER.INC`(八支共用)/`TOWNCAMP.INC`/`INSTALL.INC` |
| **中文化規模已量(L)** | `docs/re/47` §5:模組內文字 952 段 / 15,738 B(清單 `generated-text-inventory.json`)|
| **遊戲狀態的存放方式已解(D–L 基礎)** | `docs/re/43`:COMMON 區 `0x34E0`–`0x681A` 七支共用;主陣列 `ds:6822`(15×≥20 word);隊伍人數 `ds:34F8` 上限 5 |
| **`BRUN30` 的 DS 基底已解(B)** | `docs/re/41`:`DS = seg002`,`ds:XXXX` = 線性 `0x1FE00 + XXXX`。**所有 `ds:` 全域現在都有確定的檔案位移** |
| **派工表是薄包裝(B)** | `docs/re/35`,5 支共用實作;`sub_199CC` 的參數是 **4×3 網格**;派工表有進入點直接落在 trampoline 呼叫端上(`3E:44` 已閉環)|
| **五個種族 + 角色欄位名(D)** | `docs/re/34`,Humans/Trolls/Dwarfs/Elf/Gnomes;五個屬性。修正表**已排除三種存放形式** |
| **中文化落點盤點(L)** | `docs/re/18`,兩層 ≈35,800 字元;模組內嵌 362 條佔 39% |
| **世界地圖與地城(F/G)** | `docs/re/17`,`WRLDMAP` 每格 2 bytes、12,467 格 35 種圖塊;六個 `.SQZ` 一律 82 列 |

### 進行中

| 項目 | 卡在哪 |
|---|---|
| **經驗值已定位** | `CHARS.DAT` **位移 90–93,MBF 單精度**(`docs/re/150`)。引擎已改成讀寫原版記錄,旁掛的 `save/exp.json` 移除。⚠ **每一場給多少**仍未解 —— 均分是具名假設 |
| **引擎 M0–M8 全部完成** | 世界地圖、角色與存檔、戰鬥、迷宮與事件、法術與道具、中文化上線、城鎮/商店/營地/名冊。⚠ 各里程碑的未解項見 `docs/spec/03-engine-plan.md` 的狀態欄 |
| ~~海洋(值 11)的圖從哪來~~ | **問題本身是錯的**(`docs/re/132`):原版**什麼都不畫**,底色即海。全圖 12,463 格現在 100% 有來源,畫面上沒有佔位符了 |
| ~~世界地圖的可通行性~~ | **已解**(`docs/re/131`):八條規則,`WRLDMOVE.EXE` `0x10334`–`0x10572`。已實作並經畫面驗證(往東撞海洋停在 x=82)。剩下的是**值 10/12/20/21 的語意**與**呼叫端未讀** |
| **模組內 1,012 段字串的中文化** | 🔸 **10 段已譯**(酒館傳聞,`docs/re/138` §4 給了落點)。⚠ 其餘**不是 remake 的缺口**(`translations/README.md` §0):remake 有自己的介面字串,`tools/check_ui_language.py` 量到畫面上 0 條英文。那 1,012 段的價值在還原原版的**措辭**,要先把 1,085 個引用點歸戶到畫面(`docs/re/124`)|
| ~~商店清單的加註顯示~~ | **前提是錯的**(`spec/04` §5 已改寫):清單放**主視野**(61 欄)不是側欄,而列裡放的是道具名(最長 8 欄)不是地名。三個候選一個都不需要 |
| ~~DOSBox oracle~~ | **不是卡控**(`CLAUDE.md` §6)。腳本可用,實跑停在開機查表題;要用就由持有原版者依手冊作答,不用就走靜態。十二個子系統的收斂沒有一項靠它 |
| **模組內字串的中文化** | 1,012 段未動。822 段已接上引用點(`docs/re/124`,100%),還缺**把 1,085 個引用點歸戶到畫面**;障礙是 IDA 只判 ~40% 為程式碼,函式邊界在另外 60% 上不存在 |
| 重讀 `3E:44` | 該區指令邊界錯位(`docs/re/15` §2)。**不影響任何已定案的結論**,是可讀性問題 |
| 掃描擴到 `seg001`–`seg004` | 目前只掃 `seg000`;四支小模組大半內容沒掃到(`docs/re/06` §4)。四支都不在遊戲流程主線上 |

### 一句話現況

**RE 閘門已通過:`CLAUDE.md` §2.2 看板 A–L 十二個子系統全部 RE-DONE**
(`docs/re/122` §4 是索引)。規格已收攏並標 READY:
`docs/formats/01`–`07`(七份格式)、`docs/spec/01`–`02`(戰鬥、魔法)。

技術棧與畫布都已定案:**Go + Ebitengine、1024×768**(2026-08-14)。
`docs/spec/03-engine-plan.md`(架構與里程碑)與 `04-display-layout.md`
(版面、中文排版、避頭尾)標 READY —— **可以開始寫引擎程式碼**。
顯示層的核心決定是**美術 4× 整數放大、文字層走 TTF 原生解析度**:
中文塞不進放大的點陣字格。

**沒有任何外部條件卡著。** DOSBox 是輔助驗證不是卡控(`CLAUDE.md` §6):
十二個子系統的條件 3 全部由跨檔案 / 跨模組的獨立來源滿足。

`docs/spec/00-index.md`「標 READY 不等於零疑問」列的那批洞
(傷害公式的兩個係數、戰鬥屬性 14/18、法術效果類別 3、`CHARS.DAT` 位移 1、
先攻是否每回合重排)照該檔的指示辦:**實作時遇到就回 `docs/re/` 或回 IDA,不要猜。**

中文化已完成 **439 段 / 17,807 bytes(52%)**,涵蓋全部可變長資料檔;
剩下 1,012 段全在模組映像內,難點不是長度(remake 沒有等長限制)
而是還不知道每條字串的顯示位置。

⚠ **RE-DONE 不等於零疑問。** 看板的意思是四項條件都核對過,
不是「這個子系統不會再被推翻」—— J 就在 `97` 宣告、`102` 撤回、`103` 重新收斂。
§6 的表持續在長。

---

## 3. 文件索引

| 編號 | 主題 | 一句話 |
|---|---|---|
| [`00-inputs.md`](docs/re/00-inputs.md) | 輸入檔清冊 | 13 支執行檔的 SHA-256;**所有結論只對這些雜湊成立** |
| [`01-inventory.md`](docs/re/01-inventory.md) | 初始清冊 | IDA 只分析到 loader stub;UNK 段是明碼 8086 不是壓縮 |
| [`02-loader-stub.md`](docs/re/02-loader-stub.md) | loader stub | 依 `PATH=` 載 `BRUN30.EXE`、依 `LIB=` 載 `USERLIB.EXE` |
| [`03-bz-module-header.md`](docs/re/03-bz-module-header.md) | 模組標頭 | `+0x16` = 模組大小(paragraphs);重定位分兩類修補 |
| [`04-module-layout-entry.md`](docs/re/04-module-layout-entry.md) | 佈局與進入點 | `[模組區][stub]`;MZ CS:IP 指向 stub;`sub_14CB8` 是通用 MZ 載入器 |
| [`05-module-code-start.md`](docs/re/05-module-code-start.md) | 程式碼起點 | `bz` 標頭長 `0x30`;強制反組譯只到 0.7%,卡在 `INT` 內嵌參數 |
| [`06-runtime-int-abi.md`](docs/re/06-runtime-int-abi.md) | 執行期 ABI | `INT 3Eh/3Fh` 固定 3 bytes;`INT 3Dh` 會自我改寫成 far call;解鎖到 40.8% |
| [`07-dispatch-tables.md`](docs/re/07-dispatch-tables.md) | 派工表 | 三張表已取出;容量是推算未量到 |
| [`08-code-data-separation.md`](docs/re/08-code-data-separation.md) | 程式碼/資料分離 | 追蹤與線性掃描各是下界與上界;模組由執行期驅動 |
| [`09-bz-segment-map.md`](docs/re/09-bz-segment-map.md) | 節區表 | `bz` 標頭六個欄位是段邊界;資料段位置已知 |
| [`10-runtime-does-not-call-modules.md`](docs/re/10-runtime-does-not-call-modules.md) | 控制流方向 | 執行期不 far call 進模組;低涵蓋率另有原因 |
| [`11-trace-terminates-cleanly.md`](docs/re/11-trace-terminates-cleanly.md) | 追蹤終止分析 | 追蹤乾淨跑完;內嵌參數假設否證;下界確定 |
| [`12-api-3f61.md`](docs/re/12-api-3f61.md) | API 對照表 | `3F:61`(最高頻)= 描述子複製;表已開第一列 |
| [`13-api-3e42.md`](docs/re/13-api-3e42.md) | API 對照(續) | `3E:42` = 緩衝區附加;`BRUN30` 未分析區擋住後續 |
| [`14-brun-unlock-param-buffer.md`](docs/re/14-brun-unlock-param-buffer.md) | `BRUN30` 補齊 | 補到 44%;§3 結論已作廢 |
| [`15-chain-filename-and-misalignment.md`](docs/re/15-chain-filename-and-misalignment.md) | 模組轉交 + 錯位教訓 | `ds:0B06h` 的寫入端找到;強制反組譯會錯位 |
| [`16-rule-tables.md`](docs/re/16-rule-tables.md) | 規則資料表 | 怪物 36B×74、法術/道具 CSV;兩個欄位語意有 22/22 與 5/5 印證 |
| [`17-world-and-maze.md`](docs/re/17-world-and-maze.md) | 地圖與地城 | `WRLDMAP` 每格 2 bytes;`.SQZ` 逐列編碼、六檔皆 82 列 |
| [`18-text-inventory.md`](docs/re/18-text-inventory.md) | 中文化落點 | 兩層 ≈35,800 字元;資料檔 61% 不必等程式碼分析 |
| [`19-bsave-container.md`](docs/re/19-bsave-container.md) | BSAVE 容器 | 52/52;`STARTUP.BIN` 是 `0xB800` CGA 整頁;地圖 103×121 |
| [`20-cga-layout.md`](docs/re/20-cga-layout.md) | CGA 佈局 | 320×200 2bpp 交錯 |
| [`21-tile-format.md`](docs/re/21-tile-format.md) | 圖塊格式 | `GET`/`PUT` 陣列 17×17;磚牆/拱門/四角星各自成立 |
| [`22-pict-and-monst.md`](docs/re/22-pict-and-monst.md) | 大圖與怪物圖 | `PICT` 153×153;`MONST` 未解,格式不通用 |
| [`23-module-datafile-map.md`](docs/re/23-module-datafile-map.md) | 模組↔資料檔 | 哪支模組讀哪個檔;`CHARUTIL` 是角色資料專職 |
| [`24-tooling-conflict.md`](docs/re/24-tooling-conflict.md) | 工具衝突 | `trace` 會覆蓋 `unlock`;`BLOAD` 不留位址常數 |
| [`25-string-storage.md`](docs/re/25-string-storage.md) | 字串存放 | 描述子形式已確認;指標基底未解 |
| [`26-relocation-targets.md`](docs/re/26-relocation-targets.md) | 重定位目標 | 只有 7 項,11/11 對上節區欄位;否證 `25` 的解釋 |
| [`27-chars-dat.md`](docs/re/27-chars-dat.md) | 角色槽 | 25 槽×94B;技能旗標 20 個對上 `TOWN.EXE` |
| [`28-input-semantics.md`](docs/re/28-input-semantics.md) | 輸入語意 | 命令鍵慣例與五個畫面的鍵表 |
| [`29-de-eff-pairs.md`](docs/re/29-de-eff-pairs.md) | `DE*EFF` 成對檔 | 資料相同只差標頭;否證「母本/工作副本」 |
| [`30-userlib-family.md`](docs/re/30-userlib-family.md) | `USERLIB` 分族 | `bm` 族只有一支;分類要按結構不按檔名語感 |
| [`31-combat-fields.md`](docs/re/31-combat-fields.md) | 戰鬥欄位 | 等級序列當對照組;`w9` 魔法相關、`w8` 非等級 |
| [`32-dosbox-oracle.md`](docs/re/32-dosbox-oracle.md) | DOSBox oracle | 環境可用;開機關卡洩漏了屬性加值表的存在 |
| [`33-quiz-validates-columns.md`](docs/re/33-quiz-validates-columns.md) | 題庫驗證 | 題目主題對回資料欄位;欄 3 語意訂正 |
| [`34-races-and-char-fields.md`](docs/re/34-races-and-char-fields.md) | 種族與角色欄位 | 五個種族、五個屬性;`GROUPS.DAT` 欄位候選 |
| [`35-shared-impl-wrappers.md`](docs/re/35-shared-impl-wrappers.md) | 派工表結構 | 約 16% 是薄包裝;共用實作分兩類 |
| [`36-api-index.md`](docs/re/36-api-index.md) | API 索引 | 實際用到 125 個;前十佔 75% 的呼叫 |
| [`37-api-3e79-3e83.md`](docs/re/37-api-3e79-3e83.md) | 兩個高頻 API | `mov ds:0A08h, sp` 是 API 的共同開場白 |
| [`38-api-are-basic-primitives.md`](docs/re/38-api-are-basic-primitives.md) | API 的性質 | 高頻端是語言基本操作;**方向改往低頻** |
| [`39-ds0a3a-and-xref-blind-spot.md`](docs/re/39-ds0a3a-and-xref-blind-spot.md) | `ds:0A3Ah` + xref 盲點 | **xref 看不到 DS 相對存取**;`ds:0A3Ah` 是目前物件指標 |
| [`40-brun30-identity-and-ds-base.md`](docs/re/40-brun30-identity-and-ds-base.md) | 執行期的身分 | **MS BASIC Compiler Runtime 5.60** |
| [`41-ds-base-solved.md`](docs/re/41-ds-base-solved.md) | **DS 基底** | `ds:XXXX` = 線性 `0x1FE00 + XXXX`;判準是位移分布的上界 |
| [`42-module-ds-and-the-66dc-boundary.md`](docs/re/42-module-ds-and-the-66dc-boundary.md) | 模組的資料段 | 七支共用 `0x66C8`–`0x681A`;模組變數大半是 BSS,不在檔案裡 |
| [`43-common-block-and-array-indexing.md`](docs/re/43-common-block-and-array-indexing.md) | **陣列索引 + COMMON 區** | `ds:6822` 是 15×≥20 的 word 陣列;`ds:34F8` = 隊伍人數(上限 5);`ds:34E0` = 隊員名字 |
| [`44-int3d-is-used-after-all.md`](docs/re/44-int3d-is-used-after-all.md) | **`INT 3Dh` 推翻** | 模組有用,909 處;助憶碼過濾器造成的假結論 |
| [`45-int3d-family.md`](docs/re/45-int3d-family.md) | `INT 3Dh` 這一族 | 字串/暫存值操作;`sub_1A08B` 是自由串列配置器 |
| [`46-string-table-partial.md`](docs/re/46-string-table-partial.md) | 字串表(L) | 三描述子 + 文字;**等長替換現在可行**,改長度不行 |
| [`47-source-filenames-and-master-inc.md`](docs/re/47-source-filenames-and-master-inc.md) | **原始碼檔名** | 十一支的 `.BAS`/`.INC` 都在;`MASTER.INC` 解釋 COMMON 區;文字總量 15,738 B |
| [`48-monst-deinterleave.md`](docs/re/48-monst-deinterleave.md) | **`MONST*.BIN` 已解** | 八張 17×17 動畫格,以 8 word 交錯;22/22 檔驗證 |
| [`49-h-closure.md`](docs/re/49-h-closure.md) | **子系統 H = RE-DONE** | `MIO2.EXE` 是開發工具、給出讀取端;調色盤 `0x3D8=0x0E`;`.PIC` 是 `DRAW` 巨集 |
| [`50-sqz-maze-format.md`](docs/re/50-sqz-maze-format.md) | **`.SQZ` 已解** | 不是壓縮,是文字 + 跑長;81×51,六檔一致 |
| [`51-mazedata-and-world-entrances.md`](docs/re/51-mazedata-and-world-entrances.md) | **關卡表 + 世界地圖索引** | `MAZEDATA` 13×8;圖塊 24/25/27/28 = 入口(11 處零誤差);`DT*TEXT` 已確認 |
| [`52-world-map-reader-and-shared-grid.md`](docs/re/52-world-map-reader-and-shared-grid.md) | **F 的讀取端** | `(y×103+x)×2+0x6822` 逐字對上資料側;`ds:6822` 是「當前地圖」,世界 103／戰鬥 15 |
| [`53-world-tiles-towns-and-draw-renderer.md`](docs/re/53-world-tiles-towns-and-draw-renderer.md) | **地形 + 城鎮 + DRAW 渲染器** | `FASTWRLD` 9 張;`TOWNDATA` 13 城鎮(13/13、74/74 兩重驗證);`tools/draw_pic.py` |
| [`54-f-closure.md`](docs/re/54-f-closure.md) | **子系統 F = RE-DONE** | `WRLDITEM.PIC` 行 k = 圖塊 k+10(7/7 + 20/20 零不合)|
| [`55-sqz-decoder-from-code.md`](docs/re/55-sqz-decoder-from-code.md) | **`.SQZ` 解碼器** | 規則從程式碼讀出;`_` 與 `*` 是同一值;20 列殘差是我多加的約束 |
| [`56-maze-tile-classes-and-mazedata-columns.md`](docs/re/56-maze-tile-classes-and-mazedata-columns.md) | 迷宮格值 + `MAZEDATA` 欄位 | 5–10 阻擋;欄 7 = 文字記錄數−1(13/13);欄 4 未解 |
| [`57-g-closure.md`](docs/re/57-g-closure.md) | **子系統 G = RE-DONE** | `MAZEDATA` 在 `ds:365C`;欄 4 = 朝向 1北2東3南4西;`MAZEITEM` 行 k = 圖塊 k |
| [`58-key-dispatch-mechanism.md`](docs/re/58-key-dispatch-mechanism.md) | 按鍵派工機制(K) | 單字元字串常數 + 字串比對;`CAMP` 主選單 10/10 |
| [`59-de-eff-event-table.md`](docs/re/59-de-eff-event-table.md) | **`DE*EFF` 是地城事件表** | 106×5:列/欄/方向/目標/目的欄;目標 ≥100 是 `DT` 文字編號 |
| [`60-event-lookup-and-tile-19.md`](docs/re/60-event-lookup-and-tile-19.md) | **G 重新 RE-DONE** | 事件表在 `ds:0x88F0`;圖塊 19 = 隱形觸發格;負值 = 跨關卡樓梯(4/4)|
| [`61-i-closure-unused-tiles.md`](docs/re/61-i-closure-unused-tiles.md) | **子系統 I = RE-DONE** | 九張圖塊只有 `MAZEWALL` 被載入;I 掛錯東西,重新歸位後無剩餘 |
| [`62-l-localization-inventory.md`](docs/re/62-l-localization-inventory.md) | **子系統 L = RE-DONE** | 1,476 段 / 34,499 B(經 `63` 訂正);`TITLES.DAT` 是外置 UI 字串表 |
| [`63-userlib-strings-and-l-correction.md`](docs/re/63-userlib-strings-and-l-correction.md) | `USERLIB` 字串 + L 訂正 | 11/11 模組載入;單描述子格式;60 段存檔/時鐘/提示文字 |
| [`64-userlib-call-mechanism.md`](docs/re/64-userlib-call-mechanism.md) | **`USERLIB` 的呼叫機制** | `3D:00` 延遲繫結;匯出表在 `ds:0x0A1E`;19 個槽全部 ≡3 mod 8 |
| [`65-userlib-export-table.md`](docs/re/65-userlib-export-table.md) | **完整匯出表** | 靠 MZ 重定位表找到(seg 1822、間距 8);63 槽;槽 35 呼叫槽 15 畫訊息框 |
| [`66-userlib-slot-semantics.md`](docs/re/66-userlib-slot-semantics.md) | **子系統 C = RE-DONE** | 槽 34 = 存檔、33 = 狀態列、17 = 死亡、21 = 結局、35 = 訊息框、15 = 視窗框 |
| [`67-a-closure-module-handoff.md`](docs/re/67-a-closure-module-handoff.md) | **子系統 A = RE-DONE** | 轉交是 `retf` 進 `節區:0x30`(`bm` 為 `0x3A`);`ds:0A28h` 指向檔名緩衝區 |
| [`68-b-closure-dispatch-capacity.md`](docs/re/68-b-closure-dispatch-capacity.md) | **子系統 B = RE-DONE** | 表容量 105/165/≤204;`3E:A5` 必然等於 `3F:00` |
| [`69-movement-keys.md`](docs/re/69-movement-keys.md) | 移動鍵(K) | `C P S Q 1 2 3 4` 三角驗證;§2 已被 `70` 推翻 |
| [`70-key-chains-all-modules.md`](docs/re/70-key-chains-all-modules.md) | 小鍵盤轉譯層 + 七支模組比對鏈 | 描述子 = 文字 −4;`1北 2東` 從程式碼確認 |
| [`71-k-closure.md`](docs/re/71-k-closure.md) | **子系統 K = RE-DONE** | `3Dh:00` 位移 ÷8 得槽號,五筆全整除;`S`=存檔確認 |
| [`72-e-file-formats-from-readers.md`](docs/re/72-e-file-formats-from-readers.md) | 三個規則資料表的格式 | 從讀取端的元素大小與迴圈上界解出;三個檔大小零誤差 |
| [`73-monster-columns-to-combat-array.md`](docs/re/73-monster-columns-to-combat-array.md) | 怪物欄 → 戰鬥單位屬性 | 位移全是 15 的倍數,證實 15 欄二維陣列 |
| [`74-spell-and-item-columns.md`](docs/re/74-spell-and-item-columns.md) | 法術/物品欄位語意 | 物品的法術編號與點數搬進施法流程同一組變數;28+6 是兩個 100% 規則 |
| [`75-character-record-equipment.md`](docs/re/75-character-record-equipment.md) | 角色記錄的裝備欄(D×E×J)| 裝備欄存背包格號不是物品編號;哨兵 99/60/59 |
| [`76-to-hit-formula.md`](docs/re/76-to-hit-formula.md) | 命中公式 + 物品欄的雙重身分 | 「這一欄是什麼」問錯了,要問「誰在讀它」|
| [`77-runtime-arithmetic-api.md`](docs/re/77-runtime-arithmetic-api.md) | 執行期算術 API | MBF 浮點;加減乘除靠指數處理分辨 |
| [`78-damage-sequence.md`](docs/re/78-damage-sequence.md) | 傷害運算序列 | 形狀確定、配對壓在未解的對齊問題上(保留已由 `79` 撤銷)|
| [`79-alignment-resolved-damage-formula.md`](docs/re/79-alignment-resolved-damage-formula.md) | **傷害公式** | 三個假設全被否證,但否證過程證明配對成立 |
| [`80-save-write-end.md`](docs/re/80-save-write-end.md) | 存檔寫入端(D)| 記錄長度 94/90 從 `OPEN` 直接讀出;`GROUPS.DAT` 15 個欄位位移 |
| [`81-chars-record-to-combat-attributes.md`](docs/re/81-chars-record-to-combat-attributes.md) | 角色欄位 → 戰鬥屬性 | 機械配對 + 兩個已讀過的錨點驗證 |
| [`82-monster-columns-semantics.md`](docs/re/82-monster-columns-semantics.md) | 怪物十欄語意 | ⚠ 欄4/欄8 已被 `83` 推翻 |
| [`83-hp-is-attribute-3.md`](docs/re/83-hp-is-attribute-3.md) | 生命值是屬性 3 | 傷害減在哪個欄位上那個欄位就是生命值;欄8 = 經驗值 |
| [`84-pursuit-not-initiative.md`](docs/re/84-pursuit-not-initiative.md) | 追擊移動(不是先攻)| ⚠ §2 已被 `85` 推翻 |
| [`85-scanner-conflated-zero-with-unknown.md`](docs/re/85-scanner-conflated-zero-with-unknown.md) | 掃描器失敗值撞號 | 31% 的失敗被讀成「屬性 0」,而那正是我去看它的原因 |
| [`86-scanner-fixed-conclusion-restored.md`](docs/re/86-scanner-fixed-conclusion-restored.md) | 修好工具後重判 | 結論沒變、證據全換;結論正確不等於推理正確 |
| [`87-class-race-and-attribute-17.md`](docs/re/87-class-race-and-attribute-17.md) | 職業/種族碼、屬性 17 | 位移 15 = 職業(程式碼比 `'1'`)、14 = 種族;屬性 17 是條件式減傷 |
| [`88-races-and-classes.md`](docs/re/88-races-and-classes.md) | 種族/職業碼表 | `H`uman `T`roll `D`warf `E`lf `G`nome;職業是 **`Hero`** 不是「戰士」|
| [`89-record-offset-census.md`](docs/re/89-record-offset-census.md) | 記錄位移清點 | 39 個位移、8 個有語意;共用變數會混兩個檔 |
| [`90-monster-column-9-is-a-tier.md`](docs/re/90-monster-column-9-is-a-tier.md) | 怪物欄9 = 難度階級 | 三組碰撞否證「等級」;判別點是單射性不是大小 |
| [`91-e-closure.md`](docs/re/91-e-closure.md) | **子系統 E = RE-DONE** | 物品欄6 雙重身分;法術負值只出現在四個類別 |
| [`92-attribute-15-16.md`](docs/re/92-attribute-15-16.md) | 屬性 15 = 目標編號、16 = Hero 旗標 | 一個屬性存的是另一個單位的編號,那就是目標 |
| [`93-initiative.md`](docs/re/93-initiative.md) | **先攻** | 兩層迴圈依屬性 2(速度)排 `ds:6A7Ah` 順序表;補上 `73` 欄1 的缺口 |
| [`94-skill-tables.md`](docs/re/94-skill-tables.md) | 兩張技能表 | 位移 42–51 由職業決定讀哪張;`Hard Axe` 會 Axe、`Fire Hawk` 會 Fire runes |
| [`95-no-flee-command.md`](docs/re/95-no-flee-command.md) | 逃跑找不到 | 三條線指向不存在;但「回空」的證據等級不足,標假設 |
| [`96-facing-and-action-points.md`](docs/re/96-facing-and-action-points.md) | 朝向與行動點數 | 屬性 10 = 朝向,命中 `+12` = 背後攻擊;轉身 1 點、行動 3 點 |
| [`97-j-closure.md`](docs/re/97-j-closure.md) | **子系統 J = RE-DONE** | `D` = Dispell;「沒有逃跑」以窮舉 + 正對照收斂 |
| [`98-status-level-and-a-sign-error.md`](docs/re/98-status-level-and-a-sign-error.md) | 狀態/等級位移 + 命中的反號 | 位移 38 = 狀態、40 = 等級;`76` 的 `≤` 應為 `>` |
| [`99-parity-separates-the-two-records.md`](docs/re/99-parity-separates-the-two-records.md) | 奇偶性分開兩個記錄 | `CHARS` 全偶、`GROUPS` 全奇,25 筆零例外 |
| [`100-chars-attributes-closed.md`](docs/re/100-chars-attributes-closed.md) | `CHARS.DAT` 欄位表 | 位移 12/20/22 解出;整數欄 14/15 有語意 |
| [`101-groups-record-status.md`](docs/re/101-groups-record-status.md) | `GROUPS.DAT` 現況 | 「六個連續 word = 隊伍成員」被否證;剩 10 欄只能從程式碼追 |
| [`102-flee-exists-retracting-j.md`](docs/re/102-flee-exists-retracting-j.md) | **撤回 J = RE-DONE** | `'PARTY RAN!'`;正規表示式有 `run` 沒有 `ran` |
| [`103-flee-is-leaving-the-field.md`](docs/re/103-flee-is-leaving-the-field.md) | **J = RE-DONE(重新收斂)** | 逃跑 = 隊伍成員朝向全為 0;範圍 9…人數+8 對上 `43` |
| [`104-groups-fields-from-context.md`](docs/re/104-groups-fields-from-context.md) | `GROUPS.DAT` 再解四欄 | 用語境字串當標籤;9/14 有語意 |
| [`105-visibility-fields.md`](docs/re/105-visibility-fields.md) | 位移 59/83 = 能見度與來源旗標 | 槽 33 只讀四個變數,消去法;11/14 |
| [`106-clock-cascade.md`](docs/re/106-clock-cascade.md) | 位移 27/29/33 = 時鐘三級 | ⚠ 已被 `107` 訂正為四級 |
| [`107-clock-is-four-levels.md`](docs/re/107-clock-is-four-levels.md) | 時鐘是四級 + 訂正 `104` | 語境字串給的是「在哪裡被動到」,不是「它是什麼」|
| [`108-clock-labels-not-resolved.md`](docs/re/108-clock-labels-not-resolved.md) | 時鐘標籤對應未解 | 顯示端不直接讀計時器;掃描回空的第三個成因是定址形式 |
| [`109-clock-labels-resolved.md`](docs/re/109-clock-labels-resolved.md) | **時鐘標籤定案** | `ds:3010h` 存的是記錄位移;掃不到的第四種成因是「不在你掃的抽象層」|
| [`110-display-offset-census.md`](docs/re/110-display-offset-census.md) | 顯示欄位清點 + 訂正 `109` | 兩個事實之間的**距離**也是證據的一部分 |
| [`111-status-code-equals-school.md`](docs/re/111-status-code-equals-school.md) | **狀態編號 = 法術系別**;`CHARS.DAT` 完成 | `CHAINS`→`Bound`、`STILL AIR`→`Still Air`、`FREEZE`→`Frozen` 三筆全中 |
| [`112-light-counter-verified.md`](docs/re/112-light-counter-verified.md) | 位移 45 重驗通過 | 語境證據要能「夾住」目標,不能只是「靠近」|
| [`113-d-status-and-remaining.md`](docs/re/113-d-status-and-remaining.md) | D 的收束 | 剩四項都不以「標籤+值+標籤」顯示,字串法結構上無效 |
| [`114-maze-coordinates.md`](docs/re/114-maze-coordinates.md) | **位移 79/81 = 迷宮座標** | `81 × 欄 + 列` 的陣列算術;`104` 四筆錯三筆 |
| [`115-visibility-lit-and-dark.md`](docs/re/115-visibility-lit-and-dark.md) | 位移 59/61 = 能見度 | ⚠ 對應方向被 `116` 存疑 |
| [`116-contradiction-in-115.md`](docs/re/116-contradiction-in-115.md) | 位移 25 的形狀 + 質疑 `115` | 註解裡的因果詞要能指到一條指令 |
| [`117-annotation-was-right.md`](docs/re/117-annotation-was-right.md) | 註解是對的,但寫的時候不該是對的 | 排版會製造出資料裡沒有的對立;猜對與猜錯的註解長得一樣 |
| [`118-daylight.md`](docs/re/118-daylight.md) | **天色 + 位移 83 結案** | `ds:35B4h` 由時辰決定(17→2、19→1);位移 83 選自然光或攜帶光 |
| [`119-attribute-12-never-decremented.md`](docs/re/119-attribute-12-never-decremented.md) | 屬性 12 從不遞減 | 「X 需要有 Y」式的論證要去找 Y 存不存在 |
| [`120-offset-84-cleared-in-camp.md`](docs/re/120-offset-84-cleared-in-camp.md) | 位移 84 被清成 0 | 對所有候選都成立的觀察不縮小候選集合 |
| [`121-offset-84-is-strength.md`](docs/re/121-offset-84-is-strength.md) | **位移 84 = 狀態效果強度**;`CHARS.DAT` 完成 | 要分開候選,去找「只有其中一個會做」的動作 |
| [`122-d-closure.md`](docs/re/122-d-closure.md) | **子系統 D = RE-DONE,看板 12/12** | 位移 25 歸零時掃 3×3 鄰域 = 遭遇檢查;兩支移動模組同構 |
| [`123-menu-data-place-lists.md`](docs/re/123-menu-data-place-lists.md) | `MENU.EXE` 的三串 `DATA` | 兩張地城名單各以 `Ralith` 結尾 = 兩條路線;原版自己有 `Black Fort`/`Blackfort` 兩種拼法 |
| [`124-string-refs-linked.md`](docs/re/124-string-refs-linked.md) | **822 段字串全部接上引用點** | 那個掛了很久的「位移↔DS 對應」不需要解;δ 平移負對照證明 100% 不是巧合 |
| [`125-manual-confirms-spells.md`](docs/re/125-manual-confirms-spells.md) | **官方中文手冊當 oracle** | 法術法力 33/33、命中 ×4 與背擊 +12 兩源吻合;⚠ 手冊內文譯自 Apple II 版 |
| [`126-shop-price-multiplier.md`](docs/re/126-shop-price-multiplier.md) | **新發現:商店價格倍率** | `TOWNDATA.DAT` 位移 38 = MBF 倍率;`ITEMS.DAT` 欄3 是基準價;「差一點點」的偏差就是主要證據 |
| [`127-pict-contents.md`](docs/re/127-pict-contents.md) | 五張大圖的內容 | 不是遭遇畫面(藏寶圖/NPC/風景/死亡/兜帽人);`BORDER`+`EXITSPOT` 兩張圖塊由手冊確認語意 |
| [`128-wrlditem-tile-bias.md`](docs/re/128-wrlditem-tile-bias.md) | ⛔ **結論已作廢** | 偏移不是 +14;保留原文是為了留下錯誤的形狀 |
| [`129-world-tile-dispatch.md`](docs/re/129-world-tile-dispatch.md) | 世界圖塊派工鏈 | 1–9 與 11 走點陣、其餘走向量;**5/6/11 同組是水**(推翻 `53` 的猜測);遭遇集合 12/13/20–32 有程式碼證據 |
| [`130-retro-why-the-tile-map-was-missed.md`](docs/re/130-retro-why-the-tile-map-was-missed.md) | **檢討:為什麼繞了兩圈** | 先懷疑解析器再調常數;引用筆記要掃「尚未解開」;帶著假設讀 dump 會讓證據隱形 |
| [`131-world-passability.md`](docs/re/131-world-passability.md) | **可通行性八條規則** | X 限 5–98;目標 10–12 阻擋;15–18 是海岸四轉角,各擋兩個相鄰方向。為此寫了 `tools/bool_chain.py` |
| [`132-world-tile-dispatch-corrected.md`](docs/re/132-world-tile-dispatch-corrected.md) | **圖塊來源 100% 解完** | 值 11 **不畫**(底色即海);其餘走 `0xC980+4×值`;載入迴圈 `FOR I=10 TO 38` 是 `WrldItemBias=10` 的程式碼側證據 |
| [`133-chars-offset-1-is-party.md`](docs/re/133-chars-offset-1-is-party.md) | `CHARS.DAT` 位移 1 = **隊伍編號** | 答案在 `CHARUTIL` 的名冊標題(有 `Party` 欄)+ 手冊的 PARTY #5 |
| [`134-groups-load-routine-and-gold.md`](docs/re/134-groups-load-routine-and-gold.md) | **金幣在位移 19(4B 單精度)** | `MENU` 載入常式的 16 欄對照表;傳位址 vs 傳值本身就是型別證據 |
| [`135-groups-record-5-is-a-real-save.md`](docs/re/135-groups-record-5-is-a-real-save.md) | **出貨檔第 5 筆是真存檔** | 位移 1–18 = 九個成員槽;它一次印證了世界座標、朝向 3=南、時鐘下界 |
| [`136-damage-coefficients-still-unresolved.md`](docs/re/136-damage-coefficients-still-unresolved.md) | 傷害係數:`k₂` 解了、`k₁` 沒有 | `k₂` = 擲骰的 `+1`(17/17 形狀);⛔ 三條資料側 delta 路線已否證,不要再試 |
| [`137-maze-coordinate-order.md`](docs/re/137-maze-coordinate-order.md) | **迷宮座標三個來源順序一致** | 衝突的是名字不是算式(`56`/`59` 叫「列」、`114` 叫「欄」);程式碼改用 Major/Minor |
| [`138-towndata-building-types.md`](docs/re/138-towndata-building-types.md) | **`TOWNDATA` 位移 34/36** | 負值 = 治療/酒館/旅店/訓練四類;正值 = **販售的物品編號範圍**(店名逐間對得上);酒館的位移 36 是傳聞編號 |

規格:[`docs/spec/00-index.md`](docs/spec/00-index.md) —— 七份格式 + 四份實作規格,全部 READY。

工具:`tools/ida.sh`(headless 包裝)、`tools/ida/*.py`(匯出腳本)、
`tools/link_strings.py`(字串 ↔ 引用點)、`tools/bool_chain.py`(BASIC 布林鏈還原,
**不要用眼睛讀 `j<cc>`**)。
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

### ⚠ 掃字串時,間隔不規則的地方要回去看原始位元組

用「長度 ≥ N」過濾掃字串,會**安靜地吃掉短字串**,而被吃掉的那個
不留任何痕跡 —— 只有**相鄰間隔異常**會露餡(`Elf` 3 字元被漏掉,
症狀是間隔從 `0x12` 變成 `0x22`)。`docs/re/34` §1

### ⚠ 由執行期決定的東西不會出現在模組程式碼裡

兩次踩到同一個形狀:`BLOAD` 的載入位址寫在檔案標頭(`docs/re/24`)、
陣列索引的 stride 乘法在執行期裡(`docs/re/23` §5,掃 `0x24` 零個 `mul`)。
**所以 §2.1 條件 2 不能靠掃模組程式碼達成** —— 要先解派工表語意,
或走 DOSBox 動態觀察。這是技術棧的結構決定的,不是努力不夠。

### ⚠ 掃描前先看「掃了幾條指令」

`trace_module.py` 會 `del_items` 整段,洗掉 `unlock_module.py` 的成果。
之後任何掃描都只看得到 12% 的程式碼,**而輸出看起來完全正常**。
**工具回報的分母比分子重要** —— 命中 0 處可能是真的沒有,
也可能是掃描範圍被前一個工具改小了。(`docs/re/24`)

### 掃二進位樣式不要用正規表示式

`bytes([0xCD, 0x3F])` 是 `b'\xcd?'`,`?` 是量詞 —— 每個位置都命中。
用 `bytes.find` 或 `re.escape`。**抓到它靠的是量級檢查:命中數不該接近檔案大小。**
⚠ 同一次掃描裡只有一部分結果會壞(`0x3D`/`0x3E` 不是 metacharacter),更難察覺。

### 橫跨不同輸入卻不動的數字 = 查法壞了

`func=10` 出現在十二支大小差十倍的 EXE 上,是這一輪最重要的線索。
**不要把它讀成結果。**

---

## 6. 已被推翻的斷言

| 曾經寫過 | 真相 |
|---|---|
| 十一支的 loader stub「逐位元組相同」 | **只有結構相同**(函式大小序列與段大小一致);位元組不同,差異來自重定位與嵌入的模組名。`docs/re/01` §2 |
| `INT 3Dh/3Eh/3Fh` 只是「呼叫執行期」,語意未知 | 三支處理常式都已讀完(`docs/re/06`):`3Eh`/`3Fh` 固定吃 1 個參數位元組當派工索引;`3Dh` 有三種情況,其中 `CD 3D 00 oo oo` 會**把自己改寫成 `9A` far call**。⚠ 因此執行中的記憶體映像與磁碟位元組不同 |
| `GET` 陣列 17×17 的讀法適用於所有圖形素材 | **只適用於九個 98-byte 圖塊與五個 `PICT`**。`MONST*` 套不上(標頭讀法給 174 bytes,實際 734),而且它開頭是連續七個 `34` —— 「前兩個是寬高」與「整段都是資料」在形狀上分不出來。**一個格式在 N 個檔上成立,只證明它適用於那 N 個檔;驗不過時先懷疑不是同一種格式,不要調參數去湊。** `docs/re/22` §3 |
| 小圖的尺寸要靠因數分解 + 九檔互證來挑(`docs/re/20` §2) | **尺寸就寫在資料裡** —— 扣掉 BSAVE 標頭後,前 2 個 word 是 BASIC `GET` 陣列的寬(bit)與高。**看到「扣掉容器標頭後前幾個 byte 是小整數」時,先假設那是內容的標頭,不要直接拿剩餘長度做因數分解。** `docs/re/21` §3 |
| `WRLDMAP.BIN` 有 12,467 格,寬度候選 91／137 | **12,463 格,103 × 121**。先前沒扣 BSAVE 的 7-byte 標頭與 1-byte EOF;而 **7 是奇數,還讓 2-byte 交錯的奇偶性翻轉**(圖塊在 `data[0::2]` 不是 `d[1::2]`)。**判準:切分單位對了還不夠,起點錯一個 byte 會讓交錯資料整個換半邊。** `docs/re/19` §2 |
| `WRLDMAP.BIN` 是 byte 陣列,寬度在九個矩形分解裡 | **每格 2 bytes**(低位元組 99.96% 為 0),真正的格數是 12,467 不是 24,934。**ASCII 圖出現「每隔一格重複」的規律時,先懷疑切分單位不是 1 byte。** 而且 `91`/`137` 兩個候選已被肉眼否決(逐列右移)。`docs/re/17` §1 |
| 「這個技術棧大量使用參數寫在呼叫點後面」(`docs/re/35` §4 的推論)| **在共用實作這一層不成立** —— 122 個呼叫端裡只有 12 個後面是資料,其餘四支都是 trampoline。該句在 `INT` thunk 那一層仍成立(7,673 個都帶參數)。**判準:說「大量使用某慣例」之前,先數兩邊的比例。** `docs/re/35` §4.6 |
| `pop <變數>` 開場 = 讀內嵌參數(`docs/re/35` 第一版) | **只對 `sub_199CC` 成立**。`sub_11F6B`(87 個呼叫端)是 **trampoline** —— `pop` 返回位址後 `call` 它,把後面的位元組當**程式碼**執行。**`pop <變數>` 只說明拿到了返回位址,沒說明拿去做什麼** —— 要看後面是 `lods`(資料)還是 `call`(程式碼)。`docs/re/35` §4 |
| 每個派工索引是一支獨立常式(`docs/re/12`–`14` 的前提) | **約 16% 是薄包裝**(98/630) —— 它們 `call` 五支共用實作之一,真正內容接在 `call` 後面。「先找共用實作」是有效策略但只涵蓋六分之一。⚠ 該文標題原寫「多數」,**用詞過強已訂正**。`docs/re/35` §4.7 |
| 角色記錄 `+0x0B` 起那七個 9–11 的值是屬性(`docs/re/27`) | **數目對不上** —— 遊戲只有五個屬性(Speed/Strength/Intellect/Endurance/Skill)。維持未解。**判準:形狀像某個東西時,先去數那個東西有幾個。** `docs/re/34` §2 |
| `ITEMS.DAT` 欄 3 是「傷害」(`docs/re/16`) | **是型別相依的主數值** —— 武器上是傷害、護甲上是防護。開機題庫同時問了小斧的傷害與鎖甲的防護,兩者都落在欄 3。**判準:一個欄位在某一類記錄上成立,不代表整張表都是同一個語意。** `docs/re/33` §2.2 |
| `MONSTERS.DAT` 的 `w8`(值域 1–13)「像等級」(`docs/re/16`) | **是序列內的序號**。`Lvl N Fighter` 的等級跳號(1,2,3,4,6,8,10,12,15)而 `w8` 是平順的 1…9。**判準:值域對不等於語意對** —— 等級序列提供直接對照,看一眼跳不跳號就能判。`docs/re/31` §3 |
| `USERLIB`/`MIO2`/`WSIO`/`MTEST` 是同一類「原生輔助程式庫」 | **只有 `USERLIB` 是另一族**(`bm` 簽章在載入映像 +0、114 個重定位項);另外三支是標準 `bz` 模組、各 7 項。原本的分組依據是**檔名看起來像工具**而不是結構。**判準:分類要按結構,不要按檔名的語感。** `docs/re/30` §3 |
| `.MST` 與 `.BIN` 成對出現,所以是母本 vs 工作副本 | **否證**。九對逐位元組比對,差異只在 BSAVE 標頭的 segment/offset(位移 1–4),**資料本體完全相同**。**判準:兩個檔「成對出現且其中一個叫 master」不足以推出母本關係,先比對資料本體** —— 成本是九次 diff。`docs/re/29` §2 |
| 字串指標解不出是因為未重定位(`docs/re/25` §2) | **否證**。MZ 重定位表只有 7 項,全部落在 `bz` 標頭的節區欄位,沒有一項指向字串描述子。**判準:引用一個已確認的機制來解釋新現象前,先確認那個機制真的涵蓋這個現象** —— 少的那一步只要讀 7 筆資料。`docs/re/26` §2 |
| `ds:74h` 是參數累積指標、`0xC26`–`0xC30` 是 10-byte 緩衝區(`docs/re/14` §3) | **不成立**。依據的那幾行是**強制反組譯錯位**的產物(徵狀:跳躍目標 `loc_14346+1` 落在指令中間);xref 證實 `0xC26`–`0xC32` 是程式碼。**判準:讀強制反組譯的產物前,先 grep 有沒有 `loc_XXXX+N` 這種跳進指令中間的目標。** `docs/re/15` §2–§3 |
| `3E:42` 是「把 word 附加到緩衝區」的輸出動作(`docs/re/13`) | **動作對、用途錯**。它是**參數傳遞** —— `3E:44` 顯示緩衝區滿了會被回頭取出當參數。判準:一支常式頻繁被呼叫且只做一個小動作時,先問「它是不是別人的前置步驟」。`docs/re/14` §3 |
| 追蹤只到 5.4% 是因為 thunk 後接內嵌參數(`docs/re/10` §3) | **否證**。全模組只有 2 個位址解不開,且都不在 thunk 之後(距離為負)。追蹤是乾淨終止的:132 `jmp` + 3 `retn` + 2 解不開。`docs/re/11` |
| 流程追蹤到不了,是因為模組被執行期呼叫(`docs/re/08` §2) | **否證**。`BRUN30` 13 處 far 轉移目標全在自己內部、零間接。錯在用「BASIC 執行期通常這樣」解釋觀察,而不是去查機制 —— 查證成本只是一支掃描腳本。`docs/re/10` |
| `+0x0A` 的值與 `+0x16` 相同(十一支皆然) | **`MENU.EXE` 不成立**(`0x3D3` vs `0x462`)。`+0x0A` 是**最後一段的起點**;十支剛好只有六段才看起來相同。**十一分之十的巧合被讀成了規則** —— `MENU` 有 7 段這件事在 `docs/re/01` 的清冊裡一直寫著。`docs/re/09` §3 |
| `INT 3Eh` 索引 `0xA5` 是線性掃描造出的假指令(`docs/re/07` §3 第一版) | **不成立**。保守得多的流程追蹤同樣產生它。實際成立的只有「兩張表相鄰」這個觀察,而**相鄰只約束位置不約束長度**。判準:一個推論若只在某一種測量方法下成立,先換一種方法再下結論。`docs/re/08` §3 |
| `docs/re/06` §4 的「`3D` far 形式」欄位(合計 346) | **一個都不成立**。那是線性掃描的中間計數,最終資料庫裡解碼成 `int 3Dh` 的是 **0** 個 —— 遊戲模組不用 `INT 3Dh`。**中間產物不是結論**。`docs/re/07` §2 |
| 「移動鍵找不到程式碼側連結」(`docs/re/69` §2)| **推翻**:連結就在 `0x105E3`;`imm_range.py` 只掃 IDA 判成程式碼的位址,而模組區只有 40% 被判成程式碼(`70` §6)| **以 IDA 判定為前提的工具回 0 命中,不能當成「不存在」** —— 先問這支工具的分母是什麼 |
| 「字串描述子位址 = 文字位址 +2」(`docs/re/58` §2)| 實為 **−4**(`70` §2);`58` §3 的 10/10 不受影響,整條鏈一起位移 | **等距排列的資料上,「差一格」的規則驗不出來** —— 要找間距改變的邊界(`'CAMP'` 是 4 字元)|
| `bz` 標頭有 `+0x32` / `+0x34` / `+0x54` 三個欄位 | 那三個位移屬於 **`BRUN30` 的控制區塊**(`ds:0CACh`),不是模組標頭。錯在沒注意 `sub_14BDD` 中途換過 `es` 的基底。`docs/re/03` §1 |
| `ds:0A28h` 存的 `0x0B06` 是「模組本體的進入點位移」 | 是**檔名字串的指標**。`sub_14CB8` 拿它去 `INT 21h AH=3Dh` 開檔。**常數是位址還是指標,要看使用端怎麼用,不能從值的樣子猜**。`docs/re/04` §4 |
| kb 寫的「IDAPython 實測無輸出,一律寫 IDC」 | 可用,但要修正過的 image(`ida-pro-9.4-idapython:py312-v1`)。兩個獨立根因見 `~/.claude/knowledge-base/retro/ida-pro-9.4.md` |

> 這張表只增不減。**推翻一條斷言時,要同時把正文改寫成正確答案**,
> 不是在正文加註解(`rulebook/63`)。

---

### 新增(本輪)

| 原斷言 | 實際 | 錯誤形狀 |
|---|---|---|
| 「`ds:0A3Ah` 只有 3 支 API 在動」 | 62 處存取 | **用錯的工具查 → 空結果讀成「沒有」**(`docs/re/39` §1) |
| 「`0x0BE8` 是鉤子常式的位址」 | 是哨兵值,與 `0` 並列在跳過條件裡 | 看到數值像位址就去 dump,沒先確認它被當位址用過 |
| 「xref 為空是因為位址算錯」 | 位址是對的,IDA 根本不為 `o_mem` 建資料參考 | 結論對、理由錯 —— 沒實測就寫成因果 |
| 「`DS = CS`(`docs/re/40` §2)」 | 只對「單獨執行 BRUN30」的錯誤路徑成立 | **把單一情境的證據當成全域結論** |
| 「三個常數落在字串中段 → 不是結構指標」 | 用錯基底算的,反證作廢 | 拿未定案的基底去做反證 |
| 「`0x66DC` 是 linker 保留的記號值」(`docs/re/03` §3)| 語意仍未解 | 「大於檔案大小」推不出「不是位址」|
| 「遊戲模組不使用 `INT 3Dh`」(`docs/re/07` §2,標**已確認**)| **推翻**:909 處、24 個索引 | **過濾器用助憶碼,而 IDA 把 `CD 3D` 叫 `wait`** —— 同一節裡 227 與「0 個起點」自相矛盾卻沒察覺 |
| 「`06` §4 的 346 是掃描雜訊」(`docs/re/07` §2 的撤回)| **撤錯了**,346 = `3D:00`,兩種方法同值 | **撤回一個結論也需要證據**,只有推論不夠 |
| 「全遊戲文字 1,416 段 / 33,795 B」(`docs/re/62`)| 漏了 `USERLIB.EXE`,實際 1,476 段 / 34,499 B | **盤點範圍用「遊戲模組」這個角色分類定義,而不是「會不會被載入」** |
| 「`MAZEITEM.PIC` 不是逐圖塊值索引」(`docs/re/55` §6)| 第 k 行**就是**圖塊值 k | **檢定條件漏了「用到但不繪製」** —— 檢定沒過先查條件 |
| 「`ds:3518h`/`ds:351Ah` 是累加量」(`docs/re/43` §4)| 隊伍的世界座標 | 只看寫入的形狀,沒看讀取端在跟什麼比 |
| 「`ds:6822` 是 15 列 × 20 欄的主狀態陣列」(`docs/re/43` §1)| 15 是**寬度**;它是「當前地圖」陣列,`WRLDMOVE` 用 103 欄 | **算術對、標籤反了** —— 只看一支模組就替陣列命名 |
| 「`.SQZ` 的 20 列短 1 格是未解殘差」(`docs/re/50` §4)| 解碼器沒有欄數檢查,50 格合法 | **把「所有列等長」當成必然 —— 從自己的期待推規格** |
| 「`.SQZ` 是壓縮格式,壓縮法未知」(`docs/re/01`)| 是**純文字** + 跑長編碼 | **拿副檔名當證據**,清單階段沒看內容 |
| 「`MONST*.BIN` 的資料是連續的」(`docs/re/22` §2 的所有嘗試)| 是**交錯**的(8 word 一組) | **算術對不上時只想到「寬度猜錯」,沒想到「資料不連續」** |
| 「模組程式碼掃不到陣列索引」(`docs/re/23` §5)| **推翻**:掃錯指紋。word 陣列用 `add`+`shl` 不用 `mul`;改掃 `[reg+disp]` 後 CMBT 有 558 處 | **拿一個不會出現的樣式去掃,把 0 命中讀成「不存在」** |
| 「`0x66DC` 是資料段的區域結束位址」(`docs/re/42` §2 第一版)| **否證**:`MENU` 門檻 `0x66E8` 卻有完全相同的存取範圍;而且門檻是段落、位移是位元組 | **兩個數字接近就當成有關係,沒先確認單位** —— 靠「找一個該值不同的樣本」抓到 |
| 「`TOWN`/`MENU`/`MAZEMOVE` 有四個提示字母沒有對應的按鍵處理」(`docs/re/70` §5)| **三個是抽取器誤判** —— `([A-Za-z0-9])\)` 會吃掉 `(Y/N)?`、`(320K)`、`(B,G,V,R)` 的右括號前一字;第四個是收滿四字元才比一次的寶石謎題 | **抽取器的假陽性和假陰性一樣要當缺口處理** —— 「異常清單」要逐項回看原文,不要直接當待解項 |
| 「每個解除法術與它所解的法術同系別」(`docs/re/74` 起草時)| **否證**:`MELT`(系別 1 火)解的是 `FREEZE`(系別 4 冰) | **三筆裡兩筆符合就想立規則** —— 分群要看完整欄位,不是挑順眼的三筆 |
| 「`ITEMS.DAT` 欄4/欄5 有單一語意,那六筆附魔裝備是例外」(`docs/re/74` §3)| **分類不在資料裡,在呼叫端** —— 同一欄被兩段程式碼用兩種方式解讀(欄4 = 傷害/護甲 或 法術編號;欄5 = 命中加值 或 投入點數)| **「這一欄是什麼」問錯了** —— 要問「誰在讀它」;資料裡找不到分類旗標時,先看讀取端有幾個 |
| 「落單位元組的比例和普通指令的自然頻率相符,所以是普通指令」(`docs/re/79` 測量一)| **假象** —— 兩個族群疊在一起。拆成「後面接不接 `CD`」再看,`0x80` 是 10/10 全接,巧合率 1.6×10⁻⁷ | **一個分佈「看起來正常」可能是兩個族群疊出來的** —— 拆條件再看一次的成本很低 |
| 「`INT 3Dh` 參數 < 0 會直接返回」讀成「遊戲會走這條路」(`docs/re/06` §3)| 處理常式支援,但**十一支模組裡沒有任何一個負參數** | **「機制存在」不等於「被使用」** —— 描述能力時要分開寫 |
| 「D 的存檔寫入端只能靠 DOSBox 動態觀察,卡在開機手冊查詢」(多輪回報)| **推翻**:存檔是 `USERLIB` 槽 34,原生程式碼靜態就讀得到;記錄長度直接寫在 `OPEN` 的 `mov cx` 裡(`80`)| **本專案第三次「答案早在自己手上」** —— 前兩次漏查規則,這次漏查自己的成果。要說「做不到」之前先把已解出的過一遍 |
| 「`MONSTERS.DAT` 欄8 形狀像圖號(33/33/38 落在 46 張圖內)」(`docs/re/73` §4)| **否證**:`Siriadne !` 的欄8 是 **5000**。欄8 = 生命值(兩條等級序列嚴格單調)| **從前 N 筆看出來的形狀,要拿全部 N 筆去否證不是去確認** —— 這次只要看最後一筆 |
| 「`MONSTERS.DAT` 欄8 = 生命值(已確認)」(`docs/re/82`)| **推翻**:生命值是**欄4**(→ 屬性 3,`sub [di+6822h], 傷害`);欄8 → 屬性 19 在 `CMBT` 只寫不讀,是**經驗值**(`83`)| **一個測試若對兩個候選答案都會通過,它就不是判別測試** —— 「隨等級單調」在 RPG 裡每個數值欄位都成立,拿它當證據等於沒測 |
| 「`CHARS.DAT` 位移 41 → 戰鬥屬性 1」(`docs/re/81`)| **可疑**:屬性 0/1 是戰場座標,不該存在存檔裡;而那一筆的配對距離 33 bytes 是 12 筆裡唯一的離群值(`84` §3)| **機械配對的離群值要當成待查,不是雜訊** —— 錨點證明規則大體正確,不證明每一筆正確 |
| 「屬性 0 / 1 = 戰場座標」(`docs/re/84` §2)| 屬性 0 那一半來自掃描器的預設值 —— 找不到 `add` 就當 `imm=0`,317 處裡有 97 處(31%)如此。屬性 1 降為假設 | **掃描器的預設值不能和它的某個有效輸出撞號** —— 失敗值撞號會讓失敗看起來像訊號,而且是最強的那個 |
| 「壞掃描器只會多出一堆假的屬性 0」(`docs/re/85` §1 的影響表)| **還會掏空其他類**:`0x13F48` 是屬性 18、`0x14097` 是屬性 10,都被記成屬性 0(`86` §3)| **把失敗吞進某類的分類器會同時灌水那一類、掏空其他類** —— 凡引用過「次數」的結論都要重看 |
| 「`MONSTERS.DAT` 欄9 = 等級」(`docs/re/82` 起標為假設)| **否證**:`Lv5 Wizard`/`Lv6 Fighter` 同為 5、`Lv12`/`Lv13` 同為 8、`Lv15`/`Lv16` 同為 9 —— 三組碰撞。它是**難度階級**(`90`)| **找測試要找「兩個候選會給出不同答案」的地方** —— 這次是函數的**單射性**,不是數值大小 |
| 「怪物打怪物**永遠** +12 命中」(`docs/re/76` §4)| 屬性 10 是**朝向**,常數 `3` 只是**出場時全部面南**;一轉身就不成立(`96` §3)| **「常數初始化」不等於「永遠是這個值」** —— 看到 init 填常數,要先問有沒有別處會改它 |
| 「命中 `+30` 的條件是 `屬性8[防] ≤ 1`」(`docs/re/76` §1)| **反號**:`jg` 那一側才是 `+30`,正確為 **`> 1`**。屬性 8 = 狀態,所以是「非 `OK` 狀態的目標更容易被打中」(`98`)| **語意是抓反號的主要工具** —— 沒有語意的數字沒辦法「看起來不對」;補語意時要回頭重讀每一個比較,不能只把名字填進去 |
| 「兩個記錄的位移分不開,只能靠『這支模組開的是哪個檔』」(`docs/re/89` §3)| **數字本身就分得開**:`CHARS.DAT` 的整數欄位全在偶數位移、`GROUPS.DAT` 全在奇數位移(25 筆零例外);成因是記錄前綴長度的奇偶(`99`)| **說「只能靠上下文」之前,先看資料本身有沒有結構性特徵** |
| 「`TOWN` 讀到位移 89 → `CHARS.DAT` 的尾端有被用到」(`docs/re/89` §5)| 89 是奇數 → 屬於 `GROUPS.DAT`(記錄長 90,89–90 是最後一欄)| **在還沒能把兩個資料源分開之前,不要用聯集去推任何一個的結構** |
| 「`ds:3532h`–`ds:353Ch` 六個連續 word = 隊伍六個成員的角色編號」(`docs/re/101` 的工作假設)| **否證**:掃 `cmp` 立即數,它們分別被比 `10` / `21` / `34` / 七個不同值 —— 是六個不相干的純量 | **「DS 裡連續」不等於「同一個陣列」** —— BASIC 純量照宣告順序配置,相鄰是弱訊號;要看它們被當成什麼用 |
| 「這個遊戲沒有逃跑指令」+「J = RE-DONE」(`docs/re/95`、`97`)| **錯**:`CMBT` 有 `'PARTY RAN!'`(`0x122A9`),逃跑時把 `GROUPS.DAT` 位移 85 清為 0。根因:搜尋式有 `run` 沒有 `ran`;而「只有兩個轉交叢集」被誤讀成「只有兩個出口」——**一個叢集裡有兩個出口**(`102`)| **正對照要對「這一次的查詢」做,不是對「這個方法」做** —— 該拿一個已知存在的同類目標(`'PARTY DIES!'`)去測關鍵字,當場就會暴露它太窄 |
| 「`GROUPS.DAT` 位移 79/81 是迷宮內座標(因為與世界座標在 DS 裡相鄰)」(`docs/re/101` §1)| 位移 79 的語境是 `DE5EFF.BIN`/`DE51EFF.BIN`(每個迷宮一個的事件檔)→ **當前迷宮編號**(`104`)| 又一次:**「DS 裡相鄰」是弱訊號** —— 同一條判準在兩輪內被驗證兩次 |
| 「`GROUPS.DAT` 位移 31 = 當前城鎮、位移 25 = 所在地圖層」(`docs/re/104`,語境字串法)| 31 是**時鐘的第二級**(4–26)、25 是**每次時鐘推進都遞減的倒數計時器**;計時碼就放在進城/切模組的敘述附近(`107`)| **語境字串給的是「在哪裡被動到」,不是「它是什麼」** —— 越是被廣泛使用的變數越容易判錯,而那正是最重要的那些 |
| 「`GROUPS.DAT` 位移 31 = 當前城鎮」(`104`)→「時鐘的第二級」(`107`)| **= `Hour`**,從 `mov word ptr ds:3010h, 31` 直接讀出(`109`)| **「掃不到」的第四種成因:目標不在你掃的抽象層** —— 程式不提變數名,只提「第幾個位元組」;而 `80` 早就用同一個形狀解過存檔 |
| 「`Visibility =` → `mov ds:3010h, 59`,所以位移 59 = 能見度已確認」(`docs/re/109`)| 兩者相距 **2170 bytes**,是兩段不相干的程式碼;而同篇的時鐘三筆距離都在 20 bytes 內。位移 59 回到消去法等級(`110` §1)| **兩個事實之間的距離也是證據的一部分** —— 三筆緊鄰、一筆隔兩千位元組,不能寫成同一種證據 |
| 「`GROUPS.DAT` 位移 79 = 當前迷宮編號」(`docs/re/104`)| **= 迷宮內的欄(x)**,從 `81 × 位移79 + (位移81 − 1)` 的陣列算術讀出(`114`)| **一個方法錯到 3/4 時,它不是「有時候會錯」,是「不能用」** —— 能救回一筆的分界,不等於這個方法可用 |
| 「值 11(海洋)的圖形在 `ds:0CFA6h`,間距 682」(`docs/re/129` 結論 3,標**已確認**)| **那道指令不存在**:第十筆是 `cmp 0Bh / jnz`,值 11 **什麼都不畫**(`132` §1)| **一張規律的表,最後一列最可能不是同一種東西** —— 前九筆同形建立的預期,讓我把第十列「補完」了。而 §6 自己寫著「`0xCFA6` 由誰填 = 未解」—— **查不到寫入端時,第一個假設該是「它不是位址」** |
| 「`WRLDITEM.PIC` 有 30 行、7 個空行」(`docs/re/54` §2)| **29 行、6 個空行** —— 第 7 個是檔尾 CRLF 切出來的空段,不是資料 | **切分出來的最後一段要先問「它是資料還是分隔符的殘骸」** —— 偏移結論不受影響,但論證裡的每個數字都被引用了四份文件 |
| 「`MAZEDATA` 欄2 = 起始列、欄3 = 起始欄」(`docs/re/56` §3 的中文名)| 算式是對的,**中文名反了**:欄2 才是乘 81 的那一個(`137`)| **跨筆記引用座標時引用算式,不要引用名字** —— 「列/欄」在不同輪的心智模型裡會對調,而兩種讀法都畫得出圖,只是轉了 90 度 |
| 「`GROUPS.DAT` 出貨檔整份是空白(450 個 `0x20`)」(`docs/formats/02`)| **第 5 筆是完整存檔**(手冊的 PARTY #5),位移 1–18 是九個成員槽(`135`)| **把「已知的資料長相」寫成 assert,不要寫成註解** —— 這句話在文件裡躺了很多輪沒人踩到,變成測試後第一次執行就爆 |
| 「`CHARS.DAT` 位移 1 的 `'5'` 未解」(`100`、`122`、`formats/01`)| **所屬隊伍編號**(`133`)| **一個欄位解不出來時,先看畫面上有什麼欄位沒有來源** —— 答案在 `CHARUTIL` 的名冊標題字串裡,一次 dump 就有 |
| 「`WRLDITEM.PIC` 有 30 行、7 個空行」(`docs/re/54` §2)| **29 行、6 個空行** —— 第 7 個是檔尾 CRLF 切出來的空段 | **切分出來的最後一段要先問「它是資料還是分隔符的殘骸」** |
| 「地形 15/16/17/18 分別擋南東、南西…」(`docs/re/131` §2 推導表第一版)| 17 = 西南角、18 = 東南角,兩者寫反 | **推論要跟規則分開測** —— 規則表抄對了(64 種列舉全過),錯的是我從它推導出的那張角落表,而**實作與文件引用的是推導那一層** |
| 測試寫「反方向不該被擋」(`world_test.go` 第一版)| 規則是成對的:北擋 15/16→17/18、南擋 17/18→15/16,那是**雙向的牆** | **測試裡的假設也要有出處** —— 測試通過只證明實作符合我寫下的期待,而期待本身沒有被審過 |
| (方法)布林條件用眼睛讀 | `jge` 對應的是 `<` —— 讀反之後**仍然像一條合理的規則**,沒有任何訊號。改寫成 `tools/bool_chain.py` | **形狀規律的東西一律寫工具。判準不是「難不難」,是「錯了看不看得出來」** |
| 「位移 59 = 有光時的能見度、61 = 無光時」(`docs/re/115`,已確認)| 該篇的分岔條件「光源 ≤ 0」是**我補的註解**,`cmp` 不在 dump 範圍內;`MENU 0x119B2` 的可見 `cmp` 方向相反(`116` §2)| **註解裡的因果詞要能指到一條指令** —— 反組譯筆記裡最危險的不是錯誤結論,是自己補的註解,它下一輪會被當成一手事實引用 |
| 「`CHARS.DAT` 位移 84 = 狀態剩餘回合(證據充分)」(`docs/re/111`)| `CMBT` 裡對戰鬥屬性的 50 處 `dec`/`sub` **沒有一處是屬性 12**;「剩餘回合」降為假設(`119`)| **「X 需要有 Y」式的論證,要去找 Y 存不存在** —— 從設計常識推出來的東西一定有對應的程式碼事實可驗,沒去驗就等於只是複述常識 |
| 「`CHARS.DAT` 位移 84 = 狀態剩餘回合」(`docs/re/111`)| 屬性 12 的值是**一次以「法術最低消耗」為運算元的除法結果**(`INT 3F:8D`)→ **效果強度/倍率**(`121`)| **要分開候選,去找那個「只有其中一個會做」的動作** —— 讀值、清零、與狀態同時寫入,三個候選都會;除以消耗點數,只有倍率會 |
| (方法層)`104` 的語境字串法為何一半對一半錯 | 對的那筆(位移 45)有**閉合的句子結構**(`'You now have about'` + 值 + `' turns of light.'`,兩半相距 8 bytes);錯的那兩筆只有「附近出現過某字串」(`112` §1)| **語境證據要能「夾住」目標,不能只是「靠近」** —— 同一個方法,有沒有夾住決定它是證據還是巧合 |
| `docs/re/97` 在自己引用的 `95` §4 還留著兩項「未排除可能」時就宣告 J = RE-DONE | 其中一項(「走到戰場邊緣自動脫離」)**就是正確答案**(`103` §3)| **自己列出來的「未排除可能」,在沒有排除之前不能宣告完成** —— 四條件核對表逐條打勾的動作本身會讓人覺得檢查完了 |
| 差點把玩家職業 `'1'` 寫成「戰士 / Fighter」(`docs/re/87` 起草時)| 遊戲自己的用詞是 **`Hero`**;而 `Fighter` 在本作是**怪物名**(`Lvl 1 Fighter`)| **專有名詞不從語意猜,只從遊戲自己的字串取** —— 中文化的譯名一旦定錯,整份對照表要重來 |
| 「`fits = N` 表示譯文太長要縮短」(翻譯 TSV 的欄位設計)| 那一欄拿**原文位元組數**當上限,但 `TITLES`/`SPELLS`/`ITEMS`/`DT*TEXT` 都是變長文字檔,而 remake 由外部資源載入字串 —— 檔內長度**完全不構成約束**(`translations/README.md` §3)| **先問「這個上限是誰規定的」再去滿足它** —— 20 筆「超標」量的是一個在本專案不存在的限制,照著砍會白白犧牲譯文 |
| 用一支全域規則清掉「中文之間的空白」| `MONSTERS.DAT` 的 `R A L I T H` 譯成「拉 利 斯」,**那些空格是刻意的戲劇效果**,被一起壓掉,而 `trans_bytes` 還停在舊值沒跟著變 | **批次正規化會把「刻意的例外」跟「雜訊」一起清掉** —— 與掃描器預設值撞號(`docs/re/85`)同一類:規則沒有辦法分辨它沒被告知的意圖。改完要回頭比對受影響列,不能只看改了幾列 |
| 翻譯 TSV 的 `note` 欄寫「glossary 未收錄,保留原文」| 詞收進 glossary 之後那句話變成假的,而它長得像一手判讀依據 | **`rulebook/63` 的「現在式教訓會在修好那一刻變成假斷言」也適用於資料檔的註記欄** —— 不只是 markdown 正文 |
| 「模組字串中文化卡在『檔案位移 ↔ DS 位址』未解」(`docs/re/46` §5 掛著,我在 `translations/README.md` 第一版照抄)| **那個問題不需要解**。每段字串的描述子自帶 DS 指標,`extract_text.py` 從第一版就存進 `ds` 欄;822/822 全部接得上引用點(`124`)| **一個「未解」掛太久,先問它是為哪一條路線而提的** —— §5 那一列是為「改原版 EXE 的位元組」寫的,而本專案做 remake。路線換了,問題整個消失,不是變簡單。⚠ 這是第四次「答案早就在自己的 docs/tools 裡」 |
| 「等距資料上『差一格』的規則驗不出來」(`docs/re/70` §2 的判準)| 對**逐筆核對**成立;對**比較兩個假設各自的總命中數**不成立 —— 全模組統計下,真值 149 vs 舊規則 0(`124` §3)| **個案分不開的兩個假設,放大到全體看分佈可以分得很開** —— 判準要標明它適用於哪一種比較,否則會擋掉真正有效的方法 |
| 「這份中文手冊對應的就是 DOS 版」(`docs/re/125` §1 初稿,依封面與磁片標籤)| 手冊 p.4 正文寫「你必須有 **APPLE Ⅱ** 64K 的記憶容量」——**內文譯自 Apple II 版說明書,沒有為 PC 版校訂**(`125` §1 已訂正)| **包裝上的平台標示不等於內文的平台** —— 一份文件可以有多個來源層,證據等級要逐層問。手冊的每一項數值都要逐項比對,不能因為前一項對上就推定整章都對 |
| 「`ITEMS.DAT` 欄 3 = 價格」(`formats/04`)| **= 基準價**。售價 = `INT(欄3 × 該店倍率)`,倍率在 `TOWNDATA.DAT` 位移 38(`126`)| **「差一點點」的偏差不要當雜訊四捨五入掉** —— 四筆截圖價同時比 `基準價×1.3` 少 1,那不是誤差,是 MBF 表示不出精確 1.3 的指紋。能解釋「為什麼偏偏差這個量」的假設,比能解釋「大致對得上」強一個等級 |
| M1 的 `engine/internal/original/` 整包沒進版控,而 commit **成功** | `.gitignore` 的 `original/` 沒有錨定 —— 不帶開頭 `/` 的樣式會匹配**任何深度**的同名目錄。`cmd/convert` 進去了,它 import 的套件沒有 → clone 下來編不起來 | **`git add` 的「已忽略」提示會被淹沒在多檔輸出裡,而 commit 照樣成功** —— 加完檔要看 `git show --stat` 確認清單,不能只看 commit 有沒有報錯。gitignore 的目錄樣式一律加開頭 `/` |
| `WRLDITEM.PIC` 的圖塊偏移 = +11,後改 +14(`docs/re/128`)| **兩次都錯,正解是 +10**,而 `docs/re/54` §2 早在前一天就論證完整(7/7 空行 + 20/20 用到的值)。根因是我的 `SplitPIC` 把空行濾掉,30 行變 23,索引位移 —— 我沒懷疑解析器,而去調偏移補償(`130`)| **輸出對不上時先懷疑自己的解析器,不要先調常數。** 調常數會產生「局部自洽」的假解 —— 我的 +14 錨點全落在前四個空行之後、後兩個之前,那一段位移剛好固定是 4,所以「7/7 全中」是真的,卻只證明了那一個區段 |
| (方法層)「動手前先 grep」擋不住這一次 | 我**查了,也開了正確的檔案**(`52`、`53`),只是引用完 §1 就停 —— 而 `53` §5「尚未解開」明寫著這一項未解、`52` §2 的標題就是「圖塊 → 圖形的派工」| **引用一篇筆記的某一節時,至少掃過它的「結論」與「尚未解開」兩節** —— 後者往往指向續篇。這是第五次「答案早就在自己的 docs 裡」,但前四次是沒查,這一次是查了沒讀完 |
| (方法層)帶著假設讀 dump | 向量路徑的派工(`shl/shl/add 0C980h/int 3E`)就在我當輪 dump 的第一頁,我在找 `cmp 值 → mov bx` 的形狀,眼睛掃過去了 | **帶著一個假設去讀 dump,會讓不符合那個形狀的證據隱形** |
| 「地形值 5、6 是疏林 / 沼澤」(`docs/re/53` §1,依渲染出來的樣子)| **是水**。`WRLDMOVE` 的海岸線判定把 5、6、11 放進同一個判斷式(`129` §3)| **圖像看起來像什麼是最弱的一級證據。有程式碼可讀的時候不要用看圖來分類** |
| 「表格比正文正式,數字打架時信表格」(差點犯)| `MAGIC TORCH` 手冊 p.28 表格印 3、p.26 正文印 2,**資料檔是 2** —— 表格才是印錯的那個(`125` §2)| **同一份文件內部打架時不要挑「看起來比較正式的」,回到第三個來源裁決** |
| glossary 硬規則 4「專有名詞一律音譯,不保留英文」(我上一輪自己訂的)| **反過來了**:專案負責人裁定以精訊 1987 官方譯名為主,而精訊的做法是劇情專有名詞保留英文。同一批 19 列地城譯文在兩輪內被改了兩次方向 | **有官方前例存在時,先去找那份前例,不要先造自己的規則** —— 我造規則那一輪並不知道手冊存在,但「這款 1987 年在台灣發行過的遊戲有沒有官方中文資料」是動手前就該問的問題 |
| 「`MONSTERS.DAT`/`TOWNDATA.DAT` 名稱欄 16 bytes 是硬上限」(glossary 硬規則 2)| **對 remake 不成立**。原版資料檔只在資產轉換階段讀一次、轉成 JSON 進引擎(`spec/03` §3),中文字串不住在 16-byte 欄位裡 | **同一個毛病犯第二次**(第一次是 `fits` 欄拿原文位元組當上限)。**每一條「上限」都要問「是誰規定的、對哪一條路線成立」** —— RE 筆記是照 patch 原版寫的,整批搬進 remake 規格時要逐條重問 |
| 「手冊裡種族名譯作人類/矮人/巨魔/精靈/地精」(抄錄 agent 的回報)| 手冊**全篇沒有任何中文種族名**(p.9、p.49 兩張種族表都是純英文);那是 agent 自己補的註解 | **agent 回報的「對照表」要分清哪些是它從原件讀到的、哪些是它順手補的** —— 兩者在回報裡長得一模一樣,而後者沒有出處 |
| 「實跑停在開機的手冊查表題,要往下走**只有一條**合法途徑」(`docs/re/32` §3.1)| **按 Enter 就過**,題目每次隨機(`139` §1)。oracle 一路走得到世界地圖 | **「有一道關卡」與「這道關卡擋得住」是兩件事,後者要量。** 那句「只有一條途徑」從寫下到被推翻之間,一次都沒有被測過 —— 而測它只要一次 `key:Return` |
| 手冊 p.51:世界地圖 `←`/`→` 轉向、`<RETURN>` 前進 | **DOS 版是方向鍵先轉再走,`<RETURN>` 無作用**(`139` §4,五次可分辨的觀察)| 手冊「內文譯自 Apple II 版」的警告**第一次被實測打臉**,先前的分歧都只是譯名或印刷。**跨平台手冊講操作的那幾頁,信心要比講數值的那幾頁低一級** |
| `GROUPS.DAT` 時鐘的四個上限「未解且**不可從這些數字推**」(`formats/02`)| 手冊 p.32 給了曆法;回去讀進位指令後定案:**月 21(手冊的 22 是錯的)、日 34 ✅、時 4–26(手冊的 26 是上界不是長度)**(`140` §4)| **「不可推」講的是「不能從這四個數字自己推」,不是「沒有別的來源」。** ⚠ 第二層教訓:手冊給了數字之後我一度把「26 對 26」當成相符 —— **兩個數字相同不代表量的是同一件事**,先問它是上界、長度還是索引 |
| **世界地圖是 103 東西 × 121 南北,索引 `y×103+x`**(`re/19`/`54`/`formats/05`,標**已確認**)| **兩軸接反了**:121 東西 × 103 南北、索引 `x×103+y`,跨距 103 是**往東**(`141`)。格數與跨距從來沒錯 | **轉置過的地圖沒有症狀** —— 仍然是一塊連續陸地、仍然有海岸線、仍然畫得出來。`world.go` 的 `At()` 上方**早就寫著**「寫反了地圖仍然畫得出來,只是轉了 90 度」,擋不住,因為那句話警告的東西**看不出來**。⚠ 更該記的是當初的對照組錯了:我拿「重新切成 121 寬」去否證,而那個 reshape 會把圖切碎、一眼就知道不對 —— **它從來沒有碰到「轉置」這個候選**。**對照組要能區分你真正在猶豫的那兩件事** |
| **背包 15 格、空格是 `0`**(`formats/01`,標已確認)| **10 格(位移 54–72),空格哨兵是 `99`**(`144` §3);位移 74–83 是另一串字串旗標。手冊 p.39 也寫「只能帶 10 件」| **這是一個沒有症狀的 bug**:`EmptyPackSlot` 找值 0 的格子,而真存檔每一格都是 99 → **買東西一律回「背包已滿」**,而那句話完全合理。⚠ 測試沒抓到,因為測試用零值 `Character`(背包全 0)——**假資料要長得像真資料**,否則它保護的是假設不是行為 |
| 「`'*'` 究竟是無隊伍還是空槽,出貨資料分不開」(`133` §3)| 分得開:**`'0'` = 有角色但無隊伍、`'*'` = 空槽**(`144` §5)| **兩種語意在現有資料上重合時,去製造第三個樣本**,不要在同一份資料裡繼續找差別。讓原版造一個角色就拆開了 |
| 「角色名稱欄 10 bytes 但只讓輸入 9,**差的那一格用途未解**」(`formats/01`)| 不是未解:**創造**的提示寫 `(10 char)`、**改名**寫 `(9 char max)`,原版自己打架(`143` §4)| **一個「上限」有兩個來源時,兩個都要找出來。** 只讀到其中一個時,它看起來像個乾淨的事實,而「差一格」被寫成了未解項 |
| 「每次**實際位移**推進時鐘一格,純轉身不推進」(`spec/05` §7)| **每一個動作一格** —— 前進、轉身、存檔都算。實測:只存檔 = 1、6 步 0 轉 = 7、18 步 2 轉 = 21,三筆都等於動作數(`149`)| **一個「每 N 次做一件事」的計數器,分子與分母兩邊都要驗。** 我先前只驗了「時鐘會不會進位」(分母),沒驗「什麼動作算一次」(分子)。⚠ 慢的時鐘**沒有症狀**:天色、遭遇、食糧一起變慢,玩起來只覺得比較寬鬆 |
| `MAZEDATA.BIN` 欄 0/1 = 「入口座標 X / Y」(`re/51`,標已確認)| **欄 0 是南北、欄 1 是東西**。兩軸訂正之後這裡沒跟著改,於是**地城全部進不去** —— 正對照:直接讀 0/11 命中、對調 11/11 | **一次轉置錯誤會散在所有「成對的座標欄」上。** 訂正 `re/141` 時我只掃了世界地圖本身,沒掃**別處引用世界座標的地方**(`TOWNDATA.BIN`、`MAZEDATA.BIN`)。⚠ 症狀是沉默的:走到入口格什麼都不會發生,像「那一格本來就沒東西」 |
| 實作端用「把城鎮格按座標排序取第 n 個」猜城鎮身分(`spec/11` §2 標為佔位)| **`TOWNDATA.BIN` 就是那張對應表**,而且 `re/53` §2 早就三重驗證解出來了;實跑走到 (24,12) 得到 Arcania(表的第 4 列),排序給的是 Gleon(`145`)| **「答案早就在自己的 docs 裡」的第六次**,而這次的形狀不同:**我引用的那份筆記本身是對的,只是它比答案早一篇**(`52` 標未解、`53` 解掉)。⚠ 對策:一份筆記把某項標成「未解」時,**那句話的有效期只到下一篇** —— 要引用「未解」這個狀態,先看 `CONTEXT.md` §2 的現況表 |
| 「離開迷宮的做法未解」(`spec/08` §6,從 M5 掛到現在)| **走到格陣列界外就離開**,原版印 `Leaving maze ..`(`147`)| **要找「某個動作怎麼觸發」,先看有沒有對應的訊息字串。** 有訊息就表示動作存在,而訊息的鄰居會告訴你它屬於哪一類(這裡鄰居是三個切模組的訊息)。⚠ 我先做了按鍵窮舉(`ESC`/`E`/`Q` 全陰性),而 `Leaving maze` 這五個字從第一次 dump 字串就在檔案裡 |
| 「第 11 段酒館傳聞未定位」(`138` §4)| 找到了,而且**不是傳聞**:是續作《Demon's Winter》的預告,就接在第 10 段後面(`142` §5)| **掃描門檻卡掉的不是長度,是我對它應該長什麼樣子的預期。** 我在找「一段一百多 bytes 的傳聞」,而它是「一段一百多 bytes 的廣告」。⚠ 連帶:`ExtractRumors` 回傳的「10 段」被當成事實寫進註解與測試,**而那個 10 是我畫的掃描範圍的產物** —— 回傳的段數不是證據,除非範圍本身有出處 |
| 「賣出價未解,沒有讀到賣出的公式」(`spec/11` §3)| **原版沒有賣出功能**:商店指令列只有 `Item to buy, ESC to leave.`,手冊 p.36 也只寫買(`142` §4)| **一個公式找很久找不到,先問這個功能存不存在。**「找不到賣出公式」與「沒有賣出」在靜態分析上長得一模一樣,而只有後者能結案 |
| 「住宿 / 治療 / 食糧的基準價未解」(`138` §5,擱了好幾輪)| 實跑進 Green Hamlet 就讀到了:25 / 2 / 2 / 40 / 94×lv / 100×lv(`142`)| **量一個被係數乘過的量,先找係數為 1 的樣本。** 這間城鎮七間建築有五間倍率剛好 1.0 —— 而「哪一間倍率是 1」在檔案裡一直看得到,沒人想到要用它 |
| 「經驗值在 `CHARS.DAT` 裡沒有落點,只剩 90–94 五個 byte 而它們**沒有變異量可以指認**」(`144` §6、`spec/11` §7)| **位移 90–93,MBF 單精度**。五處程式碼、三支模組都讀它,其中兩處**緊接在載入 `Exp.  ` 標籤之後**(`150`)| **資料側沒有變異量時,不要去製造變異量,先去讀端找標籤。** 我當時規劃的下一步是「打一場勝仗再存檔比對 bytes」—— 那要跑贏一場隨機遭遇;而讀端只花一次位元組樣式掃描 + 兩次 `dump_range`。⚠ 「沒有變異量可以指認」講的是**這一種方法**沒有訊號,不是這個問題沒有入口 |
| 「五個基本屬性沒有畫面標籤,中文化時這四個名字要另外決定」(`100` §2)| 兩個地方都有:`TITLES.DAT` 的 `"Speed     :"`… 與 `CAMP`/`TOWN.EXE` 內的字串陣列(`150` §5.1)| **斷言「這個東西沒有名字」之前,要把所有可能放名字的地方掃過。** `TITLES.DAT` 從第一天就在 `CLAUDE.md` §4.4 的純文字檔清單上,而一次 `grep -a Intellect` 就會命中 |
| 「營地只有換裝備 / 施法 / 睡覺,而且武器防具是 `W`/`A` 兩個指令」(`spec/11` §4)| 原版選單**列著 11 個指令**,裝備是單一 `E)quip`;`H)unt` / `I)dentify` / `T)rade` / `R)eorder` 先前完全沒有(`150` §5.2)| **從字串反推選單會漏掉字串長得不像選單項的那些。** 這個畫面按一次 `Z` 就看得到 —— ⚠ 而本引擎自訂的「休息」還佔著 `R`,也就是原版 `R)eorder` 的位置:**自己加的東西佔了原版按鍵,那個鍵的真實行為就永遠測不出來** |
| 「手冊 p.32『丘陵上時間快兩倍』未觀察到」(`149` §4)| **已解**:起點或終點是山地的那一步花 2 格(`151`)。⚠ 而「起點與終點各加一」這個錯規則**對前三次量測完全一致** | **兩個候選在現有量測上一致時,要去設計一個它們會給出不同數字的操作。** 差別只有一格時鐘,畫面上完全看不出來 —— 是遭遇倒數這個逐格可讀的計數器讓它變得可量 |
| 「遭遇倒數歸零 → 觸發遭遇」(`spec/05` §7)| **不是完整條件**:實跑量到倒數還剩 **2** 就開打(`151` §5)| **一個計數器和一個事件同時出現,不等於前者是後者的唯一原因。** 這一條是在算山地成本時順帶撞見的 —— 回推時鐘發現數字對不上,才發現它 |
| 「魔法道具的分母是 26,戒指法杖約 27%–58%」(`91`)| 分母是 **100**,欄6 直接就是百分比 → 戒指法杖 **7%–15%**(`157`)| **拿「飽和的兩端」去驗一個尺度參數,驗不出東西。** `91` 用 100(必成)與 0(必不成)檢查分母,而那兩個值在任何 ≤100 的分母下都一樣 —— 要驗尺度得挑中段 |
| 「擲屬性那一段不在 `CHARUTIL` 的那兩處,因為面數都是 26」(`143` §5)| 排除對了但**理由是假的**:`3D:03` 是 `INT()` 不是亂數,`1Ah` 是累加器位址。真正的位址是 `0x10ABA`/`0x10CF7`,用 `3D:34`(`156`)| **一個「排除某處」的結論,若依據是某個常數的值,那麼常數被推翻時排除也跟著作廢。** 這裡連鎖了三篇:`77` 讀錯 → `143` 據此排除 → `152` 訂正 → 回頭一掃就找到了 |
| 屬性用 `4d3`(`town.AttrDice`/`AttrFaces`)| 是 `INT(RND×A + RND×A + B)` —— **兩個亂數相加再取整一次**,三角分佈(`156`)| **支撐集吻合不代表分佈吻合。** `4d3` 的範圍也是 4–12,但峰度明顯更高 —— 玩起來只覺得「極端值比較少」,沒有人會發現 |
| 「擲骰面數未解,⛔ 不要隨手填 100」(`136` §7,擱了很多輪)| **就是 100**,而且不是猜的:手冊把命中率印成百分比,程式算的是 `擲骰 ≤ 命中值`(`154`)| **一個常數解不出來時,先問「它的值有沒有別的觀測後果」。** 三條失敗的路線都在找「初值在檔案的哪個位移」,而 `42` §3 早就寫著模組的變數大半不在檔案裡。**後果比定義容易拿到,而且拿到的是同一個答案** |
| 傷害公式 `(武器傷害 × R₁ − 屬性17 − 防具) × k₁ + R₂ + k₂`(`spec/01` §5)| **力量整個漏掉了**,`k₁` 乘的是 `(力量 − 7)` 不是整個括號,而「R₂」那次擲骰其實是狂暴判定用的(`153`)| **一個公式的每一項都合理,不代表整條式子對。** 舊式子算出來的傷害同樣落在合理範圍 —— 分不開,只能讀程式。⚠ 能讀通是因為先補齊了兩個 API 語意 + 一個運算元解碼器;缺任何一個,整段都會讀成另一個自洽的公式 |
| `DamageFaces = 26`(引擎常數,源自 `77` §2)| **那個面數不存在**:`mov bx, 1Ah` 的 `0x1A` 是浮點累加器的**位址**。傷害擲骰的面數是武器傷害本身(或力量、或命中能力−5)(`153` §9)| **一個常數若只有一個出處,而那個出處是「某條指令的立即數」,要回去確認它是值還是位址。** 這個 26 從 RE 筆記一路活到引擎,還被 `spec/09` 借去當魔法道具的分母 —— 26 面骰算出來的東西看起來完全正常 |
| 「魔法道具的成功率分母 26 與傷害骰面同一個數」(`spec/09` §5)| 傷害根本沒有 26 面骰;而 `MagicItemMin = 26`(物品編號門檻)是真的 —— 兩個 26 是不同的東西 | **兩個地方出現同一個數字,不代表它們是同一件事。**「共用一個常數」這個決定把假的那個綁在真的那個旁邊,反而更難發現 |
| 「經驗的累加迴圈讀的是屬性 13(難度階級),不是屬性 19」(`150` §2.2)| 我認的是**金幣**那一段。經驗的迴圈在它前面 350 bytes,基底 `11Dh` = 欄 19,累加到 `ds:967C`;金幣那段累加到 `ds:96AC`(`152` §2)| **認出「某個迴圈」時要先確認它的輸出寫到哪裡,再去解釋它讀什麼。** 兩個迴圈的輸出變數不同,看一眼就分得開 —— 而我從讀端往回推語意,一路推到了錯的結論 |
| 「`MONSTERS.DAT` 欄 8 → 屬性 19 在戰鬥中**只寫不讀**」(`formats/03`、`83`)| **有讀**:`0x12C69` 的結算累加(`152` §1.1)。欄位語意不變,但「只寫不讀」這句話錯了 | **「只寫不讀」是一句關於掃描分母的話,不是關於程式的話。** 它撐了很多輪沒被踩到,因為結論(= 經驗值)剛好是對的 |
| 「`mov bx, 1Ah` + `INT 3D:03` 是亂數」(`77` §2)| `0x1A` 是**累加器的位址**(同篇 §1 自己寫著),`3D:03` 是 `INT()`;亂數是 `3D:34`(`152` §3)| **一個立即數在同一篇筆記裡有兩種讀法(位址 / 數值)時,以已經定案的那一種為準。** §1 讀出「0x1A = 累加器」,§2 把同一個 `1Ah` 當成十進位 26 —— 兩節隔不到一頁 |
| 「經驗總額只算**被打倒**的怪物」(`combat.TotalExp`)| 原版跑九個槽全部,沒有死活判斷(`152` §1.1)| **多數戰鬥打到全滅才結束,所以兩種算法在畫面上分不開** —— 只有怪物逃走的那一場會分岔。這種差別要靠讀程式,玩不出來 |
| 「戰後分經驗的第一個條件是**當前生命值** > 0」(`150` §2 初稿)| 是**屬性 10 = 朝向** > 0(還在場上)。把 `[di+6822h]` 的基底 158 照 `A(列,欄) = 0x6822 + (欄×15+列)×2` 拆開就是欄 10、列 9–13(`150` §2.1)| **`[reg+常數]` 的常數要照定址公式拆,不要用「這個值看起來像哪個屬性」去猜。** 拆開只要一次除法,而猜錯產生的規則**執行起來完全正常** —— 兩者只在「逃走但活著」那一格分岔 |
| 引擎的 `save()` 只寫 `GROUPS.DAT` | 原版的存檔常式在同一段裡開**兩個檔**(`re/80` §1);`g.members` 又是 `g.chars` 的複本,戰鬥的結果一步都沒回寫 | **「存檔成功」與「存了什麼」是兩件事。** 檔案長度對、欄位合法、原版也開得起來 —— 只是打完仗的血量與經驗全部回到上一次。⚠ 這個缺口是在替經驗值找位置時撞見的,不是測試抓到的 |

## 7. 每一輪都要做

0. **先查手上已經有的**(`grep docs/`、`grep tools/`、看術語表)
1. 更新受影響的 markdown
2. **清掉被推翻的斷言**,正文改寫成正確答案,推翻紀錄補進 §6
3. commit + push
4. 更新本檔 §2 現況一覽
