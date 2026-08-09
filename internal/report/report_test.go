package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ltorre/palworld-save-scrap/internal/sav"
)

func TestPalsClassifiesPlayerContainers(t *testing.T) {
	rank := 1
	world := &sav.World{
		Players: []sav.Player{{UID: "player", OtomoContainerID: "party", PalStorageContainerID: "box"}},
		Pals:    []sav.Pal{{CharacterID: "PartyPal", ContainerID: "party", Rank: &rank}, {CharacterID: "BoxPal", ContainerID: "box", Rank: &rank}, {CharacterID: "BasePal", BaseID: "base"}},
	}
	rows := pals(world, "")
	got := []string{rows[0].Scope, rows[1].Scope, rows[2].Scope}
	if strings.Join(got, ",") != "base,palbox,party" {
		t.Fatalf("scopes = %v", got)
	}
}

func TestWriteCreatesAllExports(t *testing.T) {
	rank := 2
	world := &sav.World{
		Players: []sav.Player{{UID: "player", Nickname: "Player", OtomoContainerID: "party", PalStorageContainerID: "box", PalCaptureCounts: map[string]int64{"Lamball": 4}}},
		Pals: []sav.Pal{
			{CharacterID: "Lamball", ContainerID: "party", Rank: &rank, PassiveSkillIDs: []string{"Philanthropist"}},
			{CharacterID: "WildPal"},
		},
	}
	dir := t.TempDir()
	if err := Write(dir, world, "", "", false); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"collection.md", "pals.csv", "capture-history.csv", "palpedia-progress.md", "breeding-candidates.md", "world.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	csvBytes, err := os.ReadFile(filepath.Join(dir, "pals.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(csvBytes), "WildPal") {
		t.Fatal("pals.csv contains a Pal outside the current collection")
	}
	if !strings.Contains(string(csvBytes), "passive_traits") || !strings.Contains(string(csvBytes), "Philanthropist") {
		t.Fatalf("pals.csv does not expose passive traits: %s", csvBytes)
	}
	progress, err := os.ReadFile(filepath.Join(dir, "palpedia-progress.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(progress), "Lamball") || !strings.Contains(string(progress), "Current unique Pal species") {
		t.Fatalf("unexpected progress report: %s", progress)
	}
	breeding, err := os.ReadFile(filepath.Join(dir, "breeding-candidates.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(breeding), "Philanthropist") || !strings.Contains(string(breeding), "Lamball") {
		t.Fatalf("unexpected breeding report: %s", breeding)
	}
}

func TestSnapshotDiffShowsCollectionAndCaptureChanges(t *testing.T) {
	markdown := string(snapshotDiffMarkdown(
		map[string]int64{"Lamball": 2, "Cattiva": 1},
		map[string]int64{"Lamball": 1, "Chikipi": 1},
		map[string]int64{"Lamball": 5, "Cattiva": 1},
		map[string]int64{"Lamball": 3},
	))
	for _, expected := range []string{"Cattiva", "Chikipi", "Paldeck capture gains", "+2"} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("diff does not contain %q: %s", expected, markdown)
		}
	}
}

func TestWriteComparisonReadsPreviousExport(t *testing.T) {
	previous := t.TempDir()
	if err := os.WriteFile(filepath.Join(previous, "pals.csv"), []byte("scope,player_uid,instance_id,character_id,level,gender,rank,owner_uid,base_id,slot\nparty,player,old,Lamball,1,male,1,player,,0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(previous, "capture-history.csv"), []byte("player_uid,character_id,captures\nplayer,Lamball,2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rank := 1
	world := &sav.World{
		Players: []sav.Player{{UID: "player", OtomoContainerID: "party", PalCaptureCounts: map[string]int64{"Lamball": 3, "Cattiva": 1}}},
		Pals:    []sav.Pal{{CharacterID: "Lamball", ContainerID: "party", Rank: &rank}, {CharacterID: "Cattiva", ContainerID: "party", Rank: &rank}},
	}
	destination := t.TempDir()
	if err := Write(destination, world, "player", previous, false); err != nil {
		t.Fatal(err)
	}
	diff, err := os.ReadFile(filepath.Join(destination, "collection-diff.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(diff), "Cattiva") || !strings.Contains(string(diff), "+1") {
		t.Fatalf("unexpected comparison: %s", diff)
	}
}

func TestPlayerFilterKeepsOnlyRequestedCollection(t *testing.T) {
	rank := 1
	world := &sav.World{
		Players: []sav.Player{
			{UID: "one", OtomoContainerID: "one-party"},
			{UID: "two", OtomoContainerID: "two-party"},
		},
		Pals: []sav.Pal{
			{CharacterID: "OnePal", ContainerID: "one-party", Rank: &rank},
			{CharacterID: "TwoPal", ContainerID: "two-party", Rank: &rank},
		},
	}
	rows := currentCollection(pals(world, "one"))
	if len(rows) != 1 || rows[0].Character != "OnePal" {
		t.Fatalf("filtered collection = %#v", rows)
	}
	if !HasPlayer(world, "one") || HasPlayer(world, "missing") {
		t.Fatal("unexpected player lookup result")
	}
}
