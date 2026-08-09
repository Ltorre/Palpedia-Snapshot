# Palworld Save Scrap

Small read-only Windows CLI for exporting a Palworld `Level.sav` and its player saves.

It writes four files to a directory you choose:

- `collection.md` — player collection summary and current party/Palbox Pals
- `pals.csv` — current player-owned Pals in the party or Palbox; this is the NotebookLM collection source
- `capture-history.csv` — per-player Paldeck capture counts
- `world.json` — complete typed export, including players, all decoded Pals, guilds, bases, and parser diagnostics

The application never modifies save files. It refuses to place its output inside the save directory.

## Windows build

Install Go, then run:

```powershell
go build -o palworld-save-scrap.exe .\cmd\palworld-save-scrap
```

Official Windows binaries are attached to each GitHub release. Check a binary with:

```powershell
.\palworld-save-scrap-windows-amd64.exe --version
```

Each release includes a standalone, source-ready release note describing its export format, use, and safety limits.

## NotebookLM template

Duplicate the shared [Palworld Palpedia NotebookLM template](https://notebook.google.com/notebook/fec4f41d-1c32-4b8d-975c-a0fbe3f7eba1), then add the generated `pals.csv` as your personal collection source. The notebook can use that CSV to identify gaps in your Palpedia and propose efficient next captures.

## Export

For a typical Steam installation, local worlds are under:

```text
C:\Users\<WindowsUser>\AppData\Local\Pal\Saved\SaveGames\<SteamID>\<WorldID>\Level.sav
```

`<WorldID>` is the world folder containing both `Level.sav` and the matching `Players` directory. The executable does not search this location automatically; pass the exact `Level.sav` you want to inspect.

```powershell
.\palworld-save-scrap.exe `
  --level "D:\path\to\Level.sav" `
  --output "D:\path\to\export" `
  --oodle-lib "D:\path\to\oo2core_9_win64.dll"
```

`Level.sav` must have its matching `Players` directory beside it, unless you pass `--players-dir` explicitly. `--output` is mandatory and must be outside the save directory. Use `--force` only to replace files in an existing export directory.

Modern saves use Oodle compression. For those saves, pass the absolute path to `oo2core_9_win64.dll` from your own Palworld installation. The DLL is not included, downloaded, copied, or changed by this project.

## Safety and scope

- Reads `Level.sav`, optional `LevelMeta.sav`, and player `.sav` files only.
- Makes no network requests at runtime.
- Exports game-internal `CharacterID` values exactly as stored; it does not guess localized Pal names.

The parser is based on the Apache-2.0 Palhelm save parser. See [NOTICE](NOTICE).
