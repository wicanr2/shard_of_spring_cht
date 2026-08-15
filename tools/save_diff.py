#!/usr/bin/env python3
"""把 oracle 寫出的存檔與出貨檔逐位元組比對。

用法:
    python3 tools/save_diff.py                 # 兩個檔都比,只列有差的欄位
    python3 tools/save_diff.py --dump groups   # 印出第 5 筆的全部欄位

`GROUPS.DAT` 450 B = 5 筆 × 90 B(docs/re/135);
`CHARS.DAT`  = 32 筆 × 94 B(docs/formats/01)。

⚠ 位移一律用**1-based**,和 docs/re/135 的表一致 —— 兩套編號混用過一次。
⚠ 值 8224(`0x2020`)= 這個欄位從來沒被寫過,不是一個數值。
"""
import argparse
import os

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
FILES = {
    'groups': ('GROUPS.DAT', 90),
    'chars': ('CHARS.DAT', 94),
}
# 已知語意,只用來讓輸出好讀(來源 docs/re/135、docs/re/149)
GROUPS_FIELDS = {
    1: '成員1', 3: '成員2', 5: '成員3', 7: '成員4', 9: '成員5',
    19: '金幣(MBF 4B)', 23: '補給', 25: '遭遇倒數',
    27: '時鐘.月', 29: '時鐘.日', 31: '時鐘.時', 33: '時鐘.時以下',
    35: '座標.東西', 37: '座標.南北', 41: '朝向', 45: '光源回合',
    59: '能見度(有光)', 61: '能見度(無光)', 83: '光源道具',
}
CHARS_FIELDS = {
    2: '狀態(0=有角色 *=空)', 3: '姓名(10B)', 13: '種族', 15: '職業',
    28: 'HP', 32: 'SP', 55: '背包(10 槽)', 75: '旗標2(10B)', 90: '剩餘技能點',
}


def word(rec, off1):
    """1-based 位移的 16-bit little-endian。"""
    i = off1 - 1
    return rec[i] | (rec[i + 1] << 8)


def records(path, size):
    d = open(path, 'rb').read()
    return [d[i:i + size] for i in range(0, len(d), size)]


def show(kind, rec_no, a, b, fields, only_diff=True):
    size = len(a)
    hits = []
    for off1 in range(1, size, 2):
        va, vb = word(a, off1), word(b, off1)
        if only_diff and va == vb:
            continue
        hits.append((off1, va, vb))
    if not hits:
        return
    print(f'--- {kind} 第 {rec_no} 筆 ---')
    for off1, va, vb in hits:
        tag = fields.get(off1, '')
        mark = '' if va == vb else '   ←'
        print(f'位移 {off1:>3}  {va:>6} → {vb:<6} {tag}{mark}')


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--dump', choices=list(FILES) + ['all'], default=None)
    ap.add_argument('--rec', type=int, default=0, help='只看某一筆(1-based)')
    ap.add_argument('--orig', default=os.path.join(ROOT, 'game/sharspri'))
    ap.add_argument('--work', default=os.path.join(ROOT, 'workplace/dosbox/game'))
    a = ap.parse_args()

    for kind, (name, size) in FILES.items():
        if a.dump and a.dump != 'all' and a.dump != kind:
            continue
        po, pw = os.path.join(a.orig, name), os.path.join(a.work, name)
        if not os.path.exists(pw):
            print(f'{name}:工作副本不存在'); continue
        ro, rw = records(po, size), records(pw, size)
        fields = GROUPS_FIELDS if kind == 'groups' else CHARS_FIELDS
        for i, (x, y) in enumerate(zip(ro, rw), 1):
            if a.rec and i != a.rec:
                continue
            if a.dump:
                show(name, i, x, y, fields, only_diff=False)
            elif x != y:
                show(name, i, x, y, fields, only_diff=True)


if __name__ == '__main__':
    main()
