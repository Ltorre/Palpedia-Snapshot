//go:build windows

package gui

import (
	"fmt"
	"image"
	"image/color"
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

	"github.com/Ltorre/palworld-save-scrap/internal/report"
	"github.com/Ltorre/palworld-save-scrap/internal/sav"
)

type language string

const (
	english language = "en"
	french  language = "fr"
)

type taskResult struct {
	kind       string
	candidates []SaveCandidate
	players    []sav.Player
	path       string
	message    string
	err        error
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
}

func Run(version string) {
	go func() {
		window := new(app.Window)
		window.Option(app.Title("Palworld Save Scrap"), app.Size(unit.Dp(1080), unit.Dp(760)))
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
	for _, field := range []*widget.Editor{&s.root, &s.level, &s.output, &s.players, &s.player, &s.compare} {
		field.SingleLine = true
	}
	s.root.SetText(DefaultSaveRoot())
	s.output.SetText(DefaultExportDir())
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
			case "compare-browse":
				s.compare.SetText(result.path)
			case "players":
				s.playersFound = result.players
				s.playerButtons = make([]widget.Clickable, len(result.players))
			case "export":
				if result.err == nil && result.path != "" {
					s.lastExportDir = result.path
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
				layout.Rigid(s.header), layout.Rigid(spacer(14)), layout.Rigid(s.saveSection), layout.Rigid(spacer(12)), layout.Rigid(s.exportSection), layout.Rigid(spacer(12)), layout.Rigid(s.statusSection),
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
		"title": "Palworld Save Scrap", "subtitle": "Personal Palpedia export", "find_save": "1. Find your save", "save_root": "Default Palworld save folder", "save_root_help": "Starts at the standard Windows Palworld save location. You may replace it with any folder containing your saves.", "scan": "Find worlds", "browse_level": "Browse for Level.sav", "selected_level": "Selected Level.sav", "no_candidates": "No world found yet. Scan the folder or browse directly to Level.sav.", "detected_worlds": "Detected worlds", "export": "2. Export for NotebookLM", "output_directory": "Export parent folder", "browse_output": "Choose folder", "choose_export_folder": "Choose an export folder", "opening_export_browser": "Opening the folder browser…", "export_folder_selected": "Export folder selected.", "export_folder_unchanged": "Export folder unchanged.", "output_help": "Required. Choose a parent folder; the tool writes only here and never inside your game save folder.", "snapshot_help": "Each export is saved in its own folder, for example export_08-09-2026 18-42. Windows cannot use : in a folder name.", "notebooklm_files": "Add to NotebookLM: collection.md, pals.csv, capture-history.csv, palpedia-progress.md, breeding-candidates.md, and collection-diff.md when comparing. Do not add world.json.", "advanced_options": "Show optional advanced options", "advanced_help": "Use these only for shared worlds, custom save layouts, or comparing this snapshot with an earlier export.", "players_directory": "Players directory (optional)", "players_help": "Only needed when Players is not beside Level.sav.", "find_players": "Find players in this save", "available_players": "Available players", "select_save_first": "Select a Level.sav file first.", "reading_players": "Reading the players in this save…", "no_players": "No players were found in this save.", "players_found": "%d player(s) found. Select one to export only that player, or leave it empty for everyone.", "player_selected": "%s selected. Clear the Player UID field to export all players.", "player_uid": "Player UID (optional)", "player_help": "Choose a detected player to export only their collection. Leave empty to export every player in the world.", "compare_directory": "Previous export folder (optional)", "browse_compare": "Choose previous export", "choose_compare_folder": "Choose the earlier export folder", "opening_compare_browser": "Opening the previous-export folder browser…", "compare_folder_selected": "Previous export folder selected.", "compare_folder_unchanged": "Previous export folder unchanged.", "compare_help": "Adds collection-diff.md using a previous export_<date time> folder.", "export_button": "Create NotebookLM export", "save_root_required": "Choose a save folder first.", "scanning": "Scanning for Palworld worlds…", "no_saves": "No Level.sav file was found in this folder.", "saves_found": "%d world(s) found. Select one below.", "opening_browser": "Opening the file browser…", "world_selected": "World selected. Choose an export parent folder and create the export.", "save_and_output_required": "Select a Level.sav file and an export parent folder.", "exporting": "Reading the save and creating a new export snapshot…", "open_export_folder": "Open export folder in Explorer",
	},
	french: {
		"title": "Palworld Save Scrap", "subtitle": "Export personnel pour le Palpédia", "find_save": "1. Trouver votre sauvegarde", "save_root": "Dossier de sauvegarde Palworld par défaut", "save_root_help": "Commence dans le dossier Windows standard de Palworld. Vous pouvez le remplacer par tout dossier contenant vos sauvegardes.", "scan": "Chercher les mondes", "browse_level": "Parcourir Level.sav", "selected_level": "Level.sav sélectionné", "no_candidates": "Aucun monde trouvé. Cherchez dans le dossier ou choisissez directement Level.sav.", "detected_worlds": "Mondes détectés", "export": "2. Exporter pour NotebookLM", "output_directory": "Dossier parent des exports", "browse_output": "Choisir un dossier", "choose_export_folder": "Choisir un dossier d’export", "opening_export_browser": "Ouverture du navigateur de dossiers…", "export_folder_selected": "Dossier d’export sélectionné.", "export_folder_unchanged": "Dossier d’export inchangé.", "output_help": "Obligatoire. Choisissez un dossier parent ; l’outil écrit uniquement ici, jamais dans le dossier de sauvegarde du jeu.", "snapshot_help": "Chaque export est créé dans son propre dossier, par exemple export_08-09-2026 18-42. Windows interdit : dans les noms de dossiers.", "notebooklm_files": "À ajouter à NotebookLM : collection.md, pals.csv, capture-history.csv, palpedia-progress.md, breeding-candidates.md et collection-diff.md lors d’une comparaison. Ne pas ajouter world.json.", "advanced_options": "Afficher les options avancées facultatives", "advanced_help": "Utilisez-les seulement pour les mondes partagés, les emplacements personnalisés ou la comparaison avec un export antérieur.", "players_directory": "Dossier Players (facultatif)", "players_help": "Nécessaire uniquement si Players n’est pas à côté de Level.sav.", "find_players": "Chercher les joueurs de cette sauvegarde", "available_players": "Joueurs disponibles", "select_save_first": "Sélectionnez d’abord un fichier Level.sav.", "reading_players": "Lecture des joueurs de cette sauvegarde…", "no_players": "Aucun joueur trouvé dans cette sauvegarde.", "players_found": "%d joueur(s) trouvé(s). Sélectionnez-en un pour n’exporter que sa collection, ou laissez vide pour tous les joueurs.", "player_selected": "%s sélectionné. Videz le champ UID du joueur pour exporter tous les joueurs.", "player_uid": "UID du joueur (facultatif)", "player_help": "Choisissez un joueur détecté pour n’exporter que sa collection. Laissez vide pour exporter tous les joueurs du monde.", "compare_directory": "Dossier d’export précédent (facultatif)", "browse_compare": "Choisir l’export précédent", "choose_compare_folder": "Choisir le dossier de l’export antérieur", "opening_compare_browser": "Ouverture du navigateur d’exports précédents…", "compare_folder_selected": "Dossier d’export précédent sélectionné.", "compare_folder_unchanged": "Dossier d’export précédent inchangé.", "compare_help": "Ajoute collection-diff.md à partir d’un dossier export_<date heure> antérieur.", "export_button": "Créer l’export NotebookLM", "save_root_required": "Choisissez d’abord un dossier de sauvegarde.", "scanning": "Recherche des mondes Palworld…", "no_saves": "Aucun fichier Level.sav trouvé dans ce dossier.", "saves_found": "%d monde(s) trouvé(s). Sélectionnez-en un ci-dessous.", "opening_browser": "Ouverture du navigateur de fichiers…", "world_selected": "Monde sélectionné. Choisissez un dossier parent puis créez l’export.", "save_and_output_required": "Sélectionnez un fichier Level.sav et un dossier parent d’exports.", "exporting": "Lecture de la sauvegarde et création d’un nouvel instantané…", "open_export_folder": "Ouvrir le dossier d’export dans l’Explorateur",
	},
}
