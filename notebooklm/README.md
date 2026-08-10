# Create your own Palworld NotebookLM notebook

Palpedia Snapshot no longer depends on a shared notebook. Your notebook belongs to your own Google account and combines the bundled game reference with your personal, read-only save export.

1. Open [NotebookLM](https://notebook.google.com/) and sign in to your Google account.
2. Create a new notebook, for example **My Palworld Palpedia**.
3. Add all 31 Markdown files in `palworld-reference`. This is the shared game reference corpus bundled with the application.
4. In Palpedia Snapshot, create a fresh export and open its `export_<date time>` folder.
5. Add every generated file from that folder except `world.json`:

   - `collection.md`
   - `pals.csv`
   - `capture-history.csv`
   - `palpedia-progress.md`
   - `breeding-candidates.md`
   - `breeding-rules.md`
   - `breeding-direct-pairs.csv`
   - `collection-diff.md` when a comparison was requested

The 31 reference files plus up to 8 personal files make 39 sources, leaving room in a 50-source notebook for additional material.

## Ask useful questions

Use prompts that name the personal export and the relevant rule files. For example:

- “Using `palpedia-progress.md` and the reference corpus, which Palpedia entries should I capture next and why?”
- “Using `pals.csv`, `breeding-candidates.md`, and `breeding-direct-pairs.csv`, identify the best available parents for preserving Philanthropist and Demon God. Do not infer child species from a median.”
- “Using `breeding-rules.md` and `breeding-direct-pairs.csv`, explain the shortest practical route I can make toward Anubis, and state which parent traits are useful to preserve.”
- “If a fact is not in the sources, say so instead of guessing.”

## Refreshing your personal data

NotebookLM uploads are snapshots. When you create a new Palpedia Snapshot export, remove or deselect the old personal export files in your notebook and add the files from the newest `export_<date time>` folder. Keep the bundled reference corpus unless a newer Palpedia Snapshot release supplies an updated corpus.

`world.json` is intentionally excluded: it is a technical troubleshooting file and is not a useful NotebookLM source.
