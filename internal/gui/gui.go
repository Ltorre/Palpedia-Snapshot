//go:build windows

package gui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"

	"github.com/Ltorre/palpedia-snapshot/internal/breeding"
	"github.com/Ltorre/palpedia-snapshot/internal/planner"
	"github.com/Ltorre/palpedia-snapshot/internal/report"
	"github.com/Ltorre/palpedia-snapshot/internal/sav"
)

type language string

const (
	english language = "en"
	french  language = "fr"
)

type taskResult struct {
	kind        string
	candidates  []SaveCandidate
	players     []sav.Player
	plannerPals []planner.Pal
	updatedAt   time.Time
	path        string
	message     string
	err         error
}

type plannerPicker struct {
	pal          planner.Pal
	male, female widget.Clickable
}

type screen struct {
	window   *app.Window
	explorer *explorer.Explorer
	theme    *material.Theme
	version  string
	language language

	root, level, output, players, player, compare                                                        widget.Editor
	advanced                                                                                             widget.Bool
	englishButton, frenchButton, scanButton                                                              widget.Clickable
	browseButton, outputBrowseButton, compareBrowseButton, playersButton, exportButton, openExportButton widget.Clickable
	candidates                                                                                           []SaveCandidate
	candidateButtons                                                                                     []widget.Clickable
	playersFound                                                                                         []sav.Player
	playerButtons                                                                                        []widget.Clickable
	list                                                                                                 layout.List
	results                                                                                              chan taskResult
	busy                                                                                                 bool
	status                                                                                               string
	statusError                                                                                          bool
	lastExportDir                                                                                        string
	plannerRefreshButton, pairButton, routeButton                                                        widget.Clickable
	plannerGold, plannerDiamond                                                                          widget.Bool
	plannerFilter, target                                                                                widget.Editor
	plannerPals                                                                                          []planner.Pal
	plannerPickers                                                                                       map[string]*plannerPicker
	selectedMale, selectedFemale                                                                         *planner.Pal
	plannerLoadedAt                                                                                      time.Time
	plannerSaveModified                                                                                  time.Time
	plannerPairResult, plannerRoute                                                                      string
}

func Run(version string) {
	go func() {
		window := new(app.Window)
		window.Option(app.Title("Palpedia Snapshot"), app.Size(unit.Dp(1080), unit.Dp(760)))
		s := newScreen(window, version)
		for {
			event := window.Event()
			s.explorer.ListenEvents(event)
			switch event := event.(type) {
			case app.DestroyEvent:
				return
			case app.FrameEvent:
				gtx := app.NewContext(new(op.Ops), event)
				s.handle(gtx)
				s.layout(gtx)
				event.Frame(gtx.Ops)
			}
		}
	}()
	app.Main()
}

func newScreen(window *app.Window, version string) *screen {
	theme := material.NewTheme()
	theme.Palette = material.Palette{
		Bg:         color.NRGBA{R: 246, G: 248, B: 252, A: 255},
		Fg:         color.NRGBA{R: 25, G: 32, B: 48, A: 255},
		ContrastBg: color.NRGBA{R: 93, G: 76, B: 205, A: 255},
		ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
	}
	s := &screen{window: window, explorer: explorer.NewExplorer(window), theme: theme, version: version, language: english, results: make(chan taskResult, 4), list: layout.List{Axis: layout.Vertical}}
	for _, field := range []*widget.Editor{&s.root, &s.level, &s.output, &s.players, &s.player, &s.compare, &s.plannerFilter, &s.target} {
		field.SingleLine = true
	}
	s.plannerPickers = make(map[string]*plannerPicker)
	s.root.SetText(DefaultSaveRoot())
	s.output.SetText(DefaultExportDir())
	s.lastExportDir = latestExportDir(s.output.Text())
	s.startScan()
	return s
}

func (s *screen) handle(gtx layout.Context) {
	for {
		select {
		case result := <-s.results:
			s.busy = false
			s.statusError = result.err != nil
			if result.err != nil {
				s.status = result.err.Error()
			} else {
				s.status = result.message
			}
			switch result.kind {
			case "scan":
				s.candidates = result.candidates
				s.candidateButtons = make([]widget.Clickable, len(result.candidates))
			case "browse":
				s.level.SetText(result.path)
			case "output-browse":
				s.output.SetText(result.path)
				if result.path != "" {
					s.lastExportDir = latestExportDir(result.path)
				}
			case "compare-browse":
				s.compare.SetText(result.path)
			case "players":
				s.playersFound = result.players
				s.playerButtons = make([]widget.Clickable, len(result.players))
			case "export":
				if result.err == nil && result.path != "" {
					s.lastExportDir = result.path
					s.startPlannerRefresh()
				}
			case "planner":
				if result.err == nil {
					s.plannerPals, s.plannerLoadedAt, s.plannerSaveModified = result.plannerPals, time.Now(), result.updatedAt
					s.plannerPickers = make(map[string]*plannerPicker)
					s.selectedMale, s.selectedFemale, s.plannerPairResult, s.plannerRoute = nil, nil, "", ""
				}
			}
		default:
			goto handled
		}
	}

handled:
	if s.englishButton.Clicked(gtx) {
		s.language = english
	}
	if s.frenchButton.Clicked(gtx) {
		s.language = french
	}
	if s.scanButton.Clicked(gtx) && !s.busy {
		s.startScan()
	}
	if s.browseButton.Clicked(gtx) && !s.busy {
		s.startBrowse()
	}
	if s.outputBrowseButton.Clicked(gtx) && !s.busy {
		s.startOutputBrowse()
	}
	if s.compareBrowseButton.Clicked(gtx) && !s.busy {
		s.startCompareBrowse()
	}
	if s.playersButton.Clicked(gtx) && !s.busy {
		s.startPlayers()
	}
	if s.exportButton.Clicked(gtx) && !s.busy {
		s.startExport()
	}
	if s.plannerRefreshButton.Clicked(gtx) && !s.busy {
		s.startPlannerRefresh()
	}
	if s.pairButton.Clicked(gtx) {
		s.calculatePair()
	}
	if s.routeButton.Clicked(gtx) {
		s.calculateRoute()
	}
	if s.openExportButton.Clicked(gtx) && s.lastExportDir != "" {
		if err := openFolder(s.lastExportDir); err != nil {
			s.statusError, s.status = true, err.Error()
		}
	}
	for index := range s.candidateButtons {
		if s.candidateButtons[index].Clicked(gtx) {
			s.level.SetText(s.candidates[index].LevelPath)
			s.statusError = false
			s.status = s.t("world_selected")
		}
	}
	for index := range s.playerButtons {
		if s.playerButtons[index].Clicked(gtx) {
			player := s.playersFound[index]
			s.player.SetText(player.UID)
			s.statusError = false
			s.status = fmt.Sprintf(s.t("player_selected"), player.Nickname)
		}
	}
	for _, picker := range s.plannerPickers {
		if picker.male.Clicked(gtx) {
			pal := picker.pal
			s.selectedMale, s.plannerPairResult, s.plannerRoute = &pal, "", ""
		}
		if picker.female.Clicked(gtx) {
			pal := picker.pal
			s.selectedFemale, s.plannerPairResult, s.plannerRoute = &pal, "", ""
		}
	}
}

func (s *screen) startScan() {
	root := strings.TrimSpace(s.root.Text())
	if root == "" {
		s.statusError, s.status = true, s.t("save_root_required")
		return
	}
	s.busy, s.statusError, s.status = true, false, s.t("scanning")
	go func() {
		candidates, err := FindSaves(root)
		message := s.t("no_saves")
		if err == nil && len(candidates) > 0 {
			message = fmt.Sprintf(s.t("saves_found"), len(candidates))
		}
		s.results <- taskResult{kind: "scan", candidates: candidates, message: message, err: err}
		s.window.Invalidate()
	}()
}

func (s *screen) startBrowse() {
	s.busy, s.statusError, s.status = true, false, s.t("opening_browser")
	go func() {
		file, err := s.explorer.ChooseFile(".sav")
		if err != nil {
			s.results <- taskResult{kind: "browse", err: err}
			s.window.Invalidate()
			return
		}
		defer file.Close()
		named, ok := file.(interface{ Name() string })
		if !ok {
			s.results <- taskResult{kind: "browse", err: fmt.Errorf("selected file path is unavailable")}
			s.window.Invalidate()
			return
		}
		s.results <- taskResult{kind: "browse", path: named.Name(), message: s.t("world_selected")}
		s.window.Invalidate()
	}()
}

func (s *screen) startPlayers() {
	levelPath := strings.TrimSpace(s.level.Text())
	if levelPath == "" {
		s.statusError, s.status = true, s.t("select_save_first")
		return
	}
	playersDir := strings.TrimSpace(s.players.Text())
	s.busy, s.statusError, s.status = true, false, s.t("reading_players")
	go func() {
		players, err := readPlayers(levelPath, playersDir)
		message := s.t("no_players")
		if err == nil && len(players) > 0 {
			message = fmt.Sprintf(s.t("players_found"), len(players))
		}
		s.results <- taskResult{kind: "players", players: players, message: message, err: err}
		s.window.Invalidate()
	}()
}

func (s *screen) startOutputBrowse() {
	s.busy, s.statusError, s.status = true, false, s.t("opening_export_browser")
	go func() {
		path, err := chooseFolder(s.t("choose_export_folder"))
		if err == errFolderSelectionAborted {
			s.results <- taskResult{kind: "output-browse", message: s.t("export_folder_unchanged")}
			s.window.Invalidate()
			return
		}
		s.results <- taskResult{kind: "output-browse", path: path, message: s.t("export_folder_selected"), err: err}
		s.window.Invalidate()
	}()
}

func (s *screen) startCompareBrowse() {
	s.busy, s.statusError, s.status = true, false, s.t("opening_compare_browser")
	go func() {
		path, err := chooseFolder(s.t("choose_compare_folder"))
		if err == errFolderSelectionAborted {
			s.results <- taskResult{kind: "compare-browse", message: s.t("compare_folder_unchanged")}
			s.window.Invalidate()
			return
		}
		s.results <- taskResult{kind: "compare-browse", path: path, message: s.t("compare_folder_selected"), err: err}
		s.window.Invalidate()
	}()
}

func (s *screen) startExport() {
	levelPath, outputDir := strings.TrimSpace(s.level.Text()), strings.TrimSpace(s.output.Text())
	if levelPath == "" || outputDir == "" {
		s.statusError, s.status = true, s.t("save_and_output_required")
		return
	}
	playersDir := strings.TrimSpace(s.players.Text())
	playerUID, compareDir := strings.TrimSpace(s.player.Text()), strings.TrimSpace(s.compare.Text())
	s.busy, s.statusError, s.status = true, false, s.t("exporting")
	go func() {
		message, exportDir, err := export(levelPath, outputDir, playersDir, playerUID, compareDir)
		s.results <- taskResult{kind: "export", message: message, path: exportDir, err: err}
		s.window.Invalidate()
	}()
}

func (s *screen) startPlannerRefresh() {
	levelPath := strings.TrimSpace(s.level.Text())
	if levelPath == "" {
		s.statusError, s.status = true, s.t("planner_select_save")
		return
	}
	playersDir, playerUID := strings.TrimSpace(s.players.Text()), strings.TrimSpace(s.player.Text())
	s.busy, s.statusError, s.status = true, false, s.t("planner_updating")
	go func() {
		world, err := loadWorld(levelPath, playersDir)
		if err == nil && !report.HasPlayer(world, playerUID) {
			err = fmt.Errorf("player %q was not found", playerUID)
		}
		var pals []planner.Pal
		if err == nil {
			pals = plannerCollection(world, playerUID)
		}
		var updatedAt time.Time
		if info, statErr := os.Stat(levelPath); statErr == nil {
			updatedAt = info.ModTime()
		}
		message := s.t("planner_ready")
		if err == nil {
			message = fmt.Sprintf(s.t("planner_loaded"), len(pals))
		}
		s.results <- taskResult{kind: "planner", plannerPals: pals, updatedAt: updatedAt, message: message, err: err}
		s.window.Invalidate()
	}()
}

func plannerCollection(world *sav.World, playerUID string) []planner.Pal {
	containers := make(map[string]bool, len(world.Players)*2)
	for _, player := range world.Players {
		if playerUID != "" && !strings.EqualFold(player.UID, playerUID) {
			continue
		}
		containers[strings.ToLower(player.OtomoContainerID)] = true
		containers[strings.ToLower(player.PalStorageContainerID)] = true
	}
	pals := make([]planner.Pal, 0, len(world.Pals))
	for _, pal := range world.Pals {
		if !containers[strings.ToLower(pal.ContainerID)] {
			continue
		}
		pals = append(pals, planner.Pal{InstanceID: pal.InstanceID, CharacterID: pal.CharacterID, Gender: pal.Gender, Level: pal.Level, Traits: append([]string(nil), pal.PassiveSkillIDs...)})
	}
	return pals
}

func (s *screen) calculatePair() {
	if s.selectedMale == nil || s.selectedFemale == nil {
		s.statusError, s.status = true, s.t("planner_select_parents")
		return
	}
	rules, err := breeding.Default()
	if err == nil {
		result, resolveErr := planner.ResolvePair(rules, *s.selectedMale, *s.selectedFemale)
		if resolveErr != nil {
			err = resolveErr
		} else {
			s.plannerPairResult = fmt.Sprintf(s.t("planner_pair_result"), result.Child, result.Rule, result.TargetRank)
			s.statusError, s.status = false, s.t("planner_pair_ready")
		}
	}
	if err != nil {
		s.statusError, s.status = true, err.Error()
	}
}

func (s *screen) calculateRoute() {
	if len(s.plannerPals) == 0 {
		s.statusError, s.status = true, s.t("planner_empty")
		return
	}
	target := strings.TrimSpace(s.target.Text())
	if target == "" {
		s.statusError, s.status = true, s.t("planner_target_required")
		return
	}
	rules, err := breeding.Default()
	if err != nil {
		s.statusError, s.status = true, err.Error()
		return
	}
	path, err := planner.ShortestPath(rules, s.plannerPals, target)
	if err != nil {
		s.statusError, s.status = true, err.Error()
		return
	}
	var out strings.Builder
	fmt.Fprintf(&out, s.t("planner_route_title"), path.Target, path.Generations)
	if len(path.Steps) == 0 {
		out.WriteString("\n" + s.t("planner_already_owned"))
	} else {
		for index, step := range path.Steps {
			fmt.Fprintf(&out, "\n%d. %s + %s → %s (%s)", index+1, step.ParentA, step.ParentB, step.Child, step.Rule)
		}
	}
	if path.Generations > 2 {
		out.WriteString("\n\n" + s.t("planner_speed_title"))
		helpers := planner.BreedingSpeedHelpers(s.plannerPals)
		if len(helpers) == 0 {
			out.WriteString("\n" + s.t("planner_speed_none"))
		}
		for _, helper := range helpers {
			label := plannerPalLabel(helper.Pal)
			switch helper.TraitID {
			case "MutationPal_Babysitter":
				fmt.Fprintf(&out, "\n• "+s.t("planner_speed_babysitter"), label)
			case "Test_PalEgg_HatchingSpeed_Up":
				fmt.Fprintf(&out, "\n• "+s.t("planner_speed_philanthropist"), label)
			}
		}
	}
	out.WriteString("\n" + s.t("planner_route_caveat"))
	s.plannerRoute, s.statusError, s.status = out.String(), false, s.t("planner_route_ready")
}

func export(levelPath, outputParent, playersDir, playerUID, compareDir string) (string, string, error) {
	levelPath, err := filepath.Abs(levelPath)
	if err != nil {
		return "", "", err
	}
	outputParent, err = filepath.Abs(outputParent)
	if err != nil {
		return "", "", err
	}
	if err := report.ValidateOutputParent(levelPath, outputParent); err != nil {
		return "", "", err
	}
	world, err := loadWorld(levelPath, playersDir)
	if err != nil {
		return "", "", err
	}
	if !report.HasPlayer(world, playerUID) {
		return "", "", fmt.Errorf("player %q was not found", playerUID)
	}
	exportDir, err := report.CreateExportDirectory(outputParent, time.Now())
	if err != nil {
		return "", "", err
	}
	if err := report.Write(exportDir, world, playerUID, compareDir, false); err != nil {
		return "", "", err
	}
	return fmt.Sprintf("Exported %d Pals to %s", len(world.Pals), exportDir), exportDir, nil
}

func latestExportDir(parent string) string {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return ""
	}
	var latest string
	var latestTime time.Time
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "export_") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr == nil && info.ModTime().After(latestTime) {
			latest, latestTime = filepath.Join(parent, entry.Name()), info.ModTime()
		}
	}
	return latest
}

func readPlayers(levelPath, playersDir string) ([]sav.Player, error) {
	world, err := loadWorld(levelPath, playersDir)
	if err != nil {
		return nil, err
	}
	players := append([]sav.Player(nil), world.Players...)
	sort.Slice(players, func(i, j int) bool {
		return strings.ToLower(players[i].Nickname) < strings.ToLower(players[j].Nickname)
	})
	return players, nil
}

func loadWorld(levelPath, playersDir string) (*sav.World, error) {
	levelPath, err := filepath.Abs(levelPath)
	if err != nil {
		return nil, err
	}
	world, err := sav.ParseLevel(levelPath, sav.Options{PlayersDir: playersDir})
	if err != nil {
		return nil, fmt.Errorf("read save: %w", err)
	}
	return world, nil
}

func (s *screen) layout(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(18), Bottom: unit.Dp(18), Left: unit.Dp(28), Right: unit.Dp(28)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return s.list.Layout(gtx, 1, func(gtx layout.Context, _ int) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(s.header), layout.Rigid(spacer(14)), layout.Rigid(s.saveSection), layout.Rigid(spacer(12)), layout.Rigid(s.exportSection), layout.Rigid(spacer(12)), layout.Rigid(s.plannerSection), layout.Rigid(spacer(12)), layout.Rigid(s.statusSection),
			)
		})
	})
}

func (s *screen) header(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.H4(s.theme, s.t("title"))
					l.Color = color.NRGBA{R: 60, G: 48, B: 140, A: 255}
					return l.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Body2(s.theme, fmt.Sprintf("%s · %s", s.t("subtitle"), s.version))
					l.Color = color.NRGBA{R: 85, G: 91, B: 110, A: 255}
					return l.Layout(gtx)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return s.languageButton(gtx, &s.englishButton, "EN", s.language == english)
				}), layout.Rigid(spacer(6)), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return s.languageButton(gtx, &s.frenchButton, "FR", s.language == french)
				}),
			)
		}),
	)
}

func (s *screen) languageButton(gtx layout.Context, button *widget.Clickable, label string, active bool) layout.Dimensions {
	style := material.Button(s.theme, button, label)
	if !active {
		style.Background, style.Color = color.NRGBA{R: 224, G: 226, B: 236, A: 255}, color.NRGBA{R: 63, G: 67, B: 85, A: 255}
	}
	return style.Layout(gtx)
}

func (s *screen) saveSection(gtx layout.Context) layout.Dimensions {
	return section(gtx, s.theme, s.t("find_save"), func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(s.caption("save_root_help")), layout.Rigid(spacer(6)), layout.Rigid(s.editor(&s.root, s.t("save_root"))), layout.Rigid(spacer(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Button(s.theme, &s.scanButton, s.t("scan")).Layout(gtx)
				}), layout.Rigid(spacer(8)), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Button(s.theme, &s.browseButton, s.t("browse_level")).Layout(gtx)
				}))
			}),
			layout.Rigid(spacer(10)), layout.Rigid(s.editor(&s.level, s.t("selected_level"))), layout.Rigid(spacer(8)), layout.Rigid(s.candidatesList),
		)
	})
}

func (s *screen) candidatesList(gtx layout.Context) layout.Dimensions {
	if len(s.candidates) == 0 {
		return s.caption("no_candidates")(gtx)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Subtitle2(s.theme, s.t("detected_worlds")).Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			list := layout.List{Axis: layout.Vertical}
			return list.Layout(gtx, len(s.candidates), func(gtx layout.Context, index int) layout.Dimensions {
				candidate := s.candidates[index]
				label := fmt.Sprintf("%s  ·  %s", filepath.Base(candidate.WorldDir), candidate.UpdatedAt.Local().Format("2006-01-02 15:04"))
				style := material.Button(s.theme, &s.candidateButtons[index], label)
				style.Background, style.Color = color.NRGBA{R: 232, G: 235, B: 250, A: 255}, color.NRGBA{R: 48, G: 54, B: 82, A: 255}
				return layout.Inset{Bottom: unit.Dp(5)}.Layout(gtx, style.Layout)
			})
		}),
	)
}

func (s *screen) exportSection(gtx layout.Context) layout.Dimensions {
	return section(gtx, s.theme, s.t("export"), func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{layout.Rigid(s.caption("output_help")), layout.Rigid(s.caption("snapshot_help")), layout.Rigid(s.caption("notebooklm_files")), layout.Rigid(spacer(6)), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, s.editor(&s.output, s.t("output_directory"))),
				layout.Rigid(spacer(8)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Button(s.theme, &s.outputBrowseButton, s.t("browse_output")).Layout(gtx)
				}),
			)
		}), layout.Rigid(spacer(10)), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.CheckBox(s.theme, &s.advanced, s.t("advanced_options")).Layout(gtx)
		})}
		if s.advanced.Value {
			children = append(children,
				layout.Rigid(s.caption("advanced_help")), layout.Rigid(spacer(6)),
				layout.Rigid(s.editor(&s.players, s.t("players_directory"))), layout.Rigid(s.caption("players_help")),
				layout.Rigid(spacer(4)), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Button(s.theme, &s.playersButton, s.t("find_players")).Layout(gtx)
				}), layout.Rigid(s.playersList),
				layout.Rigid(s.editor(&s.player, s.t("player_uid"))), layout.Rigid(s.caption("player_help")),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, s.editor(&s.compare, s.t("compare_directory"))),
						layout.Rigid(spacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Button(s.theme, &s.compareBrowseButton, s.t("browse_compare")).Layout(gtx)
						}),
					)
				}), layout.Rigid(s.caption("compare_help")),
			)
		}
		children = append(children, layout.Rigid(spacer(12)), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			button := material.Button(s.theme, &s.exportButton, s.t("export_button"))
			button.Background = color.NRGBA{R: 32, G: 125, B: 104, A: 255}
			return button.Layout(gtx)
		}))
		if s.lastExportDir != "" {
			children = append(children, layout.Rigid(spacer(8)), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Button(s.theme, &s.openExportButton, s.t("open_export_folder")).Layout(gtx)
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

func (s *screen) plannerSection(gtx layout.Context) layout.Dimensions {
	return section(gtx, s.theme, s.t("planner_title"), func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{
			layout.Rigid(s.caption("planner_help")),
			layout.Rigid(spacer(6)),
			layout.Rigid(s.plannerFreshness),
			layout.Rigid(spacer(6)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				button := material.Button(s.theme, &s.plannerRefreshButton, s.t("planner_refresh"))
				button.Background = color.NRGBA{R: 46, G: 103, B: 171, A: 255}
				return button.Layout(gtx)
			}),
		}
		if len(s.plannerPals) == 0 {
			children = append(children, layout.Rigid(spacer(6)), layout.Rigid(s.caption("planner_empty")))
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}
		children = append(children,
			layout.Rigid(spacer(12)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Subtitle2(s.theme, s.t("planner_pick_title")).Layout(gtx)
			}),
			layout.Rigid(s.caption("planner_filter_help")),
			layout.Rigid(s.editor(&s.plannerFilter, s.t("planner_filter"))),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.CheckBox(s.theme, &s.plannerGold, s.t("planner_gold")).Layout(gtx)
					}),
					layout.Rigid(spacer(12)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.CheckBox(s.theme, &s.plannerDiamond, s.t("planner_diamond")).Layout(gtx)
					}),
				)
			}),
			layout.Rigid(spacer(4)),
			layout.Rigid(s.plannerPalsList),
			layout.Rigid(spacer(8)),
			layout.Rigid(s.plannerSelection),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				button := material.Button(s.theme, &s.pairButton, s.t("planner_calculate_pair"))
				button.Background = color.NRGBA{R: 127, G: 83, B: 187, A: 255}
				return button.Layout(gtx)
			}),
		)
		if s.plannerPairResult != "" {
			children = append(children, layout.Rigid(spacer(5)), layout.Rigid(s.note(s.plannerPairResult, color.NRGBA{R: 58, G: 48, B: 132, A: 255})))
		}
		children = append(children,
			layout.Rigid(spacer(12)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Subtitle2(s.theme, s.t("planner_route_section")).Layout(gtx)
			}),
			layout.Rigid(s.caption("planner_route_help")),
			layout.Rigid(s.editor(&s.target, s.t("planner_target"))),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				button := material.Button(s.theme, &s.routeButton, s.t("planner_find_route"))
				button.Background = color.NRGBA{R: 32, G: 125, B: 104, A: 255}
				return button.Layout(gtx)
			}),
		)
		if s.plannerRoute != "" {
			children = append(children, layout.Rigid(spacer(5)), layout.Rigid(s.note(s.plannerRoute, color.NRGBA{R: 20, G: 88, B: 72, A: 255})))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

func (s *screen) plannerFreshness(gtx layout.Context) layout.Dimensions {
	if s.plannerLoadedAt.IsZero() {
		return s.note(s.t("planner_not_loaded"), color.NRGBA{R: 91, G: 98, B: 117, A: 255})(gtx)
	}
	message := fmt.Sprintf(s.t("planner_freshness"), len(s.plannerPals), s.plannerLoadedAt.Local().Format("2006-01-02 15:04"), s.plannerSaveModified.Local().Format("2006-01-02 15:04"))
	if s.lastExportDir != "" {
		message += "\n" + fmt.Sprintf(s.t("planner_last_export"), filepath.Base(s.lastExportDir))
	}
	return s.note(message, color.NRGBA{R: 39, G: 91, B: 76, A: 255})(gtx)
}

func (s *screen) plannerPalsList(gtx layout.Context) layout.Dimensions {
	visible := planner.Filter(s.plannerPals, s.plannerFilter.Text(), s.plannerGold.Value, s.plannerDiamond.Value)
	total := len(visible)
	if len(visible) == 0 {
		return s.caption("planner_no_matches")(gtx)
	}
	const limit = 40
	if len(visible) > limit {
		visible = visible[:limit]
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(s.note(fmt.Sprintf(s.t("planner_showing"), len(visible), total), color.NRGBA{R: 91, G: 98, B: 117, A: 255})),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			list := layout.List{Axis: layout.Vertical}
			return list.Layout(gtx, len(visible), func(gtx layout.Context, index int) layout.Dimensions {
				pal := visible[index]
				picker := s.plannerPicker(pal)
				return layout.Inset{Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, s.note(plannerPalLabel(pal), color.NRGBA{R: 48, G: 54, B: 82, A: 255})),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if strings.EqualFold(pal.Gender, "male") {
								return material.Button(s.theme, &picker.male, s.t("planner_choose_male")).Layout(gtx)
							}
							if strings.EqualFold(pal.Gender, "female") {
								return material.Button(s.theme, &picker.female, s.t("planner_choose_female")).Layout(gtx)
							}
							return layout.Dimensions{}
						}),
					)
				})
			})
		}),
	)
}

func (s *screen) plannerPicker(pal planner.Pal) *plannerPicker {
	key := strings.Join([]string{pal.InstanceID, pal.CharacterID, pal.Gender}, "\x00")
	if picker, ok := s.plannerPickers[key]; ok {
		return picker
	}
	picker := &plannerPicker{pal: pal}
	s.plannerPickers[key] = picker
	return picker
}

func (s *screen) plannerSelection(gtx layout.Context) layout.Dimensions {
	male, female := s.t("planner_no_parent"), s.t("planner_no_parent")
	if s.selectedMale != nil {
		male = plannerPalLabel(*s.selectedMale)
	}
	if s.selectedFemale != nil {
		female = plannerPalLabel(*s.selectedFemale)
	}
	return s.note(fmt.Sprintf(s.t("planner_selected"), male, female), color.NRGBA{R: 61, G: 67, B: 93, A: 255})(gtx)
}

func plannerPalLabel(pal planner.Pal) string {
	traits := planner.TraitSummary(pal.Traits)
	if traits == "" {
		traits = "—"
	}
	return fmt.Sprintf("%s · Lv. %d · %s · %s", pal.CharacterID, pal.Level, pal.Gender, traits)
}

func (s *screen) playersList(gtx layout.Context) layout.Dimensions {
	if len(s.playersFound) == 0 {
		return layout.Dimensions{}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Subtitle2(s.theme, s.t("available_players")).Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			list := layout.List{Axis: layout.Vertical}
			return list.Layout(gtx, len(s.playersFound), func(gtx layout.Context, index int) layout.Dimensions {
				player := s.playersFound[index]
				style := material.Button(s.theme, &s.playerButtons[index], fmt.Sprintf("%s  ·  Lv. %d", player.Nickname, player.Level))
				style.Background, style.Color = color.NRGBA{R: 232, G: 235, B: 250, A: 255}, color.NRGBA{R: 48, G: 54, B: 82, A: 255}
				return layout.Inset{Bottom: unit.Dp(5)}.Layout(gtx, style.Layout)
			})
		}),
	)
}

func (s *screen) statusSection(gtx layout.Context) layout.Dimensions {
	if s.status == "" {
		return layout.Dimensions{}
	}
	label := material.Body2(s.theme, s.status)
	if s.statusError {
		label.Color = color.NRGBA{R: 180, G: 43, B: 43, A: 255}
	} else {
		label.Color = color.NRGBA{R: 22, G: 104, B: 83, A: 255}
	}
	return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(8)}.Layout(gtx, label.Layout)
}

func (s *screen) editor(editor *widget.Editor, hint string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		style := material.Editor(s.theme, editor, hint)
		style.TextSize = unit.Sp(15)
		return style.Layout(gtx)
	}
}

func (s *screen) caption(key string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		label := material.Caption(s.theme, s.t(key))
		label.Color = color.NRGBA{R: 91, G: 98, B: 117, A: 255}
		label.WrapPolicy = text.WrapWords
		return label.Layout(gtx)
	}
}

func (s *screen) note(value string, foreground color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		label := material.Body2(s.theme, value)
		label.Color = foreground
		label.WrapPolicy = text.WrapWords
		return label.Layout(gtx)
	}
}

func section(gtx layout.Context, theme *material.Theme, title string, content layout.Widget) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.H6(theme, title)
			label.Color = color.NRGBA{R: 43, G: 49, B: 70, A: 255}
			return label.Layout(gtx)
		}), layout.Rigid(spacer(6)), layout.Rigid(content))
	})
}

func spacer(size unit.Dp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: image.Pt(0, gtx.Dp(size))} }
}

func (s *screen) t(key string) string { return translations[s.language][key] }

var translations = map[language]map[string]string{
	english: {
		"title": "Palpedia Snapshot", "subtitle": "Personal Palpedia export", "find_save": "1. Find your save", "save_root": "Default Palworld save folder", "save_root_help": "Starts at the standard Windows Palworld save location. You may replace it with any folder containing your saves.", "scan": "Find worlds", "browse_level": "Browse for Level.sav", "selected_level": "Selected Level.sav", "no_candidates": "No world found yet. Scan the folder or browse directly to Level.sav.", "detected_worlds": "Detected worlds", "export": "2. Export for NotebookLM", "output_directory": "Export parent folder", "browse_output": "Choose folder", "choose_export_folder": "Choose an export folder", "opening_export_browser": "Opening the folder browser…", "export_folder_selected": "Export folder selected.", "export_folder_unchanged": "Export folder unchanged.", "output_help": "Required. Choose a parent folder; the tool writes only here and never inside your game save folder.", "snapshot_help": "Each export is saved in its own folder, for example export_08-09-2026 18-42. Windows cannot use : in a folder name.", "notebooklm_files": "Add to NotebookLM: collection.md, pals.csv, capture-history.csv, palpedia-progress.md, breeding-candidates.md, and collection-diff.md when comparing. Do not add world.json.", "advanced_options": "Show optional advanced options", "advanced_help": "Use these only for shared worlds, custom save layouts, or comparing this snapshot with an earlier export.", "players_directory": "Players directory (optional)", "players_help": "Only needed when Players is not beside Level.sav.", "find_players": "Find players in this save", "available_players": "Available players", "select_save_first": "Select a Level.sav file first.", "reading_players": "Reading the players in this save…", "no_players": "No players were found in this save.", "players_found": "%d player(s) found. Select one to export only that player, or leave it empty for everyone.", "player_selected": "%s selected. Clear the Player UID field to export all players.", "player_uid": "Player UID (optional)", "player_help": "Choose a detected player to export only their collection. Leave empty to export every player in the world.", "compare_directory": "Previous export folder (optional)", "browse_compare": "Choose previous export", "choose_compare_folder": "Choose the earlier export folder", "opening_compare_browser": "Opening the previous-export folder browser…", "compare_folder_selected": "Previous export folder selected.", "compare_folder_unchanged": "Previous export folder unchanged.", "compare_help": "Adds collection-diff.md using a previous export_<date time> folder.", "export_button": "Create NotebookLM export", "save_root_required": "Choose a save folder first.", "scanning": "Scanning for Palworld worlds…", "no_saves": "No Level.sav file was found in this folder.", "saves_found": "%d world(s) found. Select one below.", "opening_browser": "Opening the file browser…", "world_selected": "World selected. Choose an export parent folder and create the export.", "save_and_output_required": "Select a Level.sav file and an export parent folder.", "exporting": "Reading the save and creating a new export snapshot…", "open_export_folder": "Open export folder in Explorer",
	},
	french: {
		"title": "Palpedia Snapshot", "subtitle": "Export personnel pour le Palpédia", "find_save": "1. Trouver votre sauvegarde", "save_root": "Dossier de sauvegarde Palworld par défaut", "save_root_help": "Commence dans le dossier Windows standard de Palworld. Vous pouvez le remplacer par tout dossier contenant vos sauvegardes.", "scan": "Chercher les mondes", "browse_level": "Parcourir Level.sav", "selected_level": "Level.sav sélectionné", "no_candidates": "Aucun monde trouvé. Cherchez dans le dossier ou choisissez directement Level.sav.", "detected_worlds": "Mondes détectés", "export": "2. Exporter pour NotebookLM", "output_directory": "Dossier parent des exports", "browse_output": "Choisir un dossier", "choose_export_folder": "Choisir un dossier d’export", "opening_export_browser": "Ouverture du navigateur de dossiers…", "export_folder_selected": "Dossier d’export sélectionné.", "export_folder_unchanged": "Dossier d’export inchangé.", "output_help": "Obligatoire. Choisissez un dossier parent ; l’outil écrit uniquement ici, jamais dans le dossier de sauvegarde du jeu.", "snapshot_help": "Chaque export est créé dans son propre dossier, par exemple export_08-09-2026 18-42. Windows interdit : dans les noms de dossiers.", "notebooklm_files": "À ajouter à NotebookLM : collection.md, pals.csv, capture-history.csv, palpedia-progress.md, breeding-candidates.md et collection-diff.md lors d’une comparaison. Ne pas ajouter world.json.", "advanced_options": "Afficher les options avancées facultatives", "advanced_help": "Utilisez-les seulement pour les mondes partagés, les emplacements personnalisés ou la comparaison avec un export antérieur.", "players_directory": "Dossier Players (facultatif)", "players_help": "Nécessaire uniquement si Players n’est pas à côté de Level.sav.", "find_players": "Chercher les joueurs de cette sauvegarde", "available_players": "Joueurs disponibles", "select_save_first": "Sélectionnez d’abord un fichier Level.sav.", "reading_players": "Lecture des joueurs de cette sauvegarde…", "no_players": "Aucun joueur trouvé dans cette sauvegarde.", "players_found": "%d joueur(s) trouvé(s). Sélectionnez-en un pour n’exporter que sa collection, ou laissez vide pour tous les joueurs.", "player_selected": "%s sélectionné. Videz le champ UID du joueur pour exporter tous les joueurs.", "player_uid": "UID du joueur (facultatif)", "player_help": "Choisissez un joueur détecté pour n’exporter que sa collection. Laissez vide pour exporter tous les joueurs du monde.", "compare_directory": "Dossier d’export précédent (facultatif)", "browse_compare": "Choisir l’export précédent", "choose_compare_folder": "Choisir le dossier de l’export antérieur", "opening_compare_browser": "Ouverture du navigateur d’exports précédents…", "compare_folder_selected": "Dossier d’export précédent sélectionné.", "compare_folder_unchanged": "Dossier d’export précédent inchangé.", "compare_help": "Ajoute collection-diff.md à partir d’un dossier export_<date heure> antérieur.", "export_button": "Créer l’export NotebookLM", "save_root_required": "Choisissez d’abord un dossier de sauvegarde.", "scanning": "Recherche des mondes Palworld…", "no_saves": "Aucun fichier Level.sav trouvé dans ce dossier.", "saves_found": "%d monde(s) trouvé(s). Sélectionnez-en un ci-dessous.", "opening_browser": "Ouverture du navigateur de fichiers…", "world_selected": "Monde sélectionné. Choisissez un dossier parent puis créez l’export.", "save_and_output_required": "Sélectionnez un fichier Level.sav et un dossier parent d’exports.", "exporting": "Lecture de la sauvegarde et création d’un nouvel instantané…", "open_export_folder": "Ouvrir le dossier d’export dans l’Explorateur",
	},
}

func init() {
	translations[english]["notebooklm_files"] = "Create your own notebook at notebook.google.com. First upload the 31 reference Markdown files from palpedia-snapshot-notebooklm-reference.zip, then add: collection.md, pals.csv, capture-history.csv, palpedia-progress.md, breeding-candidates.md, breeding-rules.md, breeding-direct-pairs.csv, and collection-diff.md when comparing. Do not add world.json."
	translations[french]["notebooklm_files"] = "Créez votre propre notebook sur notebook.google.com. Importez d’abord les 31 fichiers Markdown de référence depuis palpedia-snapshot-notebooklm-reference.zip, puis ajoutez : collection.md, pals.csv, capture-history.csv, palpedia-progress.md, breeding-candidates.md, breeding-rules.md, breeding-direct-pairs.csv et collection-diff.md lors d’une comparaison. Ne pas ajouter world.json."
	translations[english]["planner_title"] = "3. Breeding planner"
	translations[english]["planner_help"] = "Load the selected save to choose actual Pals, calculate an exact pair, or plan the fastest breeding-generation route to a target. This is read-only."
	translations[english]["planner_not_loaded"] = "No collection is loaded for planning yet."
	translations[english]["planner_freshness"] = "Planner collection: %d current Pals · loaded %s · selected save last changed %s."
	translations[english]["planner_last_export"] = "Latest NotebookLM export in the selected folder: %s"
	translations[english]["planner_refresh"] = "Update planner from selected save"
	translations[english]["planner_select_save"] = "Select a Level.sav before updating the planner."
	translations[english]["planner_updating"] = "Reading the selected save for the breeding planner…"
	translations[english]["planner_ready"] = "Breeding planner is ready."
	translations[english]["planner_loaded"] = "Loaded %d current Pals into the breeding planner."
	translations[english]["planner_empty"] = "No current party or Palbox Pals are loaded. Update the planner from a selected save."
	translations[english]["planner_pick_title"] = "Pick real parents"
	translations[english]["planner_filter_help"] = "Search by Pal, raw trait ID, or known trait name. Gold is rank 3; diamond is rank 4. If both are checked, either tier is included."
	translations[english]["planner_filter"] = "Filter Pals or traits"
	translations[english]["planner_gold"] = "Gold traits (rank 3)"
	translations[english]["planner_diamond"] = "Diamond traits (rank 4)"
	translations[english]["planner_no_matches"] = "No loaded Pals match these filters."
	translations[english]["planner_showing"] = "Showing %d of %d matching Pals (refine the filter to see more)."
	translations[english]["planner_choose_male"] = "Choose male"
	translations[english]["planner_choose_female"] = "Choose female"
	translations[english]["planner_no_parent"] = "Not selected"
	translations[english]["planner_selected"] = "Male parent: %s\nFemale parent: %s"
	translations[english]["planner_calculate_pair"] = "Calculate selected pair"
	translations[english]["planner_select_parents"] = "Choose one male parent and one female parent first."
	translations[english]["planner_pair_result"] = "Exact child: %s · rule: %s · generic target rank: %d"
	translations[english]["planner_pair_ready"] = "Exact breeding outcome calculated."
	translations[english]["planner_route_section"] = "Find the quickest target route"
	translations[english]["planner_route_help"] = "Enter a Pal Character ID, for example Anubis, SheepBall, or PinkCat. The route uses the fewest sequential breeding generations from your current male/female collection."
	translations[english]["planner_target"] = "Target Pal Character ID"
	translations[english]["planner_find_route"] = "Find quickest breeding route"
	translations[english]["planner_target_required"] = "Enter a target Pal Character ID."
	translations[english]["planner_route_title"] = "Route to %s · %d breeding generation(s)"
	translations[english]["planner_already_owned"] = "Already owned in the loaded collection; no breeding step is required."
	translations[english]["planner_speed_title"] = "Speed up this long route"
	translations[english]["planner_speed_none"] = "No Philanthropist or Babysitter Pal was found in the loaded party/Palbox collection."
	translations[english]["planner_speed_philanthropist"] = "%s — assign to the Breeding Farm: Philanthropist increases that Pal's breeding speed by 100%."
	translations[english]["planner_speed_babysitter"] = "%s — keep at the base: Babysitter improves Breeding Farm egg production and incubation speed by 30%."
	translations[english]["planner_route_caveat"] = "This is a species route. Passive inheritance and egg-gender RNG are not guaranteed; breed extra eggs when a later step needs a specific sex."
	translations[english]["planner_route_ready"] = "Quickest breeding route calculated."

	translations[french]["planner_title"] = "3. Planificateur d’élevage"
	translations[french]["planner_help"] = "Chargez la sauvegarde sélectionnée pour choisir vos vrais Pals, calculer une paire exacte ou trouver le chemin le plus rapide vers une cible. Lecture seule."
	translations[french]["planner_not_loaded"] = "Aucune collection n’est encore chargée pour le planificateur."
	translations[french]["planner_freshness"] = "Collection du planificateur : %d Pals actuels · chargée à %s · sauvegarde sélectionnée modifiée à %s."
	translations[french]["planner_last_export"] = "Dernier export NotebookLM du dossier sélectionné : %s"
	translations[french]["planner_refresh"] = "Mettre à jour depuis la sauvegarde sélectionnée"
	translations[french]["planner_select_save"] = "Sélectionnez un Level.sav avant de mettre à jour le planificateur."
	translations[french]["planner_updating"] = "Lecture de la sauvegarde sélectionnée pour le planificateur…"
	translations[french]["planner_ready"] = "Le planificateur d’élevage est prêt."
	translations[french]["planner_loaded"] = "%d Pals actuels chargés dans le planificateur."
	translations[french]["planner_empty"] = "Aucun Pal de l’équipe ou du Palbox n’est chargé. Mettez à jour depuis une sauvegarde sélectionnée."
	translations[french]["planner_pick_title"] = "Choisir les vrais parents"
	translations[french]["planner_filter_help"] = "Recherchez un Pal, un identifiant de trait ou un nom de trait connu. Or = rang 3 ; diamant = rang 4. Avec les deux cochés, les deux rangs sont inclus."
	translations[french]["planner_filter"] = "Filtrer les Pals ou traits"
	translations[french]["planner_gold"] = "Traits or (rang 3)"
	translations[french]["planner_diamond"] = "Traits diamant (rang 4)"
	translations[french]["planner_no_matches"] = "Aucun Pal chargé ne correspond à ces filtres."
	translations[french]["planner_showing"] = "%d Pals affichés sur %d correspondant(s) (affinez le filtre pour en voir plus)."
	translations[french]["planner_choose_male"] = "Choisir le mâle"
	translations[french]["planner_choose_female"] = "Choisir la femelle"
	translations[french]["planner_no_parent"] = "Non sélectionné"
	translations[french]["planner_selected"] = "Parent mâle : %s\nParent femelle : %s"
	translations[french]["planner_calculate_pair"] = "Calculer la paire sélectionnée"
	translations[french]["planner_select_parents"] = "Choisissez d’abord un parent mâle et un parent femelle."
	translations[french]["planner_pair_result"] = "Enfant exact : %s · règle : %s · rang cible générique : %d"
	translations[french]["planner_pair_ready"] = "Résultat exact de l’élevage calculé."
	translations[french]["planner_route_section"] = "Trouver le chemin le plus rapide"
	translations[french]["planner_route_help"] = "Entrez un Character ID de Pal, par exemple Anubis, SheepBall ou PinkCat. Le chemin utilise le moins de générations d’élevage consécutives depuis votre collection mâle/femelle."
	translations[french]["planner_target"] = "Character ID du Pal cible"
	translations[french]["planner_find_route"] = "Trouver le chemin le plus rapide"
	translations[french]["planner_target_required"] = "Entrez un Character ID de Pal cible."
	translations[french]["planner_route_title"] = "Chemin vers %s · %d génération(s) d’élevage"
	translations[french]["planner_already_owned"] = "Déjà possédé dans la collection chargée ; aucune étape d’élevage n’est nécessaire."
	translations[french]["planner_speed_title"] = "Accélérer ce long chemin"
	translations[french]["planner_speed_none"] = "Aucun Pal Philanthropist ou Babysitter n’a été trouvé dans la collection chargée (équipe/Palbox)."
	translations[french]["planner_speed_philanthropist"] = "%s — assignez-le à la Ferme d’élevage : Philanthropist augmente sa vitesse d’élevage de 100 %."
	translations[french]["planner_speed_babysitter"] = "%s — gardez-le dans la base : Babysitter augmente de 30 %% la production d’œufs et la vitesse d’incubation de la Ferme d’élevage."
	translations[french]["planner_route_caveat"] = "C’est un chemin d’espèces. L’héritage des traits et le hasard du sexe des œufs ne sont pas garantis ; produisez des œufs supplémentaires lorsqu’une étape suivante demande un sexe précis."
	translations[french]["planner_route_ready"] = "Chemin d’élevage le plus rapide calculé."
}
