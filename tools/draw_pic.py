"""最小的 BASIC DRAW 巨集解譯器,用來渲染 *.PIC(docs/re/49 §3)。

支援:U D L R E F G H(移動並畫線)、B 前綴(不畫)、N 前綴(畫完回原點)、
M±x,±y(相對移動)/ Mx,y(絕對)、C n(顏色)、S n(縮放,單位 1/4)、
TA n(旋轉角度)、P f,b(填充 —— 只標記起點,不做真的洪水填充)。
"""
import re, sys, math, pathlib

CH = " .:#"

class Canvas:
    def __init__(self, w=64, h=40):
        self.w, self.h = w, h
        self.px = [[0]*w for _ in range(h)]
    def put(self, x, y, c):
        xi, yi = int(round(x)), int(round(y))
        if 0 <= xi < self.w and 0 <= yi < self.h:
            self.px[yi][xi] = c
    def line(self, x0, y0, x1, y1, c):
        n = int(max(abs(x1-x0), abs(y1-y0))) + 1
        for i in range(n+1):
            t = i/max(n,1)
            self.put(x0+(x1-x0)*t, y0+(y1-y0)*t, c)
    def render(self):
        rows = ["".join(CH[v] for v in r) for r in self.px]
        while rows and not rows[0].strip(): rows.pop(0)
        while rows and not rows[-1].strip(): rows.pop()
        if not rows: return []
        lo = min((len(r)-len(r.lstrip())) for r in rows if r.strip())
        hi = max(len(r.rstrip()) for r in rows)
        return [r[lo:hi] for r in rows]

DIRS = {"U":(0,-1),"D":(0,1),"L":(-1,0),"R":(1,0),
        "E":(1,-1),"F":(1,1),"G":(-1,1),"H":(-1,-1)}

TOK = re.compile(r"(B|N)?(TA|[UDLREFGHMCSA P])\s*([+\-]?\d+)?(?:\s*,\s*([+\-]?\d+))?", re.I)

def draw(macro, cv, x=32, y=20):
    color, scale, ang = 3, 4, 0
    for m in TOK.finditer(macro):
        pre = (m.group(1) or "").upper()
        cmd = m.group(2).upper()
        a = m.group(3); b = m.group(4)
        n = int(a) if a is not None else 1
        if cmd == "C": color = n % 4; continue
        if cmd == "S": scale = max(n, 1); continue
        if cmd in ("A", "TA"): ang = n; continue
        if cmd == "P": cv.put(x, y, color); continue
        sx, sy = x, y
        if cmd == "M":
            if a is None: continue
            dx, dy = n, int(b or 0)
            if a.lstrip()[0] in "+-": nx, ny = x+dx, y+dy
            else: nx, ny = dx, dy
        elif cmd in DIRS:
            ux, uy = DIRS[cmd]
            d = n * scale / 4.0
            r = math.radians(ang)
            rx = ux*math.cos(r) - uy*math.sin(r)
            ry = ux*math.sin(r) + uy*math.cos(r)
            nx, ny = x + rx*d, y + ry*d
        else:
            continue
        if pre != "B":
            cv.line(sx, sy, nx, ny, color)
        if pre != "N":
            x, y = nx, ny
    return x, y

if __name__ == "__main__":
    for p in sys.argv[1:]:
        t = pathlib.Path(p).read_bytes().rstrip(b"\x1a").decode("latin1")
        segs = [l for l in t.split("\r\n") if l.strip()]
        print(f"===== {pathlib.Path(p).name}: {len(segs)} 段 =====")
        for i, s in enumerate(segs):
            cv = Canvas()
            try: draw(s, cv)
            except Exception as e: print(f"  [{i}] 解譯失敗 {e}"); continue
            rows = cv.render()
            print(f"  --- 段 {i} ({len(s)} 字元) ---")
            for r in rows[:22]: print("     |"+r+"|")
