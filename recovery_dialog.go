package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func showRecoveryDialog(owner walk.Form, getSelectedPaths func() []string, model *FileModel) {
	// Determine target archive
	archivePath := ""
	if model.inArchive {
		archivePath = model.archivePath
	} else {
		paths := getSelectedPaths()
		if len(paths) == 1 && strings.HasSuffix(strings.ToLower(paths[0]), ".nya") {
			archivePath = paths[0]
		}
	}

	if archivePath == "" || !strings.HasSuffix(strings.ToLower(archivePath), ".nya") {
		walk.MsgBox(owner, "Recovery Record", "Select a .nya archive first.\nRecovery records can only be added to .nya files.", walk.MsgBoxIconInformation)
		return
	}

	// Get file size
	fi, err := os.Stat(archivePath)
	if err != nil {
		walk.MsgBox(owner, "Error", err.Error(), walk.MsgBoxIconError)
		return
	}
	currentSize := fi.Size()

	var dlg *walk.Dialog
	var fecSlider *walk.Slider
	var fecLabel *walk.Label
	var sizeLabel *walk.Label

	updateLabels := func(pct int) {
		extra := currentSize * int64(pct) / 100
		newSize := currentSize + extra
		fecLabel.SetText(fmt.Sprintf("Recovery: %d%%", pct))
		sizeLabel.SetText(fmt.Sprintf(
			"Current size: %s\nRecovery data: +%s (%d%%)\nNew size: %s\n\n%s",
			humanSize(currentSize),
			humanSize(extra),
			pct,
			humanSize(newSize),
			recoveryDescription(pct),
		))
	}

	Dialog{
		AssignTo: &dlg,
		Title:    "Add Recovery Record — " + filepath.Base(archivePath),
		MinSize:  Size{Width: 450, Height: 300},
		Layout:   VBox{},
		Children: []Widget{
			GroupBox{
				Title:  "Recovery Record (RaptorQ FEC)",
				Layout: VBox{},
				Children: []Widget{
					Label{AssignTo: &fecLabel, Text: "Recovery: 3%"},
					Slider{
						AssignTo:    &fecSlider,
						MinValue:    0,
						MaxValue:    100,
						Value:       3,
						OnValueChanged: func() {
							updateLabels(fecSlider.Value())
						},
					},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							Label{Text: "0%"},
							HSpacer{},
							Label{Text: "3%"},
							HSpacer{},
							Label{Text: "10%"},
							HSpacer{},
							Label{Text: "50%"},
							HSpacer{},
							Label{Text: "100%"},
						},
					},
				},
			},
			Label{
				AssignTo: &sizeLabel,
				Text: fmt.Sprintf(
					"Current size: %s\nRecovery data: +%s (3%%)\nNew size: %s\n\n%s",
					humanSize(currentSize),
					humanSize(currentSize*3/100),
					humanSize(currentSize+currentSize*3/100),
					recoveryDescription(3),
				),
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "Add Recovery Record",
						OnClicked: func() {
							pct := fecSlider.Value()
							if pct == 0 {
								walk.MsgBox(owner, "Info", "Recovery percentage is 0%. No record will be added.", walk.MsgBoxIconInformation)
								dlg.Cancel()
								return
							}
							// TODO: Actually add recovery record to existing archive
							// For now: inform user to recompress with FEC
							walk.MsgBox(owner, "Info",
								fmt.Sprintf("To add %d%% recovery record, recompress the files with:\n\nnyarc -a files/ --fec %d\n\nIn-place FEC addition coming soon.", pct, pct),
								walk.MsgBoxIconInformation)
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

func recoveryDescription(pct int) string {
	if pct == 0 {
		return "No recovery — archive cannot be repaired if damaged"
	}
	if pct <= 3 {
		return "Light recovery — handles minor corruption\n(network errors, disk sector failures)"
	}
	if pct <= 10 {
		return "Standard recovery — handles moderate corruption\n(recommended for cloud storage/network transfers)"
	}
	if pct <= 50 {
		return "Strong recovery — handles significant corruption\n(recommended for long-term archival)"
	}
	return "Maximum recovery — can survive up to 50% data loss\n(recommended for critical data, cold storage)"
}
