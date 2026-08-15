// Package save 是引擎自己的存檔格式。docs/spec/18-save-format.md。
//
// 取代「寫回原版 CHARS.DAT / GROUPS.DAT」當主存檔路徑那件事(docs/spec/18 §1)——
// 原版把「事件已觸發」這類進度寫回 DE*EFF.BIN 本身(docs/re/181),而
// CLAUDE.md §8 規定 game/ 唯讀,remake 沒有那條路可走,只能另開一份存檔。
//
// ⚠ 這個套件**只管格式與 I/O**,不碰任何遊戲規則。原版的欄位型別
// (original.Character / original.Group)直接沿用,**不新增、不修改
// internal/original 的欄位定義**(docs/spec/18 §6 的邊界)。
package save

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"shardofspring/internal/original"
)

// CurrentVersion 是這個引擎認得的存檔版本。docs/spec/18 §2:
// 未知版本要**拒絕載入並講清楚**,不是盡力解析 —— 半懂的存檔會安靜弄壞進度。
const CurrentVersion = 1

// DefaultName 是主迴圈自動存讀的存檔名(saves/party.json)。
//
// ⚠ docs/spec/18 §2 的格式支援**任意檔名**的多存檔(Read/Write/List 都吃
// 任意路徑),這裡的常數只是「目前唯一被外殼(shell_scene.go)自動使用」
// 的那一個名字 —— 存檔挑選畫面(選哪一份 saves/*.json)不在這一輪的範圍內,
// 見 engine 那三個檔案裡的說明。
const DefaultName = "party"

// Progress 是原版沒有欄位可以裝、或存放位置未解的狀態。
// docs/spec/18 §3.2:每一項都要註明「原版存在哪」——未解就寫未解,不留白。
type Progress struct {
	// DisabledEvents 是已經作廢的一次性事件:key = 事件檔編號
	// (original.MazeEntry.TextFile,即 DE<N>EFF.BIN 的 N)→ 目標編號集合。
	//
	// 原版把「事件已觸發」寫回 DE*EFF.BIN 本身(docs/re/181),`R)estore Mazes`
	// 就是從 .MST 母片還原它。remake 不能那樣做(game/ 唯讀,CLAUDE.md §8),
	// 所以改記在這裡;進迷宮時(maze_scene.go 的 loadLevel)要把它套回去,
	// 否則走出迷宮再走回來,一次性事件就復活了(docs/spec/18 §1 第 3 項)。
	DisabledEvents map[string][]int `json:"disabledEvents,omitempty"`

	// Tombs 是踩過的墓(事件 701–704),餵 Eldron 氏族謎題的進度。
	// ⚠ 原版存在哪未解(docs/re/162 §5)。
	Tombs []int `json:"tombs,omitempty"`

	// ClanRewarded:氏族謎題解過了。⚠ 原版存在哪未解(docs/re/162 §5)。
	ClanRewarded bool `json:"clanRewarded,omitempty"`

	// MazeFile 是目前在哪座迷宮:值是 original.MazeEntry.MazeFile 的十進位
	// 字串(例如 DG2MAZE.SQZ → "2"),空字串 = 在世界地圖。
	//
	// ⚠ GROUPS.DAT 有迷宮座標(位移 79/81 = original.Group.MazeX/MazeY),
	// 但**沒有「哪一座」**——原版的 ds:3534 是執行期變數(docs/re/169 §4)。
	// 這個欄位補上「哪一座」,座標本身仍然沿用 Group.MazeX/MazeY
	// (不重複存一份,那樣兩邊可能不一致)。
	MazeFile string `json:"mazeFile,omitempty"`

	// MazeFacing 是在迷宮裡的朝向。
	//
	// ⚠ **不能沿用 Group.Facing**:GROUPS.DAT 位移 41 只有一格,存的是
	// 世界地圖朝向,離開迷宮回到世界地圖時還要用它;寫進迷宮朝向會讓
	// 「下次真的走出迷宮」時面向錯的方向。原版怎麼存迷宮朝向未解,
	// 這裡另外開一格是本引擎的選擇,不是 RE 結論。
	MazeFacing int `json:"mazeFacing,omitempty"`
}

// Save 是一份完整的遊戲:25 個角色 + 5 支隊伍 + 進度。docs/spec/18 §2/§3。
//
// ⚠ **一個檔 = 一整個世界**,不是「一隊 = 一檔」——原版的 5 個隊伍槽是
// 同一個世界裡的五支隊伍,共用同一份名冊(CHARS.DAT),拆開存會把共用
// 名冊拆散,那是原版沒有的語意。多存檔要用多個檔(不同檔名的 saves/*.json)。
type Save struct {
	Version int `json:"version"`

	// Chars / Groups 直接搬 original 的型別,欄位語意見 docs/formats/01、02。
	Chars  [original.CharSlots]original.Character `json:"chars"`
	Groups [original.GroupSlots]original.Group    `json:"groups"`

	// Active 是目前玩的是第幾隊(1–5;0 = 沒有,例如剛匯入、還沒選過隊伍)。
	Active int `json:"active"`

	Progress Progress `json:"progress"`
}

// Path 組出 <dir>/<name>.json。
func Path(dir, name string) string {
	return filepath.Join(dir, name+".json")
}

// Write 把 s 寫成 JSON。目錄不存在就建。
//
// ⚠ 選 JSON 不是 gob:這是文化資產保存專案,玩家看得懂、修得動、
// 出問題查得出來(docs/spec/18 §2)。存檔很小,省不了多少 bytes。
func Write(path string, s *Save) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("建立存檔目錄:%w", err)
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("編碼存檔:%w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("寫入存檔:%w", err)
	}
	// 先寫暫存檔再 rename:半份 JSON 比舊檔還糟(讀不回來也救不了),
	// rename 在同一個檔案系統上是原子的。
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("寫入存檔:%w", err)
	}
	return nil
}

// Read 讀一份存檔。
//
// ⚠ 未知版本**拒絕載入並講清楚**,不嘗試解析(docs/spec/18 §2)——
// 半懂的存檔會安靜地弄壞進度。呼叫端要判斷「檔案不存在」(表示還沒存過,
// 不是錯誤)可以用 os.IsNotExist(err)。
func Read(path string) (*Save, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Save
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("存檔格式錯誤(%s):%w", path, err)
	}
	if s.Version != CurrentVersion {
		return nil, fmt.Errorf(
			"存檔版本 %d,這個引擎只認得版本 %d —— 拒絕載入,不嘗試解析"+
				"(docs/spec/18 §2:半懂的存檔會安靜弄壞進度)",
			s.Version, CurrentVersion)
	}
	return &s, nil
}

// List 列出 dir 底下所有 *.json 存檔(回傳不含副檔名的名字,已排序)。
//
// dir 不存在時回空清單,不是錯誤 —— 那只表示還沒存過任何東西。
func List(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".json"))
	}
	sort.Strings(names)
	return names, nil
}

// Import 讀原版的 CHARS.DAT + GROUPS.DAT,包成一份 *Save。docs/spec/18 §4。
//
// ⚠ **一次性匯入,不是執行期格式。** Progress 全部取零值 ——
// 等於「一次性事件都還沒觸發」。原版把觸發紀錄存在 DE*EFF.BIN,
// 我們不讀那個檔案,所以匯入的存檔會讓已經開過的寶箱/事件重新可觸發。
// **這件事要在呼叫端的畫面上講清楚**,這裡只負責把位元組轉成 *Save。
func Import(charsPath, groupsPath string) (*Save, error) {
	cb, err := os.ReadFile(charsPath)
	if err != nil {
		return nil, fmt.Errorf("讀 %s:%w", charsPath, err)
	}
	chars, err := original.ParseChars(cb)
	if err != nil {
		return nil, err
	}
	gb, err := os.ReadFile(groupsPath)
	if err != nil {
		return nil, fmt.Errorf("讀 %s:%w", groupsPath, err)
	}
	groups, err := original.ParseGroups(gb)
	if err != nil {
		return nil, err
	}
	s := &Save{Version: CurrentVersion}
	copy(s.Chars[:], chars)
	copy(s.Groups[:], groups)
	return s, nil
}
