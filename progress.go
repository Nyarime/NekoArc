package main

import (
	"fmt"
	"sync"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type ProgressDialog struct {
	dlg      *walk.Dialog
	bar      *walk.ProgressBar
	label    *walk.Label
	mu       sync.Mutex
	closed   bool
}

func ShowProgressDialog(owner walk.Form, title string) *ProgressDialog {
	pd := &ProgressDialog{}

	go func() {
		Dialog{
			AssignTo: &pd.dlg,
			Title:    title,
			MinSize:  Size{Width: 400, Height: 120},
			Layout:   VBox{},
			Children: []Widget{
				Label{AssignTo: &pd.label, Text: "Processing..."},
				ProgressBar{AssignTo: &pd.bar, MaxValue: 100},
			},
		}.Run(owner)
		pd.mu.Lock()
		pd.closed = true
		pd.mu.Unlock()
	}()

	return pd
}

func (pd *ProgressDialog) SetProgress(percent int, msg string) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	if pd.closed || pd.dlg == nil {
		return
	}
	if pd.bar != nil {
		pd.bar.SetValue(percent)
	}
	if pd.label != nil && msg != "" {
		pd.label.SetText(msg)
	}
}

func (pd *ProgressDialog) SetMessage(msg string) {
	pd.SetProgress(-1, msg)
}

func (pd *ProgressDialog) Close() {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	if !pd.closed && pd.dlg != nil {
		pd.dlg.Accept()
	}
}

// Simple progress wrapper for operations
func withProgress(owner walk.Form, title string, work func(update func(int, string))) {
	var pd *ProgressDialog
	done := make(chan struct{})

	// Run dialog in main thread
	pd = &ProgressDialog{}
	var bar *walk.ProgressBar
	var label *walk.Label

	go func() {
		// Do work
		work(func(pct int, msg string) {
			if bar != nil {
				bar.Synchronize(func() {
					bar.SetValue(pct)
				})
			}
			if label != nil && msg != "" {
				label.Synchronize(func() {
					label.SetText(msg)
				})
			}
		})
		close(done)
	}()

	Dialog{
		AssignTo: &pd.dlg,
		Title:    title,
		MinSize:  Size{Width: 450, Height: 100},
		MaxSize:  Size{Width: 450, Height: 100},
		Layout:   VBox{},
		Children: []Widget{
			Label{AssignTo: &label, Text: "Processing..."},
			ProgressBar{AssignTo: &bar, MaxValue: 100},
		},
	}.Run(owner)

	_ = fmt.Sprintf // keep import
	<-done
}
