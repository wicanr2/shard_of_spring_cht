// Package magic 是法術與魔法道具的規則。
//
// 規則出自 docs/spec/02-magic.md 與 docs/spec/09-magic-items.md;
// 每一條在下面註明章節。
package magic

import (
	"fmt"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
)

// SkillBase 是技能旗標在角色記錄裡的起點:MID$(記錄, 系別 + 41, 1)
// (docs/spec/09 §1)。
//
// ⚠ **索引是「系別」不是「法術編號」。** 系別只有 1–5(五種 runes),
// 33 個法術共用這五個旗標。寫成法術編號會讓大部分法術查到界外,
// 而界外在 Go 裡是 panic、在原版裡是讀到別的欄位 —— 兩邊都不是靜默,
// 但錯的方向不同,所以這裡明寫。
const SkillBase = 41

// WizardClass 是可以施法的職業碼(docs/formats/01 位移 15)。
const WizardClass = '2'

// MagicItemMin 是「魔法道具」的物品編號門檻:**大於**它才走道具發動
// (docs/spec/02 §4)。
const MagicItemMin = 26

// Fail 說明為什麼施不出來。
type Fail int

const (
	OK Fail = iota
	FailNotWizard
	FailNoSkill
	FailNoPoints
	FailBelowOneLevel
)

// docs/spec/19-module-text.md(F1):FailNotWizard 照 CAMP:133,
// FailNoSkill 照 CAMP:82(「You don't know that spell!」),
// FailNoPoints 照 CAMP:85(spec/19 §1.1 的範例句,一字不差)。
// FailBelowOneLevel 在這批清冊裡沒有對應的原文,沿用既有措辭。
func (f Fail) String() string {
	switch f {
	case FailNotWizard:
		return "這位角色不是巫師。"
	case FailNoSkill:
		return "你不會那個法術!"
	case FailNoPoints:
		return "你需要更多法力點數!"
	case FailBelowOneLevel:
		return "投入的點數不足一級"
	}
	return ""
}

// CanCast 檢查三個閘門(docs/spec/09 §1)。
func CanCast(c original.Character, s original.Spell, invest int) Fail {
	if c.Class != WizardClass {
		return FailNotWizard
	}
	i := s.School + SkillBase - 1 // Skills 是 0-based 切片,記錄位移是 1-based
	if s.School < 1 || i-SkillBase < 0 || i-SkillBase >= len(c.Skills) {
		return FailNoSkill
	}
	if c.Skills[s.School-1] == '0' {
		return FailNoSkill
	}
	if invest > c.SP {
		return FailNoPoints
	}
	if Level(s, invest) < 1 {
		return FailBelowOneLevel
	}
	return OK
}

// Level 回傳這次施法的等級:INT(投入 ÷ 每級單價)(docs/spec/09 §2)。
func Level(s original.Spell, invest int) int {
	if s.UnitCost <= 0 {
		return invest // 單價 0 的法術:投多少算多少級
	}
	return invest / s.UnitCost
}

// Power 回傳威力:每點威力 × **投入點數**(不是等級,docs/spec/09 §2)。
//
// ⚠ 兩者只在單價 = 1 時相同。寫成「× 等級」不會報錯,
// 只會讓單價 > 1 的法術全部變弱。
func Power(s original.Spell, invest int) int { return s.Power * invest }

// StatusMagnitude 回傳狀態強度:每級單價 ÷ 投入(docs/spec/09 §4)。
//
// ⚠ **投得越多、值越小。** 而狀態強度在傷害公式裡是減項
// (docs/spec/01 §5),所以值小 = 被打得越痛。這個方向反直覺,是照抄的。
func StatusMagnitude(s original.Spell, invest int) int {
	if invest <= 0 {
		return 0
	}
	return s.UnitCost / invest
}

// Result 是一次施法的結果。
type Result struct {
	Fail       Fail
	Message    string
	Unresolved bool // 效果未解 —— 訊息會標出來,而且不套用任何數值
	// NoEffect = 這一次什麼都沒發生(威力 0、沒有狀態可解、沒有被束縛)。
	//
	// ⚠ 有這個旗標是因為**原版對同一件事有好幾句話**:戰鬥是
	// CMBT:128/129、營地是 CAMP:106/108,措辭與標點都不同。
	// 呼叫端照自己的情境挑句子,⛔ 不要在這裡把它們併成一句。
	NoEffect bool
	// WindWalk = 這一次是風行術:全隊傳送到世界座標 (8, 8) 並離開地城
	// (docs/re/193)。**效果由呼叫端做** —— 這個套件看不到世界地圖,
	// 也不該看得到。
	WindWalk bool
}

// SpellWindWalk 是「風行術」在 `assets/data/spells.json` 的索引。
//
// ⚠ **原版比的是 `ds:7430 == 22`,那是 `SPELLS.DAT` 的列號(1 起算)**;
// 轉出的 `index` 是 0 起算,所以這裡是 21(docs/re/193 §1)。
const SpellWindWalk = 21

// WindWalkX / WindWalkY 是風行術的落點。
//
// 翠綠村在 (9, 8),所以這是**它西邊一格** —— 踩在城鎮格上會直接進城,
// 而這個法術要做的是回到文明世界,不是進城(docs/re/193 §3)。
const (
	WindWalkX = 8
	WindWalkY = 8
)

// MsgWindWalk 是 CAMP:109。原版在傳送**之前**印它,所以句尾的刪節號
// 接的就是「回到世界地圖」那一幕。
const MsgWindWalk = "一陣怒吼般的狂風吹襲隊伍,當聲響平息之後……"

// 效果類別。docs/formats/04。
const (
	EffGroupDamage  = 1
	EffSingleDamage = 2
	EffToHit        = 3 // 命中能力(屬性 9,docs/re/171 §3)
	EffStrength     = 4
	EffHitPoints    = 5
	EffSpeed        = 6
	EffProtect      = 7
	EffRaise        = 8
	EffCure         = 9
	EffUnbind       = 10
	EffBind         = 11
	EffUtility      = 12
	EffTransference = 13 // ⚠ 僅一例,**未解**
)

// 法術結果的字面,照 `translations/module-text/CMBT.tsv`(F3)。
const (
	MsgDies         = "，並死亡!"       // 122 `and Dies!`
	MsgNoDifference = "沒有感到任何變化。" // 128+129 `notices no` + `difference.`
	// 134–138 `is not bound in` + `chains and` + `still air and` + `ice and`
	// + `notices no effect` —— 原版把三種束縛狀態逐一列出來。
	MsgNotBound = "沒有被鐵鍊束縛,凝滯的空氣,冰霜,沒有感到效果"
)

// 束縛類的狀態值。`CHARS.DAT` 位移 1 的狀態碼:
// 1 中毒、2 束縛、3 凝滯、4 冰封、5 陣亡(docs/formats/01)。
//
// ⚠ **解除束縛吃三種,不是只有「束縛」**:CMBT 的訊息把它們並列成
// `is not bound in chains and still air and ice`(CMBT:134–137)——
// `chains` = 束縛、`still air` = 凝滯、`ice` = 冰封。信心:**證據充分**
// (模組自己的字串,不是從技能名推的)。
const (
	StatusBound  = 2 // chains
	StatusStill  = 3 // still air
	StatusFrozen = 4 // ice
)

// IsBound 回傳這個狀態算不算「被束縛」——解除束縛只對這三種有效。
func IsBound(status int) bool {
	return status == StatusBound || status == StatusStill || status == StatusFrozen
}

// Apply 把一個法術套到目標身上,回傳訊息。
//
// ⚠ **沒有 default 分支吞掉未知類別**(docs/spec/09 §7 驗收 8)——
// 未知的類別要走到 unresolved 那一條,在訊息上看得見。
func Apply(s original.Spell, invest int, caster *combat.Unit,
	targets []*combat.Unit) Result {

	p := Power(s, invest)
	name := s.Name
	switch s.Effect {
	case EffGroupDamage:
		died := 0
		for _, t := range targets {
			combat.Apply(t, p)
			if !t.Alive() {
				died++
			}
		}
		msg := fmt.Sprintf("%s 對全體造成 %d 點傷害", name, p)
		if died > 0 {
			msg += MsgDies // CMBT:122「and Dies!」
		}
		return Result{Message: msg}

	case EffSingleDamage:
		msg := fmt.Sprintf("%s 造成 %d 點傷害", name, p)
		if len(targets) > 0 {
			combat.Apply(targets[0], p)
			if !targets[0].Alive() {
				msg += MsgDies
			}
		}
		return Result{Message: msg}

	case EffToHit, EffStrength, EffHitPoints, EffSpeed:
		// 類別 → 屬性欄是**讀到的**(docs/re/171 §3):
		// 3 → 屬性 9(命中能力)、4 → 6(力量)、5 → 3(生命值)、6 → 2(速度)。
		// ⚠ 四個裡三個本來就確認過,所以第四個的讀法可信 —— 不是從
		// `Becomes clumsy` 這個名字猜的(docs/spec/09 §3 當初明文禁止那樣做)。
		for _, t := range targets {
			switch s.Effect {
			case EffToHit:
				t.ToHit += p
			case EffStrength:
				t.Str += p
			case EffHitPoints:
				t.HP += p
				if t.HP < 0 {
					t.HP = 0
				}
			case EffSpeed:
				t.Speed += p
			}
		}
		if p == 0 {
			// CMBT:128/129「notices no difference.」—— 威力算出來是 0,
			// 屬性一點都沒動。⛔ 不要印「力量 +0」,那看起來像有效果。
			return Result{Message: name + "：" + MsgNoDifference, NoEffect: true}
		}
		attr := map[int]string{
			EffStrength: "力量", EffHitPoints: "生命值", EffSpeed: "速度",
		}[s.Effect]
		return Result{Message: fmt.Sprintf("%s：%s %+d", name, attr, p)}

	case EffProtect:
		for _, t := range targets {
			t.ArmSkin += p
		}
		return Result{Message: fmt.Sprintf("%s 提供 %d 點防護", name, p)}

	case EffRaise:
		for _, t := range targets {
			if !t.Alive() {
				t.HP = 1
				t.Status = 0
			}
		}
		// CAMP:104「Lives !」
		return Result{Message: name + "：活過來了!"}

	case EffCure:
		cured := 0
		for _, t := range targets {
			if t.Status != 0 {
				cured++
			}
			t.Status = 0
			t.StatMag = 0
		}
		if cured == 0 {
			// CMBT:132/133 —— 本來就沒有狀態,治了等於沒治。
			return Result{Message: name + "：" + MsgNoDifference, NoEffect: true}
		}
		// CAMP:105「Is cured.」
		return Result{Message: name + "：被治癒了。"}

	case EffUnbind:
		freed := 0
		for _, t := range targets {
			if IsBound(t.Status) {
				t.Status = 0
				t.StatMag = 0
				freed++
			}
		}
		if freed == 0 {
			// CMBT:134–138「is not bound in chains and still air and ice and
			// notices no effect」
			return Result{Message: name + "：" + MsgNotBound, NoEffect: true}
		}
		// CMBT:130「Breaks free!」
		return Result{Message: name + "：掙脫了!"}

	case EffBind:
		for _, t := range targets {
			t.Status = s.School
			t.StatMag = StatusMagnitude(s, invest)
		}
		return Result{Message: fmt.Sprintf("%s %s", name, s.HitMsg)}

	case EffUtility:
		// 類別 12 有三個成員:風行術(21)、魔法火炬(31)、水晶燈術(32)。
		// 只有第一個解出來了(docs/re/193),另外兩個是照明,規則未讀。
		if s.Index == SpellWindWalk {
			return Result{WindWalk: true, Message: MsgWindWalk}
		}
		return Result{Message: name + "(非戰鬥效用)"}

	case EffTransference:
		// ⚠ 僅一例,群內無對照,結構上無法再分(docs/spec/02「未解」)。
		return Result{Unresolved: true,
			Message: name + " 發動,但效果未解(類別 13,全遊戲僅此一個)"}
	}

	// 走到這裡表示資料裡有規格沒列的類別 —— 讓它看得見,不要靜默。
	return Result{Unresolved: true,
		Message: fmt.Sprintf("%s 的效果類別 %d 不在規格裡", name, s.Effect)}
}

// ItemBreaks 判定一件魔法道具這次用完會不會壞掉(docs/re/190)。
//
// **分母是 100,欄6 就是百分比**(docs/re/157):原版在 `CMBT 0x17C67` 擲
// `INT(RND × ds:977E) + 1` 再比 `≤ 欄6`,而 ds:977E = 100(docs/re/154)。
//
// ⚠ **欄6 是「壞掉」的機率,不是「發動」的機率**(docs/re/190)。
// `157` 讀對了判定式卻讀錯了那一支做什麼 —— 結論在**旗標的消費端**,
// 而那一端印的是 `Item Breaks !` 並把背包那一格清掉。
//
// ⚠ 讀反了**沒有症狀**:戒指有時有效有時沒效,在這類遊戲裡看起來完全正常。
// 資料面的印證:火把／油燈欄6 = 100(一用就沒了 ✅,不是「必定發動」)、
// 鑰匙欄6 = 0(永遠不壞 ✅,不是「永遠打不開門」)。
//
// ⚠ 這裡曾經用 26,理由是「與傷害公式的骰面同一個數」—— 而那個 26 是幻影
// (`mov bx, 1Ah` 的 0x1A 是浮點累加器的位址,docs/re/153 §9)。
func ItemBreaks(itemIndex, breakChance int, r combat.Rand) bool {
	if itemIndex <= MagicItemMin {
		return false // 不是魔法道具
	}
	return r.Roll(combat.ToHitFaces) <= breakChance
}
