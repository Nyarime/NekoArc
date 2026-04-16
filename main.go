package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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
	}

	goUp := func() {
		if model.inArchive {
			navigate(filepath.Dir(model.archivePath))
		} else if currentDir != "" {
			parent := filepath.Dir(currentDir)
			if parent != currentDir {
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
		} else if strings.HasSuffix(strings.ToLower(item.Name), ".nya") && !model.inArchive {
			navigateArchive(item.Path)
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
		paths := getSelectedPaths()
		if len(paths) == 0 {
			// No selection: open file dialog
			dlg := new(walk.FileDialog)
			dlg.Title = "Select files to compress"
			dlg.FilePath = currentDir
			if ok, _ := dlg.ShowOpenMultiple(mw); ok && len(dlg.FilePaths) > 0 {
				paths = dlg.FilePaths
			}
		}
		if len(paths) > 0 {
			showPackDialog(mw, paths)
		}
	}

	doExtract := func() {
		if model.inArchive {
			showExtractDialog(mw, model.archivePath)
			return
		}
		// Check if a .nya is selected
		paths := getSelectedPaths()
		if len(paths) == 1 && strings.HasSuffix(strings.ToLower(paths[0]), ".nya") {
			showExtractDialog(mw, paths[0])
			return
		}
		dlg := new(walk.FileDialog)
		dlg.Title = "Select archive"
		dlg.Filter = "Nyarc Archives (*.nya)|*.nya|All Files (*.*)|*.*"
		if ok, _ := dlg.ShowOpen(mw); ok {
			showExtractDialog(mw, dlg.FilePath)
		}
	}

	getNyaPath := func() string {
		if model.inArchive {
			return model.archivePath
		}
		paths := getSelectedPaths()
		if len(paths) == 1 && strings.HasSuffix(strings.ToLower(paths[0]), ".nya") {
			return paths[0]
		}
		dlg := new(walk.FileDialog)
		dlg.Title = "Select .nya archive"
		dlg.Filter = "Nyarc Archives (*.nya)|*.nya"
		if ok, _ := dlg.ShowOpen(mw); ok {
			return dlg.FilePath
		}
		return ""
	}

	doTestFn := func() {
		path := getNyaPath()
		if path == "" {
			return
		}
		count, ok, err := doTest(path)
		if err != nil {
			walk.MsgBox(mw, "Error", err.Error(), walk.MsgBoxIconError)
		} else if ok {
			walk.MsgBox(mw, "Test", fmt.Sprintf("OK: %d files, integrity verified", count), walk.MsgBoxIconInformation)
		} else {
			walk.MsgBox(mw, "Test", fmt.Sprintf("FAILED: %d files, archive corrupted", count), walk.MsgBoxIconWarning)
		}
	}

	doRepairFn := func() {
		path := getNyaPath()
		if path == "" {
			return
		}
		total, damaged, repaired, err := doRepair(path)
		if err != nil {
			walk.MsgBox(mw, "Repair", err.Error(), walk.MsgBoxIconError)
		} else if damaged == 0 {
			walk.MsgBox(mw, "Repair", fmt.Sprintf("No damage found (%d chunks verified)", total), walk.MsgBoxIconInformation)
		} else {
			walk.MsgBox(mw, "Repair", fmt.Sprintf("%d chunks, %d damaged, %d repaired", total, damaged, repaired), walk.MsgBoxIconInformation)
		}
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
			return
		}
		paths := getSelectedPaths()
		if len(paths) == 0 {
			return
		}
		msg := fmt.Sprintf("Delete %d item(s)?\n", len(paths))
		for i, p := range paths {
			if i < 5 {
				msg += filepath.Base(p) + "\n"
			}
		}
		if len(paths) > 5 {
			msg += fmt.Sprintf("... and %d more", len(paths)-5)
		}
		if walk.MsgBox(mw, "Confirm Delete", msg, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) == walk.DlgCmdYes {
			for _, p := range paths {
				os.RemoveAll(p)
			}
			model.SetDir(currentDir)
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
			label = fmt.Sprintf("[%s] %d file(s), %s", filepath.Base(model.archivePath), count-1, humanSize(totalSize))
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
	var levelEdit *walk.NumberEdit
	var fecEdit *walk.NumberEdit
	var passwordEdit *walk.LineEdit
	var solidCheck *walk.CheckBox
	var sfxCheck *walk.CheckBox

	var totalSize int64
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if info.IsDir() {
			filepath.Walk(f, func(_ string, fi os.FileInfo, _ error) error {
				if fi != nil && !fi.IsDir() {
					totalSize += fi.Size()
				}
				return nil
			})
		} else {
			totalSize += info.Size()
		}
	}

	Dialog{
		AssignTo: &dlg,
		Title:    fmt.Sprintf("Add to archive (%d items, %s)", len(files), humanSize(totalSize)),
		MinSize:  Size{Width: 420, Height: 320},
		Layout:   VBox{},
		Children: []Widget{
			GroupBox{
				Title:  "Archive parameters",
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: "Compression level (1-19):"},
					NumberEdit{AssignTo: &levelEdit, Value: 9, MinValue: 1, MaxValue: 19},
					Label{Text: "FEC recovery (%):"},
					NumberEdit{AssignTo: &fecEdit, Value: 100, MinValue: 0, MaxValue: 500},
					Label{Text: "Password:"},
					LineEdit{AssignTo: &passwordEdit, PasswordMode: true},
				},
			},
			GroupBox{
				Title:  "Options",
				Layout: VBox{},
				Children: []Widget{
					CheckBox{AssignTo: &solidCheck, Text: "Create solid archive"},
					CheckBox{AssignTo: &sfxCheck, Text: "Create SFX archive"},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "OK",
						OnClicked: func() {
							fdlg := new(walk.FileDialog)
							fdlg.Title = "Save archive to folder"
							if ok, _ := fdlg.ShowBrowseFolder(owner); ok {
								err := doPack(PackOptions{
									Inputs:   files,
									Output:   fdlg.FilePath,
									Level:    int(levelEdit.Value()),
									FEC:      int(fecEdit.Value()),
									Password: passwordEdit.Text(),
									Solid:    solidCheck.Checked(),
									SFX:      sfxCheck.Checked(),
								})
								if err != nil {
									walk.MsgBox(owner, "Error", err.Error(), walk.MsgBoxIconError)
								} else {
									walk.MsgBox(owner, "Done", "Archive created successfully", walk.MsgBoxIconInformation)
								}
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
	fdlg := new(walk.FileDialog)
	fdlg.Title = "Extract to folder"
	fdlg.FilePath = filepath.Dir(archivePath)
	if ok, _ := fdlg.ShowBrowseFolder(owner); ok {
		err := doExtract(archivePath, fdlg.FilePath)
		if err != nil {
			walk.MsgBox(owner, "Error", err.Error(), walk.MsgBoxIconError)
		} else {
			walk.MsgBox(owner, "Done", "Extracted successfully to:\n"+fdlg.FilePath, walk.MsgBoxIconInformation)
		}
	}
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
	details := ""
	for _, f := range files {
		totalOrig += f.OriginalSize
		details += fmt.Sprintf("  %s  (%s)\n", f.Path, humanSize(int64(f.OriginalSize)))
	}
	ratio := float64(0)
	if totalOrig > 0 {
		ratio = float64(fi.Size()) / float64(totalOrig) * 100
	}
	walk.MsgBox(owner, "Archive Info", fmt.Sprintf(
		"Archive: %s\nArchive size: %s\nFiles: %d\nOriginal size: %s\nCompression ratio: %.1f%%\n\n%s",
		filepath.Base(path), humanSize(fi.Size()), len(files), humanSize(int64(totalOrig)), ratio, details,
	), walk.MsgBoxIconInformation)
}

// ─── Sortable File model ───

type FileModel struct {
	walk.SorterBase
	walk.TableModelBase
	items       []FileEntry
	onUpdate    func()
	inArchive   bool
	archivePath string
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
		// Return any existing directory path for folder icon
		return item.Path
	}
	return item.Path
}

func (m *FileModel) addParentEntry(dir string) {
	if dir == "" {
		return
	}
	parent := filepath.Dir(dir)
	if parent == dir {
		return
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
	m.items = []FileEntry{{Name: "..", Path: filepath.Dir(path), IsDir: true}}
	for _, f := range files {
		m.items = append(m.items, FileEntry{
			Name: f.Path,
			Path: f.Path,
			Size: int64(f.OriginalSize),
		})
	}
	m.inArchive = true
	m.archivePath = path
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
