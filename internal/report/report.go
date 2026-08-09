package report

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Ltorre/palworld-save-scrap/internal/sav"
)

type palRow struct {
	Scope      string
	PlayerUID  string
	InstanceID string
	Character  string
	Level      int32
	Gender     string
	Rank       string
	OwnerUID   string
	BaseID     string
	Slot       int
}

func ValidateOutputDirectory(levelPath, outputDir string, force bool) error {
	levelDir := filepath.Dir(levelPath)
	rel, err := filepath.Rel(levelDir, outputDir)
	if err != nil {
		return err
	}
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
		return fmt.Errorf("output directory must be outside the save directory")
	}
	if st, err := os.Stat(outputDir); err == nil {
		if !st.IsDir() {
			return fmt.Errorf("output path is not a directory: %s", outputDir)
		}
		entries, readErr := os.ReadDir(outputDir)
		if readErr != nil {
			return readErr
		}
		if len(entries) > 0 && !force {
			return fmt.Errorf("output directory is not empty; choose another directory or use --force")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func HasPlayer(world *sav.World, playerUID string) bool {
	if playerUID == "" {
		return true
	}
	for _, player := range world.Players {
		if strings.EqualFold(player.UID, playerUID) {
			return true
		}
	}
	return false
}

func Write(outputDir string, world *sav.World, playerUID, compareDir string, force bool) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	worldJSON, err := json.MarshalIndent(world, "", "  ")
	if err != nil {
		return err
	}
	rows := pals(world, playerUID)
	collectionRows := currentCollection(rows)
	if err := writeBytes(filepath.Join(outputDir, "world.json"), append(worldJSON, '\n'), force); err != nil {
		return err
	}
	if err := writeBytes(filepath.Join(outputDir, "pals.csv"), palsCSV(collectionRows), force); err != nil {
		return err
	}
	if err := writeBytes(filepath.Join(outputDir, "capture-history.csv"), capturesCSV(world, playerUID), force); err != nil {
		return err
	}
	if err := writeBytes(filepath.Join(outputDir, "palpedia-progress.md"), palpediaProgress(world, collectionRows, playerUID), force); err != nil {
		return err
	}
	if compareDir != "" {
		diff, err := collectionDiff(world, collectionRows, playerUID, compareDir)
		if err != nil {
			return err
		}
		if err := writeBytes(filepath.Join(outputDir, "collection-diff.md"), diff, force); err != nil {
			return err
		}
	}
	return writeBytes(filepath.Join(outputDir, "collection.md"), markdown(world, rows, playerUID), force)
}

func currentCollection(rows []palRow) []palRow {
	collection := make([]palRow, 0, len(rows))
	for _, row := range rows {
		if row.Scope == "party" || row.Scope == "palbox" {
			collection = append(collection, row)
		}
	}
	return collection
}

func pals(world *sav.World, playerUID string) []palRow {
	containers := make(map[string]containerOwner, len(world.Players)*2)
	for _, player := range world.Players {
		if playerUID != "" && !strings.EqualFold(player.UID, playerUID) {
			continue
		}
		if player.OtomoContainerID != "" {
			containers[strings.ToLower(player.OtomoContainerID)] = containerOwner{player.UID, "party"}
		}
		if player.PalStorageContainerID != "" {
			containers[strings.ToLower(player.PalStorageContainerID)] = containerOwner{player.UID, "palbox"}
		}
	}
	rows := make([]palRow, 0, len(world.Pals))
	for _, pal := range world.Pals {
		scope, playerUID := "unassigned", ""
		if owner, ok := containers[strings.ToLower(pal.ContainerID)]; ok {
			scope, playerUID = owner.scope, owner.playerUID
		} else if pal.BaseID != "" {
			scope = "base"
		} else if pal.ContainerID == "" {
			scope = "world"
		}
		rank := ""
		if pal.Rank != nil {
			rank = strconv.Itoa(*pal.Rank)
		}
		rows = append(rows, palRow{scope, playerUID, pal.InstanceID, pal.CharacterID, pal.Level, pal.Gender, rank, pal.OwnerUID, pal.BaseID, pal.SlotIndex})
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.Join([]string{rows[i].PlayerUID, rows[i].Scope, rows[i].Character, rows[i].InstanceID}, "\x00") < strings.Join([]string{rows[j].PlayerUID, rows[j].Scope, rows[j].Character, rows[j].InstanceID}, "\x00")
	})
	return rows
}

type containerOwner struct{ playerUID, scope string }

func palsCSV(rows []palRow) []byte {
	var out bytes.Buffer
	w := csv.NewWriter(&out)
	_ = w.Write([]string{"scope", "player_uid", "instance_id", "character_id", "level", "gender", "rank", "owner_uid", "base_id", "slot"})
	for _, row := range rows {
		_ = w.Write([]string{row.Scope, row.PlayerUID, row.InstanceID, row.Character, strconv.Itoa(int(row.Level)), row.Gender, row.Rank, row.OwnerUID, row.BaseID, strconv.Itoa(row.Slot)})
	}
	w.Flush()
	return out.Bytes()
}

func capturesCSV(world *sav.World, playerUID string) []byte {
	var out bytes.Buffer
	w := csv.NewWriter(&out)
	_ = w.Write([]string{"player_uid", "character_id", "captures"})
	for _, player := range world.Players {
		if playerUID != "" && !strings.EqualFold(player.UID, playerUID) {
			continue
		}
		keys := make([]string, 0, len(player.PalCaptureCounts))
		for key := range player.PalCaptureCounts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			_ = w.Write([]string{player.UID, key, strconv.FormatInt(player.PalCaptureCounts[key], 10)})
		}
	}
	w.Flush()
	return out.Bytes()
}

func palpediaProgress(world *sav.World, collection []palRow, playerUID string) []byte {
	byPlayer := make(map[string][]palRow)
	for _, row := range collection {
		byPlayer[row.PlayerUID] = append(byPlayer[row.PlayerUID], row)
	}
	var out strings.Builder
	out.WriteString("# Palpedia progress\n\n")
	out.WriteString("This is a read-only snapshot of current owned Pals and the Paldeck capture records stored in the selected player's save. Add this file and `pals.csv` to the shared NotebookLM template to identify the best next captures.\n\n")
	for _, player := range world.Players {
		if playerUID != "" && !strings.EqualFold(player.UID, playerUID) {
			continue
		}
		owned := make(map[string]int)
		for _, row := range byPlayer[player.UID] {
			owned[row.Character]++
		}
		captureCount := make(map[string]int64)
		captures := int64(0)
		for character, count := range player.PalCaptureCounts {
			if count > 0 {
				captureCount[character] = count
				captures += count
			}
		}
		name := player.Nickname
		if name == "" {
			name = "Unnamed player"
		}
		fmt.Fprintf(&out, "## %s\n\n- Player UID: `%s`\n- Current unique Pal species: %d\n- Paldeck capture-record species: %d\n- Captures recorded: %d\n", name, player.UID, len(owned), len(captureCount), captures)
		if player.UniquePalsCaptured != nil {
			fmt.Fprintf(&out, "- Game-reported captured species: %d\n", *player.UniquePalsCaptured)
		}
		if player.CaptureTotal != nil {
			fmt.Fprintf(&out, "- Game-reported lifetime captures: %d\n", *player.CaptureTotal)
		}

		characters := sortedKeys(owned)
		out.WriteString("\n### Currently owned species\n\n| Character ID | Current Pals | Captures recorded |\n| --- | ---: | ---: |\n")
		for _, character := range characters {
			fmt.Fprintf(&out, "| %s | %d | %d |\n", character, owned[character], captureCount[character])
		}
		if len(characters) == 0 {
			out.WriteString("| _No party or Palbox Pals found_ | 0 | 0 |\n")
		}

		noLongerOwned := make([]string, 0)
		for character := range captureCount {
			if owned[character] == 0 {
				noLongerOwned = append(noLongerOwned, character)
			}
		}
		sort.Strings(noLongerOwned)
		out.WriteString("\n### Captured but not currently in party or Palbox\n\n| Character ID | Captures recorded |\n| --- | ---: |\n")
		for _, character := range noLongerOwned {
			fmt.Fprintf(&out, "| %s | %d |\n", character, captureCount[character])
		}
		if len(noLongerOwned) == 0 {
			out.WriteString("| _None_ | 0 |\n")
		}
		out.WriteByte('\n')
	}
	return []byte(out.String())
}

func sortedKeys(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func collectionDiff(world *sav.World, current []palRow, playerUID, compareDir string) ([]byte, error) {
	previousCollection, err := collectionSnapshot(filepath.Join(compareDir, "pals.csv"), playerUID)
	if err != nil {
		return nil, fmt.Errorf("read previous collection: %w", err)
	}
	previousCaptures, err := captureSnapshot(filepath.Join(compareDir, "capture-history.csv"), playerUID)
	if err != nil {
		return nil, fmt.Errorf("read previous capture history: %w", err)
	}
	return snapshotDiffMarkdown(collectionCounts(current), previousCollection, captureCounts(world, playerUID), previousCaptures), nil
}

func collectionCounts(rows []palRow) map[string]int64 {
	counts := make(map[string]int64)
	for _, row := range rows {
		counts[row.Character]++
	}
	return counts
}

func captureCounts(world *sav.World, playerUID string) map[string]int64 {
	counts := make(map[string]int64)
	for _, player := range world.Players {
		if playerUID != "" && !strings.EqualFold(player.UID, playerUID) {
			continue
		}
		for character, count := range player.PalCaptureCounts {
			if count > 0 {
				counts[character] += count
			}
		}
	}
	return counts
}

func collectionSnapshot(path, playerUID string) (map[string]int64, error) {
	rows, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64)
	for _, row := range rows {
		if playerUID != "" && !strings.EqualFold(row["player_uid"], playerUID) {
			continue
		}
		if character := row["character_id"]; character != "" {
			counts[character]++
		}
	}
	return counts, nil
}

func captureSnapshot(path, playerUID string) (map[string]int64, error) {
	rows, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64)
	for _, row := range rows {
		if playerUID != "" && !strings.EqualFold(row["player_uid"], playerUID) {
			continue
		}
		character := row["character_id"]
		count, parseErr := strconv.ParseInt(row["captures"], 10, 64)
		if character == "" || parseErr != nil || count < 0 {
			return nil, fmt.Errorf("invalid capture-history row for %q", character)
		}
		counts[character] += count
	}
	return counts, nil
}

func readCSV(path string) ([]map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]string, 0)
	for {
		record, readErr := reader.Read()
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return nil, readErr
		}
		if len(record) != len(header) {
			return nil, fmt.Errorf("invalid CSV column count")
		}
		row := make(map[string]string, len(header))
		for index, key := range header {
			row[key] = record[index]
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func snapshotDiffMarkdown(current, previous, currentCaptures, previousCaptures map[string]int64) []byte {
	var out strings.Builder
	out.WriteString("# Collection snapshot comparison\n\n")
	fmt.Fprintf(&out, "- Previous current Pals: %d\n- Current Pals: %d\n- Previous unique species: %d\n- Current unique species: %d\n\n", sumCounts(previous), sumCounts(current), len(previous), len(current))
	writeCountDiff(&out, "## Added to party or Palbox", current, previous, "Previous Pals", "Current Pals")
	writeCountDiff(&out, "## No longer in party or Palbox", previous, current, "Current Pals", "Previous Pals")
	writeCountDiff(&out, "## Paldeck capture gains", currentCaptures, previousCaptures, "Previous captures", "Current captures")
	return []byte(out.String())
}

func writeCountDiff(out *strings.Builder, heading string, primary, baseline map[string]int64, baselineLabel, primaryLabel string) {
	type change struct {
		character string
		before    int64
		after     int64
		delta     int64
	}
	changes := make([]change, 0)
	for character, after := range primary {
		before := baseline[character]
		if after > before {
			changes = append(changes, change{character, before, after, after - before})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].character < changes[j].character })
	out.WriteString(heading + "\n\n")
	out.WriteString("| Character ID | " + baselineLabel + " | " + primaryLabel + " | Change |\n| --- | ---: | ---: | ---: |\n")
	for _, item := range changes {
		fmt.Fprintf(out, "| %s | %d | %d | +%d |\n", item.character, item.before, item.after, item.delta)
	}
	if len(changes) == 0 {
		fmt.Fprintf(out, "| _None_ | 0 | 0 | 0 |\n")
	}
	out.WriteByte('\n')
}

func sumCounts(counts map[string]int64) int64 {
	total := int64(0)
	for _, count := range counts {
		total += count
	}
	return total
}

func markdown(world *sav.World, rows []palRow, playerUID string) []byte {
	byPlayer := make(map[string][]palRow)
	for _, row := range rows {
		if row.Scope == "party" || row.Scope == "palbox" {
			byPlayer[row.PlayerUID] = append(byPlayer[row.PlayerUID], row)
		}
	}
	var out strings.Builder
	out.WriteString("# Palworld collection export\n\n")
	out.WriteString("Read-only export from the supplied save files. `pals.csv` contains the selected player's current party and Palbox collection. `world.json` contains the complete parsed world view.\n\n")
	fmt.Fprintf(&out, "- Players: %d\n- Decoded Pals: %d\n- Guilds: %d\n- Bases: %d\n\n", len(world.Players), len(world.Pals), len(world.Guilds), len(world.Bases))
	for _, player := range world.Players {
		if playerUID != "" && !strings.EqualFold(player.UID, playerUID) {
			continue
		}
		current := byPlayer[player.UID]
		party, palbox := 0, 0
		unique := map[string]struct{}{}
		for _, row := range current {
			if row.Scope == "party" {
				party++
			} else {
				palbox++
			}
			unique[row.Character] = struct{}{}
		}
		name := player.Nickname
		if name == "" {
			name = "Unnamed player"
		}
		fmt.Fprintf(&out, "## %s\n\n- UID: `%s`\n- Level: %d\n- Current party: %d\n- Current Palbox: %d\n- Current species: %d\n", name, player.UID, player.Level, party, palbox, len(unique))
		if player.UniquePalsCaptured != nil {
			fmt.Fprintf(&out, "- Paldeck species captured: %d\n", *player.UniquePalsCaptured)
		}
		if player.CaptureTotal != nil {
			fmt.Fprintf(&out, "- Lifetime captures: %d\n", *player.CaptureTotal)
		}
		out.WriteString("\n| Storage | Pal | Level | Gender | Rank | Slot |\n| --- | --- | ---: | --- | ---: | ---: |\n")
		for _, row := range current {
			fmt.Fprintf(&out, "| %s | %s | %d | %s | %s | %d |\n", row.Scope, row.Character, row.Level, row.Gender, row.Rank, row.Slot)
		}
		if len(current) == 0 {
			out.WriteString("| - | No party or Palbox Pals identified | | | | |\n")
		}
		out.WriteByte('\n')
	}
	return []byte(out.String())
}

func writeBytes(path string, data []byte, force bool) error {
	if _, err := os.Stat(path); err == nil {
		if !force {
			return fmt.Errorf("refusing to overwrite %s without --force", filepath.Base(path))
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".palworld-save-scrap-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Chmod(0o644)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if force {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return os.Rename(tmpPath, path)
}
