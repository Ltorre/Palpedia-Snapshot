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
	rows := pals(world)
	got := []string{rows[0].Scope, rows[1].Scope, rows[2].Scope}
	if strings.Join(got, ",") != "base,palbox,party" {
		t.Fatalf("scopes = %v", got)
	}
}

func TestWriteCreatesAllExports(t *testing.T) {
	rank := 2
	world := &sav.World{
		Players: []sav.Player{{UID: "player", Nickname: "Player", OtomoContainerID: "party", PalStorageContainerID: "box", PalCaptureCounts: map[string]int64{"Lamball": 4}}},
		Pals:    []sav.Pal{{CharacterID: "Lamball", ContainerID: "party", Rank: &rank}},
	}
	dir := t.TempDir()
	if err := Write(dir, world, false); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"collection.md", "pals.csv", "capture-history.csv", "world.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
