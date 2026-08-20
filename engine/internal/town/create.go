package town

import (
	"math"

	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

// CreateAdjustRounds 是創角時「重擲屬性」的輪數上限:**3**。
//
// 原版實跑:`Adjustment 1` → ESC → `Adjustment 2` → ESC → `Adjustment 3`
// → ESC → 直接進 `Choose class`(2026-08-18,兩次獨立實跑)。
// **ESC 不論有沒有選項目都消耗一輪。**
//
// ⚠ 這是規則不是介面:重擲**嚴格有利**(屬性是 2…13 的三角分佈,
// 不滿意就再擲,而且沒有代價),所以「無限重擲」等於讓玩家把五項刷到滿。
const CreateAdjustRounds = 3

// SkillPageSize 是技能點分配畫面**一頁幾項**:5(原版實跑 `r3e3-page2.png`)。
//
// ⚠ 這不只是版面 —— **一頁 5 項正是「按一個鍵就選得到」的理由**:
// 編號是頁內的 1–5,十項擠一頁的話「10」要按兩個鍵,單鍵選取就不成立。
const SkillPageSize = 5

// 角色創造。流程出自 `CHARUTIL.EXE` 的實跑(docs/re/143):
//
//	C)reate → Choose Race: H)uman D)warf T)roll E)lf G)nome
//	        → 五項屬性一次擲出,種族修正**另外一欄**顯示
//	        → `Enter #'s to Be rerolled Or ESC when Done`(輸入 1–5 重擲該項)
//	        → 職業 → 名稱
//
// ⚠ 畫面把**擲出的值**與**種族修正**分開顯示,所以修正不是骰進去的,
// 是事後加的 —— 這也是為什麼手冊 p.49 那張表叫「修正」而不是「加值範圍」。

// Roller 是擲一項屬性用的亂數來源。傳進來而不是內建,
// 是為了讓測試能用固定序列(docs/spec/07 的可重現性要求)。
type Roller interface {
	Roll(faces int) int // 回 1…faces
}

// 擲屬性的算式。**形狀是讀出來的**(docs/re/156):
//
//	屬性 = INT(RND × A + RND × A + B)
//
// 兩次亂數用**同一個範圍**(原版第二次乘法沒有重載 di),
// 而 `INT()` **只在總和之後做一次** —— 那讓它是個對稱三角分佈。
//
// ⚠ `INT(x + y + B)` 與 `INT(x) + INT(y) + B` 的支撐集幾乎一樣、分佈不同。
// 先前用 `4d3`(四顆骰的和)支撐集也吻合,但峰度明顯更高 ——
// **支撐集相同而分佈不同,玩起來只覺得「極端值比較少」**,沒有人會發現。
//
// A 與 B 都是**讀出來的**:CHARUTIL 的 DGROUP 常數 `ds:6C2A` 與 `ds:6C2E`
// 有編譯期初值,直接躺在檔案裡(docs/re/178)。支撐集 2…13、平均 7.5。
//
// ⚠ 先前這裡寫 `A = 5、B = 4`,是從 15 個實測樣本(4…12)反推的 ——
// **那個反推把「觀察到的最小值」當成了下界**。真正的下界 2 只有
// 5.6% 的機率出現,15 個樣本裡看不到它一點也不奇怪。
const (
	AttrRangeA  = 6.0 // ds:6C2A
	AttrOffsetB = 2   // ds:6C2E
)

// SkillPoints 回傳創造時可用的技能點數。
//
// **= 智能**(手冊 p.12 + 實跑:智能 3 的巨魔點掉成本 2 的技能後剩 1,
// docs/re/143 §3)。⚠ 只有一個樣本,分不開「= 智能」與「= 智能 + 0」。
func SkillPoints(intellect int) int { return intellect }

// SkillCost 回傳一項技能的點數成本。
//
// **兩張表都是讀出來的**:`TITLES.DAT` 第 67–86 列就是創造畫面印出來的技能表,
// 每一列尾端括號裡的數字就是成本(docs/re/178 §1)。
//
//	"1)  Sword           (2)"   →  1 號技能,成本 2
//
// ⚠ 先前這裡只有戰士表的前五項(從實跑觀察到的),其餘十五項回一個佔位值 ——
// 那五項與本表**逐項吻合**,所以它同時是這張表的正對照。
func SkillCost(class rules.Class, n int) (int, bool) {
	t := heroSkillCost
	if class == rules.ClassWizard {
		t = wizardSkillCost
	}
	if n < 1 || n > len(t) {
		return 0, false
	}
	return t[n-1], true
}

// 技能成本。索引 = 技能編號 − 1。⚠ 同一格在兩張表裡是不同的技能。
var (
	// 劍 斧 釘錘 空手 夜視 策略 護甲 技擊 打獵 說服
	heroSkillCost = [10]int{2, 2, 1, 2, 2, 2, 4, 3, 2, 2}
	// 火 金 風 冰 靈 武器知識 藥水知識 物品知識 怪物知識 降魔
	wizardSkillCost = [10]int{5, 5, 5, 5, 5, 2, 2, 2, 2, 3}
)

// FloatRoller 是能給 [0,1) 浮點的亂數來源。原版的屬性算式要的是浮點,
// 不是骰子(docs/re/156)。
type FloatRoller interface{ Float01() float64 }

// RollAttribute 擲一項屬性(**不含**種族修正)。**首擲**,有 `INT 3D:03`
// 截尾,取整仍然是 `floor`(docs/re/185 §2.1:A 類,算式不變)。
//
//	INT(RND × A + RND × A + B)
func RollAttribute(r FloatRoller) int {
	return int(math.Floor(r.Float01()*AttrRangeA + r.Float01()*AttrRangeA + AttrOffsetB))
}

// RerollAttribute 重擲一項屬性(ESC 重擲,對應畫面上的編號 1–5)。
//
// 公式形狀與 RollAttribute 相同,但**沒有** `INT 3D:03` 那道截尾 ——
// `CHARUTIL 0x10CF7`(重擲)比 `0x10ABA`(首擲)少了 `mov bx,1Ah / INT 3D:03`
// 那三個位元組,只剩最後的 `INT 3F:77`。`3F:77` 沒配 `INT()` 時是
// **四捨五入**,不是截尾(docs/re/184 §5、185 §2.1 逐位元組核對)。
//
//	round(RND × A + RND × A + B)
//
// ⚠ **這是原版真實行為,不是要跟首擲統一成同一顆骰**:重擲期望值 8.0
// 比首擲 7.5 高 0.5,而且**只有重擲擲得出上界 14**(首擲上界 13)
// (docs/re/185 §2 表列 #1)。⛔ 不要把兩者合併成同一個函式。
func RerollAttribute(r FloatRoller) int {
	return toInt(r.Float01()*AttrRangeA + r.Float01()*AttrRangeA + AttrOffsetB)
}

// Rolled 是一次擲出的五項屬性,**種族修正尚未加上去**。
// 欄位順序與創造畫面相同(Speed / Strength / Intellect / Endurance / Skill)。
type Rolled struct {
	Speed, Str, Int, End, Skill int
}

// RollAll 擲出五項。
func RollAll(r FloatRoller) Rolled {
	return Rolled{
		Speed: RollAttribute(r), Str: RollAttribute(r), Int: RollAttribute(r),
		End: RollAttribute(r), Skill: RollAttribute(r),
	}
}

// Reroll 重擲第 n 項(1–5,對應畫面上的編號)。超出範圍時原樣回傳。
//
// ⚠ 用的是 RerollAttribute(四捨五入),**不是** RollAttribute(截尾)——
// 兩者是原版兩條不同的指令路徑(docs/re/185 §2.1)。
func Reroll(v Rolled, n int, r FloatRoller) Rolled {
	switch n {
	case 1:
		v.Speed = RerollAttribute(r)
	case 2:
		v.Str = RerollAttribute(r)
	case 3:
		v.Int = RerollAttribute(r)
	case 4:
		v.End = RerollAttribute(r)
	case 5:
		v.Skill = RerollAttribute(r)
	}
	return v
}

// CreateResult 說明一次創造為什麼失敗。
type CreateResult int

const (
	CreateOK CreateResult = iota
	CreateBadClass
	CreateNoSlot
	CreateBadName
)

// docs/spec/19-module-text.md(F1):CreateNoSlot 照
// translations/module-text/CHARUTIL.tsv 第 22 列
// (「You must remove a character before adding any more!」)。
func (r CreateResult) String() string {
	switch r {
	case CreateBadClass:
		return "這個種族不能選這個職業"
	case CreateNoSlot:
		return "必須先移除一位角色才能再新增!"
	case CreateBadName:
		return "名稱不能空白"
	}
	return ""
}

// Create 依種族、職業、擲出的屬性與名稱造一個角色,寫進名冊第一個空槽。
//
// ⚠ **初始生命值 = 體能**(手冊 p.13「它也是你一開始的生命點數」;
// 創造畫面上 `Endurance 5` 對應 `H.P.: 5`,兩個來源一致)。
// **初始法力值 = 智能**(手冊 p.12,與初始生命同一組句子)。
// ⚠ 證據等級比初始生命低一級 —— 見 struct 裡的註解。
func Create(chars []original.Character, race rules.Race, class rules.Class,
	v Rolled, name string) (int, CreateResult) {

	if !rules.AllowsClass(race, class) {
		return 0, CreateBadClass
	}
	if len([]rune(name)) == 0 || len([]rune(name)) > NameMaxRunes {
		return 0, CreateBadName
	}
	slot := -1
	for i := range chars {
		if !chars[i].Occupied() {
			slot = i
			break
		}
	}
	if slot < 0 {
		return 0, CreateNoSlot
	}
	spd, str, intel, end, skill := rules.ApplyRaceModifiers(
		race, v.Speed, v.Str, v.Int, v.End, v.Skill)

	c := original.Character{
		Party: original.NoParty, // '0' = 有角色但無隊伍(docs/re/144 §5)
		Name:  name,
		ID:    slot + 1,
		Race:  byte(race), Class: byte(class),
		Speed: spd, Str: str, Int: intel, End: end, ToHit: skill,
		MaxHP: end, HP: end, // 初始生命 = 體能
		// 初始法力 = 智能。手冊 p.12 把兩者寫在同一組句子裡:
		// 「智能:學技能時所擁有的點數**及巫師一開始時的法力點數**」、
		// 「體能:…同時,它也是你一開始的生命點數」。
		// ⚠ 這是**第 3 級證據**(手冊),而初始生命那一條有反組譯佐證、
		// 這一條沒有 —— 兩者寫在同一行不代表證據等級相同(docs/re/178 §3)。
		// ⚠ 戰士沒有法力,所以只給巫師。
		MaxSP: wizardSP(class, intel), SP: wizardSP(class, intel),
		Weapon: original.NotEquipped, Armor: original.NotEquipped,
		Level:  1,
		Skills: raceSkills(race, class),
		// 背包十格全空 → 十個「未辨識」(docs/re/167 §2)
		Identified: "0000000000",
		SkillPts:   intel, // 技能點數 = 智能,一點都還沒花(docs/re/144 §4)
	}
	// 背包十格全空。⚠ 哨兵是 99 不是 0 —— 填 0 會讓每一格都看起來裝著第 0 號道具。
	for i := range c.Pack {
		c.Pack[i] = original.NotEquipped
	}
	chars[slot] = c
	return slot + 1, CreateOK
}

// wizardSP 回傳初始法力:巫師 = 智能,戰士 0。
func wizardSP(class rules.Class, intel int) int {
	if class == rules.ClassWizard {
		return intel
	}
	return 0
}

// raceSkills 回傳十個技能旗標,把種族附贈的技能打開。
//
// ⚠ 同一格在戰士表與巫師表是**不同的技能**(docs/formats/01),
// 所以旗標要配上職業才有意義 —— 手冊 p.49 給的附贈技能名已經在
// rules.Races 裡換算成該職業表的編號。
func raceSkills(race rules.Race, class rules.Class) string {
	flags := []byte("0000000000")
	for _, n := range rules.Races[race].Skills {
		if n >= 1 && n <= len(flags) {
			flags[n-1] = '1'
		}
	}
	return string(flags)
}

// ── 營地的 R)eorder 與 T)rade ────────────────────────────────────────
//
// `R)eorder` 的操作介面**已經讀出來了**(docs/re/203):3×3 字母格、
// 只能搬到空格 —— 見上面的 `MoveToSlot`。`T)rade` 仍然只讀到選單上有那個字母。

// FormationSlots 是陣型格子的數量:`GROUPS.DAT` 位移 1–18 的**九個成員槽**
// (docs/formats/02)。原版的 `R)eorder` 把它畫成 3×3 的字母格
// `A B C / D E F / G H I`(CAMP:71/72/73,docs/re/203)。
const FormationSlots = original.MemberSlots

// MoveToSlot 把第 memberIdx 位成員(0 起算,依槽位順序)搬到第 slot 格
// (0 起算,對應字母 A–I)。
//
// ⚠ **只能搬到空格**(`CAMP 0x1151B` 的 `cmp [di+683Ch], 99 / jl → 重問`)——
// 原版不是「兩人互換」,是「移到一個沒人的位置」。九格裡最多五個人,
// 所以永遠有空位。
//
// ⚠ 搬完之後**成員的先後順序**跟著槽位走 —— 而先後順序就是戰場站位
// (docs/re/160),所以這個指令不只是換顯示。
func MoveToSlot(g *original.Group, memberIdx, slot int) bool {
	if slot < 0 || slot >= FormationSlots {
		return false
	}
	from, seen := -1, 0
	for i, v := range g.Members {
		if !original.ValidMemberID(v) {
			continue
		}
		if seen == memberIdx {
			from = i
			break
		}
		seen++
	}
	if from < 0 || from == slot {
		return false
	}
	if original.ValidMemberID(g.Members[slot]) {
		return false // 那一格有人
	}
	g.Members[slot], g.Members[from] = g.Members[from], g.Members[slot]
	return true
}

// TradeResult 說明一次傳遞的結果。
type TradeResult int

const (
	TradeOK TradeResult = iota
	TradeEmptySlot
	TradeNoRoom
	TradeSamePerson
)

func (r TradeResult) String() string {
	switch r {
	// 措辭照 CAMP.tsv 第 37/38 列(F3):`NO ROOM !` / ` OK`。
	case TradeOK:
		return "完成"
	case TradeEmptySlot:
		return "那一格是空的"
	case TradeNoRoom:
		return "沒有空間!"
	case TradeSamePerson:
		return "不能交給自己"
	}
	return "?"
}

// Trade 把 from 的第 slot 格交給 to。
//
// ⚠ **裝備欄存的是背包格號**(docs/formats/01 位移 34/36)——
// 傳出去的那一格若正裝備著,要先卸下,否則裝備欄會指向一個空格,
// 而那個錯誤不會報錯,只會讓角色空手打人。與 D)rop 是同一條(docs/spec/11 §4)。
func Trade(from, to *original.Character, slot int) TradeResult {
	if from == to {
		return TradeSamePerson
	}
	if slot < 0 || slot >= original.PackSlots || from.Pack[slot] == original.NotEquipped {
		return TradeEmptySlot
	}
	dst := -1
	for i, v := range to.Pack {
		if v == original.NotEquipped {
			dst = i
			break
		}
	}
	if dst < 0 {
		return TradeNoRoom
	}
	if from.Weapon == slot {
		from.Weapon = original.NotEquipped
	}
	if from.Armor == slot {
		from.Armor = original.NotEquipped
	}
	to.Pack[dst] = from.Pack[slot]
	from.Pack[slot] = original.NotEquipped
	return TradeOK
}

// SkillName 回傳一項技能的中文名稱(docs/spec/11 §7,精訊 1987 官方譯名,
// translations/glossary.md §2/§3)。順序跟 SkillCost 用的是**同一張表**
// (heroSkillCost / wizardSkillCost 的技能編號),這裡只是配上顯示名稱,
// 不是另外一份獨立資料 —— 用途是 docs/spec/16 §4 的 `P)rint` 畫面。
func SkillName(class rules.Class, n int) (string, bool) {
	t := heroSkillNames
	if class == rules.ClassWizard {
		t = wizardSkillNames
	}
	if n < 1 || n > len(t) {
		return "", false
	}
	return t[n-1], true
}

// 技能名稱。索引 = 技能編號 − 1,順序與 heroSkillCost / wizardSkillCost 相同。
var (
	// 劍 斧 釘錘 空手 夜視 策略 護甲 技擊 打獵 說服
	heroSkillNames = [10]string{
		"劍", "斧", "釘頭鎚", "空手道", "夜視",
		"策略", "護甲", "技擊術", "打獵", "說服能力",
	}
	// 火 金 風 冰 靈 武器知識 藥水知識 物品知識 怪物知識 降魔
	wizardSkillNames = [10]string{
		"火誌", "金誌", "風誌", "冰誌", "精神之誌",
		"武器知識", "藥劑知識", "物品知識", "怪物知識", "降魔術",
	}
)

// ── 沒有效果的技能(docs/spec/14 §13.1)────────────────────────────────

// SkillNoEffect 回答「這一項技能在本引擎裡有沒有效果」。
//
// **技能點是不可逆的支出** —— 玩家花掉的點數換不回來,所以「這一項目前
// 什麼都不做」必須在**他按下去之前**看得到,不能只寫在文件裡。
//
// 三項的成因相同(原版行為沒有 RE 出來),但暴露程度先前差很多:
// 說服有標在商店畫面上,夜視與怪物知識**連提示都沒有**。
//
//	戰士 5  夜視        記錄位移 46 —— 全模組沒有讀到消費端
//	戰士 10 說服能力    位移 51 —— 怎麼降價未解(ShopUnresolved 也列著)
//	巫師 9  怪物知識    位移 50 —— 沒有讀到消費端
//
// ⛔ **巫師 8「物品知識」不列進來。** 它在**原版自己**就是死技能 ——
// `LoreFor` 的第三段(編號 > 56)接不到任何真實道具(docs/re/167 §4)。
// 引擎照抄原版的行為就是正確的;標上去等於引擎在替原版下註解,
// 而那是另一件事。**「引擎還沒做」與「原版就沒做」要分開講。**
func SkillNoEffect(class rules.Class, n int) bool {
	if class == rules.ClassWizard {
		return n == SkillMonsterLore
	}
	return n == SkillNightVision || n == SkillPersuasion
}

// 三項沒有效果的技能編號。⚠ 巫師與戰士**同一格是不同技能**,
// 所以 SkillMonsterLore 與 SkillHunting 都是 9 並不是筆誤。
const (
	SkillNightVision = 5  // 戰士:夜視
	SkillPersuasion  = 10 // 戰士:說服能力
	SkillMonsterLore = 9  // 巫師:怪物知識
)

// SkillNoEffectMark 是技能清單上的記號;SkillNoEffectNote 是底下那一行說明。
const (
	SkillNoEffectMark = "!"
	SkillNoEffectNote = "(標 ! 的原版行為未解,學了目前沒有效果)"
)
