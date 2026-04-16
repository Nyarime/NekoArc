package main

import (
	"path/filepath"
	"strings"
)

// NavState represents the current navigation state
type NavState struct {
	// What we're looking at
	Mode        NavMode
	DirPath     string // filesystem directory path (empty = drive list)
	ArchivePath string // physical path to archive file
	SubDir      string // current subdirectory within archive (/ separated)

	// Display
	DisplayPath string // what to show in address bar
	Title       string // window title
}

type NavMode int

const (
	NavFilesystem NavMode = iota // browsing filesystem
	NavArchive                   // inside an archive
)

// NavStack manages navigation history
type NavStack struct {
	current NavState
	history []NavState
}

func NewNavStack(startDir string) *NavStack {
	return &NavStack{
		current: NavState{
			Mode:        NavFilesystem,
			DirPath:     startDir,
			DisplayPath: startDir,
			Title:       "NekoArc",
		},
	}
}

// EnterDir navigates into a filesystem directory
func (ns *NavStack) EnterDir(dir string) NavState {
	ns.pushHistory()
	ns.current = NavState{
		Mode:        NavFilesystem,
		DirPath:     dir,
		DisplayPath: dir,
		Title:       "NekoArc",
	}
	return ns.current
}

// EnterArchive opens an archive from filesystem
func (ns *NavStack) EnterArchive(archivePath string) NavState {
	ns.pushHistory()
	ns.current = NavState{
		Mode:        NavArchive,
		ArchivePath: archivePath,
		SubDir:      "",
		DisplayPath: archivePath,
		Title:       filepath.Base(archivePath) + " - NekoArc",
	}
	return ns.current
}

// EnterNestedArchive opens an archive inside another archive
func (ns *NavStack) EnterNestedArchive(physicalPath, logicalName string) NavState {
	ns.pushHistory()
	logicalPath := ns.current.DisplayPath + string(filepath.Separator) + logicalName
	ns.current = NavState{
		Mode:        NavArchive,
		ArchivePath: physicalPath,
		SubDir:      "",
		DisplayPath: logicalPath,
		Title:       logicalName + " - NekoArc",
	}
	return ns.current
}

// EnterSubDir navigates into a subdirectory within an archive
func (ns *NavStack) EnterSubDir(subDir string) NavState {
	ns.current.SubDir = subDir
	return ns.current
}

// GoUp navigates up one level
func (ns *NavStack) GoUp() NavState {
	if ns.current.Mode == NavArchive {
		if ns.current.SubDir != "" {
			// Go up within archive subdirectory
			parent := ""
			// Normalize to /
			sub := strings.ReplaceAll(ns.current.SubDir, "\\", "/")
			if idx := strings.LastIndex(sub, "/"); idx >= 0 {
				parent = sub[:idx]
			}
			ns.current.SubDir = parent
			return ns.current
		}
		// At archive root → go back to previous state
		if len(ns.history) > 0 {
			ns.current = ns.history[len(ns.history)-1]
			ns.history = ns.history[:len(ns.history)-1]
			return ns.current
		}
		// No history → go to archive's directory
		return ns.EnterDir(filepath.Dir(ns.current.ArchivePath))
	}

	// Filesystem
	if ns.current.DirPath == "" {
		return ns.current // already at root
	}
	parent := filepath.Dir(ns.current.DirPath)
	if parent == ns.current.DirPath {
		// At drive root → show drive list
		return ns.EnterDir("")
	}
	return ns.EnterDir(parent)
}

// GetDisplayPath returns what to show in address bar
func (ns *NavStack) GetDisplayPath() string {
	if ns.current.Mode == NavArchive && ns.current.SubDir != "" {
		return ns.current.DisplayPath + string(filepath.Separator) + ns.current.SubDir
	}
	return ns.current.DisplayPath
}

func (ns *NavStack) pushHistory() {
	ns.history = append(ns.history, ns.current)
	// Limit history size
	if len(ns.history) > 50 {
		ns.history = ns.history[1:]
	}
}
