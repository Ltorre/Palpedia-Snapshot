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
	"gioui.org/op/paint"
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

type themeMode string

const (
	lightTheme themeMode = "light"
	darkTheme  themeMode = "dark"
)

type workspaceView string

const (
	noWorkspace       workspaceView = ""
	notebookWorkspace workspaceView = "notebook"
	plannerWorkspace  workspaceView = "planner"
	routeWorkspace    workspaceView = "route"
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
	window    *app.Window
	explorer  *explorer.Explorer
	theme     *material.Theme
	version   string
	language  language
	themeMode themeMode

	root, level, output, players, player, compare                                                          widget.Editor
	advanced                                                                                               widget.Bool
	englishButton, frenchButton, lightButton, darkButton, scanButton, changeSaveButton, changeExportButton widget.Clickable
	notebookViewButton, plannerViewButton, routeViewButton                                                 widget.Clickable
	browseButton, outputBrowseButton, compareBrowseButton, playersButton, exportButton, openExportButton   widget.Clickable
	candidates                                                                                             []SaveCandidate
	candidateButtons                                                                                       []widget.Clickable
	playersFound                                                                                           []sav.Player
	playerButtons                                                                                          []widget.Clickable
	list, plannerList                                                                                      layout.List
	results                                                                                                chan taskResult
	busy                                                                                                   bool
	status                                                                                                 string
	statusError                                                                                            bool
	lastExportDir                                                                                          string
	showSaveSetup, showExportSetup                                                                         bool
	activeView                                                                                             workspaceView
	plannerRefreshButton, pairButton, routeButton, plannerSortLevelAsc, plannerSortLevelDesc               widget.Clickable
	plannerGold, plannerDiamond, plannerMale, plannerFemale                                                widget.Bool
	plannerSort                                                                                            planner.SortOrder
	plannerFilter, target                                                                                  widget.Editor
	plannerPals                                                                                            []planner.Pal
	plannerPickers                                                                                         map[string]*plannerPicker
	targetSuggestionButtons                                                                                map[string]*widget.Clickable
	traitHovers                                                                                            map[string]*widget.Clickable
	selectedMale, selectedFemale                                                                           *planner.Pal
	plannerLoadedAt                                                                                        time.Time
	plannerSaveModified                                                                                    time.Time
	plannerPairResult, plannerRoute, plannerRouteNotice                                                    string
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
	s := &screen{window: window, explorer: explorer.NewExplorer(window), theme: theme, version: version, language: english, themeMode: lightTheme, showSaveSetup: true, showExportSetup: true, results: make(chan taskResult, 4), list: layout.List{Axis: layout.Vertical}, plannerList: layout.List{Axis: layout.Vertical}}
	s.applyTheme(lightTheme)
	for _, field := range []*widget.Editor{&s.root, &s.level, &s.output, &s.players, &s.player, &s.compare, &s.plannerFilter, &s.target} {
		field.SingleLine = true
	}
	s.plannerPickers = make(map[string]*plannerPicker)
	s.targetSuggestionButtons = make(map[string]*widget.Clickable)
	s.traitHovers = make(map[string]*widget.Clickable)
	s.root.SetText(DefaultSaveRoot())
	s.output.SetText(DefaultExportDir())
	s.lastExportDir = latestExportDir(s.output.Text())
	s.startScan()
	return s
}

func (s *screen) applyTheme(mode themeMode) {
	s.themeMode = mode
	if mode == darkTheme {
		s.theme.Palette = material.Palette{Bg: color.NRGBA{R: 22, G: 25, B: 34, A: 255}, Fg: color.NRGBA{R: 237, G: 239, B: 247, A: 255}, ContrastBg: color.NRGBA{R: 150, G: 130, B: 255, A: 255}, ContrastFg: color.NRGBA{R: 20, G: 18, B: 30, A: 255}}
		return
	}
	s.theme.Palette = material.Palette{Bg: color.NRGBA{R: 246, G: 248, B: 252, A: 255}, Fg: color.NRGBA{R: 25, G: 32, B: 48, A: 255}, ContrastBg: color.NRGBA{R: 93, G: 76, B: 205, A: 255}, ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}
}

func (s *screen) isDark() bool { return s.themeMode == darkTheme }

func (s *screen) primaryText() color.NRGBA {
	if s.isDark() {
		return color.NRGBA{R: 237, G: 239, B: 247, A: 255}
	}
	return color.NRGBA{R: 43, G: 49, B: 70, A: 255}
}

func (s *screen) mutedText() color.NRGBA {
	if s.isDark() {
		return color.NRGBA{R: 174, G: 181, B: 202, A: 255}
	}
	return color.NRGBA{R: 91, G: 98, B: 117, A: 255}
}

func (s *screen) surface() color.NRGBA {
	if s.isDark() {
		return color.NRGBA{R: 43, G: 48, B: 64, A: 255}
	}
	return color.NRGBA{R: 232, G: 235, B: 250, A: 255}
}

func (s *screen) border() color.NRGBA {
	if s.isDark() {
		return color.NRGBA{R: 92, G: 101, B: 130, A: 255}
	}
	return color.NRGBA{R: 173, G: 178, B: 207, A: 255}
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
				s.showSaveSetup = result.path == ""
				if result.path != "" {
					s.activeView = noWorkspace
				}
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
					s.showExportSetup = false
					s.startPlannerRefresh()
				}
			case "planner":
				if result.err == nil {
					s.plannerPals, s.plannerLoadedAt, s.plannerSaveModified = result.plannerPals, time.Now(), result.updatedAt
					s.plannerPickers = make(map[string]*plannerPicker)
					s.selectedMale, s.selectedFemale, s.plannerPairResult, s.plannerRoute, s.plannerRouteNotice = nil, nil, "", "", ""
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
	if s.lightButton.Clicked(gtx) {
		s.applyTheme(lightTheme)
	}
	if s.darkButton.Clicked(gtx) {
		s.applyTheme(darkTheme)
	}
	if s.changeSaveButton.Clicked(gtx) {
		s.showSaveSetup = true
		s.activeView = noWorkspace
	}
	if s.changeExportButton.Clicked(gtx) {
		s.showExportSetup = true
	}
	if s.notebookViewButton.Clicked(gtx) {
		s.activeView = notebookWorkspace
	}
	if s.plannerViewButton.Clicked(gtx) {
		s.activeView = plannerWorkspace
		s.ensurePlannerLoaded()
	}
	if s.routeViewButton.Clicked(gtx) {
		s.activeView = routeWorkspace
		s.ensurePlannerLoaded()
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
	if s.plannerSortLevelAsc.Clicked(gtx) {
		s.plannerSort = planner.SortByLevelAscending
	}
	if s.plannerSortLevelDesc.Clicked(gtx) {
		s.plannerSort = planner.SortByLevelDescending
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
			s.showSaveSetup = false
			s.activeView = noWorkspace
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
			s.selectedMale, s.plannerPairResult, s.plannerRoute, s.plannerRouteNotice = &pal, "", "", ""
			s.plannerMale.Value, s.plannerFemale.Value = false, true
		}
		if picker.female.Clicked(gtx) {
			pal := picker.pal
			s.selectedFemale, s.plannerPairResult, s.plannerRoute, s.plannerRouteNotice = &pal, "", "", ""
			s.plannerMale.Value, s.plannerFemale.Value = true, false
		}
	}
	for characterID, button := range s.targetSuggestionButtons {
		if button.Clicked(gtx) {
			if rules, err := breeding.Default(); err == nil {
				s.target.SetText(rules.DisplayName(characterID))
				s.plannerRoute, s.plannerRouteNotice = "", ""
			}
		}
	}
}

func (s *screen) ensurePlannerLoaded() {
	if len(s.plannerPals) == 0 && !s.busy {
		s.startPlannerRefresh()
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
			rules, ruleErr := breeding.Default()
			if ruleErr != nil {
				err = ruleErr
			} else {
				pals = plannerCollection(world, playerUID, rules)
			}
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

func plannerCollection(world *sav.World, playerUID string, rules *breeding.Rules) []planner.Pal {
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
		pals = append(pals, planner.Pal{InstanceID: pal.InstanceID, CharacterID: pal.CharacterID, DisplayName: rules.DisplayName(pal.CharacterID), Gender: pal.Gender, Level: pal.Level, Traits: append([]string(nil), pal.PassiveSkillIDs...)})
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
			s.plannerPairResult = fmt.Sprintf(s.t("planner_pair_result"), rules.DisplayName(result.Child), result.Rule, result.TargetRank)
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
	s.plannerRoute, s.plannerRouteNotice = "", ""
	path, err := planner.ShortestPath(rules, s.plannerPals, target)
	if err != nil {
		s.statusError, s.status = true, err.Error()
		return
	}
	if path.Generations == 0 {
		s.plannerRouteNotice = fmt.Sprintf(s.t("planner_already_owned_inheritance"), rules.DisplayName(path.Target))
		path, err = planner.ShortestPathAsIfUnowned(rules, s.plannerPals, path.Target)
		if err != nil {
			s.plannerRoute = s.t("planner_inheritance_no_route") + "\n" + s.t("planner_route_caveat")
			s.statusError, s.status = false, s.t("planner_route_ready")
			return
		}
	}
	var out strings.Builder
	fmt.Fprintf(&out, s.t("planner_route_title"), rules.DisplayName(path.Target), path.Generations)
	for index, step := range path.Steps {
		fmt.Fprintf(&out, "\n%d. %s + %s → %s (%s)", index+1, rules.DisplayName(step.ParentA), rules.DisplayName(step.ParentB), rules.DisplayName(step.Child), step.Rule)
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
	paint.Fill(gtx.Ops, s.theme.Palette.Bg)
	return layout.Inset{Top: unit.Dp(18), Bottom: unit.Dp(18), Left: unit.Dp(28), Right: unit.Dp(28)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return s.list.Layout(gtx, 1, func(gtx layout.Context, _ int) layout.Dimensions {
			children := []layout.FlexChild{layout.Rigid(s.header), layout.Rigid(spacer(14))}
			if s.showSaveSetup || strings.TrimSpace(s.level.Text()) == "" {
				children = append(children, layout.Rigid(s.saveSection))
			} else {
				children = append(children,
					layout.Rigid(s.selectedSaveSummary),
					layout.Rigid(spacer(12)),
					layout.Rigid(s.workspaceChooser),
				)
				switch s.activeView {
				case notebookWorkspace:
					children = append(children, layout.Rigid(spacer(12)))
					if s.showExportSetup {
						children = append(children, layout.Rigid(s.exportSection))
					} else {
						children = append(children, layout.Rigid(s.exportSummary))
					}
				case plannerWorkspace:
					children = append(children, layout.Rigid(spacer(12)), layout.Rigid(s.plannerSection))
				case routeWorkspace:
					children = append(children, layout.Rigid(spacer(12)), layout.Rigid(s.quickestRouteSection))
				}
			}
			children = append(children, layout.Rigid(spacer(12)), layout.Rigid(s.statusSection))
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func (s *screen) header(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.H4(s.theme, s.t("title"))
					l.Color = s.theme.Palette.ContrastBg
					return l.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Body2(s.theme, fmt.Sprintf("%s · %s", s.t("subtitle"), s.version))
					l.Color = s.mutedText()
					return l.Layout(gtx)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.End}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return s.languageButton(gtx, &s.englishButton, "EN", s.language == english)
						}), layout.Rigid(spacer(6)), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return s.languageButton(gtx, &s.frenchButton, "FR", s.language == french)
						}),
					)
				}),
				layout.Rigid(spacer(6)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return s.languageButton(gtx, &s.lightButton, s.t("theme_light"), s.themeMode == lightTheme)
						}), layout.Rigid(spacer(6)), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return s.languageButton(gtx, &s.darkButton, s.t("theme_dark"), s.themeMode == darkTheme)
						}),
					)
				}),
			)
		}),
	)
}

func (s *screen) languageButton(gtx layout.Context, button *widget.Clickable, label string, active bool) layout.Dimensions {
	style := material.Button(s.theme, button, label)
	if !active {
		style.Background, style.Color = s.surface(), s.primaryText()
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

func (s *screen) selectedSaveSummary(gtx layout.Context) layout.Dimensions {
	return section(gtx, s.theme, s.t("selected_save_summary"), func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, s.note(filepath.Base(s.level.Text()), s.primaryText())),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return s.languageButton(gtx, &s.changeSaveButton, s.t("change_save"), false)
			}),
		)
	})
}

func (s *screen) workspaceChooser(gtx layout.Context) layout.Dimensions {
	return section(gtx, s.theme, s.t("workspace_title"), func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return s.workspaceButton(gtx, &s.notebookViewButton, s.t("workspace_notebook"), s.activeView == notebookWorkspace)
			}),
			layout.Rigid(spacer(8)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return s.workspaceButton(gtx, &s.plannerViewButton, s.t("workspace_planner"), s.activeView == plannerWorkspace)
			}),
			layout.Rigid(spacer(8)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return s.workspaceButton(gtx, &s.routeViewButton, s.t("workspace_route"), s.activeView == routeWorkspace)
			}),
		)
	})
}

func (s *screen) workspaceButton(gtx layout.Context, button *widget.Clickable, label string, active bool) layout.Dimensions {
	style := material.Button(s.theme, button, label)
	if !active {
		style.Background, style.Color = s.surface(), s.primaryText()
	}
	return style.Layout(gtx)
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
				style.Background, style.Color = s.surface(), s.primaryText()
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
			button.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
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

func (s *screen) exportSummary(gtx layout.Context) layout.Dimensions {
	return section(gtx, s.theme, s.t("export_summary"), func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, s.note(filepath.Base(s.lastExportDir), s.primaryText())),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return s.languageButton(gtx, &s.changeExportButton, s.t("change_export"), false)
			}),
		)
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
				button.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
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
			layout.Rigid(s.plannerParentCards),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				button := material.Button(s.theme, &s.pairButton, s.t("planner_calculate_pair"))
				button.Background = color.NRGBA{R: 127, G: 83, B: 187, A: 255}
				button.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
				return button.Layout(gtx)
			}),
		)
		if s.plannerPairResult != "" {
			children = append(children, layout.Rigid(spacer(5)), layout.Rigid(s.note(s.plannerPairResult, s.primaryText())))
		}
		children = append(children,
			layout.Rigid(spacer(8)),
			layout.Rigid(s.caption("planner_filter_help")),
			layout.Rigid(s.caption("planner_trait_help")),
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
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.CheckBox(s.theme, &s.plannerMale, s.t("planner_male")).Layout(gtx)
					}),
					layout.Rigid(spacer(12)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.CheckBox(s.theme, &s.plannerFemale, s.t("planner_female")).Layout(gtx)
					}),
				)
			}),
			layout.Rigid(spacer(4)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return s.plannerSortButton(gtx, &s.plannerSortLevelAsc, s.t("planner_sort_level_asc"), s.plannerSort == planner.SortByLevelAscending)
					}),
					layout.Rigid(spacer(8)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return s.plannerSortButton(gtx, &s.plannerSortLevelDesc, s.t("planner_sort_level_desc"), s.plannerSort == planner.SortByLevelDescending)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if gender := s.plannerRequiredGender(); gender != "" {
					return s.caption("planner_opposite_" + gender)(gtx)
				}
				return layout.Dimensions{}
			}),
			layout.Rigid(spacer(4)),
			layout.Rigid(s.plannerPalsList),
		)
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

func (s *screen) quickestRouteSection(gtx layout.Context) layout.Dimensions {
	return section(gtx, s.theme, s.t("planner_route_section"), func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{
			layout.Rigid(s.caption("planner_route_help")),
			layout.Rigid(spacer(6)),
			layout.Rigid(s.plannerFreshness),
			layout.Rigid(spacer(6)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				button := material.Button(s.theme, &s.plannerRefreshButton, s.t("planner_refresh"))
				button.Background = color.NRGBA{R: 46, G: 103, B: 171, A: 255}
				button.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
				return button.Layout(gtx)
			}),
		}
		if len(s.plannerPals) == 0 {
			children = append(children, layout.Rigid(spacer(6)), layout.Rigid(s.caption("planner_empty")))
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}
		children = append(children,
			layout.Rigid(spacer(12)),
			layout.Rigid(s.routeTargetPicker),
		)
		if s.plannerRoute != "" {
			children = append(children, layout.Rigid(spacer(12)), layout.Rigid(s.routeResultCard))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

func (s *screen) routeTargetPicker(gtx layout.Context) layout.Dimensions {
	return s.outlinedCard(gtx, s.t("planner_target_picker"), func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(s.caption("planner_target_picker_help")),
			layout.Rigid(spacer(8)),
			layout.Rigid(s.editor(&s.target, s.t("planner_target"))),
			layout.Rigid(s.plannerTargetSuggestions),
			layout.Rigid(spacer(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				button := material.Button(s.theme, &s.routeButton, s.t("planner_find_route"))
				button.Background = color.NRGBA{R: 32, G: 125, B: 104, A: 255}
				button.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
				return button.Layout(gtx)
			}),
		)
	})
}

func (s *screen) routeResultCard(gtx layout.Context) layout.Dimensions {
	return s.outlinedCard(gtx, s.t("planner_route_result"), func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, 0, 3)
		if s.plannerRouteNotice != "" {
			children = append(children, layout.Rigid(s.note(s.plannerRouteNotice, s.routeNoticeColor())), layout.Rigid(spacer(8)))
		}
		children = append(children, layout.Rigid(s.routeBody(s.plannerRoute)))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

func (s *screen) outlinedCard(gtx layout.Context, title string, content layout.Widget) layout.Dimensions {
	return widget.Border{Color: s.border(), CornerRadius: unit.Dp(8), Width: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					label := material.H6(s.theme, title)
					label.Color = s.primaryText()
					return label.Layout(gtx)
				}),
				layout.Rigid(spacer(8)),
				layout.Rigid(content),
			)
		})
	})
}

func (s *screen) routeBody(value string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		label := material.Body1(s.theme, value)
		label.Color = s.primaryText()
		label.WrapPolicy = text.WrapWords
		return label.Layout(gtx)
	}
}

func (s *screen) routeNoticeColor() color.NRGBA {
	if s.isDark() {
		return color.NRGBA{R: 111, G: 226, B: 172, A: 255}
	}
	return color.NRGBA{R: 20, G: 132, B: 85, A: 255}
}

func (s *screen) plannerFreshness(gtx layout.Context) layout.Dimensions {
	if s.plannerLoadedAt.IsZero() {
		return s.note(s.t("planner_not_loaded"), s.mutedText())(gtx)
	}
	message := fmt.Sprintf(s.t("planner_freshness"), len(s.plannerPals), s.plannerLoadedAt.Local().Format("2006-01-02 15:04"), s.plannerSaveModified.Local().Format("2006-01-02 15:04"))
	if s.lastExportDir != "" {
		message += "\n" + fmt.Sprintf(s.t("planner_last_export"), filepath.Base(s.lastExportDir))
	}
	return s.note(message, s.primaryText())(gtx)
}

func (s *screen) plannerTargetSuggestions(gtx layout.Context) layout.Dimensions {
	rules, err := breeding.Default()
	if err != nil {
		return layout.Dimensions{}
	}
	suggestions := rules.Suggestions(s.target.Text(), 6)
	if len(suggestions) == 0 {
		return layout.Dimensions{}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(s.caption("planner_target_suggestions")),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, suggestionChildren(s, suggestions)...)
		}),
	)
}

func suggestionChildren(s *screen, suggestions []breeding.Suggestion) []layout.FlexChild {
	children := make([]layout.FlexChild, 0, len(suggestions))
	for _, suggestion := range suggestions {
		suggestion := suggestion
		button, ok := s.targetSuggestionButtons[suggestion.CharacterID]
		if !ok {
			button = new(widget.Clickable)
			s.targetSuggestionButtons[suggestion.CharacterID] = button
		}
		label := suggestion.DisplayName
		if suggestion.DisplayName != suggestion.CharacterID {
			label += " · " + suggestion.CharacterID
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			style := material.Button(s.theme, button, label)
			style.Background, style.Color = s.surface(), s.primaryText()
			return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return widget.Border{Color: s.border(), CornerRadius: unit.Dp(5), Width: unit.Dp(1)}.Layout(gtx, style.Layout)
			})
		}))
	}
	return children
}

func (s *screen) plannerPalsList(gtx layout.Context) layout.Dimensions {
	filters := planner.FilterOptions{
		Query:          s.plannerFilter.Text(),
		GoldOnly:       s.plannerGold.Value,
		DiamondOnly:    s.plannerDiamond.Value,
		MaleOnly:       s.plannerMale.Value,
		FemaleOnly:     s.plannerFemale.Value,
		RequiredGender: s.plannerRequiredGender(),
		SortOrder:      s.plannerSort,
	}
	total := len(planner.FilterWithOptions(s.plannerPals, filters))
	filters.Deduplicate = true
	visible := planner.FilterWithOptions(s.plannerPals, filters)
	if len(visible) == 0 {
		return s.caption("planner_no_matches")(gtx)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(s.note(fmt.Sprintf(s.t("planner_showing"), len(visible), total), s.mutedText())),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			maxHeight := gtx.Dp(unit.Dp(320))
			if gtx.Constraints.Max.Y > maxHeight {
				gtx.Constraints.Max.Y = maxHeight
			}
			return s.plannerList.Layout(gtx, len(visible), func(gtx layout.Context, index int) layout.Dimensions {
				pal := visible[index]
				picker := s.plannerPicker(pal)
				return layout.Inset{Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return widget.Border{Color: s.border(), CornerRadius: unit.Dp(6), Width: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1, s.plannerPalDetails(pal, "list")),
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
				})
			})
		}),
	)
}

func (s *screen) plannerRequiredGender() string {
	if s.selectedMale != nil && s.selectedFemale == nil {
		return "female"
	}
	if s.selectedFemale != nil && s.selectedMale == nil {
		return "male"
	}
	return ""
}

func (s *screen) plannerSortButton(gtx layout.Context, button *widget.Clickable, label string, active bool) layout.Dimensions {
	style := material.Button(s.theme, button, label)
	if !active {
		style.Background, style.Color = s.surface(), s.primaryText()
	}
	return style.Layout(gtx)
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

func (s *screen) plannerParentCards(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return s.plannerParentCard(gtx, s.t("planner_male_parent"), s.selectedMale)
		}),
		layout.Rigid(spacer(8)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return s.plannerParentCard(gtx, s.t("planner_female_parent"), s.selectedFemale)
		}),
	)
}

func (s *screen) plannerParentCard(gtx layout.Context, title string, pal *planner.Pal) layout.Dimensions {
	return widget.Border{Color: s.border(), CornerRadius: unit.Dp(6), Width: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					label := material.Caption(s.theme, title)
					label.Color = s.mutedText()
					return label.Layout(gtx)
				}),
			}
			if pal == nil {
				children = append(children, layout.Rigid(s.note(s.t("planner_no_parent"), s.mutedText())))
			} else {
				children = append(children, layout.Rigid(s.plannerPalDetails(*pal, title)))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func plannerPalLabel(pal planner.Pal) string {
	return fmt.Sprintf("%s · Lv. %d · %s", planner.PalName(pal), planner.PalLevel(pal), pal.Gender)
}

func (s *screen) plannerPalDetails(pal planner.Pal, location string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{layout.Rigid(s.note(plannerPalLabel(pal), s.primaryText()))}
		if len(pal.Traits) > 0 {
			children = append(children, layout.Rigid(s.traitLabels(location+"\x00"+pal.InstanceID, pal.Traits)))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

func (s *screen) traitLabels(location string, traits []string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, 0, len(traits))
		for index, trait := range traits {
			index, trait := index, trait
			key := fmt.Sprintf("%s\x00%d\x00%s", location, index, trait)
			hover := s.traitHover(key)
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				label := material.Caption(s.theme, planner.TraitName(trait))
				label.Color = s.traitColor(trait)
				traitLabel := func(gtx layout.Context) layout.Dimensions {
					if index < len(traits)-1 {
						return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, label.Layout)
					}
					return label.Layout(gtx)
				}
				return hover.Layout(gtx, traitLabel)
			}))
		}
		labelRow := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
		for index, trait := range traits {
			key := fmt.Sprintf("%s\x00%d\x00%s", location, index, trait)
			if effect := planner.TraitEffect(trait); effect != "" && s.traitHover(key).Hovered() {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(layout.Context) layout.Dimensions { return labelRow }),
					layout.Rigid(s.note(effect, s.mutedText())),
				)
			}
		}
		return labelRow
	}
}

func (s *screen) traitHover(key string) *widget.Clickable {
	if hover, ok := s.traitHovers[key]; ok {
		return hover
	}
	hover := new(widget.Clickable)
	s.traitHovers[key] = hover
	return hover
}

func (s *screen) traitColor(trait string) color.NRGBA {
	switch planner.Tier(trait) {
	case planner.Gold:
		if s.isDark() {
			return color.NRGBA{R: 247, G: 207, B: 93, A: 255}
		}
		return color.NRGBA{R: 167, G: 114, B: 0, A: 255}
	case planner.Diamond:
		if s.isDark() {
			return color.NRGBA{R: 111, G: 226, B: 172, A: 255}
		}
		return color.NRGBA{R: 20, G: 132, B: 85, A: 255}
	default:
		return s.mutedText()
	}
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
				style.Background, style.Color = s.surface(), s.primaryText()
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
		label.Color = s.primaryText()
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
		label.Color = s.mutedText()
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
			label.Color = theme.Palette.Fg
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
	translations[english]["theme_light"] = "Light"
	translations[english]["theme_dark"] = "Dark"
	translations[french]["theme_light"] = "Clair"
	translations[french]["theme_dark"] = "Sombre"
	translations[english]["selected_save_summary"] = "Selected save"
	translations[english]["change_save"] = "Change save"
	translations[english]["export_summary"] = "Latest NotebookLM export"
	translations[english]["change_export"] = "Edit export"
	translations[french]["selected_save_summary"] = "Sauvegarde sélectionnée"
	translations[french]["change_save"] = "Changer de sauvegarde"
	translations[french]["export_summary"] = "Dernier export NotebookLM"
	translations[french]["change_export"] = "Modifier l’export"
	translations[english]["notebooklm_files"] = "Create your own notebook at notebook.google.com. First upload the 31 reference Markdown files from palpedia-snapshot-notebooklm-reference.zip, then add: collection.md, pals.csv, capture-history.csv, palpedia-progress.md, breeding-candidates.md, breeding-rules.md, breeding-direct-pairs.csv, and collection-diff.md when comparing. Do not add world.json."
	translations[french]["notebooklm_files"] = "Créez votre propre notebook sur notebook.google.com. Importez d’abord les 31 fichiers Markdown de référence depuis palpedia-snapshot-notebooklm-reference.zip, puis ajoutez : collection.md, pals.csv, capture-history.csv, palpedia-progress.md, breeding-candidates.md, breeding-rules.md, breeding-direct-pairs.csv et collection-diff.md lors d’une comparaison. Ne pas ajouter world.json."
	translations[english]["workspace_title"] = "Choose a workspace"
	translations[english]["workspace_notebook"] = "NotebookLM export"
	translations[english]["workspace_planner"] = "Breeding planner"
	translations[english]["workspace_route"] = "Quickest route"
	translations[french]["workspace_title"] = "Choisir un espace"
	translations[french]["workspace_notebook"] = "Export NotebookLM"
	translations[french]["workspace_planner"] = "Planificateur d’élevage"
	translations[french]["workspace_route"] = "Chemin le plus rapide"
	translations[english]["planner_title"] = "Breeding planner"
	translations[english]["planner_help"] = "Load the selected save to choose actual Pals and calculate the exact child of a selected pair. This is read-only."
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
	translations[english]["planner_filter_help"] = "Search a Pal name (for example Mammorest), a raw game ID, or a passive trait. Filter by sex or sort by level when useful. Gold is rank 3; diamond is rank 4. If both are checked, either tier is included."
	translations[english]["planner_trait_help"] = "Hover a trait to see its in-game effect."
	translations[english]["planner_filter"] = "Search Pal name or trait"
	translations[english]["planner_gold"] = "Gold traits (rank 3)"
	translations[english]["planner_diamond"] = "Diamond traits (rank 4)"
	translations[english]["planner_male"] = "Male"
	translations[english]["planner_female"] = "Female"
	translations[english]["planner_sort_level_asc"] = "Level ↑"
	translations[english]["planner_sort_level_desc"] = "Level ↓"
	translations[english]["planner_opposite_male"] = "Choose a male parent for the selected female."
	translations[english]["planner_opposite_female"] = "Choose a female parent for the selected male."
	translations[english]["planner_no_matches"] = "No loaded Pals match these filters."
	translations[english]["planner_showing"] = "Showing %d highest-level unique Pal(s) from %d matching Pal(s). Scroll this list to browse the collection."
	translations[english]["planner_choose_male"] = "Choose male"
	translations[english]["planner_choose_female"] = "Choose female"
	translations[english]["planner_no_parent"] = "Not selected"
	translations[english]["planner_male_parent"] = "Male parent"
	translations[english]["planner_female_parent"] = "Female parent"
	translations[english]["planner_selected"] = "Male parent: %s\nFemale parent: %s"
	translations[english]["planner_calculate_pair"] = "Calculate selected pair"
	translations[english]["planner_select_parents"] = "Choose one male parent and one female parent first."
	translations[english]["planner_pair_result"] = "Exact child: %s · rule: %s · generic target rank: %d"
	translations[english]["planner_pair_ready"] = "Exact breeding outcome calculated."
	translations[english]["planner_route_section"] = "Find the quickest target route"
	translations[english]["planner_route_help"] = "Enter a Pal name such as Anubis or Mammorest. Game Character IDs such as SheepBall or PinkCat also work. The route uses the fewest sequential breeding generations from your current male/female collection."
	translations[english]["planner_target_picker"] = "Choose the Pal you want to breed"
	translations[english]["planner_target_picker_help"] = "Start typing to find a Pal by its Palpedia name. Select a matching suggestion to avoid spelling or internal-ID mistakes."
	translations[english]["planner_target"] = "Target Pal name or Character ID"
	translations[english]["planner_target_suggestions"] = "Matching Pals — choose one"
	translations[english]["planner_find_route"] = "Find quickest breeding route"
	translations[english]["planner_target_required"] = "Enter a target Pal Character ID."
	translations[english]["planner_route_result"] = "Your breeding route"
	translations[english]["planner_route_title"] = "Route to %s · %d breeding generation(s)"
	translations[english]["planner_already_owned"] = "Already owned in the loaded collection; no breeding step is required."
	translations[english]["planner_already_owned_inheritance"] = "You already own %s. The route below deliberately ignores those copies, so you can breed a new one for passive-trait inheritance."
	translations[english]["planner_inheritance_no_route"] = "No route could be formed without using your existing target Pal. Keep the target you own, or add more male/female Pals to the collection and refresh."
	translations[english]["planner_speed_title"] = "Speed up this long route"
	translations[english]["planner_speed_none"] = "No Philanthropist or Babysitter Pal was found in the loaded party/Palbox collection."
	translations[english]["planner_speed_philanthropist"] = "%s — assign to the Breeding Farm: Philanthropist increases that Pal's breeding speed by 100%."
	translations[english]["planner_speed_babysitter"] = "%s — keep at the base: Babysitter improves Breeding Farm egg production and incubation speed by 30%."
	translations[english]["planner_route_caveat"] = "This is a species route. Passive inheritance and egg-gender RNG are not guaranteed; breed extra eggs when a later step needs a specific sex."
	translations[english]["planner_route_ready"] = "Quickest breeding route calculated."

	translations[french]["planner_title"] = "Planificateur d’élevage"
	translations[french]["planner_help"] = "Chargez la sauvegarde sélectionnée pour choisir vos vrais Pals et calculer l’enfant exact d’une paire. Lecture seule."
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
	translations[french]["planner_filter_help"] = "Recherchez un nom de Pal (par exemple Mammorest), un identifiant du jeu ou un trait passif. Filtrez par sexe ou triez par niveau si nécessaire. Or = rang 3 ; diamant = rang 4. Avec les deux cochés, les deux rangs sont inclus."
	translations[french]["planner_trait_help"] = "Survolez un trait pour voir son effet en jeu."
	translations[french]["planner_filter"] = "Rechercher un Pal ou trait"
	translations[french]["planner_gold"] = "Traits or (rang 3)"
	translations[french]["planner_diamond"] = "Traits diamant (rang 4)"
	translations[french]["planner_male"] = "Mâle"
	translations[french]["planner_female"] = "Femelle"
	translations[french]["planner_sort_level_asc"] = "Niveau ↑"
	translations[french]["planner_sort_level_desc"] = "Niveau ↓"
	translations[french]["planner_opposite_male"] = "Choisissez un parent mâle pour la femelle sélectionnée."
	translations[french]["planner_opposite_female"] = "Choisissez un parent femelle pour le mâle sélectionné."
	translations[french]["planner_no_matches"] = "Aucun Pal chargé ne correspond à ces filtres."
	translations[french]["planner_showing"] = "%d Pal(s) unique(s) au niveau le plus élevé affiché(s) parmi %d correspondant(s). Faites défiler cette liste pour parcourir la collection."
	translations[french]["planner_choose_male"] = "Choisir le mâle"
	translations[french]["planner_choose_female"] = "Choisir la femelle"
	translations[french]["planner_no_parent"] = "Non sélectionné"
	translations[french]["planner_male_parent"] = "Parent mâle"
	translations[french]["planner_female_parent"] = "Parent femelle"
	translations[french]["planner_selected"] = "Parent mâle : %s\nParent femelle : %s"
	translations[french]["planner_calculate_pair"] = "Calculer la paire sélectionnée"
	translations[french]["planner_select_parents"] = "Choisissez d’abord un parent mâle et un parent femelle."
	translations[french]["planner_pair_result"] = "Enfant exact : %s · règle : %s · rang cible générique : %d"
	translations[french]["planner_pair_ready"] = "Résultat exact de l’élevage calculé."
	translations[french]["planner_route_section"] = "Trouver le chemin le plus rapide"
	translations[french]["planner_route_help"] = "Entrez un nom de Pal comme Anubis ou Mammorest. Les Character IDs du jeu, comme SheepBall ou PinkCat, fonctionnent aussi. Le chemin utilise le moins de générations d’élevage consécutives depuis votre collection mâle/femelle."
	translations[french]["planner_target_picker"] = "Choisir le Pal à obtenir"
	translations[french]["planner_target_picker_help"] = "Commencez à écrire pour rechercher un Pal par son nom du Palpédia. Sélectionnez une suggestion pour éviter les erreurs d’orthographe ou d’identifiant interne."
	translations[french]["planner_target"] = "Nom ou Character ID du Pal cible"
	translations[french]["planner_target_suggestions"] = "Pals correspondants — choisissez-en un"
	translations[french]["planner_find_route"] = "Trouver le chemin le plus rapide"
	translations[french]["planner_target_required"] = "Entrez un Character ID de Pal cible."
	translations[french]["planner_route_result"] = "Votre chemin d’élevage"
	translations[french]["planner_route_title"] = "Chemin vers %s · %d génération(s) d’élevage"
	translations[french]["planner_already_owned"] = "Déjà possédé dans la collection chargée ; aucune étape d’élevage n’est nécessaire."
	translations[french]["planner_already_owned_inheritance"] = "Vous possédez déjà %s. Le chemin ci-dessous ignore volontairement ces exemplaires afin d’en élever un nouveau pour l’héritage des traits passifs."
	translations[french]["planner_inheritance_no_route"] = "Aucun chemin n’a pu être formé sans utiliser le Pal cible que vous possédez. Gardez cet exemplaire, ou ajoutez des Pals mâles/femelles à la collection puis actualisez-la."
	translations[french]["planner_speed_title"] = "Accélérer ce long chemin"
	translations[french]["planner_speed_none"] = "Aucun Pal Philanthropist ou Babysitter n’a été trouvé dans la collection chargée (équipe/Palbox)."
	translations[french]["planner_speed_philanthropist"] = "%s — assignez-le à la Ferme d’élevage : Philanthropist augmente sa vitesse d’élevage de 100 %."
	translations[french]["planner_speed_babysitter"] = "%s — gardez-le dans la base : Babysitter augmente de 30 %% la production d’œufs et la vitesse d’incubation de la Ferme d’élevage."
	translations[french]["planner_route_caveat"] = "C’est un chemin d’espèces. L’héritage des traits et le hasard du sexe des œufs ne sont pas garantis ; produisez des œufs supplémentaires lorsqu’une étape suivante demande un sexe précis."
	translations[french]["planner_route_ready"] = "Chemin d’élevage le plus rapide calculé."
}
