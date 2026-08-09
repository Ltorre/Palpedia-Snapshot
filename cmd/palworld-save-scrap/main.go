package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Ltorre/palworld-save-scrap/internal/report"
	"github.com/Ltorre/palworld-save-scrap/internal/sav"
)

func main() {
	var levelPath, outputDir, playersDir, oodleLibrary string
	var force bool
	flag.StringVar(&levelPath, "level", "", "path to Level.sav")
	flag.StringVar(&outputDir, "output", "", "empty output directory")
	flag.StringVar(&playersDir, "players-dir", "", "path to Players directory (defaults to the sibling Players directory)")
	flag.StringVar(&oodleLibrary, "oodle-lib", "", "absolute path to oo2core_9_win64.dll for PlM saves")
	flag.BoolVar(&force, "force", false, "allow writing into a non-empty output directory")
	flag.Parse()
	if levelPath == "" && flag.NArg() == 1 {
		levelPath = flag.Arg(0)
	}
	if levelPath == "" || outputDir == "" {
		fmt.Fprintln(os.Stderr, "usage: palworld-save-scrap --level <Level.sav> --output <directory> [--players-dir <Players>] [--oodle-lib <dll>] [--force]")
		os.Exit(2)
	}
	if oodleLibrary != "" {
		absolute, err := filepath.Abs(oodleLibrary)
		if err != nil {
			fail(err)
		}
		if err := os.Setenv("PALWORLD_SCRAP_OODLE_LIB", absolute); err != nil {
			fail(err)
		}
	}

	levelPath, err := filepath.Abs(levelPath)
	if err != nil {
		fail(err)
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		fail(err)
	}
	if err := report.ValidateOutputDirectory(levelPath, outputDir, force); err != nil {
		fail(err)
	}

	world, err := sav.ParseLevel(levelPath, sav.Options{PlayersDir: playersDir})
	if err != nil {
		fail(fmt.Errorf("read save: %w", err))
	}
	if err := report.Write(outputDir, world, force); err != nil {
		fail(err)
	}
	fmt.Printf("Exported %d players and %d Pals to %s\n", len(world.Players), len(world.Pals), outputDir)
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
