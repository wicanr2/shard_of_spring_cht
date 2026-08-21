#!/usr/bin/env python3
"""從已保存的手繪母稿建立可重生的 68×68 現代主題資產。

只在 Docker 的 Pillow image 內執行。未知地形不猜語意：以水彩底材承接
原版圖塊的非黑輪廓，讓索引與秘密資訊保持原樣。
"""

from pathlib import Path
import argparse
import math
from PIL import Image, ImageChops, ImageOps

TILE = 68
ATLAS = TILE * 4


def seamless(im: Image.Image, band: int = 18) -> Image.Image:
    """建立無鏡射對稱的 4×4 可平鋪材質頁。"""
    out = ImageOps.fit(im.convert("RGBA"), (ATLAS, ATLAS),
                       method=Image.Resampling.LANCZOS, centering=(0.5, 0.5))
    # 把原本的外緣移到中央，再只混合中央接縫；新外緣來自原圖中央，
    # 因此週期銜接，且不會產生鏡像萬花筒。
    out = ImageChops.offset(out, ATLAS // 2, ATLAS // 2)
    mid = ATLAS // 2
    src = out.copy()
    px, sp = out.load(), src.load()
    for y in range(ATLAS):
        for i, x in enumerate(range(mid - band, mid + band)):
            # 交叉淡化兩側的完整像素帶，不能把兩個端點拉成 36 px 色帶。
            px[x, y] = blend(sp[x + band, y], sp[x - band, y], i / (2 * band - 1))
    src = out.copy()
    sp = src.load()
    for x in range(ATLAS):
        for i, y in enumerate(range(mid - band, mid + band)):
            px[x, y] = blend(sp[x, y + band], sp[x, y - band], i / (2 * band - 1))
    return out


def texture_panel(sheet: Image.Image, col: int) -> Image.Image:
    """三欄母板各取中央正方形，避開欄界與上下可能的生成邊緣。"""
    w = sheet.width // 3
    x0, x1 = col * w, (col + 1) * w
    side = min(w, sheet.height)
    y0 = (sheet.height - side) // 2
    return sheet.crop((x0, y0, x1, y0 + side))


def blend(a: tuple[int, ...], b: tuple[int, ...], t: float) -> tuple[int, ...]:
    return tuple(round(a[i] * (1 - t) + b[i] * t) for i in range(4))


def edge_distance(x: int, y: int, mask: int) -> int:
    """回傳到未連接邊的距離；bit N/E/S/W = 1/2/4/8。"""
    ds = []
    if not mask & 1:
        ds.append(y)
    if not mask & 2:
        ds.append(TILE - 1 - x)
    if not mask & 4:
        ds.append(TILE - 1 - y)
    if not mask & 8:
        ds.append(x)
    return min(ds, default=TILE)


def build_world_autotiles(sheet: Image.Image, out: Path) -> None:
    """建立現代主題專用的 4 向 16-mask 接邊資產。"""
    grass = seamless(texture_panel(sheet, 0))
    grass_alt = ImageChops.offset(grass, TILE, TILE * 2)
    forest = seamless(texture_panel(sheet, 1))
    ocean = seamless(texture_panel(sheet, 2))
    ocean_alt = ocean.copy()
    auto = out / "world_auto"
    save(grass, auto / "grass0.png")
    save(grass_alt, auto / "grass1.png")
    save(ocean, auto / "ocean0.png")
    save(ocean_alt, auto / "ocean1.png")

    gp, fp, op = grass.load(), forest.load(), ocean.load()
    sand = (190, 154, 91, 255)
    for mask in range(16):
        ft = Image.new("RGBA", (ATLAS, ATLAS))
        ct = Image.new("RGBA", (ATLAS, ATLAS))
        ftp, ctp = ft.load(), ct.load()
        for y in range(ATLAS):
            for x in range(ATLAS):
                lx, ly = x % TILE, y % TILE
                # 森林中心保留樹冠；沒有森林鄰格的邊緣在 14 px 內融入草地。
                d = edge_distance(lx, ly, mask)
                forest_a = min(1.0, max(0.0, d / 14.0))
                ftp[x, y] = blend(gp[x, y], fp[x, y], forest_a)

                # 海岸 mask 的 bit 表示該方向是海洋。水由邊緣進入 12 px，
                # 只留 4 px 清楚沙岸；共用邊保持純海水以無縫銜接。
                depths = []
                if mask & 1:
                    depths.append(ly)
                if mask & 2:
                    depths.append(TILE - 1 - lx)
                if mask & 4:
                    depths.append(TILE - 1 - ly)
                if mask & 8:
                    depths.append(lx)
                # 固定相位的細微起伏只改岸線內緣；格邊前 7 px 仍是純海水，
                # 因此相鄰海洋格保持無縫，卻不會形成僵硬的直角方框。
                wobble = 2.2 * math.sin((x + y * 0.37) * math.pi / 17.0)
                shore_d = min(depths, default=TILE) - wobble
                if shore_d < 10:
                    ctp[x, y] = op[x, y]
                elif shore_d < 13:
                    ctp[x, y] = blend(op[x, y], sand, (shore_d - 10) / 3.0)
                elif shore_d < 17:
                    ctp[x, y] = blend(sand, gp[x, y], (shore_d - 13) / 4.0)
                else:
                    ctp[x, y] = gp[x, y]
        save(ft, auto / "forest" / f"m{mask:02d}.png")
        save(ct, auto / "coast" / f"m{mask:02d}.png")


def cell(im: Image.Image, col: int, row: int, cols=6, rows=4) -> Image.Image:
    x0, x1 = round(im.width * col / cols), round(im.width * (col + 1) / cols)
    y0, y1 = round(im.height * row / rows), round(im.height * (row + 1) / rows)
    # 去掉生成母板的白色分隔線。
    pad = max(2, min(x1 - x0, y1 - y0) // 90)
    return im.crop((x0 + pad, y0 + pad, x1 - pad, y1 - pad)).resize(
        (TILE, TILE), Image.Resampling.LANCZOS
    ).convert("RGBA")


def raw_cell(im: Image.Image, col: int, row: int, cols: int, rows: int) -> Image.Image:
    """切出母稿格但不先壓成正方形，避免人物被縱向壓扁。"""
    x0, x1 = round(im.width * col / cols), round(im.width * (col + 1) / cols)
    y0, y1 = round(im.height * row / rows), round(im.height * (row + 1) / rows)
    return im.crop((x0, y0, x1, y1)).convert("RGBA")


def original_ink(base: Image.Image, original: Path) -> Image.Image:
    if not original.exists():
        return base
    src = Image.open(original).convert("RGB").resize((TILE, TILE), Image.Resampling.NEAREST)
    # 原版黑底不畫；所有非黑像素化成半透明褐色墨線。輪廓保留原始定位，
    # 顏色不沿用 CGA，避免在水彩底上突兀。
    lum = ImageOps.grayscale(src)
    mask = lum.point(lambda p: 0 if p < 20 else min(180, 55 + p // 2))
    ink = Image.new("RGBA", (TILE, TILE), (45, 28, 16, 0))
    ink.putalpha(mask)
    return Image.alpha_composite(base, ink)


def save(im: Image.Image, path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    im.save(path, optimize=True)


def build_world(root: Path, sheet: Image.Image, out: Path) -> None:
    known = {
        1: (0, 0), 2: (1, 0), 3: (2, 0), 4: (3, 0),
        5: (4, 0), 6: (5, 0), 7: (4, 2), 8: (5, 2), 9: (0, 3),
        10: (2, 3), 11: (1, 2), 12: (1, 3), 13: (3, 2),
        20: (3, 3), 21: (5, 3),
    }
    fallback = cell(sheet, 0, 0)
    for n in range(1, 39):
        if n in (14, 19, 22, 33, 34):  # 原始資料沒有這些值。
            continue
        if n in known:
            im = cell(sheet, *known[n])
        elif n in (15, 16, 17, 18, 35, 36, 37, 38):
            turns = {15: 0, 16: 1, 17: 2, 18: 3, 35: 0, 36: 1, 37: 2, 38: 3}[n]
            im = cell(sheet, 2, 2).rotate(90 * turns, resample=Image.Resampling.BICUBIC)
        else:
            im = fallback.copy()
        # 已知與未知都疊回原版輪廓；已知圖的 alpha 較低但仍可回查。
        im = original_ink(im, root / "assets" / "gfx" / "world" / f"t{n:02d}.png")
        save(im, out / "world" / f"t{n:02d}.png")
    natural = Image.open(root / "art" / "modern" / "candidates" / "world-natural-textures-v2.png")
    build_world_autotiles(natural, out)


def build_walk(sheet: Image.Image, out: Path) -> None:
    # 母板 8 欄依序 N,NE,E,SE,S,SW,W,NW；引擎只使用 N/E/S/W。
    cols = [0, 2, 4, 6]
    for facing, col in enumerate(cols):
        for gait in range(2):
            # 母稿每格是高矩形；先壓成 68×68 會把人物變矮、邊緣發糊。
            # 保留原比例切背景，再縮進正方形 sprite。
            im = fit_sprite(remove_dark_background(raw_cell(sheet, col, gait, 8, 2)), 64)
            seg = facing * 2 + 1 + gait
            save(im, out / "walk" / f"w{seg}.png")


def contact(paths: list[Path], cols: int, cell_px: int, out: Path, first_frame=False) -> None:
    rows = (len(paths) + cols - 1) // cols
    dst = Image.new("RGBA", (cols * cell_px, rows * cell_px), (238, 226, 195, 255))
    for i, path in enumerate(paths):
        im = Image.open(path).convert("RGBA")
        if first_frame and im.width > im.height:
            im = im.crop((0, 0, im.height, im.height))
        im = im.resize((cell_px - 8, cell_px - 8), Image.Resampling.NEAREST)
        dst.alpha_composite(im, ((i % cols) * cell_px + 4, (i // cols) * cell_px + 4))
    save(dst, out)


def build_references(root: Path) -> None:
    ref = root / "art" / "modern" / "references"
    contact(
        [root / "assets" / "gfx" / "monst" / f"monst{n:02d}.png" for n in range(1, 23)],
        11, 96, ref / "monsters-indexed.png", first_frame=True,
    )
    maze = sorted((root / "assets" / "gfx" / "maze").glob("t*.png"))
    contact(maze, 6, 96, ref / "maze-indexed.png")


def remove_light_background(im: Image.Image) -> Image.Image:
    rgba = im.convert("RGBA")
    px = rgba.load()
    width, height = rgba.size

    # 母稿把透明棋盤烘進圖片。單純依顏色全圖刪除會把骷髏、刀刃與風元素的
    # 淺色細節一起挖空；改由四邊開始，只刪除與外圍相連的淺灰背景。
    def is_background(x: int, y: int) -> bool:
        r, g, b, _ = px[x, y]
        return min(r, g, b) > 185 and max(r, g, b) - min(r, g, b) < 28

    pending = []
    seen = set()
    for x in range(width):
        pending.extend(((x, 0), (x, height - 1)))
    for y in range(height):
        pending.extend(((0, y), (width - 1, y)))
    while pending:
        x, y = pending.pop()
        if (x, y) in seen or not is_background(x, y):
            continue
        seen.add((x, y))
        if x:
            pending.append((x - 1, y))
        if x + 1 < width:
            pending.append((x + 1, y))
        if y:
            pending.append((x, y - 1))
        if y + 1 < height:
            pending.append((x, y + 1))

    for x, y in seen:
        r, g, b, _ = px[x, y]
        px[x, y] = (r, g, b, 0)
    return rgba


def remove_dark_background(im: Image.Image) -> Image.Image:
    rgba = im.convert("RGBA")
    px = rgba.load()
    for y in range(rgba.height):
        for x in range(rgba.width):
            r, g, b, a = px[x, y]
            if max(r, g, b) < 18:
                a = 0
            px[x, y] = (r, g, b, a)
    return rgba


def fit_sprite(im: Image.Image, extent: int = 64) -> Image.Image:
    alpha = im.getchannel("A")
    box = alpha.getbbox()
    if not box:
        return Image.new("RGBA", (TILE, TILE), (0, 0, 0, 0))
    im = im.crop(box)
    im.thumbnail((extent, extent), Image.Resampling.LANCZOS)
    dst = Image.new("RGBA", (TILE, TILE), (0, 0, 0, 0))
    dst.alpha_composite(im, ((TILE - im.width) // 2, TILE - im.height - 3))
    return dst


def split_foreground_row(sheet: Image.Image, row: int, cols: int, rows: int) -> list[Image.Image]:
    """以角色之間的透明谷線切 atlas，不假設生成母稿真的等寬對齊。"""
    band = remove_light_background(raw_cell(sheet, 0, row, 1, rows))
    alpha = band.getchannel("A")
    occupied = [any(alpha.getpixel((x, y)) for y in range(alpha.height))
                for x in range(alpha.width)]
    runs = []
    start = None
    for x, used in enumerate(occupied + [True]):
        if not used and start is None:
            start = x
        elif used and start is not None:
            if x - start >= 4:
                runs.append((start, x - 1))
            start = None

    bounds = [0]
    radius = band.width / cols * 0.48
    for k in range(1, cols):
        expected = band.width * k / cols
        candidates = [r for r in runs
                      if abs(((r[0] + r[1]) / 2) - expected) <= radius]
        if not candidates:
            raise ValueError(f"第 {row + 1} 列找不到第 {k} 條怪物分隔谷線")
        gap = min(candidates,
                  key=lambda r: abs(((r[0] + r[1]) / 2) - expected))
        bounds.append((gap[0] + gap[1]) // 2)
    bounds.append(band.width)
    if bounds != sorted(bounds) or len(set(bounds)) != len(bounds):
        raise ValueError(f"第 {row + 1} 列怪物分隔谷線順序錯誤:{bounds}")
    return [band.crop((bounds[i], 0, bounds[i + 1], band.height))
            for i in range(cols)]


def build_monsters(sheet: Image.Image, out: Path) -> None:
    sprites = split_foreground_row(sheet, 0, 11, 2) + split_foreground_row(sheet, 1, 11, 2)
    for i, im in enumerate(sprites):
        # 生成母稿的角色中心沒有精準落在等寬 11 欄；先以透明谷線逐隻切開，
        # 再依 alpha 邊界等比例放進 68×68，避免切入鄰居或截掉武器／尾巴。
        save(fit_sprite(im, 66), out / "monst" / f"monst{i+1:02d}.png")


def build_maze(root: Path, sheet: Image.Image, out: Path) -> None:
    names = sorted(p.name for p in (root / "assets" / "gfx" / "maze").glob("t*.png"))
    for i, name in enumerate(names):
        save(cell(sheet, i % 6, i // 6, cols=6, rows=5), out / "maze" / name)


def build_combat(root: Path, out: Path) -> None:
    world = out / "world"
    bases = {0: 9, 1: 3, 2: 11, 3: 10, 4: 7, 5: 12, 6: 11, 7: 10, 8: 13}
    for slot, tile in bases.items():
        im = Image.open(world / f"t{tile:02d}.png").convert("RGBA")
        im = original_ink(im, root / "assets" / "gfx" / "combat" / f"c{slot}.png")
        save(im, out / "combat" / f"c{slot}.png")


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", default="/workspace")
    args = ap.parse_args()
    root = Path(args.root)
    candidates = root / "art" / "modern" / "candidates"
    out = root / "engine" / "assets" / "modern"
    build_world(root, Image.open(candidates / "world-tiles-book-v1.png"), out)
    build_walk(Image.open(candidates / "party-walk-book-v1.png").convert("RGBA"), out)
    tent = fit_sprite(remove_dark_background(Image.open(candidates / "tent-book-v1.png")))
    save(tent, out / "walk" / "w0.png")
    build_monsters(Image.open(candidates / "monsters-book-v1.png"), out)
    build_maze(root, Image.open(candidates / "maze-tiles-book-v1.png"), out)
    build_combat(root, out)
    build_references(root)


if __name__ == "__main__":
    main()
