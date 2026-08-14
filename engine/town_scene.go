package main

import (
	"fmt"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/layout"
	"shardofspring/internal/original"
	"shardofspring/internal/town"
	"shardofspring/internal/ui"
	"shardofspring/internal/world"
)

// 城鎮、商店與營地的畫面。docs/spec/11-town-camp-roster.md。
//
// ⚠ **清單放主視野**(61 欄),不放側欄(30 欄)——
// docs/spec/04 §5:那個「放不下」的未決項的前提是錯的。

type townMode int

const (
	townClosed townMode = iota
	townBuildings          // 建築清單
	townShop               // 商店品項
	townCamp               // 營地
)

type townState struct {
	mode  townMode
	name  string          // 城鎮名
	shops []original.Shop // 這個城鎮的商店
	shop  original.Shop   // 進去的那一間
	page  int             // 商品分頁(原版有 `+ for next page`,手冊 p.31)
	msg   string
}

// shopPageSize 是一頁的商品數。主視野高 612、行高 26 → 扣掉標題約 20 列。
const shopPageSize = 20

// enterTown 從世界地圖進城。回 false 表示這一格沒有城鎮。
//
// ⚠ 世界地圖只知道「這裡有城鎮」(值 30/31/32),**不知道是哪一個** ——
// 座標與 TOWNDATA 的對應未解。這裡按城鎮座標的排序取第 n 個,
// 是**佔位**,畫面上會標出來。
func (g *Game) enterTown(x, y int) bool {
	names := original.Towns(g.shops)
	if len(names) == 0 {
		return false
	}
	idx := g.townIndexAt(x, y)
	if idx < 0 || idx >= len(names) {
		return false
	}
	name := names[idx]
	ts := &townState{mode: townBuildings, name: name}
	for _, s := range g.shops {
		if s.Town == name {
			ts.shops = append(ts.shops, s)
		}
	}
	ts.msg = "⚠ 世界座標 → 城鎮的對應未解,這裡按座標排序取第 " +
		fmt.Sprint(idx+1) + " 個"
	g.town = ts
	return true
}

// townIndexAt 回傳這個座標是第幾個城鎮(依 y、x 排序)。
func (g *Game) townIndexAt(x, y int) int {
	type pt struct{ x, y int }
	var pts []pt
	for yy := 0; yy < world.H; yy++ {
		for xx := 0; xx < world.W; xx++ {
			if v := g.world.At(xx, yy); v >= 30 && v <= 32 {
				pts = append(pts, pt{xx, yy})
			}
		}
	}
	sort.Slice(pts, func(a, b int) bool {
		if pts[a].y != pts[b].y {
			return pts[a].y < pts[b].y
		}
		return pts[a].x < pts[b].x
	})
	for i, p := range pts {
		if p.x == x && p.y == y {
			return i
		}
	}
	return -1
}

// townKey 處理城鎮畫面的一次按鍵。
func (g *Game) townKey(k ebiten.Key) {
	ts := g.town
	switch ts.mode {
	case townBuildings:
		if i := int(k - ebiten.KeyA); i >= 0 && i < len(ts.shops) {
			ts.shop, ts.mode, ts.msg, ts.page = ts.shops[i], townShop, "", 0
		}
		// ⚠ 營地用 Z 不用 C —— 建築清單的字母是 A 起算,
		// 而 `C) Hamlet Hospital` 已經佔掉 C。同一個鍵兩個意思是
		// 「按下去做了別的事」,而畫面上看不出哪裡錯。
		if k == ebiten.KeyZ {
			ts.mode, ts.msg = townCamp, ""
		}
	case townShop:
		// 分頁:原版就是 `+ for next page` / `- for last page`(手冊 p.31)
		switch k {
		case ebiten.KeyEqual, ebiten.KeyKPAdd:
			if (ts.page+1)*shopPageSize < len(g.itemList) {
				ts.page++
			}
			return
		case ebiten.KeyMinus, ebiten.KeyKPSubtract:
			if ts.page > 0 {
				ts.page--
			}
			return
		}
		if i := int(k - ebiten.KeyA); i >= 0 && i < shopPageSize {
			if n := ts.page*shopPageSize + i; n < len(g.itemList) {
				g.buyItem(n)
			}
		}
		if k == ebiten.KeyEscape {
			ts.mode, ts.msg = townBuildings, ""
		}
	case townCamp:
		if k == ebiten.KeyR {
			g.members = town.Rest(g.members)
			g.party.Clock.Tick()
			ts.msg = "全隊休息了一會兒(恢復量未解,每次 " +
				fmt.Sprint(town.CampRestHeal) + " 點)"
		}
		if k == ebiten.KeyEscape {
			ts.mode, ts.msg = townBuildings, ""
		}
	}
}

// buyItem 讓隊伍第一位成員買第 i 件道具。
//
// ⚠ **買給誰未解** —— 原版的商店介面沒有讀到選人的步驟。
// 這裡固定給第一位,畫面上標出來。
func (g *Game) buyItem(i int) {
	if len(g.members) == 0 {
		return
	}
	it := g.itemList[i]
	price := town.Price(it.BasePrice, g.town.shop.PriceMult)
	gold := g.group.Gold
	r := town.Buy(&gold, &g.members[0], it.Index, price)
	if r != town.BuyOK {
		g.town.msg = r.String()
		return
	}
	g.group.Gold = gold
	g.town.msg = fmt.Sprintf("%s 買下 %s(%d 金幣)", g.members[0].Name, it.Name, price)
}

// drawTown 畫城鎮畫面。清單在主視野(61 欄)。
func (g *Game) drawTown(dst *ebiten.Image) {
	ts, p := g.town, g.panel
	if ts == nil || ts.mode == townClosed || p == nil {
		return
	}
	lh := p.LineHeight()
	x := float64(layout.View.X + ui.PanelPad)
	y := float64(layout.View.Y + ui.PanelPad)

	switch ts.mode {
	case townBuildings:
		p.Draw(dst, ts.name, x, y)
		y += lh * 1.5
		for i, s := range ts.shops {
			p.Draw(dst, fmt.Sprintf("%c) %s", 'A'+i, s.Name), x, y)
			y += lh
		}
		y += lh * 0.5
		p.Draw(dst, "Z) 營地", x, y)

	case townShop:
		p.Draw(dst, fmt.Sprintf("%s　價格倍率 %.2f", ts.shop.Name, ts.shop.PriceMult), x, y)
		y += lh * 1.5
		lo := ts.page * shopPageSize
		for i := 0; i < shopPageSize && lo+i < len(g.itemList); i++ {
			it := g.itemList[lo+i]
			p.Draw(dst, fmt.Sprintf("%c) %s", 'A'+i, ui.PadTo(it.Name, 20)), x, y)
			p.DrawRight(dst, fmt.Sprint(town.Price(it.BasePrice, ts.shop.PriceMult)),
				x+380, y)
			y += lh
		}
		pages := (len(g.itemList) + shopPageSize - 1) / shopPageSize
		p.Draw(dst, fmt.Sprintf("第 %d／%d 頁　+ 下一頁　- 上一頁",
			ts.page+1, pages), x, y+lh*0.5)

	case townCamp:
		p.Draw(dst, "營地", x, y)
		y += lh * 1.5
		p.Draw(dst, "R) 休息", x, y)
		y += lh * 2
		for _, u := range town.Unresolved {
			p.Draw(dst, "⚠ "+u, x, y)
			y += lh
		}
	}

	if ts.msg != "" {
		// 訊息面板 30 欄,折行不截斷(docs/spec/04 §5)
		my := float64(layout.Message.Y + ui.PanelPad)
		for _, ln := range ui.Wrap(ts.msg, 30) {
			p.Draw(dst, ln, float64(layout.Message.X+ui.PanelPad), my)
			my += lh
		}
	}
}
