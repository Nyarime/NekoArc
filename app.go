package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nyarime/nyarc/pkg/nya"
	rt "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	startupFile   string
	startupAction string
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet — test binding
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, Welcome to NekoArc!", name)
}

// === Pack ===

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
	ext := ".nya"
	if opts.Format != "" && opts.Format != "nya" {
		ext = "." + opts.Format
	}
	output := strings.TrimSuffix(base, filepath.Ext(base)) + ext

	// 使用CLI调用(最简单的集成方式)
	args := []string{"-a", opts.Input}
	if opts.Format != "" && opts.Format != "nya" {
		args = append(args, "--format", opts.Format)
	}
	if opts.Level > 0 {
		args = append(args, "--level", fmt.Sprint(opts.Level))
	}
	if opts.FEC > 0 && opts.Format == "nya" {
		args = append(args, "--fec", fmt.Sprint(opts.FEC))
	}
	if opts.Password != "" {
		args = append(args, "--password", opts.Password)
	}
	if opts.Solid {
		args = append(args, "--solid")
	}
	if opts.SFX {
		args = append(args, "--sfx")
	}

	cmd := exec.Command("nyarc", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{
			Success:  false,
			Message:  fmt.Sprintf("Pack failed: %v\n%s", err, string(out)),
			Duration: time.Since(start).Seconds(),
		}
	}

	return Result{
		Success:  true,
		Message:  fmt.Sprintf("Created %s\n%s", output, string(out)),
		Duration: time.Since(start).Seconds(),
	}
}

// === Extract ===

func (a *App) Extract(filePath string) Result {
	start := time.Now()

	ext := strings.ToLower(filepath.Ext(filePath))
	var args []string
	if ext == ".nya" {
		args = []string{"-u", filePath}
	} else {
		args = []string{"-x", filePath}
	}

	cmd := exec.Command("nyarc", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{
			Success:  false,
			Message:  fmt.Sprintf("Extract failed: %v\n%s", err, string(out)),
			Duration: time.Since(start).Seconds(),
		}
	}

	return Result{
		Success:  true,
		Message:  fmt.Sprintf("Extracted!\n%s", string(out)),
		Duration: time.Since(start).Seconds(),
	}
}

// === Repair ===

func (a *App) Repair(filePath string) Result {
	start := time.Now()

	cmd := exec.Command("nyarc", "-r", filePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{
			Success:  false,
			Message:  fmt.Sprintf("Repair failed: %v\n%s", err, string(out)),
			Duration: time.Since(start).Seconds(),
		}
	}

	return Result{
		Success:  true,
		Message:  fmt.Sprintf("Repaired!\n%s", string(out)),
		Duration: time.Since(start).Seconds(),
	}
}

// === Test ===

func (a *App) Test(filePath string) Result {
	start := time.Now()

	cmd := exec.Command("nyarc", "-t", filePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{
			Success:  false,
			Message:  fmt.Sprintf("Test failed: %v\n%s", err, string(out)),
			Duration: time.Since(start).Seconds(),
		}
	}

	return Result{
		Success:  true,
		Message:  string(out),
		Duration: time.Since(start).Seconds(),
	}
}

// === File Dialog ===

func (a *App) OpenFileDialog() string {
	path, err := rt.OpenFileDialog(a.ctx, rt.OpenDialogOptions{
		Title: "Select file or archive",
		Filters: []rt.FileFilter{
			{DisplayName: "Archives", Pattern: "*.nya;*.zip;*.rar;*.7z;*.tar;*.gz;*.bz2;*.xz"},
			{DisplayName: "All Files", Pattern: "*"},
		},
	})
	if err != nil {
		return ""
	}
	return path
}

func (a *App) OpenDirectoryDialog() string {
	path, err := rt.OpenDirectoryDialog(a.ctx, rt.OpenDialogOptions{
		Title: "Select folder to pack",
	})
	if err != nil {
		return ""
	}
	return path
}

func (a *App) GetFileInfo(path string) map[string]interface{} {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	return map[string]interface{}{
		"name":  info.Name(),
		"size":  info.Size(),
		"isDir": info.IsDir(),
		"ext":   filepath.Ext(path),
		"path":  path,
	}
}

func (a *App) Version() string {
	return "NekoArc v0.1.0 (Nyarc Engine v0.6.0)"
}

func init() { _ = nya.HumanSize }

func (a *App) GetStartupFile() string {
	return a.startupFile
}

func (a *App) GetStartupAction() string {
	return a.startupAction
}

// PackWithProgress 带进度的打包
func (a *App) PackWithProgress(opts PackOptions) Result {
	start := time.Now()
	if opts.Input == "" {
		return Result{Success: false, Message: "No input selected"}
	}

	// 发送进度事件
	rt.EventsEmit(a.ctx, "progress", map[string]interface{}{
		"stage": "packing",
		"percent": 0,
	})

	r := a.Pack(opts)

	rt.EventsEmit(a.ctx, "progress", map[string]interface{}{
		"stage": "done",
		"percent": 100,
	})

	r.Duration = time.Since(start).Seconds()
	return r
}
