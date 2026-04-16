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

func NewApp() *App                            { return &App{} }
func (a *App) startup(ctx context.Context)    { a.ctx = ctx }
func (a *App) GetStartupFile() string         { return a.startupFile }
func (a *App) GetStartupAction() string       { return a.startupAction }
func (a *App) Version() string                { return "NekoArc v0.2.0 (Nyarc v0.6.2)" }
func (a *App) Greet(name string) string       { return fmt.Sprintf("Hello %s!", name) }

// === Types ===

type PackOptions struct {
	Inputs   []string `json:"inputs"`
	Output   string   `json:"output"`
	Format   string   `json:"format"`
	Level    int      `json:"level"`
	FEC      int      `json:"fec"`
	Password string   `json:"password"`
	Solid    bool     `json:"solid"`
	SFX      bool     `json:"sfx"`
}

type Result struct {
	Success  bool    `json:"success"`
	Message  string  `json:"message"`
	Duration float64 `json:"duration"`
}

// === Pack (supports multiple files) ===

func (a *App) Pack(opts PackOptions) Result {
	start := time.Now()
	if len(opts.Inputs) == 0 {
		return Result{Success: false, Message: "No files selected"}
	}

	// Output name from first input
	firstName := filepath.Base(opts.Inputs[0])
	ext := ".nya"
	if opts.Format != "" && opts.Format != "nya" {
		ext = "." + opts.Format
	}
	outName := strings.TrimSuffix(firstName, filepath.Ext(firstName)) + ext

	// Output directory: user choice or same as first input
	outDir := opts.Output
	if outDir == "" {
		outDir = filepath.Dir(opts.Inputs[0])
	}
	output := filepath.Join(outDir, outName)

	level := opts.Level
	if level == 0 {
		level = 9
	}
	fec := opts.FEC
	if fec == 0 {
		fec = 100
	}

	f, err := os.Create(output)
	if err != nil {
		return fail(err, start)
	}

	var w *nya.Writer
	if opts.Password != "" {
		w = nya.NewWriterOpts(f, fec, level, opts.Solid, []byte(opts.Password))
	} else if opts.Solid {
		w = nya.NewWriterOpts(f, fec, level, true)
	} else {
		w = nya.NewWriter(f, fec, level)
	}

	// Add ALL files
	for _, input := range opts.Inputs {
		if err := w.AddFile(input); err != nil {
			f.Close()
			return fail(err, start)
		}
	}
	w.Close()
	f.Close()

	info, _ := os.Stat(output)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}

	if opts.SFX {
		nya.CreateSFX(output, "")
	}

	return Result{
		Success:  true,
		Message:  fmt.Sprintf("OK: %d files -> %s (%s)", len(opts.Inputs), output, nya.HumanSize(int(size))),
		Duration: time.Since(start).Seconds(),
	}
}

// === Extract (outputs to archive directory or chosen dir) ===

func (a *App) Extract(fp string, destDir string) Result {
	start := time.Now()
	if fp == "" {
		return Result{Success: false, Message: "No file selected"}
	}

	// Default: extract to archive's directory
	dir := destDir
	if dir == "" {
		dir = filepath.Dir(fp)
	}

	r, err := nya.Open(fp)
	if err != nil {
		return fail(err, start)
	}
	if err := r.Extract(dir); err != nil {
		return fail(err, start)
	}
	return Result{
		Success:  true,
		Message:  fmt.Sprintf("OK: Extracted to %s", dir),
		Duration: time.Since(start).Seconds(),
	}
}

// === Repair ===

func (a *App) Repair(fp string) Result {
	start := time.Now()
	if fp == "" {
		return Result{Success: false, Message: "No file selected"}
	}
	result, err := nya.Repair(fp, "")
	if err != nil {
		return fail(err, start)
	}
	return Result{
		Success: true,
		Message: fmt.Sprintf("OK: %d chunks, %d damaged, %d recovered",
			result.TotalChunks, result.CorruptedChunks, result.RepairedChunks),
		Duration: time.Since(start).Seconds(),
	}
}

// === Test ===

func (a *App) Test(fp string) Result {
	start := time.Now()
	r, err := nya.Open(fp)
	if err != nil {
		return fail(err, start)
	}
	if r.Verify() {
		return Result{Success: true, Message: "OK: Archive integrity verified", Duration: time.Since(start).Seconds()}
	}
	return Result{Success: false, Message: "ERR: Archive corrupted", Duration: time.Since(start).Seconds()}
}

// === Estimate ===

type Estimate struct {
	InputSize    int64  `json:"inputSize"`
	OutputSize   int64  `json:"outputSize"`
	FECSize      int64  `json:"fecSize"`
	RecoveryRate string `json:"recoveryRate"`
}

func (a *App) EstimateSize(paths []string, fecPercent int) Estimate {
	var total int64
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.IsDir() {
			filepath.Walk(p, func(_ string, fi os.FileInfo, _ error) error {
				if fi != nil && !fi.IsDir() {
					total += fi.Size()
				}
				return nil
			})
		} else {
			total += info.Size()
		}
	}

	if fecPercent == 0 {
		fecPercent = 100
	}
	// Estimate: compressed ~= original (random data worst case)
	// FEC adds fecPercent% overhead
	fecSize := total * int64(fecPercent) / 100
	outputSize := total + fecSize

	K := 32
	repairCount := K * fecPercent / 100
	if repairCount < 1 {
		repairCount = 1
	}
	totalSymbols := K + repairCount
	maxLoss := float64(repairCount) / float64(totalSymbols) * 100

	return Estimate{
		InputSize:    total,
		OutputSize:   outputSize,
		FECSize:      fecSize,
		RecoveryRate: fmt.Sprintf("%.0f%%", maxLoss),
	}
}

// === File Dialogs ===

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

func (a *App) OpenNyaFileDialog() string {
	p, _ := rt.OpenFileDialog(a.ctx, rt.OpenDialogOptions{
		Title:   "Select .nya archive",
		Filters: []rt.FileFilter{{DisplayName: "Nyarc Archives", Pattern: "*.nya"}},
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

func (a *App) GetFileInfo(path string) map[string]interface{} {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	return map[string]interface{}{
		"name": info.Name(), "size": info.Size(), "isDir": info.IsDir(),
		"ext": filepath.Ext(path), "path": path,
	}
}

func fail(err error, start time.Time) Result {
	return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
}
