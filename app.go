package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/nyarime/nyarc/pkg/nya"
)

// ─── File browser helpers ───

type FileEntry struct {
	Name    string
	Path    string
	Size    int64
	IsDir   bool
	ModTime string
}

func listDir(dir string) []FileEntry {
	if dir == "" {
		return listDrives()
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var result []FileEntry
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, FileEntry{
			Name:    e.Name(),
			Path:    filepath.Join(dir, e.Name()),
			Size:    info.Size(),
			IsDir:   e.IsDir(),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result
}

func listDrives() []FileEntry {
	if runtime.GOOS != "windows" {
		return []FileEntry{{Name: "/", Path: "/", IsDir: true}}
	}
	var drives []FileEntry
	for c := 'A'; c <= 'Z'; c++ {
		p := string(c) + ":\\"
		if _, err := os.Stat(p); err == nil {
			drives = append(drives, FileEntry{Name: p, Path: p, IsDir: true})
		}
	}
	return drives
}

// ─── Archive operations ───

type PackOptions struct {
	Inputs   []string
	Output   string
	Level    int
	FEC      int
	Password string
	Solid    bool
	SFX      bool
}

func doPack(opts PackOptions) error {
	if len(opts.Inputs) == 0 {
		return fmt.Errorf("no files selected")
	}

	firstName := filepath.Base(opts.Inputs[0])
	outName := strings.TrimSuffix(firstName, filepath.Ext(firstName)) + ".nya"

	outDir := opts.Output
	if outDir == "" {
		outDir = filepath.Dir(opts.Inputs[0])
	}
	output := filepath.Join(outDir, outName)

	level := opts.Level
	if level == 0 { level = 9 }
	fec := opts.FEC
	if fec == 0 { fec = 100 }

	f, err := os.Create(output)
	if err != nil {
		return err
	}

	var w *nya.Writer
	if opts.Password != "" {
		w = nya.NewWriterOpts(f, fec, level, opts.Solid, []byte(opts.Password))
	} else if opts.Solid {
		w = nya.NewWriterOpts(f, fec, level, true)
	} else {
		w = nya.NewWriter(f, fec, level)
	}

	for _, input := range opts.Inputs {
		if err := w.AddFile(input); err != nil {
			f.Close()
			return err
		}
	}
	w.Close()
	f.Close()

	if opts.SFX {
		nya.CreateSFX(output, "")
	}
	return nil
}

func doExtract(archivePath, destDir string) error {
	if destDir == "" {
		destDir = filepath.Dir(archivePath)
	}
	r, err := nya.Open(archivePath)
	if err != nil {
		return err
	}
	return r.Extract(destDir)
}

func doTest(path string) (int, bool, error) {
	r, err := nya.Open(path)
	if err != nil {
		return 0, false, err
	}
	count := len(r.List())
	return count, r.Verify(), nil
}

func doRepair(path string) (total, damaged, repaired int, err error) {
	result, err := nya.Repair(path, "")
	if err != nil {
		return 0, 0, 0, err
	}
	return result.TotalChunks, result.CorruptedChunks, result.RepairedChunks, nil
}

func humanSize(b int64) string {
	return nya.HumanSize(int(b))
}
