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

	currentDir := ""
	home, _ := os.UserHomeDir()
	if home != "" {
		currentDir = home
	}

	model = NewFileModel(currentDir)

	navigate := func(dir string) {
		currentDir = dir
		if addressBar != nil {
			addressBar.SetText(currentDir)
		}
		model.SetDir(currentDir)
		if table != nil {
			table.SetCurrentIndex(0)
		}
	}

	navigateArchive := func(path string) {
		currentDir = path
		if addressBar != nil {
			addressBar.SetText(currentDir)
		}
		model.SetArchive(path)
		if table != nil {
			table.SetCurrentIndex(0)
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

	if err := (MainWindow{
		AssignTo: &mw,
		Title:    "NekoArc",
		MinSize:  Size{Width: 700, Height: 500},
		Size:     Size{Width: 900, Height: 650},
		Layout:   VBox{MarginsZero: true, SpacingZero: true},
		Children: []Widget{
			// ─── Toolbar ───
			Composite{
				Layout: HBox{Margins: Margins{Left: 4, Top: 4, Right: 4, Bottom: 4}},
				Children: []Widget{
					PushButton{
						Text: "Add",
						OnClicked: func() {
							dlg := new(walk.FileDialog)
							dlg.Title = "Select files to compress"
							dlg.FilePath = currentDir
							if ok, _ := dlg.ShowOpenMultiple(mw); ok && len(dlg.FilePaths) > 0 {
								showPackDialog(mw, dlg.FilePaths)
							}
						},
					},
					PushButton{
						Text: "Extract",
						OnClicked: func() {
							if model.inArchive {
								showExtractDialog(mw, model.archivePath)
								return
							}
							dlg := new(walk.FileDialog)
							dlg.Title = "Select archive"
							dlg.Filter = "Nyarc Archives (*.nya)|*.nya|All Files (*.*)|*.*"
							if ok, _ := dlg.ShowOpen(mw); ok {
								showExtractDialog(mw, dlg.FilePath)
							}
						},
					},
					PushButton{
						Text: "Test",
						OnClicked: func() {
							path := ""
							if model.inArchive {
								path = model.archivePath
							} else {
								dlg := new(walk.FileDialog)
								dlg.Title = "Select .nya archive to test"
								dlg.Filter = "Nyarc Archives (*.nya)|*.nya"
								if ok, _ := dlg.ShowOpen(mw); !ok {
									return
								} else {
									path = dlg.FilePath
								}
							}
							count, ok, err := doTest(path)
							if err != nil {
								walk.MsgBox(mw, "Error", err.Error(), walk.MsgBoxIconError)
							} else if ok {
								walk.MsgBox(mw, "Test", fmt.Sprintf("OK: %d files, integrity verified", count), walk.MsgBoxIconInformation)
							} else {
								walk.MsgBox(mw, "Test", fmt.Sprintf("FAILED: %d files, archive corrupted", count), walk.MsgBoxIconWarning)
							}
						},
					},
					PushButton{
						Text: "Repair",
						OnClicked: func() {
							path := ""
							if model.inArchive {
								path = model.archivePath
							} else {
								dlg := new(walk.FileDialog)
								dlg.Title = "Select .nya archive to repair"
								dlg.Filter = "Nyarc Archives (*.nya)|*.nya"
								if ok, _ := dlg.ShowOpen(mw); !ok {
									return
								} else {
									path = dlg.FilePath
								}
							}
							total, damaged, repaired, err := doRepair(path)
							if err != nil {
								walk.MsgBox(mw, "Repair", err.Error(), walk.MsgBoxIconError)
							} else if damaged == 0 {
								walk.MsgBox(mw, "Repair", fmt.Sprintf("No damage found (%d chunks verified)", total), walk.MsgBoxIconInformation)
							} else {
								walk.MsgBox(mw, "Repair", fmt.Sprintf("%d chunks, %d damaged, %d repaired", total, damaged, repaired), walk.MsgBoxIconInformation)
							}
						},
					},
					PushButton{
						Text: "Info",
						OnClicked: func() {
							if model.inArchive {
								showInfoDialog(mw, model.archivePath)
								return
							}
							dlg := new(walk.FileDialog)
							dlg.Title = "Select .nya archive"
							dlg.Filter = "Nyarc Archives (*.nya)|*.nya"
							if ok, _ := dlg.ShowOpen(mw); ok {
								showInfoDialog(mw, dlg.FilePath)
							}
						},
					},
				},
			},
			// ─── Address bar ───
			Composite{
				Layout: HBox{Margins: Margins{Left: 4, Top: 0, Right: 4, Bottom: 4}},
				Children: []Widget{
					PushButton{
						Text:    "..",
						MaxSize: Size{Width: 30},
						OnClicked: func() {
							goUp()
						},
					},
					LineEdit{
						AssignTo: &addressBar,
						Text:     currentDir,
						OnKeyDown: func(key walk.Key) {
							if key == walk.KeyReturn {
								dir := addressBar.Text()
								// Check if it's an archive
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
				AssignTo:         &table,
				AlternatingRowBG: true,
				ColumnsOrderable: true,
				Columns: []TableViewColumn{
					{Title: "Name", Width: 300},
					{Title: "Size", Width: 100, Alignment: AlignFar},
					{Title: "Modified", Width: 150},
				},
				Model: model,
				OnItemActivated: func() {
					idx := table.CurrentIndex()
					if idx < 0 || idx >= len(model.items) {
						return
					}
					item := model.items[idx]
					if item.IsDir {
						navigate(item.Path)
					} else if strings.HasSuffix(strings.ToLower(item.Name), ".nya") {
						navigateArchive(item.Path)
					} else if !model.inArchive {
						// Open file with system default application
						exec.Command("cmd", "/c", "start", "", item.Path).Start()
					}
				},
				OnCurrentIndexChanged: func() {
					// Force refresh
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

	updateStatus := func() {
		count := len(model.items)
		var totalSize int64
		for _, f := range model.items {
			if !f.IsDir {
				totalSize += f.Size
			}
		}
		label := fmt.Sprintf("%d items, %s", count, humanSize(totalSize))
		if model.inArchive {
			label = fmt.Sprintf("[%s] %s", filepath.Base(model.archivePath), label)
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
		if info, err := os.Stat(f); err == nil {
			totalSize += info.Size()
		}
	}

	Dialog{
		AssignTo: &dlg,
		Title:    fmt.Sprintf("Add to archive (%d files, %s)", len(files), humanSize(totalSize)),
		MinSize:  Size{Width: 400, Height: 300},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: "Compression level (1-19):"},
					NumberEdit{AssignTo: &levelEdit, Value: 9, MinValue: 1, MaxValue: 19},
					Label{Text: "FEC recovery (%):"},
					NumberEdit{AssignTo: &fecEdit, Value: 100, MinValue: 0, MaxValue: 500},
					Label{Text: "Password:"},
					LineEdit{AssignTo: &passwordEdit, PasswordMode: true},
					Label{Text: ""},
					CheckBox{AssignTo: &solidCheck, Text: "Solid archive"},
					Label{Text: ""},
					CheckBox{AssignTo: &sfxCheck, Text: "Create SFX"},
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
							fdlg.Title = "Save to folder"
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
									walk.MsgBox(owner, "Pack Error", err.Error(), walk.MsgBoxIconError)
								} else {
									walk.MsgBox(owner, "Pack", "Archive created successfully", walk.MsgBoxIconInformation)
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
			walk.MsgBox(owner, "Extract Error", err.Error(), walk.MsgBoxIconError)
		} else {
			walk.MsgBox(owner, "Extract", "Extracted successfully to:\n"+fdlg.FilePath, walk.MsgBoxIconInformation)
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
		walk.MsgBox(owner, "Info", fmt.Sprintf(
			"File: %s\nSize: %s\nNot a valid .nya archive: %s",
			path, humanSize(fi.Size()), err.Error(),
		), walk.MsgBoxIconInformation)
		return
	}

	files := r.List()
	var totalOrig uint64
	details := ""
	for _, f := range files {
		totalOrig += f.OriginalSize
		details += fmt.Sprintf("  %s (%s)\n", f.Path, humanSize(int64(f.OriginalSize)))
	}

	ratio := float64(0)
	if totalOrig > 0 {
		ratio = float64(fi.Size()) / float64(totalOrig) * 100
	}

	msg := fmt.Sprintf(
		"Archive: %s\nSize: %s\nFiles: %d\nOriginal size: %s\nRatio: %.1f%%\n\nContents:\n%s",
		filepath.Base(path),
		humanSize(fi.Size()),
		len(files),
		humanSize(int64(totalOrig)),
		ratio,
		details,
	)
	walk.MsgBox(owner, "Archive Info", msg, walk.MsgBoxIconInformation)
}

// ─── Sortable File model for TableView ───

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
	return m
}

func (m *FileModel) SetDir(dir string) {
	m.items = listDir(dir)
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
	m.items = nil
	for _, f := range files {
		m.items = append(m.items, FileEntry{
			Name:    f.Path,
			Path:    f.Path,
			Size:    int64(f.OriginalSize),
			IsDir:   false,
			ModTime: "",
		})
	}
	m.inArchive = true
	m.archivePath = path
	m.doSort()
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
		if item.IsDir {
			return "[" + item.Name + "]"
		}
		return item.Name
	case 1:
		if item.IsDir {
			return ""
		}
		return humanSize(item.Size)
	case 2:
		return item.ModTime
	}
	return ""
}

// Sort implements walk.Sorter
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
		// Directories always first
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
