package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lxn/walk"
	"github.com/nyarime/nyarc/pkg/nya"
	. "github.com/lxn/walk/declarative"
)

func main() {
	var mw *walk.MainWindow
	var addressBar *walk.LineEdit
	var table *walk.TableView
	var statusBar *walk.StatusBarItem
	var model *FileModel
	var tb *walk.ToolBar

	currentDir := ""
	home, _ := os.UserHomeDir()
	if home != "" {
		currentDir = home
	}

	model = NewFileModel(currentDir)
	initIcons()

	navigate := func(dir string) {
		currentDir = dir
		if addressBar != nil {
			addressBar.SetText(currentDir)
		}
		model.SetDir(currentDir)
		if table != nil {
			table.Invalidate()
		}
		if mw != nil {
			mw.SetTitle("NekoArc")
		}
	}

	navigateArchive := func(path string) {
		currentDir = path
		if addressBar != nil {
			addressBar.SetText(currentDir)
		}
		model.SetArchive(path)
		if table != nil {
			table.Invalidate()
		}
		if mw != nil {
			mw.SetTitle(filepath.Base(path) + " - NekoArc")
		}
	}

	navigateGenericArchive := func(path string) {
		currentDir = path
		if addressBar != nil {
			addressBar.SetText(currentDir)
		}
		entries, err := listGenericArchive(path)
		if err != nil {
			walk.MsgBox(mw, "Error", "Cannot browse archive: "+err.Error(), walk.MsgBoxIconError)
			return
		}
		model.SetGenericArchive(path, entries)
		if table != nil { table.Invalidate() }
		if mw != nil { mw.SetTitle(filepath.Base(path) + " - NekoArc") }
	}

	goUp := func() {
		if model.inArchive {
			navigate(filepath.Dir(model.archivePath))
		} else if currentDir != "" {
			parent := filepath.Dir(currentDir)
			if parent == currentDir {
				// At drive root (C:\) → go to "My Computer" (drive list)
				currentDir = ""
				if addressBar != nil {
					addressBar.SetText("")
				}
				model.SetDir("")
				if table != nil { table.Invalidate() }
				if mw != nil { mw.SetTitle("NekoArc") }
			} else {
				navigate(parent)
			}
		}
	}

	activateItem := func() {
		idx := table.CurrentIndex()
		if idx < 0 || idx >= len(model.items) {
			return
		}
		item := model.items[idx]
		if item.Name == ".." {
			goUp()
		} else if item.IsDir {
			navigate(item.Path)
		} else if isArchiveFile(item.Path) && !model.inArchive {
			if strings.HasSuffix(strings.ToLower(item.Name), ".nya") {
				navigateArchive(item.Path)
			} else {
				// Other archives: list contents via archiver
				navigateGenericArchive(item.Path)
			}
		} else if !model.inArchive {
			exec.Command("cmd", "/c", "start", "", item.Path).Start()
		}
	}

	// Get selected file paths from table
	getSelectedPaths := func() []string {
		indices := table.SelectedIndexes()
		var paths []string
		for _, i := range indices {
			if i >= 0 && i < len(model.items) && model.items[i].Name != ".." {
				paths = append(paths, model.items[i].Path)
			}
		}
		return paths
	}

	doAdd := func() {
		if model.inArchive {
			// Add files to current archive
			dlg := new(walk.FileDialog)
			dlg.Title = "Add files to archive"
			if ok, _ := dlg.ShowOpenMultiple(mw); ok && len(dlg.FilePaths) > 0 {
				if err := archiveAddFiles(model.archivePath, dlg.FilePaths); err != nil {
					walk.MsgBox(mw, "Error", err.Error(), walk.MsgBoxIconError)
				} else {
					walk.MsgBox(mw, "Done", fmt.Sprintf("Added %d file(s)", len(dlg.FilePaths)), walk.MsgBoxIconInformation)
					navigateArchive(model.archivePath)
				}
			}
			return
		}
		paths := getSelectedPaths()
		if len(paths) == 0 {
			walk.MsgBox(mw, "Add", "Select files or folders in the file browser first", walk.MsgBoxIconInformation)
			return
		}
		showPackDialog(mw, paths)
	}

	doExtract := func() {
		if model.inArchive {
			showExtractDialog(mw, model.archivePath)
			return
		}
		paths := getSelectedPaths()
		if len(paths) == 1 && isArchiveFile(paths[0]) {
			showExtractDialog(mw, paths[0])
			return
		}
		walk.MsgBox(mw, "Extract", "Select an archive file (.nya/.zip/.rar/.7z) in the file browser", walk.MsgBoxIconInformation)
	}

	getNyaPath := func() string {
		if model.inArchive {
			return model.archivePath
		}
		paths := getSelectedPaths()
		if len(paths) == 1 && strings.HasSuffix(strings.ToLower(paths[0]), ".nya") {
			return paths[0]
		}
		walk.MsgBox(mw, "Info", "Select a .nya archive in the file browser", walk.MsgBoxIconInformation)
		return ""
	}

	doTestFn := func() {
		path := getNyaPath()
		if path == "" {
			return
		}
		log, count, ok, err := doTest(path)
		if err != nil {
			walk.MsgBox(mw, "Error", err.Error(), walk.MsgBoxIconError)
			log.Show(mw, "Test")
		} else if ok {
			walk.MsgBox(mw, "Test", fmt.Sprintf("OK: %d files, integrity verified", count), walk.MsgBoxIconInformation)
		} else {
			walk.MsgBox(mw, "Test", fmt.Sprintf("FAILED: %d files, archive corrupted", count), walk.MsgBoxIconWarning)
			log.Show(mw, "Test")
		}
	}

	doRepairFn := func() {
		path := getNyaPath()
		if path == "" {
			return
		}
		log, total, damaged, repaired, err := doRepair(path)
		if err != nil {
			walk.MsgBox(mw, "Repair", err.Error(), walk.MsgBoxIconError)
		} else if damaged == 0 {
			walk.MsgBox(mw, "Repair", fmt.Sprintf("No damage found (%d chunks verified)", total), walk.MsgBoxIconInformation)
		} else {
			walk.MsgBox(mw, "Repair", fmt.Sprintf("%d chunks, %d damaged, %d repaired", total, damaged, repaired), walk.MsgBoxIconInformation)
		}
		log.Show(mw, "Repair")
	}

	doInfoFn := func() {
		path := getNyaPath()
		if path == "" {
			return
		}
		showInfoDialog(mw, path)
	}

	doDeleteFn := func() {
		if model.inArchive {
			// Delete files from archive
			indices := table.SelectedIndexes()
			names := getArchiveRelPaths(model.items, indices)
			if len(names) == 0 { return }
			msg := fmt.Sprintf("Delete %d file(s) from archive?\n", len(names))
			for i, n := range names {
				if i < 5 { msg += n + "\n" }
			}
			if walk.MsgBox(mw, "Delete from archive", msg, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) == walk.DlgCmdYes {
				if err := archiveDeleteFiles(model.archivePath, names); err != nil {
					walk.MsgBox(mw, "Error", err.Error(), walk.MsgBoxIconError)
				} else {
					// Refresh archive view
					navigateArchive(model.archivePath)
				}
			}
			return
		}
		paths := getSelectedPaths()
		if len(paths) == 0 {
			return
		}
		msg := fmt.Sprintf("Delete %d item(s)?\n\n", len(paths))
		for i, p := range paths {
			if i < 5 {
				msg += filepath.Base(p) + "\n"
			}
		}
		if len(paths) > 5 {
			msg += fmt.Sprintf("... and %d more\n", len(paths)-5)
		}
		msg += "\nThis will permanently delete the files."
		if walk.MsgBox(mw, "Confirm Delete", msg, walk.MsgBoxYesNo|walk.MsgBoxIconWarning) == walk.DlgCmdYes {
			for _, p := range paths {
				os.RemoveAll(p)
			}
			model.SetDir(currentDir)
			if table != nil { table.Invalidate() }
		}
	}

	if err := (MainWindow{
		AssignTo: &mw,
		Title:    "NekoArc",
		MinSize:  Size{Width: 700, Height: 500},
		Size:     Size{Width: 900, Height: 650},
		Layout:   VBox{MarginsZero: true, SpacingZero: true},
		// ─── Menu bar (WinRAR style) ───
		MenuItems: []MenuItem{
			Menu{
				Text: "&File",
				Items: []MenuItem{
					Action{Text: "&Open Archive...", OnTriggered: func() {
						dlg := new(walk.FileDialog)
						dlg.Filter = "Nyarc Archives (*.nya)|*.nya|All Files (*.*)|*.*"
						if ok, _ := dlg.ShowOpen(mw); ok {
							navigateArchive(dlg.FilePath)
						}
					}},
					Separator{},
					Action{Text: "E&xit", OnTriggered: func() { mw.Close() }},
				},
			},
			Menu{
				Text: "&Commands",
				Items: []MenuItem{
					Action{Text: "&Add to archive...", Shortcut: Shortcut{Modifiers: walk.ModAlt, Key: walk.KeyA}, OnTriggered: func() { doAdd() }},
					Action{Text: "&Extract to...", Shortcut: Shortcut{Modifiers: walk.ModAlt, Key: walk.KeyE}, OnTriggered: func() { doExtract() }},
					Action{Text: "&Test archive", Shortcut: Shortcut{Modifiers: walk.ModAlt, Key: walk.KeyT}, OnTriggered: func() { doTestFn() }},
					Action{Text: "&Repair archive", OnTriggered: func() { doRepairFn() }},
					Action{Text: "Archive &info", Shortcut: Shortcut{Modifiers: walk.ModAlt, Key: walk.KeyI}, OnTriggered: func() { doInfoFn() }},
					Separator{},
					Action{Text: "&Delete files", Shortcut: Shortcut{Key: walk.KeyDelete}, OnTriggered: func() { doDeleteFn() }},
				},
			},
			Menu{
				Text: "&Help",
				Items: []MenuItem{
					Action{Text: "&About NekoArc", OnTriggered: func() {
						walk.MsgBox(mw, "About NekoArc", "NekoArc v0.3.0\n\nNyarc archive manager with FEC self-repair\nhttps://github.com/Nyarime/NekoArc", walk.MsgBoxIconInformation)
					}},
				},
			},
		},
		Children: []Widget{
			// ─── Toolbar ───
			ToolBar{
				AssignTo:    &tb,
				ButtonStyle: ToolBarButtonImageAboveText,
				MaxTextRows: 1,
			},
			// ─── Address bar ───
			Composite{
				Layout: HBox{Margins: Margins{Left: 2, Top: 0, Right: 2, Bottom: 2}},
				Children: []Widget{
					PushButton{
						Text:    "^",
						MaxSize: Size{Width: 28},
						OnClicked: func() { goUp() },
					},
					LineEdit{
						AssignTo: &addressBar,
						Text:     currentDir,
						OnKeyDown: func(key walk.Key) {
							if key == walk.KeyReturn {
								dir := addressBar.Text()
								if strings.HasSuffix(strings.ToLower(dir), ".nya") {
									if _, err := os.Stat(dir); err == nil {
										navigateArchive(dir)
										return
									}
								}
								if info, err := os.Stat(dir); err == nil && info.IsDir() {
									navigate(dir)
								}
							}
						},
					},
				},
			},
			// ─── File list ───
			TableView{
				AssignTo:              &table,
				AlternatingRowBG:      true,
				ColumnsOrderable:      true,
				MultiSelection:        true,
				LastColumnStretched:    true,
			Columns: []TableViewColumn{
					{Title: "Name", Width: 280},
					{Title: "Size", Width: 90, Alignment: AlignFar},
					{Title: "Type", Width: 120},
					{Title: "Modified", Width: 140},
				},
				Model:           model,
				OnItemActivated: func() { activateItem() },
				ContextMenuItems: []MenuItem{
					Action{Text: "Add to archive...", OnTriggered: func() { doAdd() }},
					Action{Text: "Extract to...", OnTriggered: func() { doExtract() }},
					Separator{},
					Action{Text: "Test archive", OnTriggered: func() { doTestFn() }},
					Action{Text: "Repair archive", OnTriggered: func() { doRepairFn() }},
					Action{Text: "Archive info", OnTriggered: func() { doInfoFn() }},
					Separator{},
					Action{Text: "Open", OnTriggered: func() { activateItem() }},
					Action{Text: "Rename", OnTriggered: func() {
						if !model.inArchive { return }
						paths := getSelectedPaths()
						if len(paths) != 1 { return }
						// Simple rename dialog
						var dlg *walk.Dialog
						var nameEdit *walk.LineEdit
						Dialog{
							AssignTo: &dlg,
							Title: "Rename",
							MinSize: Size{Width: 350, Height: 100},
							Layout: VBox{},
							Children: []Widget{
								LineEdit{AssignTo: &nameEdit, Text: filepath.Base(paths[0])},
								Composite{Layout: HBox{}, Children: []Widget{
									HSpacer{},
									PushButton{Text: "OK", OnClicked: func() {
										// TODO: implement rename inside archive
										walk.MsgBox(dlg, "Info", "Rename inside archive coming soon", walk.MsgBoxIconInformation)
										dlg.Accept()
									}},
									PushButton{Text: "Cancel", OnClicked: func() { dlg.Cancel() }},
								}},
							},
						}.Run(mw)
					}},
					Action{Text: "Copy to...", OnTriggered: func() {
						paths := getSelectedPaths()
						if len(paths) == 0 { return }
						fdlg := new(walk.FileDialog)
						fdlg.Title = "Copy to folder"
						if ok, _ := fdlg.ShowBrowseFolder(mw); ok {
							for _, src := range paths {
								dst := filepath.Join(fdlg.FilePath, filepath.Base(src))
								copyFileOrDir(src, dst)
							}
							walk.MsgBox(mw, "Done", fmt.Sprintf("Copied %d item(s)", len(paths)), walk.MsgBoxIconInformation)
						}
					}},
					Separator{},
					Action{Text: "Delete", OnTriggered: func() { doDeleteFn() }},
				},
			},
		},
		StatusBarItems: []StatusBarItem{
			{AssignTo: &statusBar, Width: 0},
		},
	}.Create()); err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Populate toolbar with icons
	addTBAction := func(text string, icon *walk.Icon, handler func()) {
		action := walk.NewAction()
		action.SetText(text)
		if icon != nil {
			action.SetImage(icon)
		}
		action.Triggered().Attach(handler)
		tb.Actions().Add(action)
	}
	addTBAction("Add", iconAdd, func() { doAdd() })
	addTBAction("Extract", iconExtract, func() { doExtract() })
	addTBAction("Test", iconTest, func() { doTestFn() })
	addTBAction("Repair", iconRepair, func() { doRepairFn() })
	addTBAction("Info", iconInfo, func() { doInfoFn() })
	sep := walk.NewSeparatorAction()
	tb.Actions().Add(sep)
	addTBAction("Delete", iconDelete, func() { doDeleteFn() })

	updateStatus := func() {
		count := len(model.items)
		dirs := 0
		var totalSize int64
		for _, f := range model.items {
			if f.Name == ".." {
				continue
			}
			if f.IsDir {
				dirs++
			} else {
				totalSize += f.Size
			}
		}
		files := count - dirs
		if model.items != nil && model.items[0].Name == ".." {
			files--
		}
		label := fmt.Sprintf("%d folder(s), %d file(s), %s", dirs, files, humanSize(totalSize))
		if model.inArchive {
			label = model.archiveInfo
		}
		statusBar.SetText(label)
	}
	model.onUpdate = updateStatus
	updateStatus()

	mw.Run()
}

// ─── Pack dialog ───

func showPackDialog(owner walk.Form, files []string) {
	var dlg *walk.Dialog
	var archiveNameEdit *walk.LineEdit
	var levelEdit *walk.NumberEdit
	var fecEdit *walk.NumberEdit
	var passwordEdit *walk.LineEdit
	var passwordConfirm *walk.LineEdit
	var solidCheck *walk.CheckBox
	var sfxCheck *walk.CheckBox
	var splitEdit *walk.LineEdit
	var destDirEdit *walk.LineEdit

	var totalSize int64
	var fileCount int
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if info.IsDir() {
			filepath.Walk(f, func(_ string, fi os.FileInfo, _ error) error {
				if fi != nil && !fi.IsDir() {
					totalSize += fi.Size()
					fileCount++
				}
				return nil
			})
		} else {
			totalSize += info.Size()
			fileCount++
		}
	}

	defaultName := strings.TrimSuffix(filepath.Base(files[0]), filepath.Ext(files[0]))
	defaultDir := filepath.Dir(files[0])

	Dialog{
		AssignTo: &dlg,
		Title:    "Archive name and parameters",
		MinSize:  Size{Width: 520, Height: 450},
		Layout:   VBox{},
		Children: []Widget{
			// Archive name at top (like WinRAR)
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: "Archive name:"},
					LineEdit{AssignTo: &archiveNameEdit, Text: defaultName + ".nya"},
					Label{Text: "Destination:"},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							LineEdit{AssignTo: &destDirEdit, Text: defaultDir},
							PushButton{Text: "...", MaxSize: Size{Width: 30}, OnClicked: func() {
								fdlg := new(walk.FileDialog)
								fdlg.Title = "Select destination"
								if ok, _ := fdlg.ShowBrowseFolder(owner); ok {
									destDirEdit.SetText(fdlg.FilePath)
								}
							}},
						},
					},
				},
			},
			TabWidget{
				Pages: []TabPage{
					// ─── General tab ───
					{
						Title:  "General",
						Layout: VBox{},
						Children: []Widget{
							GroupBox{
								Title:  "Compression",
								Layout: Grid{Columns: 2},
								Children: []Widget{
									Label{Text: "Compression level (1-19):"},
									NumberEdit{AssignTo: &levelEdit, Value: 9, MinValue: 1, MaxValue: 19},
								},
							},
							GroupBox{
								Title:  "Options",
								Layout: VBox{},
								Children: []Widget{
									CheckBox{AssignTo: &solidCheck, Text: "Create solid archive"},
									CheckBox{AssignTo: &sfxCheck, Text: "Create SFX (self-extracting) archive"},
								},
							},
							GroupBox{
								Title:  "Split to volumes",
								Layout: Grid{Columns: 2},
								Children: []Widget{
									Label{Text: "Volume size (1G, 500M, 0=no split):"},
									LineEdit{AssignTo: &splitEdit, Text: "0"},
								},
							},
							Label{Text: fmt.Sprintf("Selected: %d files, %s", fileCount, humanSize(totalSize))},
						},
					},
					// ─── Advanced tab ───
					{
						Title:  "Advanced",
						Layout: VBox{},
						Children: []Widget{
							GroupBox{
								Title:  "FEC Recovery (RaptorQ)",
								Layout: Grid{Columns: 2},
								Children: []Widget{
									Label{Text: "Recovery data (%):"},
									NumberEdit{AssignTo: &fecEdit, Value: 3, MinValue: 0, MaxValue: 500},
									Label{Text: "", ColumnSpan: 2},
									Label{Text: "0% = no recovery data (not recommended)", ColumnSpan: 2},
									Label{Text: "10% = can recover minor corruption (recommended)", ColumnSpan: 2},
									Label{Text: "100% = can recover from 50% corruption (extreme)", ColumnSpan: 2},
								},
							},
							GroupBox{
								Title:  "Encryption",
								Layout: Grid{Columns: 2},
								Children: []Widget{
									Label{Text: "Password:"},
									LineEdit{AssignTo: &passwordEdit, PasswordMode: true},
									Label{Text: "Confirm:"},
									LineEdit{AssignTo: &passwordConfirm, PasswordMode: true},
								},
							},
						},
					},
				},
			},
			// OK / Cancel
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "OK",
						OnClicked: func() {
							// Validate password
							if passwordEdit.Text() != "" && passwordEdit.Text() != passwordConfirm.Text() {
								walk.MsgBox(owner, "Error", "Passwords do not match", walk.MsgBoxIconError)
								return
							}
							outDir := destDirEdit.Text()
							log, err := doPack(PackOptions{
								Inputs:   files,
								Output:   outDir,
								Level:    int(levelEdit.Value()),
								FEC:      int(fecEdit.Value()),
								Password: passwordEdit.Text(),
								Solid:    solidCheck.Checked(),
								SFX:      sfxCheck.Checked(),
								SplitSize: splitEdit.Text(),
							})
							if err != nil {
								walk.MsgBox(owner, "Error", err.Error(), walk.MsgBoxIconError)
							} else {
								walk.MsgBox(owner, "Done", "Archive created successfully", walk.MsgBoxIconInformation)
							}
							if log.HasIssues() {
								log.Show(owner, "Pack")
							}
							dlg.Accept()
						},
					},
					PushButton{
						Text:      "Cancel",
						OnClicked: func() { dlg.Cancel() },
					},
				},
			},
		},
	}.Run(owner)
}

// ─── Extract dialog ───

func showExtractDialog(owner walk.Form, archivePath string) {
	var dlg *walk.Dialog
	var destEdit *walk.LineEdit
	var passwordEdit *walk.LineEdit

	defaultDest := filepath.Dir(archivePath)
	baseName := strings.TrimSuffix(filepath.Base(archivePath), filepath.Ext(archivePath))
	defaultDest = filepath.Join(defaultDest, baseName)

	Dialog{
		AssignTo: &dlg,
		Title:    "Extraction path and options",
		MinSize:  Size{Width: 450, Height: 300},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: "Destination:"},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							LineEdit{AssignTo: &destEdit, Text: defaultDest},
							PushButton{Text: "...", MaxSize: Size{Width: 30}, OnClicked: func() {
								fdlg := new(walk.FileDialog)
								fdlg.Title = "Select destination folder"
								if ok, _ := fdlg.ShowBrowseFolder(owner); ok {
									destEdit.SetText(fdlg.FilePath)
								}
							}},
						},
					},
					Label{Text: "Password (if encrypted):"},
					LineEdit{AssignTo: &passwordEdit, PasswordMode: true},
				},
			},
			Label{Text: fmt.Sprintf("Archive: %s", filepath.Base(archivePath))},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "OK",
						OnClicked: func() {
							dest := destEdit.Text()
							os.MkdirAll(dest, 0755)
							log, err := doExtract(archivePath, dest)
							if err != nil {
								walk.MsgBox(owner, "Error", err.Error(), walk.MsgBoxIconError)
							} else {
								walk.MsgBox(owner, "Done", "Extracted successfully to:\n"+dest, walk.MsgBoxIconInformation)
							}
							if log.HasIssues() {
								log.Show(owner, "Extract")
							}
							dlg.Accept()
						},
					},
					PushButton{
						Text:      "Cancel",
						OnClicked: func() { dlg.Cancel() },
					},
				},
			},
		},
	}.Run(owner)
}

// ─── Info dialog ───

func showInfoDialog(owner walk.Form, path string) {
	fi, err := os.Stat(path)
	if err != nil {
		walk.MsgBox(owner, "Error", err.Error(), walk.MsgBoxIconError)
		return
	}
	r, err := nya.Open(path)
	if err != nil {
		walk.MsgBox(owner, "Info", fmt.Sprintf("File: %s\nSize: %s\nError: %s", path, humanSize(fi.Size()), err.Error()), walk.MsgBoxIconInformation)
		return
	}

	files := r.List()
	var totalOrig uint64
	for _, f := range files {
		totalOrig += f.OriginalSize
	}

	ratio := float64(0)
	if totalOrig > 0 {
		ratio = float64(fi.Size()) / float64(totalOrig) * 100
	}

	// Count dirs vs files
	dirCount := 0
	fileCount := 0
	for _, f := range files {
		if f.EntryType == 1 {
			dirCount++
		} else {
			fileCount++
		}
	}

	var dlg *walk.Dialog

	Dialog{
		AssignTo: &dlg,
		Title:    fmt.Sprintf("Archive: %s", filepath.Base(path)),
		MinSize:  Size{Width: 400, Height: 350},
		Layout:   VBox{},
		Children: []Widget{
			TabWidget{
				Pages: []TabPage{
					{
						Title:  "Info",
						Layout: Grid{Columns: 2, Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}},
						Children: []Widget{
							Label{Text: "NekoArc Archive", Font: Font{Bold: true}, ColumnSpan: 2},
							Label{Text: "", ColumnSpan: 2},
							Label{Text: "Format:"},
							Label{Text: fmt.Sprintf("NYA v%d.%d / Zstd+RaptorQ", r.Header.VersionMajor, r.Header.VersionMinor)},
							Label{Text: "Folders:"},
							Label{Text: fmt.Sprintf("%d", dirCount)},
							Label{Text: "Files:"},
							Label{Text: fmt.Sprintf("%d", fileCount)},
							Label{Text: "Total size:"},
							Label{Text: fmt.Sprintf("%s (%d bytes)", humanSize(int64(totalOrig)), totalOrig)},
							Label{Text: "Packed size:"},
							Label{Text: fmt.Sprintf("%s (%d bytes)", humanSize(fi.Size()), fi.Size())},
							Label{Text: "Compression ratio:"},
							Label{Text: fmt.Sprintf("%.1f%%", ratio)},
							Label{Text: "", ColumnSpan: 2},
							Label{Text: "FEC recovery:"},
							Label{Text: func() string {
								if r.FecLen > 0 && totalOrig > 0 {
									pct := float64(r.FecLen) / float64(totalOrig) * 100
									return fmt.Sprintf("%.0f%% (RaptorQ)", pct)
								}
								if r.FecLen > 0 {
									return "Enabled (RaptorQ)"
								}
								return "None"
							}()},
							Label{Text: "Encryption:"},
							Label{Text: func() string {
								if r.Header.Flags&0x01 != 0 {
									return "AES-256-GCM"
								}
								return "None"
							}()},
						},
					},
					{
						Title:  "Comment",
						Layout: VBox{},
						Children: []Widget{
							TextEdit{Text: "(No comment)", ReadOnly: true},
						},
					},
					{
						Title:  "SFX",
						Layout: VBox{},
						Children: []Widget{
							Label{Text: "Self-extracting module: Not present"},
						},
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{Text: "OK", OnClicked: func() { dlg.Accept() }},
				},
			},
		},
	}.Run(owner)
}

// ─── Sortable File model ───

type FileModel struct {
	walk.SorterBase
	walk.TableModelBase
	items       []FileEntry
	onUpdate    func()
	inArchive   bool
	archivePath string
	archiveInfo string
	sortCol     int
	sortAsc     bool
}

func NewFileModel(dir string) *FileModel {
	m := &FileModel{sortCol: 0, sortAsc: true}
	m.items = listDir(dir)
	m.addParentEntry(dir)
	return m
}

// Image returns the file path for walk to resolve the shell icon automatically
func (m *FileModel) Image(row int) interface{} {
	if row < 0 || row >= len(m.items) {
		return nil
	}
	item := m.items[row]
	if item.Name == ".." || item.IsDir {
		if item.Path != "" {
			return item.Path
		}
		// For "My Computer" level, return a generic folder
		return "C:\\Windows"
	}
	// For archive entries with relative paths, create a dummy path with correct extension
	if m.inArchive && !filepath.IsAbs(item.Path) {
		return "C:\\dummy" + filepath.Ext(item.Name)
	}
	return item.Path
}

func (m *FileModel) addParentEntry(dir string) {
	if dir == "" {
		return // Already at "My Computer"
	}
	parent := filepath.Dir(dir)
	if parent == dir {
		// At drive root → parent is "My Computer" (empty = drive list)
		parent = ""
	}
	entry := FileEntry{Name: "..", Path: parent, IsDir: true}
	m.items = append([]FileEntry{entry}, m.items...)
}

func (m *FileModel) SetDir(dir string) {
	m.items = listDir(dir)
	m.addParentEntry(dir)
	m.inArchive = false
	m.archivePath = ""
	m.doSort()
	m.PublishRowsReset()
	if m.onUpdate != nil {
		m.onUpdate()
	}
}

func (m *FileModel) SetArchive(path string) {
	r, err := nya.Open(path)
	if err != nil {
		return
	}
	files := r.List()
	fi, _ := os.Stat(path)
	archiveSize := int64(0)
	if fi != nil {
		archiveSize = fi.Size()
	}
	m.items = []FileEntry{{Name: "..", Path: filepath.Dir(path), IsDir: true}}
	var totalOrig uint64
	for _, f := range files {
		totalOrig += f.OriginalSize
		m.items = append(m.items, FileEntry{
			Name:    f.Path,
			Path:    f.Path,
			Size:    int64(f.OriginalSize),
			ModTime: func() string {
				if f.MTimeNano > 0 {
					return time.Unix(0, f.MTimeNano).Format("2006-01-02 15:04")
				}
				return ""
			}(),
		})
	}
	m.inArchive = true
	m.archivePath = path
	m.archiveInfo = fmt.Sprintf("NYA archive, unpacked size: %s, packed: %s, ratio: %.0f%%",
		humanSize(int64(totalOrig)), humanSize(archiveSize),
		func() float64 {
			if totalOrig == 0 { return 0 }
			return float64(archiveSize) / float64(totalOrig) * 100
		}())
	m.PublishRowsReset()
	if m.onUpdate != nil {
		m.onUpdate()
	}
}

func (m *FileModel) RowCount() int { return len(m.items) }

func (m *FileModel) Value(row, col int) interface{} {
	if row < 0 || row >= len(m.items) {
		return ""
	}
	item := m.items[row]
	switch col {
	case 0:
		if item.Name == ".." {
			return ".."
		}
		return item.Name
	case 1:
		if item.IsDir {
			return ""
		}
		return humanSize(item.Size)
	case 2:
		if item.IsDir {
			return "File folder"
		}
		ext := ""
		for i := len(item.Name) - 1; i >= 0; i-- {
			if item.Name[i] == '.' {
				ext = strings.ToUpper(item.Name[i+1:]) + " File"
				break
			}
		}
		if ext == "" {
			ext = "File"
		}
		return ext
	case 3:
		return item.ModTime
	}
	return ""
}

func (m *FileModel) Sort(col int, order walk.SortOrder) error {
	m.sortCol = col
	m.sortAsc = order == walk.SortAscending
	m.doSort()
	m.PublishRowsReset()
	return m.SorterBase.Sort(col, order)
}

func (m *FileModel) doSort() {
	col := m.sortCol
	asc := m.sortAsc
	sort.SliceStable(m.items, func(i, j int) bool {
		a, b := m.items[i], m.items[j]
		// ".." always first
		if a.Name == ".." {
			return true
		}
		if b.Name == ".." {
			return false
		}
		// Dirs before files
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		var less bool
		switch col {
		case 0:
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case 1:
			less = a.Size < b.Size
		case 2:
			// Type sort by extension
			less = strings.ToLower(filepath.Ext(a.Name)) < strings.ToLower(filepath.Ext(b.Name))
		case 3:
			less = a.ModTime < b.ModTime
		default:
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		if !asc {
			return !less
		}
		return less
	})
}

func (m *FileModel) SetGenericArchive(path string, entries []FileEntry) {
	m.items = []FileEntry{{Name: "..", Path: filepath.Dir(path), IsDir: true}}
	m.items = append(m.items, entries...)
	m.inArchive = true
	m.archivePath = path

	var totalSize int64
	for _, e := range entries {
		if !e.IsDir { totalSize += e.Size }
	}
	fi, _ := os.Stat(path)
	archiveSize := int64(0)
	if fi != nil { archiveSize = fi.Size() }

	m.archiveInfo = fmt.Sprintf("Archive: %s, %d files, unpacked: %s, packed: %s",
		filepath.Ext(path), len(entries), humanSize(totalSize), humanSize(archiveSize))
	m.PublishRowsReset()
	if m.onUpdate != nil { m.onUpdate() }
}
