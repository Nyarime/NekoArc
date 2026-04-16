package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nyarime/nyarc/pkg/nya"
)

// archiveDeleteFiles removes files from a .nya archive
// by extracting, removing, and recompressing
func archiveDeleteFiles(archivePath string, filesToDelete []string) error {
	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "nekoarc-edit-*")
	if err != nil {
		return fmt.Errorf("cannot create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract all
	r, err := nya.Open(archivePath)
	if err != nil {
		return fmt.Errorf("cannot open archive: %w", err)
	}
	if err := r.Extract(tmpDir); err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}

	// Build delete set
	deleteSet := make(map[string]bool)
	for _, f := range filesToDelete {
		deleteSet[f] = true
	}

	// Remove files
	for _, f := range filesToDelete {
		target := filepath.Join(tmpDir, f)
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("cannot delete %s: %w", f, err)
		}
	}

	// Recompress to temp file
	tmpArchive := archivePath + ".tmp"
	f, err := os.Create(tmpArchive)
	if err != nil {
		return fmt.Errorf("cannot create temp archive: %w", err)
	}

	w := nya.NewWriter(f, 3, 9) // default FEC 3%, level 9

	// Walk temp dir and add all remaining files
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == tmpDir {
			return nil
		}
		return w.AddFile(path)
	})
	if err != nil {
		f.Close()
		os.Remove(tmpArchive)
		return fmt.Errorf("recompress failed: %w", err)
	}

	w.Close()
	f.Close()

	// Replace original
	if err := os.Rename(tmpArchive, archivePath); err != nil {
		os.Remove(tmpArchive)
		return fmt.Errorf("cannot replace archive: %w", err)
	}

	return nil
}

// archiveAddFiles adds files to a .nya archive
func archiveAddFiles(archivePath string, newFiles []string) error {
	tmpDir, err := os.MkdirTemp("", "nekoarc-edit-*")
	if err != nil {
		return fmt.Errorf("cannot create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract existing
	r, err := nya.Open(archivePath)
	if err != nil {
		return fmt.Errorf("cannot open archive: %w", err)
	}
	if err := r.Extract(tmpDir); err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}

	// Copy new files into temp dir
	for _, src := range newFiles {
		dst := filepath.Join(tmpDir, filepath.Base(src))
		if err := copyFileOrDir(src, dst); err != nil {
			return fmt.Errorf("cannot copy %s: %w", src, err)
		}
	}

	// Recompress
	tmpArchive := archivePath + ".tmp"
	f, err := os.Create(tmpArchive)
	if err != nil {
		return fmt.Errorf("cannot create temp archive: %w", err)
	}

	w := nya.NewWriter(f, 3, 9)
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == tmpDir {
			return err
		}
		return w.AddFile(path)
	})
	if err != nil {
		f.Close()
		os.Remove(tmpArchive)
		return err
	}
	w.Close()
	f.Close()

	return os.Rename(tmpArchive, archivePath)
}

// getArchiveRelPaths returns relative paths of selected items inside archive
func getArchiveRelPaths(items []FileEntry, indices []int) []string {
	var paths []string
	for _, i := range indices {
		if i >= 0 && i < len(items) && items[i].Name != ".." {
			paths = append(paths, items[i].Name)
		}
	}
	return paths
}
