package main

import (
	"fmt"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type LogLevel int

const (
	LogInfo LogLevel = iota
	LogWarning
	LogError
)

type LogEntry struct {
	Level   LogLevel
	Message string
	File    string
}

type DiagLog struct {
	entries []LogEntry
}

func NewDiagLog() *DiagLog {
	return &DiagLog{}
}

func (d *DiagLog) Info(msg, file string) {
	d.entries = append(d.entries, LogEntry{LogInfo, msg, file})
}

func (d *DiagLog) Warn(msg, file string) {
	d.entries = append(d.entries, LogEntry{LogWarning, msg, file})
}

func (d *DiagLog) Error(msg, file string) {
	d.entries = append(d.entries, LogEntry{LogError, msg, file})
}

func (d *DiagLog) ErrorCount() int {
	count := 0
	for _, e := range d.entries {
		if e.Level == LogError {
			count++
		}
	}
	return count
}

func (d *DiagLog) WarnCount() int {
	count := 0
	for _, e := range d.entries {
		if e.Level == LogWarning {
			count++
		}
	}
	return count
}

func (d *DiagLog) HasIssues() bool {
	for _, e := range d.entries {
		if e.Level != LogInfo {
			return true
		}
	}
	return false
}

// Show diagnostic dialog (like WinRAR's diagnostic info)
func (d *DiagLog) Show(owner walk.Form, title string) {
	if len(d.entries) == 0 {
		return
	}

	var dlg *walk.Dialog
	var tv *walk.TableView
	model := &LogModel{entries: d.entries}

	Dialog{
		AssignTo: &dlg,
		Title:    "NekoArc: Diagnostic Information",
		MinSize:  Size{Width: 650, Height: 400},
		Size:     Size{Width: 700, Height: 450},
		Layout:   VBox{},
		Children: []Widget{
			TableView{
				AssignTo:         &tv,
				AlternatingRowBG: true,
				Columns: []TableViewColumn{
					{Title: "Info", Width: 350},
					{Title: "File", Width: 280},
				},
				Model: model,
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{Text: fmt.Sprintf("Errors: %d  Warnings: %d", d.ErrorCount(), d.WarnCount())},
					HSpacer{},
					PushButton{
						Text: "Copy to Clipboard",
						OnClicked: func() {
							var sb strings.Builder
							for _, e := range d.entries {
								prefix := "!"
								if e.Level == LogWarning {
									prefix = "WARNING:"
								} else if e.Level == LogError {
									prefix = "ERROR:"
								}
								sb.WriteString(fmt.Sprintf("%s %s\t%s\n", prefix, e.Message, e.File))
							}
							walk.Clipboard().SetText(sb.String())
						walk.MsgBox(dlg, "Copied", "Diagnostic info copied to clipboard", walk.MsgBoxIconInformation)
						},
					},
					PushButton{
						Text:      "Close",
						OnClicked: func() { dlg.Accept() },
					},
				},
			},
		},
	}.Run(owner)
}

// LogModel for TableView
type LogModel struct {
	walk.TableModelBase
	entries []LogEntry
}

func (m *LogModel) RowCount() int { return len(m.entries) }

func (m *LogModel) Value(row, col int) interface{} {
	if row < 0 || row >= len(m.entries) {
		return ""
	}
	e := m.entries[row]
	switch col {
	case 0:
		prefix := ""
		switch e.Level {
		case LogWarning:
			prefix = "WARNING: "
		case LogError:
			prefix = "ERROR: "
		}
		return prefix + e.Message
	case 1:
		return e.File
	}
	return ""
}
