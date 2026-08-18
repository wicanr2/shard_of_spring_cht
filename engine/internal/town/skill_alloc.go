package town

import (
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

// 技能點分配。docs/spec/20-skill-allocation.md。
//
// 規則層絕大部分已經是既有的:成本表 SkillCost、名稱表 SkillName
// (create.go,只讀不改)、旗標欄 Character.Skills、點數欄 Character.SkillPts。
// 這裡只補「花」的那一半:LearnSkill。

// LearnResult 說明一次「學技能」的結果。
type LearnResult int

const (
	LearnOK LearnResult = iota
	LearnBadNumber
	// LearnRemoved:原本學過,這一次按取消掉並退點(見 LearnSkill)。
	LearnRemoved
	LearnNotEnough
)

// docs/spec/20 §1:LearnBadNumber 對應原版「Mumble, mumble…」一類的
// 亂數字回應——這裡沿用引擎其他地方的中文措辭,不硬翻那句(它本來就
// 不是給玩家準確資訊用的)。
//
// ⚠ `LearnNotEnough` 走的是 TOWN:54;`LearnRemoved` 的原版措辭未解
// (畫面上只有 `(or remove)` 這個提示)。`LearnBadNumber` 同樣是引擎的措辭。
func (r LearnResult) String() string {
	switch r {
	case LearnBadNumber:
		return "沒有這個編號的技能。"
	case LearnRemoved:
		return "取消了這項技能,點數退回。"
	case LearnNotEnough:
		// TOWN:54「Not enough IQ !」——「IQ」指的是**剩餘技能點數**
		// (`CHARS.DAT` 位移 89,創造時 = 智能),不是智能本身。
		// 原版比的就是「可用點數 < 該技能的成本」(docs/re/196 §2)。
		return "智能不夠!"
	}
	return ""
}

// LearnSkill 讓角色學一項技能(編號 1–10),扣掉點數。
//
// 職業從 c.Class 取,**不由呼叫端傳**——heroSkillCost / wizardSkillCost
// 同一格是不同技能,傳錯職業會學到另一項,而且畫面上看起來完全正常
// (docs/spec/20 §1)。
//
// ⚠ **具名假設,不是讀到的判斷式**(docs/spec/20 §4):已經學過 / 點數
// 不足這兩種情況,這裡選擇**擋下、不扣點**。理由:
//
//   - Skills 是二值旗標('0'/'1'),重複學一次拿不到任何東西,扣點只是白扣;
//   - SkillPts(位移 89)是單一 byte,扣成負數會在 Bytes() 環繞成一個很大
//     的正數,把存檔寫壞。
//
// 要裁決這兩種情況原版實際怎麼處理,得去讀 TOWN.EXE sub_11D09
// (輸入迴圈,docs/re/183 §7 列為未讀)。
func LearnSkill(c *original.Character, n int) LearnResult {
	class := rules.Class(c.Class)
	cost, ok := SkillCost(class, n)
	if !ok {
		return LearnBadNumber
	}
	// ⚠ **已經學過 = 取消,並把點數退回來** —— 原版的分配畫面右欄
	// 就寫著 `Enter: # to choose (or remove)`,而實跑證實了它:
	// 智能 9 學 Sword(成本 2)→ 剩 7 → 再按一次 → 剩 9、星號消失
	// (2026-08-18,workplace/dosbox/shots/r3e1-learn.png / r3e2-remove.png)。
	//
	// ⚠ **種族技能能不能這樣退,沒有量過** —— 那是白送的旗標,
	// 退掉會憑空多出點數。這裡照畫面上寫的規則一視同仁,
	// ⛔ 不替原版發明一條它沒宣告的例外。
	if n <= len(c.Skills) && c.Skills[n-1] == '1' {
		flags := skillFlags(c.Skills)
		flags[n-1] = '0'
		c.Skills = string(flags)
		c.SkillPts += cost
		return LearnRemoved
	}
	if c.SkillPts < cost {
		return LearnNotEnough
	}
	flags := skillFlags(c.Skills)
	flags[n-1] = '1'
	c.Skills = string(flags)
	c.SkillPts -= cost
	return LearnOK
}

// skillFlags 把技能旗標補滿 10 格再回傳可改的複本。
func skillFlags(s string) []byte {
	flags := []byte(s)
	for len(flags) < 10 {
		flags = append(flags, '0')
	}
	return flags[:10]
}
