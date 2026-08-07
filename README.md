<div align="center">
  <img src="assets/go-genres-icon.png" alt="go-genres icon">


  <h1>GO·GENRES</h1>

  <p><strong>Fetch, normalize, and write MP3 genre tags from multiple metadata sources.</strong></p>

  <p>
    <a href="https://go.dev/"><img src="https://img.shields.io/badge/go-1.26+-01acd7?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.26+"></a>
    <a href="https://www.last.fm/user/deathrashed"><img src="https://img.shields.io/badge/Last.fm-✓-01acd7?style=for-the-badge&logo=lastdotfm&logoColor=white" alt="Last.fm"></a>
    <a href="https://www.metal-archives.com/"><img src="https://img.shields.io/badge/Metallum-✓-01acd7?style=for-the-badge&logo=data:image/svg%2bxml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxNiIgaGVpZ2h0PSIxNiIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJub25lIiBzdHJva2U9IndoaXRlIiBzdHJva2Utd2lkdGg9IjIiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIgc3Ryb2tlLWxpbmVqb2luPSJyb3VuZCI+PHBvbHlsaW5lIHBvaW50cz0iMTIgMiAyIDcgMiAxNyAxMiAyMiAyMiAxNyAyMiA3IDEyIDIiLz48bGluZSB4MT0iMTIiIHkxPSIyMiIgeDI9IjEyIiB5Mj0iNyIvPjxwb2x5bGluZSBwb2ludHM9IjkgMTIgMTIgMTUgMTUgMTIiLz48L3N2Zz4=&logoColor=white" alt="Metallum"></a>
    <a href="https://open.spotify.com/"><img src="https://img.shields.io/badge/Spotify-✓-01acd7?style=for-the-badge&logo=spotify&logoColor=white" alt="Spotify"></a>
    <a href="https://www.discogs.com/"><img src="https://img.shields.io/badge/Discogs-✓-01acd7?style=for-the-badge&logo=discogs&logoColor=white" alt="Discogs"></a>
    <a href="https://github.com/deathrashed/go-genres"><img src="https://img.shields.io/badge/version-1-01acd7?style=for-the-badge" alt="Version 1"></a>
  </p>


  <p>
    <a href="#quick-start">Quick Start</a> |
    <a href="#usage">Usage</a> |
    <a href="#providers">Providers</a> |
    <a href="#project-structure">Structure</a> |
    <a href="#building">Building</a>
  </p>
</div>

---

## <img src="https://api.iconify.design/mdi:rocket-launch-outline.svg?color=%2301acd7" height="22"> Quick Start

```bash
git clone https://github.com/deathrashed/go-genres.git
cd go-genres

# Build everything
./menu/build.sh
./discogs/build.sh
./spotify/build.sh
./lastfm/build.sh
./metallum/build.sh

# Launch the TUI
./menu/bin/genres
```

## <img src="https://api.iconify.design/mdi:play-box-multiple-outline.svg?color=%2301acd7" height="22"> Usage

Launch the interactive TUI menu:

```bash
./menu/bin/genres
```

Or run a provider directly:

```bash
./discogs/bin/discogs-genres /path/to/music
./spotify/bin/spotify-genres /path/to/music
./lastfm/bin/lastfm-genres /path/to/music
./metallum/bin/metallum-genres /path/to/music
```

### Flags

Each provider supports:

| Flag | Description |
| --- | --- |
| `--write` | Actually update MP3 tags (default: dry-run) |
| `--undo` | Restore the most recent tag write session |

### Global Settings

Settings are stored in `~/.config/genres/global-settings.json` and shared across all providers. Configure them from any provider's `Settings` menu.

| Setting | Description |
| --- | --- |
| **Auto Backup** | Enable/disable automatic file backups before writing tags |
| **Prompt Before Write** | Ask for confirmation before saving a backup |
| **Clear All Backups** | Remove all stored backup files |
| **Downloads Path** | Configurable path for the Downloads quick-action (default: `/Volumes/Eksternal/Music/Downloads`) |
| **Swinsian Integration** | Toggle macOS Swinsian player integration |

### 🍎 Swinsian Integration

When enabled in Settings, two extra input options appear:

- **Playing** — Tags the entire album folder of the currently playing track in Swinsian
- **Selected** — Tags each selected track's album folder

### Undo & Backups

Every tag write creates a timestamped backup in `~/.config/genres/undo/`. Use `u` from the main menu to restore the latest session or manage old backups. Backups can be disabled or set to prompt before creation in Settings.

## <img src="https://api.iconify.design/mdi:server.svg?color=%2301acd7" height="22"> Providers

| Source | Description | Binary |
| --- | --- | --- |
| <img src="https://api.iconify.design/mdi:lastfm.svg?color=%2301acd7" height="16"> **Last.fm** | Fetches genre tags from Last.fm artist tags | `lastfm/bin/lastfm-genres` |
| <img src="https://api.iconify.design/mdi:axe.svg?color=%2301acd7" height="16"> **Metallum** | Scrapes Metal Archives for genre info | `metallum/bin/metallum-genres` |
| <img src="https://api.iconify.design/mdi:spotify.svg?color=%2301acd7" height="16"> **Spotify** | Pulls genre data from the Spotify API | `spotify/bin/spotify-genres` |
| <img src="data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI+PHBhdGggZmlsbD0iIzAxYWNkNyIgZD0iTTEuNzQyIDExLjk4MmMwLTUuNjY4IDQuNjEtMTAuMjc4IDEwLjI3Ni0xMC4yNzggMS44MjQgMCAzLjUzNy40OCA1LjAyNSAxLjMxN2wuODE0LTEuNDg4QTExLjkgMTEuOSAwIDAgMCAxMi4xOS4wMDNoLS4xOTVDNS40MS4wMTMuMDcyIDUuMzEgMCAxMS44ODVWMTJhMTEuOTggMTEuOTggMCAwIDAgMy43NzUgOC43MmwxLjE4NS0xLjI4YTEwLjI1IDEwLjI1IDAgMCAxLTMuMjE4LTcuNDU5em0xOC42Mi04LjU3N2wtMS4xNTQgMS4yNGExMC4yNSAxMC4yNSAwIDAgMSAzLjA4OCA3LjMzN2MwIDUuNjY2LTQuNjEgMTAuMjc2LTEwLjI3NiAxMC4yNzYtMS43ODMgMC0zLjQ2LS40NTYtNC45MjItMS4yNThsLS44NTQgMS41MjJBMTEuOTUgMTEuOTUgMCAwIDAgMTIgMjMuOTk4YzYuNjI2IDAgMTIuMDAxLTUuMzczIDEyLjAwMS0xMmExMS45OCAxMS45OCAwIDAgMC0zLjYzOC04LjU5M20tMTguNCA4LjU3N2ExMC4wMyAxMC4wMyAwIDAgMCAzLjE0NiA3LjI5NWwxLjE4LTEuMjc2YTguMyA4LjMgMCAwIDEtMi41ODYtNi4wMTljMC00LjU4NiAzLjczLTguMzE1IDguMzE1LTguMzE1YzEuNDgzIDAgMi44NzUuMzkxIDQuMDgyIDEuMDc1bC44MzUtMS41MjZhMTAgMTAgMCAwIDAtNC45MTctMS4yODlDNi40NzUgMS45MjUgMS45NjMgNi40MzcgMS45NjMgMTEuOTgybTE4LjM3IDBjMCA0LjU4Ni0zLjczIDguMzE1LTguMzE1IDguMzE1YTguMyA4LjMgMCAwIDEtMy45NjItMS4wMDVsLS44NTIgMS41MTZhMTAgMTAgMCAwIDAgNC44MTQgMS4yMjljNS41NDMgMCAxMC4wNTUtNC41MTIgMTAuMDU1LTEwLjA1NWMwLTIuODA4LTEuMTU3LTUuMzQ3LTMuMDE2LTcuMTczbC0xLjE4MyAxLjI3NGE4LjI4IDguMjggMCAwIDEgMi40NiA1Ljg5OW0tMS45NDggMGE2LjM3IDYuMzcgMCAwIDEtNi4zNjUgNi4zNjQgNi4zIDYuMyAwIDAgMS0zLjAwNi0uNzU2bC0uODQ4IDEuNTA3YTguMDQgOC4wNCAwIDAgMCAzLjg1NC45NzdjNC40NjQgMCA4LjA5NS0zLjYzIDguMDk1LTguMDk0YTguMDcgOC4wNyAwIDAgMC0yLjM5LTUuNzM4bC0xLjE3OSAxLjI2N2E2LjM2IDYuMzYgMCAwIDEgMS44MzkgNC40NzNtLTE0LjQ1OSAwYzAgMi4zMDEuOTY3IDQuMzgyIDIuNTE1IDUuODU4bDEuMTczLTEuMjdhNi4zNCA2LjM0IDAgMCAxLTEuOTYtNC41ODggNi4zNyA2LjM3IDAgMCAxIDYuMzY0LTYuMzY0IDYuMyA2LjMgMCAwIDEgMy4xNDQuODM1bC44My0xLjUxN2E4LjA2IDguMDYgMCAwIDAtMy45NzQtMS4wNDhjLTQuNDYxIDAtOC4wOTIgMy42My04LjA5MiA4LjA5NG0xMi41MyAwYTQuNDQgNC40NCAwIDAgMS00LjQzOCA0LjQzNyA0LjQgNC40IDAgMCAxLTIuMDYxLS41MDlsLS44MzUgMS40ODhhNi4xIDYuMSAwIDAgMCAyLjg5Ni43MjcgNi4xNSA2LjE1IDAgMCAwIDYuMTQzLTYuMTQzIDYuMTIgNi4xMiAwIDAgMC0xLjc2OC00LjMwOGwtMS4xNjIgMS4yNWE0LjQzIDQuNDMgMCAwIDEgMS4yMjQgMy4wNThtLTEwLjU4MSAwYTYuMTIgNi4xMiAwIDAgMCAxLjg4OCA0LjQyNWwxLjE1Ny0xLjI1bC4wMTQuMDE0YTQuNDIgNC40MiAwIDAgMS0xLjM1NS0zLjE4NyA0LjQzNiA0LjQzNiAwIDAgMSA0LjQzNy00LjQzNyA0LjQgNC40IDAgMCAxIDIuMjE3LjU5OGwuODItMS40OThhNi4xIDYuMSAwIDAgMC0zLjAzNy0uODA2Yy0zLjM4NC0uMDA1LTYuMTQxIDIuNzUzLTYuMTQxIDYuMTQxbTYuNjggMGEuNTM4LjUzOCAwIDAgMS0xLjA3NCAwIC41MzcuNTM3IDAgMSAxIDEuMDc1IDBtLTMuOTQgMGEzLjQgMy40IDAgMSAxIDYuODAxIDAgMy40IDMuNCAwIDAgMS02LjggMG0uMTQ5IDBhMy4yNTYgMy4yNTYgMCAwIDAgMy4yNTIgMy4yNTIgMy4yNTUgMy4yNTUgMCAwIDAgMy4yNTQtMy4yNTIgMy4yNTMgMy4yNTMgMCAxIDAtNi41MDYgMCIvPjwvc3ZnPg==" height="16"> **Discogs** | Queries Discogs API for release genres | `discogs/bin/discogs-genres` |

All providers normalize genre tags to a consistent set and write them to MP3 files via [id3v2](https://github.com/bogem/id3v2).

## <img src="https://api.iconify.design/mdi:file-tree.svg?color=%2301acd7" height="22"> Project Structure

```text
~/go-genres/
├── assets/                       # Project assets (icon, etc.)
├── menu/                         # TUI launcher (terminal UI)
│   ├── main.go                   #   Interactive menu entrypoint
│   ├── build.sh                  #   Build script
│   └── bin/genres                #   Compiled binary
├── shared/                       # Shared libraries
│   ├── normalize.go              #   Genre normalization logic
│   ├── undo.go                   #   Tag restore / undo support
│   ├── settings.go               #   Global settings (backup, paths, Swinsian)
│   ├── swinsian.go               #   Swinsian AppleScript integration
│   └── undo_test.go
├── discogs/                      # Discogs provider
│   ├── main.go
│   ├── build.sh
│   └── bin/discogs-genres
├── spotify/                      # Spotify provider
│   ├── main.go
│   ├── build.sh
│   └── bin/spotify-genres
├── lastfm/                       # Last.fm provider
│   ├── main.go
│   ├── build.sh
│   └── bin/lastfm-genres
├── metallum/                     # Metal Archives provider
│   ├── main.go
│   ├── build.sh
│   └── bin/metallum-genres
├── go.mod                        # Root Go module
├── go.sum                        # Dependency checksums
└── README.md
```

## <img src="https://api.iconify.design/mdi:code-braces.svg?color=%2301acd7" height="22"> Building

Build a single provider:

```bash
bash discogs/build.sh
```

Build all providers and the menu:

```bash
for d in menu discogs spotify lastfm metallum; do bash "$d/build.sh"; done
```

<details>
<summary><strong>Development Commands</strong></summary>

| Command | Purpose |
| --- | --- |
| `go build ./...` | Build all packages |
| `go test ./shared` | Run shared library tests |
| `go vet ./...` | Static analysis |
</details>

## <img src="https://api.iconify.design/mdi:github.svg?color=%2301acd7" height="22"> Repository

<https://github.com/deathrashed/go-genres>
