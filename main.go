package main

import (
	"fmt"
	"os"
	"path/filepath"
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
						Text: "📦 Add",
						OnClicked: func() {
							dlg := new(walk.FileDialog)
							dlg.Title = "Select files to compress"
							dlg.FilePath = currentDir
							if ok, _ := dlg.ShowOpenMultiple(mw); ok {
								showPackDialog(mw, dlg.FilePaths)
							}
						},
					},
					PushButton{
						Text: "📂 Extract",
						OnClicked: func() {
							dlg := new(walk.FileDialog)
							dlg.Title = "Select archive"
							dlg.Filter = "Nyarc Archives (*.nya)|*.nya|All Files (*.*)|*.*"
							if ok, _ := dlg.ShowOpen(mw); ok {
								showExtractDialog(mw, dlg.FilePath)
							}
						},
					},
					PushButton{
						Text: "🔍 Test",
						OnClicked: func() {
							dlg := new(walk.FileDialog)
							dlg.Title = "Select .nya archive to test"
							dlg.Filter = "Nyarc Archives (*.nya)|*.nya"
							if ok, _ := dlg.ShowOpen(mw); ok {
								count, ok, err := doTest(dlg.FilePath)
								if err != nil {
									walk.MsgBox(mw, "Error", err.Error(), walk.MsgBoxIconError)
								} else if ok {
									walk.MsgBox(mw, "Test", fmt.Sprintf("OK: %d files, integrity verified", count), walk.MsgBoxIconInformation)
								} else {
									walk.MsgBox(mw, "Test", fmt.Sprintf("FAILED: %d files, archive corrupted", count), walk.MsgBoxIconWarning)
								}
							}
						},
					},
					PushButton{
						Text: "🔧 Repair",
						OnClicked: func() {
							dlg := new(walk.FileDialog)
							dlg.Title = "Select .nya archive to repair"
							dlg.Filter = "Nyarc Archives (*.nya)|*.nya"
							if ok, _ := dlg.ShowOpen(mw); ok {
								total, damaged, repaired, err := doRepair(dlg.FilePath)
								if err != nil {
									walk.MsgBox(mw, "Repair", err.Error(), walk.MsgBoxIconError)
								} else if damaged == 0 {
									walk.MsgBox(mw, "Repair", fmt.Sprintf("No damage found (%d chunks verified)", total), walk.MsgBoxIconInformation)
								} else {
									walk.MsgBox(mw, "Repair", fmt.Sprintf("%d chunks, %d damaged, %d repaired", total, damaged, repaired), walk.MsgBoxIconInformation)
								}
							}
						},
					},
					PushButton{
						Text: "ℹ️ Info",
						OnClicked: func() {
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
					LineEdit{
						AssignTo: &addressBar,
						Text:     currentDir,
						OnKeyDown: func(key walk.Key) {
							if key == walk.KeyReturn {
								dir := addressBar.Text()
								if info, err := os.Stat(dir); err == nil && info.IsDir() {
									currentDir = dir
									model.SetDir(currentDir)
									table.SetCurrentIndex(0)
								}
							}
						},
					},
					PushButton{
						Text:    "⬆️",
						MaxSize: Size{Width: 40},
						OnClicked: func() {
							parent := filepath.Dir(currentDir)
							if parent != currentDir {
								currentDir = parent
								addressBar.SetText(currentDir)
								model.SetDir(currentDir)
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
						currentDir = item.Path
						addressBar.SetText(currentDir)
						model.SetDir(currentDir)
					} else if strings.HasSuffix(strings.ToLower(item.Name), ".nya") {
						showInfoDialog(mw, item.Path)
					}
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
		statusBar.SetText(fmt.Sprintf("%d items, %s", count, humanSize(totalSize)))
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
							// Choose output directory
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

	// Try to open as .nya
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

	msg := fmt.Sprintf(
		"Archive: %s\nSize: %s\nFiles: %d\nOriginal size: %s\nRatio: %.1f%%\n\nContents:\n%s",
		filepath.Base(path),
		humanSize(fi.Size()),
		len(files),
		humanSize(int64(totalOrig)),
		float64(fi.Size())/float64(totalOrig)*100,
		details,
	)
	walk.MsgBox(owner, "Archive Info", msg, walk.MsgBoxIconInformation)
}

// ─── File model for TableView ───

type FileModel struct {
	walk.TableModelBase
	items    []FileEntry
	onUpdate func()
}

func NewFileModel(dir string) *FileModel {
	m := &FileModel{}
	m.items = listDir(dir)
	return m
}

func (m *FileModel) SetDir(dir string) {
	m.items = listDir(dir)
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
			return "📁 " + item.Name
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
