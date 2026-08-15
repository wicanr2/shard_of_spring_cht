#!/usr/bin/env python3
"""世界地圖路徑規劃 → DOSBox oracle 的按鍵時間軸。

用法:
    python3 tools/world_route.py --to 41,16
    python3 tools/world_route.py --from 8,8 --facing S --to 41,16 --shot w1

輸出兩行:第一行是人看的摘要,第二行是可直接餵給 tools/dosbox_run.sh 的 timeline。

座標與可通行性依 docs/spec/05-world-scene.md §1/§3(軸向訂正見 docs/re/141)。
⚠ 「先轉再走」:方向鍵按下去,面向不同就只轉身,面向相同才前進(docs/re/139 §4),
   而**轉身也花一格時間**(docs/re/149)—— 所以按鍵數 = 動作數,不是步數。
"""
import argparse
from collections import deque

W, H = 121, 103          # 東西 × 南北
MIN_Y, MAX_Y = 5, 98     # 南北軸的硬界(規則 1)
N, E, S, Wd = 1, 2, 3, 4
DELTA = {N: (0, -1), E: (1, 0), S: (0, 1), Wd: (-1, 0)}
KEYSYM = {N: 'Up', E: 'Right', S: 'Down', Wd: 'Left'}
NAME = {'N': N, 'E': E, 'S': S, 'W': Wd}
# 這幾種地形會離開世界地圖(城鎮 / 地城入口),路徑預設繞開
LEAVE = {24, 25, 27, 28, 30, 31, 32}


def load(path='game/sharspri/WRLDMAP.BIN'):
    d = open(path, 'rb').read()[7:-1]
    return [d[i] for i in range(0, len(d), 2)]


def tile(c, x, y):
    return c[x * H + y] if 0 <= x < W and 0 <= y < H else 0


def passable(c, fx, fy, tx, ty, dr):
    """docs/spec/05 §3 的八條規則。"""
    if not (MIN_Y <= ty <= MAX_Y) or not (0 <= tx < W):
        return False
    cur, dst = tile(c, fx, fy), tile(c, tx, ty)
    if 10 <= dst <= 12:
        return False
    if dst in (20, 21) and not (1 <= cur <= 4):
        return False
    if cur in (20, 21) and 15 <= dst <= 18:
        return False
    block = {N: ((15, 16), (17, 18)), E: ((15, 18), (16, 17)),
             S: ((17, 18), (15, 16)), Wd: ((16, 17), (15, 18))}
    a, b = block[dr]
    return not (cur in a and dst in b)


def route(c, start, goal, avoid_leave=True):
    prev = {start: None}
    q = deque([start])
    while q:
        x, y = q.popleft()
        if (x, y) == goal:
            break
        for dr, (dx, dy) in DELTA.items():
            n = (x + dx, y + dy)
            if n in prev or not passable(c, x, y, n[0], n[1], dr):
                continue
            if avoid_leave and n != goal and tile(c, *n) in LEAVE:
                continue
            prev[n] = ((x, y), dr)
            q.append(n)
    if goal not in prev:
        return None, None
    dirs, cells, cur = [], [], goal
    while prev[cur]:
        p, dr = prev[cur]
        dirs.append(dr)
        cells.append(cur)
        cur = p
    return dirs[::-1], cells[::-1]


def keys(dirs, facing):
    """先轉再走:面向不同先送一次同鍵轉身,再送一次前進。"""
    out = []
    for dr in dirs:
        if facing != dr:
            out.append(KEYSYM[dr])
            facing = dr
        out.append(KEYSYM[dr])
    return out, facing


BOOT = 'wait:8;key:Return;wait:3;key:Return;wait:3;type:L;wait:4;type:5;wait:6'


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--from', dest='src', default='8,8')
    ap.add_argument('--facing', default='S')
    ap.add_argument('--to', required=True)
    ap.add_argument('--step-wait', type=float, default=2)
    ap.add_argument('--shot', default='')
    ap.add_argument('--save', action='store_true', help='走完按 S 存檔')
    ap.add_argument('--no-boot', action='store_true')
    a = ap.parse_args()

    c = load()
    src = tuple(int(v) for v in a.src.split(','))
    dst = tuple(int(v) for v in a.to.split(','))
    dirs, cells = route(c, src, dst)
    if dirs is None:
        raise SystemExit(f'走不到 {dst}')
    ks, facing = keys(dirs, NAME[a.facing.upper()])
    turns = len(ks) - len(dirs)
    tiles = [tile(c, *p) for p in cells]
    hills = sum(1 for t in tiles if t in (7, 8, 9))
    actions = len(ks) + (1 if a.save else 0)
    print(f'# {src} → {dst}  步數 {len(dirs)} 轉身 {turns} 按鍵 {len(ks)} '
          f'動作 {actions}(含存檔)  沿途山地格 {hills}  終點面向 {facing}')
    print(f'# 沿途圖塊 {tiles}')

    tl = [] if a.no_boot else [BOOT]
    tl += [f'key:{k};wait:{a.step_wait:g}' for k in ks]
    if a.save:
        tl.append('type:S;wait:3')
    if a.shot:
        tl.append(f'shot:{a.shot}')
    print(';'.join(tl))


if __name__ == '__main__':
    main()
