package main

import (
	"fmt"
	"io"
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
	SFX       bool
	SplitSize string // e.g. "1G", "500M"
}

func doPack(opts PackOptions) (*DiagLog, error) {
	log := NewDiagLog()
	if len(opts.Inputs) == 0 {
		return log, fmt.Errorf("no files selected")
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
	if fec == 0 { fec = 3 }

	log.Info(fmt.Sprintf("Creating archive: %s", output), "")
	log.Info(fmt.Sprintf("Compression level: %d, FEC: %d%%", level, fec), "")

	var ws io.WriteSeeker

	splitSize := nya.ParseVolumeSize(opts.SplitSize)
	if splitSize > 0 {
		ws = nya.NewVolumeWriter(output, splitSize)
		log.Info(fmt.Sprintf("Split volume: %s per volume", opts.SplitSize), output)
	} else {
		f, err := os.Create(output)
		if err != nil {
			log.Error("Cannot create output file: "+err.Error(), output)
			return log, err
		}
		defer f.Close()
		ws = f
	}

	var w *nya.Writer
	if opts.Password != "" {
		w = nya.NewWriterOpts(ws, fec, level, opts.Solid, []byte(opts.Password))
	} else if opts.Solid {
		w = nya.NewWriterOpts(ws, fec, level, true)
	} else {
		w = nya.NewWriter(ws, fec, level)
	}

	for _, input := range opts.Inputs {
		if err := w.AddFile(input); err != nil {
			log.Error("Failed to add: "+err.Error(), input)
			
			return log, err
		}
		log.Info("Added", input)
	}
	w.Close()
	if c, ok := ws.(interface{Close()error}); ok {
		c.Close()
	}

	if opts.SFX {
		nya.CreateSFX(output, "")
		log.Info("Created SFX", output)
	}

	si, _ := os.Stat(output)
	if si != nil {
		log.Info(fmt.Sprintf("Archive size: %s", humanSize(si.Size())), output)
	}
	return log, nil
}

func doExtract(archivePath, destDir string) (*DiagLog, error) {
	log := NewDiagLog()
	if destDir == "" {
		destDir = filepath.Dir(archivePath)
	}
	log.Info("Extracting to: "+destDir, archivePath)
	r, err := nya.Open(archivePath)
	if err != nil {
		log.Error("Cannot open archive: "+err.Error(), archivePath)
		return log, err
	}
	for _, f := range r.List() {
		log.Info("Extracting", f.Path)
	}
	if err := r.Extract(destDir); err != nil {
		log.Error("Extract failed: "+err.Error(), archivePath)
		return log, err
	}
	log.Info(fmt.Sprintf("Extracted %d files", len(r.List())), destDir)
	return log, nil
}

func doTest(path string) (*DiagLog, int, bool, error) {
	log := NewDiagLog()
	r, err := nya.Open(path)
	if err != nil {
		log.Error("Cannot open: "+err.Error(), path)
		return log, 0, false, err
	}
	files := r.List()
	for _, f := range files {
		log.Info(fmt.Sprintf("Testing %s (%s)", f.Path, humanSize(int64(f.OriginalSize))), path)
	}
	ok := r.Verify()
	if ok {
		log.Info("All integrity checks passed", path)
	} else {
		log.Error("Archive integrity check FAILED", path)
	}
	return log, len(files), ok, nil
}

func doRepair(path string) (*DiagLog, int, int, int, error) {
	log := NewDiagLog()
	log.Info("Repairing archive", path)
	result, err := nya.Repair(path, "")
	if err != nil {
		log.Error("Repair failed: "+err.Error(), path)
		return log, 0, 0, 0, err
	}
	if result.CorruptedChunks == 0 {
		log.Info(fmt.Sprintf("%d chunks verified, no damage", result.TotalChunks), path)
	} else {
		log.Warn(fmt.Sprintf("%d damaged chunks found", result.CorruptedChunks), path)
		if result.RepairedChunks > 0 {
			log.Info(fmt.Sprintf("%d chunks repaired", result.RepairedChunks), path)
		}
		failed := result.CorruptedChunks - result.RepairedChunks
		if failed > 0 {
			log.Error(fmt.Sprintf("%d chunks could not be repaired", failed), path)
		}
	}
	return log, result.TotalChunks, result.CorruptedChunks, result.RepairedChunks, nil
}

func humanSize(b int64) string {
	return nya.HumanSize(int(b))
}

// copyFileOrDir copies a file or directory
func copyFileOrDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil { return err }
	if info.IsDir() {
		return filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
			if err != nil { return err }
			rel, _ := filepath.Rel(src, path)
			target := filepath.Join(dst, rel)
			if fi.IsDir() {
				return os.MkdirAll(target, 0755)
			}
			return copyFile(path, target)
		})
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil { return err }
	defer in.Close()
	os.MkdirAll(filepath.Dir(dst), 0755)
	out, err := os.Create(dst)
	if err != nil { return err }
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
