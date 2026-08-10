# Palpedia Snapshot

Read-only Windows app for exporting a Palworld `Level.sav` and its player saves. Run the executable normally to open the graphical interface; the command-line mode remains available when arguments are supplied.

Each export creates a separate timestamped folder in the parent directory you choose, for example `export_08-09-2026 18-42`. The minute separator is `-` because Windows does not allow `:` in folder names.

Add these files from that folder to NotebookLM:

- `collection.md` — player collection summary and current party/Palbox Pals
- `pals.csv` — current player-owned Pals in the party or Palbox, including their passive trait IDs; this is the NotebookLM collection source
- `capture-history.csv` — per-player Paldeck capture counts
- `palpedia-progress.md` — NotebookLM-ready current-collection and capture-history summary
- `breeding-candidates.md` — current Pals grouped by passive trait for breeding planning
- `breeding-rules.md` — exact child-species rule, so NotebookLM does not infer a median-based outcome
- `breeding-direct-pairs.csv` — exact child species for every currently possible male/female species pair
- `collection-diff.md` — changes since a previous export, generated with `--compare` (only when comparing)

Do **not** add `world.json` to NotebookLM. It is a complete technical export for troubleshooting, not a NotebookLM source.

## Breeding planner preview

The `v3.0.0-rc1` branch adds in-app, read-only planning tools. The app first asks only for a `Level.sav`; after choosing it, select **NotebookLM export**, **Breeding planner**, or **Quickest route**. Only the chosen workspace opens, while the other choices stay available above it. **Change save** reopens the save selector at any time.

The planner can:

- browse every matching Pal in a scrollable collection list; search by player-facing name, raw game ID, or known trait; filter by male/female; and sort by level;
- restrict the view to bundled gold (rank 3) or diamond (rank 4) passive-trait catalogs, including Philanthropist, Babysitter, and Demon God;
- select a real male and female Pal, keeping their individual traits visible, then calculate the exact offspring species; and
- calculate a textual route to a target Pal name or Character ID using the fewest sequential breeding generations from the loaded male/female collection.

The planner translates all 299 bundled game IDs into their Palpedia-facing names: for example, `BOSS_GrassMammoth` displays as **Mammorest**. It still accepts raw IDs when useful.

To keep the picker compact, identical species/sex/passive-trait combinations are represented by only their highest-level Pal. Gold traits use yellow labels and diamond traits green labels. Standard passive IDs are translated too: for example, `CraftSpeed_up1` displays as **Serious** and `PAL_ALLAttack_up1` as **Brave**. Hover any known trait to see its in-game effect. The header includes a light/dark theme switch, and the selected save/export setup collapses into small summaries that can be reopened when needed.

The quickest-route workspace uses dedicated outlined cards for target selection and the final route. It offers clickable name suggestions as you type. Its result is a family tree: each male/female pair sits above the child it breeds. Each non-target node provides two decisions:

- **Avoid species** removes that species from the route entirely and searches for an alternative.
- **Breed new** removes only the owned copies from the starting collection, then adds that Pal’s own parent branch above the current tree. It does not replace the rest of the route. Use it again on a grandparent to add another earlier generation when you want to improve that Pal’s traits too.

The family tree is displayed in generation-aligned rows, so both parents always appear on the same layer even if one branch needs more earlier generations. You can also limit the initial route to Pals with gold and/or diamond passive traits, then click **Use all Pals again** to reset every route decision. This lets you steer a species route while deciding which individual passive traits to preserve. If the target is already in the collection, the app clearly says so, then calculates the shortest route while deliberately excluding those copies. This keeps the route useful when breeding a new target to inherit a different set of passive traits. The selected male and female parents remain in dedicated cards above the scrollable Pal list.

For a route longer than two generations, it also identifies the Philanthropist and Babysitter Pals in the loaded collection and explains how to use them to speed up the breeding farm.

The route is a species plan, not a promise of a specific egg: passive inheritance and egg gender remain game RNG. The UI calls this out so it is suitable for planning, rather than overstating certainty.

The application never modifies save files. It refuses to place its output inside the save directory.

## Windows build

Clone the repository with its decoder source, then run the build helper before the Go build:

```powershell
git clone --recurse-submodules https://github.com/Ltorre/palpedia-snapshot.git
cd palpedia-snapshot
# Run from Git Bash, WSL, or the GitHub Actions Linux runner.
./scripts/build-ooz-windows.sh
go build -o palpedia-snapshot.exe .\cmd\palpedia-snapshot
```

Official GitHub releases attach both the Windows binary and `palpedia-snapshot-notebooklm-reference.zip`, the 31-file NotebookLM corpus. Check a binary with:

```powershell
.\palpedia-snapshot-windows-amd64.exe --version
```

Each release includes a standalone, source-ready release note describing its export format, use, and safety limits.

## Graphical interface

Open the Windows executable without arguments and it will:

![Palpedia Snapshot GUI preview](docs/gui-preview.svg)

- start at the standard save location, `C:\Users\<WindowsUser>\AppData\Local\Pal\Saved\SaveGames`;
- find `Level.sav` files there, or let you browse for one manually;
- export to a new timestamped folder inside `Documents\Palpedia Snapshot Exports` by default, safely outside your game saves, or let you choose a parent destination with the folder picker;
- offer **Open export folder in Explorer** after a successful export, ready for drag-and-drop into NotebookLM;
- switch between English and French; and
- keep shared-world and comparison controls under clearly labelled optional advanced options.

For a shared world, use **Find players in this save** in the advanced options, then select a player. Leave the Player UID empty to export every player.

The export parent directory may be on another drive, such as `I:\Palworld export`; a different Windows volume is always outside the game save location. Keep prior `export_<date time>` folders there and select one with **Choose previous export** when creating a comparison.

## Create your own NotebookLM notebook

Palpedia Snapshot ships the whole [Palworld reference corpus](notebooklm/palworld-reference): 31 compact Markdown sources covering 299 Pals, skills, items, equipment, structures, technology, map, tools, tier lists, and breeding context. There is no shared public notebook to duplicate.

1. Open [NotebookLM](https://notebook.google.com/) and sign in with your own Google account.
2. Create a notebook, then add all 31 Markdown files in `notebooklm/palworld-reference`.
3. Create a fresh Palpedia Snapshot export and add every personal file from its `export_<date time>` folder except `world.json`.
4. Ask questions that name both the personal files and relevant reference sources. For breeding, include `breeding-rules.md`, `breeding-direct-pairs.csv`, and `breeding-candidates.md` so NotebookLM uses the exact result instead of inferring a median.

The corpus plus all eight optional personal export files totals 39 sources, leaving room in a 50-source notebook. When a new save export is created, replace the old personal export files in NotebookLM with the newest set. See the dedicated [NotebookLM setup guide](notebooklm/README.md) for file lists, refresh instructions, and prompts.

`passive_traits` contains the game’s exact passive-skill IDs. This keeps the export reliable across game updates; the bundled corpus maps the relevant reference information while raw IDs remain available for precise filtering.

## Export

For a typical Steam installation, local worlds are under:

```text
C:\Users\<WindowsUser>\AppData\Local\Pal\Saved\SaveGames\<SteamID>\<WorldID>\Level.sav
```

`<WorldID>` is the world folder containing both `Level.sav` and the matching `Players` directory. The graphical interface searches this location automatically; in command-line mode, pass the exact `Level.sav` you want to inspect.

```powershell
.\palpedia-snapshot.exe `
  --level "D:\path\to\Level.sav" `
  --output "D:\path\to\export"
```

`Level.sav` must have its matching `Players` directory beside it, unless you pass `--players-dir` explicitly. `--output` is mandatory, must be outside the save directory, and is the parent directory for a new `export_<date time>` snapshot. Existing snapshots are never replaced.

### Shared worlds: export one player's collection

List the player IDs in the save, then use the chosen UID with `--player`:

```powershell
.\palpedia-snapshot.exe --level "D:\path\to\Level.sav" --list-players

.\palpedia-snapshot.exe `
  --level "D:\path\to\Level.sav" `
  --output "D:\path\to\personal-export" `
  --player "player-uid-from-list"
```

With `--player`, `pals.csv`, `capture-history.csv`, and `collection.md` contain only that player's data. `world.json` remains the complete world export.

### Compare two collection snapshots

Point `--compare` at an earlier export directory. The new export will include `collection-diff.md`, showing additions and removals from party/Palbox plus Paldeck capture gains.

```powershell
.\palpedia-snapshot.exe `
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
