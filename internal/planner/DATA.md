# Passive-tier filter data

`traits.go` contains compact mappings from Palworld save-game passive IDs to the rank-3 (gold) and rank-4 (diamond) filters shown in the planner. It retains raw IDs in all displays, so traits outside this compact catalog remain searchable rather than being hidden.

The mappings were checked on 2026-08-10 against the public [Palworld Wiki passive-skill list](https://palworld.wiki.gg/wiki/Passive_Skills/List) and [PalMods game-ID reference](https://www.palmods.gg/docs/authors/game-ids/passive-skills). The data is bundled for offline use; Palpedia Snapshot makes no network request at runtime.
