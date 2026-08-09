package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Ltorre/palworld-save-scrap/internal/gui"
	"github.com/Ltorre/palworld-save-scrap/internal/report"
	"github.com/Ltorre/palworld-save-scrap/internal/sav"
)

var version = "dev"

func main() {
	if len(os.Args) == 1 {
		gui.Run(version)
		return
	}
	runCLI()
}

func runCLI() {
	var levelPath, outputDir, playersDir, playerUID, compareDir string
	var force bool
	var showVersion bool
	var listPlayers bool
	flag.StringVar(&levelPath, "level", "", "path to Level.sav")
	flag.StringVar(&outputDir, "output", "", "empty output directory")
	flag.StringVar(&playersDir, "players-dir", "", "path to Players directory (defaults to the sibling Players directory)")
	flag.BoolVar(&force, "force", false, "allow writing into a non-empty output directory")
	flag.BoolVar(&showVersion, "version", false, "print version")
	flag.BoolVar(&listPlayers, "list-players", false, "list players found in the save")
	flag.StringVar(&playerUID, "player", "", "player UID to export; use --list-players to find it")
	flag.StringVar(&compareDir, "compare", "", "previous export directory to compare with the new export")
	flag.Parse()
	if showVersion {
		fmt.Println(version)
		return
	}
	if levelPath == "" && flag.NArg() == 1 {
		levelPath = flag.Arg(0)
	}
	if levelPath == "" || (!listPlayers && outputDir == "") || (listPlayers && (playerUID != "" || compareDir != "")) {
		fmt.Fprintln(os.Stderr, "usage: palworld-save-scrap --level <Level.sav> --output <directory> [--player <UID>] [--compare <previous-export>] [--players-dir <Players>] [--force]\n       palworld-save-scrap --level <Level.sav> --list-players [--players-dir <Players>]")
		os.Exit(2)
	}
	levelPath, err := filepath.Abs(levelPath)
	if err != nil {
		fail(err)
	}
	world, err := sav.ParseLevel(levelPath, sav.Options{PlayersDir: playersDir})
	if err != nil {
		fail(fmt.Errorf("read save: %w", err))
	}
	if listPlayers {
		printPlayers(world.Players)
		return
	}
	if !report.HasPlayer(world, playerUID) {
		fail(fmt.Errorf("player %q was not found; use --list-players to choose a player UID", playerUID))
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		fail(err)
	}
	if err := report.ValidateOutputDirectory(levelPath, outputDir, force); err != nil {
		fail(err)
	}
	if err := report.Write(outputDir, world, playerUID, compareDir, force); err != nil {
		fail(err)
	}
	if playerUID == "" {
		fmt.Printf("Exported %d players and %d Pals to %s\n", len(world.Players), len(world.Pals), outputDir)
	} else {
		fmt.Printf("Exported player %s to %s\n", playerUID, outputDir)
	}
}

func printPlayers(players []sav.Player) {
	sort.Slice(players, func(i, j int) bool { return players[i].Nickname < players[j].Nickname })
	if len(players) == 0 {
		fmt.Println("No players found.")
		return
	}
	fmt.Println("PLAYER UID\tLEVEL\tNAME")
	for _, player := range players {
		fmt.Printf("%s\t%d\t%s\n", player.UID, player.Level, player.Nickname)
	}
}

func fail(err error) {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		fmt.Fprintln(os.Stderr, pathErr.Err)
	} else {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(1)
}
