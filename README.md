# 🐱 NekoArc

Next-generation archive manager with self-healing FEC protection.

GUI for [Nyarc](https://github.com/Nyarime/Nyarc) — the archive format that recovers from 50% data corruption.

## Features

- 📦 **Pack** — Create .nya, .zip, .tar.gz, .rar archives
- 📂 **Extract** — Open any format (.nya, .zip, .rar, .7z, .tar, .gz, .bz2, .xz)
- 🔧 **Repair** — Recover damaged .nya archives (up to 50% corruption)
- 🔍 **Test** — Verify archive integrity with BLAKE3 checksums
- 🔒 **Encrypt** — AES-256-GCM password protection
- ⚡ **Fast** — Zstd compression, 1GB in 30 seconds

## Install

**Windows:** Download `NekoArc-installer.exe` from [Releases](https://github.com/Nyarime/NekoArc/releases)

**Linux:**
```sh
# From source
git clone https://github.com/Nyarime/NekoArc.git
cd NekoArc && wails build
```

**CLI only:** See [Nyarc](https://github.com/Nyarime/Nyarc)

## Screenshots

Dark theme with drag & drop interface.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go + [Nyarc](https://github.com/Nyarime/Nyarc) |
| Frontend | Vue 3 + Tailwind CSS |
| Framework | [Wails](https://wails.io) v2 |
| FEC | [GoFEC](https://github.com/Nyarime/GoFEC) (RaptorQ + LDPC) |
| Compression | Zstd (klauspost) |
| Hash | BLAKE3 |
| Encryption | AES-256-GCM |

## System Requirements

- **Windows:** 7 SP1+ (GUI), XP SP3 planned for CLI
- **macOS:** 10.13+
- **Linux:** GTK3 + WebKit2GTK

## License

MIT

## Related

- [Nyarc](https://github.com/Nyarime/Nyarc) — CLI archive tool
- [GoFEC](https://github.com/Nyarime/GoFEC) — FEC engine (RaptorQ + LDPC)
