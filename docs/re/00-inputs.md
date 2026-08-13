# 00 — 輸入檔清冊與雜湊

> 本專案**所有** RE 結論只對本表列出的雜湊成立。
> 引用 IDA 位址時必須同時指明是哪一支執行檔（見 CLAUDE.md §2.1 條件 4）。
>
> 建立日期:2026-08-13。信心等級:**已確認**（直接計算所得）。

## 1. 來源封裝

| 項目 | 值 |
|---|---|
| 來源 | archive.org item `msdos_Shard_of_Spring_1986` |
| 檔名 | `Shard_of_Spring_1986.zip` |
| 大小 | 199,245 bytes |
| MD5 | `994eccfa6477ab53e9f3a6799eebf2e6` |
| SHA-1 | `69c0018ed74313d8b752acca8b4bf3eb292b1763` |
| SHA-256 | `4982ea6c05134e0afd763f090e68393ba2d070624700b4c1aa7eed25631f262c` |
| 封裝方式 | TorrentZip（`TORRENTZIPPED-AEA6754A`）|
| 內容 | `sharspri/` 底下 100 檔、421,839 bytes 未壓縮 |
| 檔案時間戳 | 1996-12-24 23:32（全部一致，是重新封裝的時間，不是原始發行日）|

MD5 與 SHA-1 與 archive.org metadata 記載一致（下載當下核對）。

## 2. 執行檔（RE 對象）

位置:`game/sharspri/`（唯讀，gitignore）

| 檔案 | 大小 | SHA-256 |
|---|---:|---|
| `START.EXE` | 3,440 | `89c17ef3326e7ccba06411f67b3136dc60891e4ff1ffbddf258c13e231f4248e` |
| `MENU.EXE` | 21,008 | `e565d17c7def0a7e99502b1fb01fbf4ab0fd3d01b88b18ff5b5c8f2efe52b321` |
| `TOWN.EXE` | 18,176 | `434213ae50b7480971180a95bf0dec32cc58be11de2a88ea6d2bb2e3a4df0f5e` |
| `CMBT.EXE` | 44,704 | `f1b01861f916bdfe63e122f4ab1bfa8272f2a4f94ce01a6ebbbb51a1e5aeacac` |
| `CAMP.EXE` | 22,512 | `55dc3ca5079668663312e120ab54e85197ef7514ad8b2f5c6ff5c2e0e8378559` |
| `MAZEMOVE.EXE` | 23,168 | `b4f2ef3f58d80b0aa1f723fb581b64dec64cbcc31f19ccb7e083046730eaea23` |
| `WRLDMOVE.EXE` | 10,096 | `953363f7f0c500404477514def474b931a8173931068de716a51bf1e186e6c32` |
| `CHARUTIL.EXE` | 15,440 | `4e7c7df790ab2c7f24d3344e78b13ff479cf71e0257bc3c33e6a7b5103ee392f` |
| `MTEST.EXE` | 3,904 | `9ddd8f103919d92a9e70bdde526c4f544ae73ace3b116c46b8b7080b5d19ecd4` |
| `USERLIB.EXE` | 30,813 | `c94626ab58d3b7cbbcb27d496e2dc9cf232e71bd58cc784127731f96d752a6cb` |
| `MIO2.EXE` | 4,816 | `2c035d8916e25eaad38d48a365beb3931a50b2a06c205ca5482c0e28207b0338` |
| `WSIO.EXE` | 4,544 | `22623c9e3e396df5edd3e661dae2d52ad7669062d3c5e5c66baaf216e2febc66` |
| `BRUN30.EXE` | 70,680 | `b9ebf91c480d43093987b2e6dc6289fd59a9210fe1d8fc3c289ed7d022dffc60` |

重算方式:

```bash
cd game/sharspri && sha256sum *.EXE
```

## 3. 位址體系

- 本專案只用 **IDA database 的 linear address**。
- 引用格式:`<執行檔>:<linear address>`，例如 `TOWN.EXE:0x1A2C`。
- **不使用 Ghidra**（CLAUDE.md §3.5），因此不需要標注是哪一套位址。
- 若某處必須寫 `segment:offset`（例如引用反組譯文字），明確標示，
  並同時給出對應的 linear address。

## 4. 尚未納入本表的檔案

`game/sharspri/` 底下另有 87 個資料檔（`.DAT` / `.BIN` / `.PIC` / `.SQZ` / `.MST` / `.SOS`）。
它們的雜湊等進入對應子系統的 RE 時，在該子系統的筆記裡列出，不預先全數登錄
（避免一張沒人讀的長表）。

清單與尺寸見 CLAUDE.md §4.4 與 §4.5。
