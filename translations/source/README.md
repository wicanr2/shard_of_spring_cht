# String Extraction Report

## Summary

- dungeon-text.tsv: 87 entries, 11048 bytes
- titles.tsv: 113 entries, 1230 bytes
- items.tsv: 114 entries, 1075 bytes
- spells.tsv: 68 entries, 848 bytes
- monsters.tsv: 74 entries, 728 bytes

**Total: 456 entries, 14929 bytes**

## Notes

- SPELLS.DAT row 33 (`ET CETERA`) is a sentinel marker (last row always included in extraction)
- All monster names are properly null/space-trimmed

## Anomalies Found

- TITLES.DAT row 63: Missing closing quote: `"done.` (original file format issue)
- TITLES.DAT row 112: EOF marker (0x1A) attached (not a quote issue)
