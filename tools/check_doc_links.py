"""檢查(並可選修正)`docs/` 裡的相對 markdown 連結。

為什麼需要:筆記互相引用時只記得住編號,記不住檔名後綴。
手打的檔名有相當比例是**憑印象拼的**,而壞連結在 GitHub 上不會報錯 ——
點下去才發現,而那時通常是別人在點。

修正規則:連結的檔名若不存在,但**編號前綴(`^\\d+-`)在同一個目錄裡唯一命中**
一個實際存在的檔,就改成那個檔名。前綴命中 0 個或 2 個以上時**不動**,
只列出來 —— 猜不到的不要猜。

用法:
    python3 tools/check_doc_links.py            # 只檢查
    python3 tools/check_doc_links.py --fix      # 檢查並修正唯一命中的
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
LINK = re.compile(r"\]\(([^)#][^)]*\.md)\)")
NUM = re.compile(r"^(\d+)-")


def index_by_number(d):
    """目錄裡「編號 → 檔名清單」。"""
    out = {}
    if not d.is_dir():
        return out
    for f in d.iterdir():
        m = NUM.match(f.name)
        if m:
            out.setdefault(m.group(1), []).append(f.name)
    return out


def main(fix):
    ok = broken = fixed = 0
    for p in sorted(ROOT.joinpath("docs").rglob("*.md")):
        s = p.read_text(encoding="utf-8")
        orig = s
        for m in LINK.finditer(orig):
            rel = m.group(1)
            if (p.parent / rel).resolve().exists():
                ok += 1
                continue
            broken += 1
            target = pathlib.Path(rel)
            num = NUM.match(target.name)
            cands = index_by_number((p.parent / target.parent).resolve()).get(
                num.group(1), []) if num else []
            if len(cands) == 1:
                new = str(target.parent / cands[0])
                s = s.replace(f"]({rel})", f"]({new})")
                fixed += 1
                print(f"  {p.relative_to(ROOT)}: {rel} → {new}")
            else:
                print(f"⚠ {p.relative_to(ROOT)}: {rel} "
                      f"—— 編號命中 {len(cands)} 個,不猜")
        if fix and s != orig:
            p.write_text(s, encoding="utf-8")
    print(f"\n連結 {ok + broken} 條:正常 {ok}、壞 {broken}"
          + (f"、已修 {fixed}" if fix else ""))
    return 0 if broken == 0 or fix else 1


if __name__ == "__main__":
    sys.exit(main("--fix" in sys.argv))
