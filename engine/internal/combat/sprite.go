package combat

// 單位的圖檔對應。docs/re/220。
//
// 原版把 `MONST*.BIN` 的八格圖依序編號成**圖組**(屬性 11):
//
//	圖組 = 檔號 × 8 + 25
//
// 讀出來的兩個定點:`CMBT 0x118F4` 把隊伍戰士的屬性 11 設成 `29h` = **41**、
// `0x11962` 把巫師設成 `39h` = **57**;而 `CMBT` 的字串表裡有兩個**寫死的檔名**
// `monst2.bin` 與 `monst4.bin`。41 → 檔 2、57 → 檔 4,間隔 16 = 兩個檔 × 8。
// 步長 8 直接讀得到:載完一個檔之後 `0x1159C` 做 `ds:9494 += 8`。
//
// 反過來就是 **`檔號 = MONSTERS.DAT 欄6`**(怪物)。⚠ 內容對照七項全中:
// 檔 15 蝙蝠、16 眼鏡蛇、17 火焰、18 劍身(金屬)、20 幽靈(靈)、
// 21 旋風(風)、22 龍 —— 逐一對上 `MONSTERS.DAT` 同編號的怪。
//
// ⚠ **欄6 沒有 2 與 4**,正是因為那兩個檔是**隊員**的圖。

// 隊員的圖檔:CMBT 寫死這兩個檔名。
const (
	SpriteFileFighter = 2
	SpriteFileWizard  = 4
)

// SpriteFile 回傳這個單位要用哪一個 `MONST<N>.BIN`;0 = 沒有對應的圖。
//
// ⚠ 怪物的 `Kind` 存的是**欄6**(build.go 的 `m.Class`),
// 隊員的 `Kind` 存的是**圖組**(41／57)—— 兩者不同尺度,不要直接比。
func SpriteFile(u Unit) int {
	if !u.IsMonster {
		if u.Kind == KindWizard {
			return SpriteFileWizard
		}
		return SpriteFileFighter
	}
	if u.Kind < 1 || u.Kind > MonstFiles {
		return 0
	}
	return u.Kind
}

// MonstFiles 是原版附的 `MONST*.BIN` 檔數。
const MonstFiles = 22
