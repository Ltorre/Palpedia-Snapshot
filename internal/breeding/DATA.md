# Breeding rule data

`breeding-rules.json` is a compact, factual rule snapshot used only by Palpedia Snapshot. It includes Palworld Character IDs, player-facing Pal names, breeding ranks, tie-break priorities, generic-output exclusions, and special parent combinations. It deliberately excludes descriptions, images, and other site content.

The data was derived on 2026-08-09 from the public Palworld.gg calculator data used by its [breeding calculator](https://palworld.gg/breeding-calculator). The 299 bundled display names were checked on 2026-08-10 against the public [Palworld Wiki](https://palworld.wiki.gg/). It is bundled so the application makes no network request and keeps a reproducible rule set for each release. Rules can change with Palworld updates; a newer application release may be needed after a game update.
