package combat

// 戰鬥訊息的字面回歸測試(F3,docs/spec/19-module-text.md)。
//
// 這一組釘的不是行為而是**措辭**:訊息要說原版說的話
// (`translations/module-text/CMBT.tsv` 第 69–82 列),不是實作時自己寫的中文。
// 沒有這一組,改回「擊中／落空／倒下」不會有任何測試變紅 ——
// 而 `tools/check_module_text.py` 只看得到「這串字有沒有出現在檔案裡」,
// 看不到它有沒有真的被組進訊息。

import (
	"strings"
	"testing"

	"shardofspring/internal/original"
)

// newMessageField 擺一個結果可預測的場面:隊員拿短劍打生命 1 的地精。
func newMessageField(toHit int) *Field {
	f := &Field{Rand: NewRand(11), Items: map[int]Item{
		2: {Main: 6, Bonus: 0, Name: "短劍"},
	}}
	f.Units[MonsterBase] = Unit{Name: "地精", HP: 1, Facing: South,
		IsMonster: true, X: 11, Y: 12}
	f.Units[PartyBase] = Unit{Name: "凱恩", HP: 20, Facing: North,
		Str: 12, ToHit: toHit, Weapon: 2, Armor: original.NotEquipped, X: 11, Y: 13}
	return f
}

const (
	toHitAlways = 30 // 門檻 120,d100 擲不出不中(值域 1…101)
	toHitNever  = 0  // 門檻 0,擲骰最小 1 → 必不中
)

func TestAttackMessageHitAndKill(t *testing.T) {
	f := newMessageField(toHitAlways)
	if _, hit, dmg := f.Attack(PartyBase, MonsterBase); !hit || dmg < 1 {
		t.Fatalf("fixture 失效:應該命中且有傷害,得到 hit=%v dmg=%d", hit, dmg)
	}
	got := f.Log[len(f.Log)-1]
	for _, want := range []string{"凱恩", "攻擊", "地精", "使用", "短劍",
		"命中造成", "點傷害。", "牠死了!"} {
		if !strings.Contains(got, want) {
			t.Errorf("訊息少了 %q:%s", want, got)
		}
	}
	// 怪物死的是「牠」不是「他」—— 原版就是兩段不同的字串。
	if strings.Contains(got, "他死了") {
		t.Errorf("怪物倒下應該用「牠死了!」:%s", got)
	}
}

func TestAttackMessageMiss(t *testing.T) {
	f := newMessageField(toHitNever)
	if _, hit, _ := f.Attack(PartyBase, MonsterBase); hit {
		t.Fatal("fixture 失效:門檻 0 應該必不中")
	}
	got := f.Log[len(f.Log)-1]
	if !strings.Contains(got, "但沒打中!") {
		t.Errorf("落空要說「但沒打中!」:%s", got)
	}
	// 落空這一句仍然帶著武器 —— 原版的 `with` 在命中判定之前就印出去了。
	if !strings.Contains(got, "使用 短劍") {
		t.Errorf("落空的句子也該有武器:%s", got)
	}
}

func TestAttackMessageNoDamage(t *testing.T) {
	f := newMessageField(toHitAlways)
	f.Units[MonsterBase].ArmSkin = 20 // 擲多少都減到 0 以下
	_, hit, dmg := f.Attack(PartyBase, MonsterBase)
	if !hit || dmg != 0 {
		t.Fatalf("fixture 失效:應該命中但傷害 0,得到 hit=%v dmg=%d", hit, dmg)
	}
	got := f.Log[len(f.Log)-1]
	if !strings.Contains(got, "沒有造成傷害。") {
		t.Errorf("傷害 0 要說「沒有造成傷害。」:%s", got)
	}
	if strings.Contains(got, "點傷害") {
		t.Errorf("傷害 0 不該同時印數字:%s", got)
	}
	if f.Units[MonsterBase].HP != 1 {
		t.Errorf("傷害 0 不該扣血,得到 %d", f.Units[MonsterBase].HP)
	}
}

// TestAttackMessagePartyMemberDies:隊員倒下用「他死了!」。
func TestAttackMessagePartyMemberDies(t *testing.T) {
	f := newMessageField(toHitAlways)
	// 反過來打:怪物拿同一把武器打生命 1 的隊員。
	f.Units[MonsterBase].ToHit = toHitAlways
	f.Units[MonsterBase].Str = 12
	f.Units[MonsterBase].Weapon = 2
	f.Units[PartyBase].HP = 1
	if _, hit, _ := f.Attack(MonsterBase, PartyBase); !hit {
		t.Fatal("fixture 失效:應該命中")
	}
	got := f.Log[len(f.Log)-1]
	if !strings.Contains(got, "他死了!") {
		t.Errorf("隊員倒下應該用「他死了!」:%s", got)
	}
}

// TestAttackMessageBareHandHasNoWeaponPhrase:赤手空拳不印「使用 …」。
//
// ⚠ 這是**引擎的選擇**,不是讀出來的原版行為(field.go 的 weaponPhrase 有說明)。
// 釘在這裡是為了讓它變成一個有意識的決定 —— 哪天查證了原版怎麼印,
// 改的時候會有一條測試提醒這裡曾經是空白。
func TestAttackMessageBareHandHasNoWeaponPhrase(t *testing.T) {
	f := newMessageField(toHitAlways)
	f.Units[PartyBase].Weapon = original.NotEquipped
	f.Attack(PartyBase, MonsterBase)
	if got := f.Log[len(f.Log)-1]; strings.Contains(got, "使用") {
		t.Errorf("赤手空拳不該印武器:%s", got)
	}
}
