#!/usr/bin/env python3
"""從已保存的手繪母稿建立可重生的 68×68 現代主題資產。

只在 Docker 的 Pillow image 內執行。未知地形不猜語意：以水彩底材承接
原版圖塊的非黑輪廓，讓索引與秘密資訊保持原樣。
"""

from pathlib import Path
import argparse
from PIL import Image, ImageOps

TILE = 68


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
