# Palworld Save Scrap

Read-only Windows app for exporting a Palworld `Level.sav` and its player saves. Run the executable normally to open the graphical interface; the command-line mode remains available when arguments are supplied.

Each export creates a separate timestamped folder in the parent directory you choose, for example `export_08-09-2026 18-42`. The minute separator is `-` because Windows does not allow `:` in folder names.

Add these files from that folder to NotebookLM:

- `collection.md` — player collection summary and current party/Palbox Pals
- `pals.csv` — current player-owned Pals in the party or Palbox, including their passive trait IDs; this is the NotebookLM collection source
- `capture-history.csv` — per-player Paldeck capture counts
- `palpedia-progress.md` — NotebookLM-ready current-collection and capture-history summary
- `breeding-candidates.md` — current Pals grouped by passive trait for breeding planning
- `collection-diff.md` — changes since a previous export, generated with `--compare` (only when comparing)

Do **not** add `world.json` to NotebookLM. It is a complete technical export for troubleshooting, not a NotebookLM source.

The application never modifies save files. It refuses to place its output inside the save directory.

## Windows build

Clone the repository with its decoder source, then run the build helper before the Go build:

```powershell
git clone --recurse-submodules https://github.com/Ltorre/palworld-save-scrap.git
cd palworld-save-scrap
# Run from Git Bash, WSL, or the GitHub Actions Linux runner.
./scripts/build-ooz-windows.sh
go build -o palworld-save-scrap.exe .\cmd\palworld-save-scrap
```

Official Windows binaries are attached to each GitHub release. Check a binary with:

```powershell
.\palworld-save-scrap-windows-amd64.exe --version
```

Each release includes a standalone, source-ready release note describing its export format, use, and safety limits.

## Graphical interface

The `v2.0.0-rc1` branch contains the preview of the graphical Windows app. Open the executable without arguments and it will:

![Palworld Save Scrap GUI preview](docs/gui-preview.svg)

- start at the standard save location, `C:\Users\<WindowsUser>\AppData\Local\Pal\Saved\SaveGames`;
- find `Level.sav` files there, or let you browse for one manually;
- export to a new timestamped folder inside `Documents\Palworld Save Scrap Exports` by default, safely outside your game saves, or let you choose a parent destination with the folder picker;
- offer **Open export folder in Explorer** after a successful export, ready for drag-and-drop into NotebookLM;
- switch between English and French; and
- keep shared-world and comparison controls under clearly labelled optional advanced options.

For a shared world, use **Find players in this save** in the advanced options, then select a player. Leave the Player UID empty to export every player.

The export parent directory may be on another drive, such as `I:\Palworld export`; a different Windows volume is always outside the game save location. Keep prior `export_<date time>` folders there and select one with **Choose previous export** when creating a comparison.

## NotebookLM template

Duplicate the shared [Palworld Palpedia NotebookLM template](https://notebook.google.com/notebook/fec4f41d-1c32-4b8d-975c-a0fbe3f7eba1), then add the generated `pals.csv`, `palpedia-progress.md`, and `breeding-candidates.md` as your personal collection sources. The notebook can use them to identify gaps in your Palpedia, propose efficient next captures, and find breeding parents carrying useful passive traits.

`passive_traits` contains the game’s exact passive-skill IDs. This keeps the export reliable across game updates; NotebookLM can interpret their effects from its passive-skills reference source.

## Export

For a typical Steam installation, local worlds are under:

```text
C:\Users\<WindowsUser>\AppData\Local\Pal\Saved\SaveGames\<SteamID>\<WorldID>\Level.sav
```

`<WorldID>` is the world folder containing both `Level.sav` and the matching `Players` directory. The graphical interface searches this location automatically; in command-line mode, pass the exact `Level.sav` you want to inspect.

```powershell
.\palworld-save-scrap.exe `
  --level "D:\path\to\Level.sav" `
  --output "D:\path\to\export"
```

`Level.sav` must have its matching `Players` directory beside it, unless you pass `--players-dir` explicitly. `--output` is mandatory, must be outside the save directory, and is the parent directory for a new `export_<date time>` snapshot. Existing snapshots are never replaced.

### Shared worlds: export one player's collection

List the player IDs in the save, then use the chosen UID with `--player`:

```powershell
.\palworld-save-scrap.exe --level "D:\path\to\Level.sav" --list-players

.\palworld-save-scrap.exe `
  --level "D:\path\to\Level.sav" `
  --output "D:\path\to\personal-export" `
  --player "player-uid-from-list"
```

With `--player`, `pals.csv`, `capture-history.csv`, and `collection.md` contain only that player's data. `world.json` remains the complete world export.

### Compare two collection snapshots

Point `--compare` at an earlier export directory. The new export will include `collection-diff.md`, showing additions and removals from party/Palbox plus Paldeck capture gains.

```powershell
.\palworld-save-scrap.exe `
  --level "D:\path\to\Level.sav" `
  --output "D:\path\to\new-export" `
  --compare "D:\path\to\previous-export" `
  --player "player-uid-from-list"
```

Modern `PlM` saves use Oodle compression. The Windows executable includes the open-source Ooz decoder, so no game DLL, Steam installation, or extra path is required. On first use it places a hash-checked decoder helper in the Windows user cache, runs it only against a temporary copy of the compressed data, and removes that temporary data afterwards. It never copies, changes, or writes into game save files.

## Safety and scope

- Reads `Level.sav`, optional `LevelMeta.sav`, and player `.sav` files only.
- Makes no network requests at runtime.
- Bundles the GPLv3 Ooz decoder; the program source and release binaries are distributed under GPLv3. Generated exports and the user's save data are not covered by that license.
- Exports game-internal `CharacterID` values exactly as stored; it does not guess localized Pal names.

The parser is based on the Apache-2.0 Palhelm save parser and the embedded decoder is built from the GPLv3 Ooz source. See [NOTICE](NOTICE).
