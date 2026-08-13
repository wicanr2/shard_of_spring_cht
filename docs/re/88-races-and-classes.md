# 88 — 五個種族、兩個職業:位移 14 / 15 的碼表

日期:2026-08-14
接續:[`87-class-race-and-attribute-17.md`](87-class-race-and-attribute-17.md)
子系統:**D. 角色資料**

## 結論

`CHARUTIL` 的字串區把 [`87`](87-class-race-and-attribute-17.md) 留下的兩個問號填掉了。

| `CHARS.DAT` 位移 14 | 種族 |
|:-:|---|
| `H` | **Human** |
| `T` | **Troll** |
| `D` | **Dwarf** |
| `E` | **Elf** |
| `G` | **Gnome** |

| 位移 15 | 職業 |
|:-:|---|
| `1` | **Hero** |
| `2` | **Wizard** |

## 1. 三條獨立的線

| 線 | 內容 |
|---|---|
| 字串 | `CHARUTIL` 的 `ds:0x6EA8`–`0x6ED2` 連續五個:`Human` `Troll` `Dwarf` `Elf` `Gnome`;`ds:0x6DD0`/`0x6DD8` 兩個:`Hero` `Wizard` |
| 按鍵鏈 | `CHARUTIL` 的比對鏈裡出現 **`H D T E G`**,而且出現兩次([`70`](70-key-chains-all-modules.md) §5)|
| 存檔資料 | 五個預設角色的位移 14 是 `H D T E G` —— **五個全不同** |

**五個種族的英文首字母剛好互不重複**,所以單字元就夠當代碼。
畫面標頭也對得上:`'#) Name        Race   Class   Level   Status'`。

## 2. 五個預設角色

| # | 名稱 | 種族 | 職業 | 法術系別旗標(位移 42–46)|
|---:|---|---|---|---|
| 0 | `Segrono` | Human | Hero | — |
| 1 | `Hard Axe` | Dwarf | Hero | — |
| 2 | `Grod` | Troll | Hero | — |
| 3 | `Fire Hawk` | Elf | Wizard | 有 |
| 4 | `Richtatha` | Gnome | Wizard | 有 |

三個 Hero + 兩個 Wizard,五個不同種族 —— **出貨隊伍剛好把五個種族各用一次**。

## 3. ⚠ 「戰士」是錯的,遊戲叫它 `Hero`

[`87`](87-class-race-and-attribute-17.md) §4 寫過:

> `'1'` 這三筆的名字看起來像戰士,**但那是從名字猜的,不算**。

**忍住那一步是對的** —— 遊戲自己的用詞是 **`Hero`**,不是 Fighter 或 Warrior。
而 `MONSTERS.DAT` 裡**確實有** `Lvl 1 Fighter`、`Lvl 2 Fighter`……
所以「Fighter」在這個遊戲裡是**怪物的名字**,不是玩家職業。

**猜下去會同時搞錯兩件事**:玩家職業叫錯,還會把玩家職業和怪物名混為一談。
中文化時 `Hero` 與 `Fighter` 必須用兩個不同的詞。

> **判準:專有名詞不從語意猜,只從遊戲自己的字串取。**
> 這一條對中文化尤其致命 —— 譯名一旦定錯,整份對照表都要重來。

## 4. 屬性 17 的判讀更新

[`87`](87-class-race-and-attribute-17.md) §1:屬性 17 只有位移 15 = `'1'` 時才取值。

**= 只有 `Hero` 有的減傷項**,來源是位移 48 那串旗標的第一位
(三個 Hero 都是 `1`,兩個 Wizard 都是 `0`)。

⚠ 那個旗標**代表什麼能力**仍然未解 —— 知道「誰有」不等於知道「那是什麼」。

## 5. D 的現況

`CHARS.DAT` 94 bytes 裡有語意的位置:**10 個**
(名稱、種族、職業、命中能力、當前生命值、武器格號、防具格號、
法術系別旗標 ×5、`0`/`1` 旗標串、背包)。

還缺:其餘位移、`CHARUTIL` 的建角色寫入端、位移 48 那串旗標的其餘位元。
