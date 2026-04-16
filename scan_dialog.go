package main

import (
	"fmt"
	"os"

	"github.com/lxn/walk"
	"github.com/nyarime/nyarc/pkg/nya"
	. "github.com/lxn/walk/declarative"
)

func showScanResults(owner walk.Form, path string) {
	entries, err := nya.ScanBinary(path)
	if err != nil {
		walk.MsgBox(owner, "Scan Error", err.Error(), walk.MsgBoxIconError)
		return
	}

	if len(entries) == 0 {
		walk.MsgBox(owner, "Scan", "No known signatures found", walk.MsgBoxIconInformation)
		return
	}

	var dlg *walk.Dialog
	model := &ScanModel{entries: entries}

	Dialog{
		AssignTo: &dlg,
		Title:    fmt.Sprintf("Binary Scan: %d signatures found", len(entries)),
		MinSize:  Size{Width: 650, Height: 400},
		Layout:   VBox{},
		Children: []Widget{
			TableView{
				AlternatingRowBG: true,
				Columns: []TableViewColumn{
					{Title: "Offset", Width: 100},
					{Title: "Type", Width: 100},
					{Title: "Description", Width: 380},
				},
				Model: model,
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{Text: fmt.Sprintf("%d signatures in %s", len(entries), humanSize(fileSize(path)))},
					HSpacer{},
					PushButton{Text: "Close", OnClicked: func() { dlg.Accept() }},
				},
			},
		},
	}.Run(owner)
}

type ScanModel struct {
	walk.TableModelBase
	entries []nya.BinEntry
}

func (m *ScanModel) RowCount() int { return len(m.entries) }

func (m *ScanModel) Value(row, col int) interface{} {
	if row < 0 || row >= len(m.entries) {
		return ""
	}
	e := m.entries[row]
	switch col {
	case 0:
		return fmt.Sprintf("0x%X", e.Offset)
	case 1:
		return e.Type
	case 2:
		return e.Description
	}
	return ""
}

func fileSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return fi.Size()
}
