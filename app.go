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
	if opts.Output != "" { output = filepath.Join(opts.Output, filepath.Base(output)) }

	if opts.Format != "" && opts.Format != "nya" {
		output = strings.TrimSuffix(base, filepath.Ext(base)) + "." + opts.Format
	}

	// 直接调用nya包
	if opts.Format == "" || opts.Format == "nya" {
		level := opts.Level
		if level == 0 { level = 9 }
		fec := opts.FEC
		if fec == 0 { fec = 10 }

		w, err := nya.NewWriter(output, level, fec, opts.Solid)
		if err != nil {
			return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
		}
		if opts.Password != "" {
			w.SetPassword([]byte(opts.Password))
		}

		err = w.AddPath(opts.Input)
		if err != nil {
			return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
		}
		err = w.Close()
		if err != nil {
			return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
		}

		info, _ := os.Stat(output)
		size := int64(0)
		if info != nil { size = info.Size() }

		if opts.SFX {
			nya.CreateSFX(output, "")
		}

		return Result{
			Success: true,
			Message: fmt.Sprintf("✅ %s → %s (%s)", opts.Input, output, nya.HumanSize(int(size))),
			Duration: time.Since(start).Seconds(),
		}
	}

	// 非nya格式: 用archiver
	// 简单起见先调CLI
	return Result{Success: false, Message: "Non-nya format: use CLI for now", Duration: time.Since(start).Seconds()}
}

func (a *App) Extract(filePath string, destDir string) Result {
	start := time.Now()
	if filePath == "" {
		return Result{Success: false, Message: "No file selected"}
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if destDir != "" {
		os.Chdir(destDir)
	}

	if ext == ".nya" {
		r, err := nya.Open(filePath)
		if err != nil {
			return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
		}
		dir := "."
		if destDir != "" { dir = destDir }
		err = r.Extract(dir)
		if err != nil {
			return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
		}
		return Result{
			Success: true,
			Message: fmt.Sprintf("✅ Extracted to %s", dir),
			Duration: time.Since(start).Seconds(),
		}
	}

	// 通解: 用archiver
	return Result{Success: false, Message: "Non-nya extract: coming soon", Duration: time.Since(start).Seconds()}
}

func (a *App) Repair(filePath string) Result {
	start := time.Now()
	if filePath == "" {
		return Result{Success: false, Message: "No file selected"}
	}

	result, err := nya.Repair(filePath, "")
	if err != nil {
		return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
	}

	return Result{
		Success: true,
		Message: fmt.Sprintf("✅ Repaired! %d chunks, %d damaged, %d recovered",
			result.TotalChunks, result.CorruptedChunks, result.RepairedChunks),
		Duration: time.Since(start).Seconds(),
	}
}

func (a *App) Test(filePath string) Result {
	start := time.Now()
	r, err := nya.Open(filePath)
	if err != nil {
		return Result{Success: false, Message: err.Error(), Duration: time.Since(start).Seconds()}
	}
	ok := r.Verify()
	msg := "✅ Archive OK"
	if !ok { msg = "❌ Archive corrupted" }
	return Result{Success: ok, Message: msg, Duration: time.Since(start).Seconds()}
}

func (a *App) OpenFileDialog() string {
	path, err := rt.OpenFileDialog(a.ctx, rt.OpenDialogOptions{
		Title: "Select file",
		Filters: []rt.FileFilter{
			{DisplayName: "All Files", Pattern: "*"},
			{DisplayName: "Archives", Pattern: "*.nya;*.zip;*.rar;*.7z;*.tar;*.gz;*.bz2;*.xz"},
		},
	})
	if err != nil { return "" }
	return path
}

func (a *App) OpenMultipleFilesDialog() []string {
	paths, err := rt.OpenMultipleFilesDialog(a.ctx, rt.OpenDialogOptions{
		Title: "Select files",
		Filters: []rt.FileFilter{
			{DisplayName: "All Files", Pattern: "*"},
		},
	})
	if err != nil { return nil }
	return paths
}

func (a *App) OpenDirectoryDialog() string {
	path, err := rt.OpenDirectoryDialog(a.ctx, rt.OpenDialogOptions{
		Title: "Select folder",
	})
	if err != nil { return "" }
	return path
}

func (a *App) GetFileInfo(path string) map[string]interface{} {
	info, err := os.Stat(path)
	if err != nil { return nil }
	return map[string]interface{}{
		"name":  info.Name(),
		"size":  info.Size(),
		"isDir": info.IsDir(),
		"ext":   filepath.Ext(path),
		"path":  path,
	}
}

func (a *App) GetStartupFile() string   { return a.startupFile }
func (a *App) GetStartupAction() string { return a.startupAction }
func (a *App) Version() string          { return "NekoArc v0.1.0 (Nyarc v0.6.2)" }
