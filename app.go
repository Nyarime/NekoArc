package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nyarime/nyarc/pkg/nya"
	rt "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx           context.Context
	startupFile   string
	startupAction string
}

func NewApp() *App { return &App{} }
func (a *App) startup(ctx context.Context) { a.ctx = ctx }
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, Welcome to NekoArc!", name)
}

type PackOptions struct {
	Input    string `json:"input"`
	Output   string `json:"output"`
	Format   string `json:"format"`
	Level    int    `json:"level"`
	FEC      int    `json:"fec"`
	Password string `json:"password"`
	Solid    bool   `json:"solid"`
	SFX      bool   `json:"sfx"`
}

type Result struct {
	Success  bool    `json:"success"`
	Message  string  `json:"message"`
	Duration float64 `json:"duration"`
}

func (a *App) Pack(opts PackOptions) Result {
	start := time.Now()
	if opts.Input == "" {
		return Result{Success: false, Message: "No input selected"}
	}

	base := filepath.Base(opts.Input)
	output := strings.TrimSuffix(base, filepath.Ext(base)) + ".nya"
	if opts.Output != "" {
		output = filepath.Join(opts.Output, filepath.Base(output))
	}

	level := opts.Level
	if level == 0 { level = 9 }
	fec := opts.FEC
	if fec == 0 { fec = 100 }

	f, err := os.Create(output)
	if err != nil {
		return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
	}

	var w *nya.Writer
	if opts.Password != "" {
		w = nya.NewWriterOpts(f, fec, level, opts.Solid, []byte(opts.Password))
	} else if opts.Solid {
		w = nya.NewWriterOpts(f, fec, level, true)
	} else {
		w = nya.NewWriter(f, fec, level)
	}

	err = w.AddFile(opts.Input)
	if err != nil {
		f.Close()
		return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
	}
	w.Close()
	f.Close()

	info, _ := os.Stat(output)
	size := int64(0)
	if info != nil { size = info.Size() }

	if opts.SFX {
		nya.CreateSFX(output, "")
	}

	return Result{
		Success:  true,
		Message:  fmt.Sprintf("OK: %s -> %s (%s)", opts.Input, output, nya.HumanSize(int(size))),
		Duration: time.Since(start).Seconds(),
	}
}

func (a *App) Extract(fp string) Result {
	start := time.Now()
	if fp == "" {
		return Result{Success: false, Message: "No file selected"}
	}

	r, err := nya.Open(fp)
	if err != nil {
		return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
	}
	dir := "."
	err = r.Extract(dir)
	if err != nil {
		return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
	}
	return Result{
		Success: true,
		Message: fmt.Sprintf("OK: Extracted to %s", dir),
		Duration: time.Since(start).Seconds(),
	}
}

func (a *App) Repair(fp string) Result {
	start := time.Now()
	if fp == "" {
		return Result{Success: false, Message: "No file selected"}
	}
	result, err := nya.Repair(fp, "")
	if err != nil {
		return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
	}
	return Result{
		Success: true,
		Message: fmt.Sprintf("OK: %d chunks, %d damaged, %d recovered",
			result.TotalChunks, result.CorruptedChunks, result.RepairedChunks),
		Duration: time.Since(start).Seconds(),
	}
}

func (a *App) Test(fp string) Result {
	start := time.Now()
	r, err := nya.Open(fp)
	if err != nil {
		return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
	}
	ok := r.Verify()
	if ok {
		return Result{Success: true, Message: "OK: Archive OK", Duration: time.Since(start).Seconds()}
	}
	return Result{Success: false, Message: "ERR: Archive corrupted", Duration: time.Since(start).Seconds()}
}

func (a *App) OpenFileDialog() string {
	p, _ := rt.OpenFileDialog(a.ctx, rt.OpenDialogOptions{
		Title:   "Select file",
		Filters: []rt.FileFilter{{DisplayName: "All Files", Pattern: "*"}},
	})
	return p
}

func (a *App) OpenMultipleFilesDialog() []string {
	p, _ := rt.OpenMultipleFilesDialog(a.ctx, rt.OpenDialogOptions{
		Title:   "Select files",
		Filters: []rt.FileFilter{{DisplayName: "All Files", Pattern: "*"}},
	})
	return p
}

func (a *App) OpenDirectoryDialog() string {
	p, _ := rt.OpenDirectoryDialog(a.ctx, rt.OpenDialogOptions{Title: "Select folder"})
	return p
}

func (a *App) GetFileInfo(path string) map[string]interface{} {
	info, err := os.Stat(path)
	if err != nil { return nil }
	return map[string]interface{}{
		"name": info.Name(), "size": info.Size(), "isDir": info.IsDir(),
		"ext": filepath.Ext(path), "path": path,
	}
}

func (a *App) GetStartupFile() string   { return a.startupFile }
func (a *App) GetStartupAction() string { return a.startupAction }
func (a *App) Version() string          { return "NekoArc v0.1.0 (Nyarc v0.6.2)" }

func (a *App) OpenNyaFileDialog() string {
	p, _ := rt.OpenFileDialog(a.ctx, rt.OpenDialogOptions{
		Title: "Select .nya archive",
		Filters: []rt.FileFilter{
			{DisplayName: "Nyarc Archives", Pattern: "*.nya"},
		},
	})
	return p
}

func (a *App) OpenArchiveDialog() string {
	p, _ := rt.OpenFileDialog(a.ctx, rt.OpenDialogOptions{
		Title: "Select archive",
		Filters: []rt.FileFilter{
			{DisplayName: "Archives", Pattern: "*.nya;*.zip;*.rar;*.7z;*.tar;*.gz;*.bz2;*.xz"},
			{DisplayName: "All Files", Pattern: "*"},
		},
	})
	return p
}
