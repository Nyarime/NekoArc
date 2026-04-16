package main

import (
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

var (
	shell32            = syscall.NewLazyDLL("shell32.dll")
	procSHGetFileInfoW = shell32.NewProc("SHGetFileInfoW")
	user32             = syscall.NewLazyDLL("user32.dll")
	procDestroyIcon    = user32.NewProc("DestroyIcon")
)

const (
	SHGFI_ICON            = 0x100
	SHGFI_SMALLICON       = 0x1
	SHGFI_USEFILEATTRIBUTES = 0x10
	SHGFI_SYSICONINDEX    = 0x4000
	FILE_ATTRIBUTE_DIRECTORY = 0x10
	FILE_ATTRIBUTE_NORMAL = 0x80
)

type SHFILEINFO struct {
	HIcon         win.HICON
	IIcon         int32
	DwAttributes  uint32
	SzDisplayName [260]uint16
	SzTypeName    [80]uint16
}

func shGetFileInfo(path string, attrs uint32, flags uint32) SHFILEINFO {
	var info SHFILEINFO
	pathPtr, _ := syscall.UTF16PtrFromString(path)
	procSHGetFileInfoW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(attrs),
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info),
		uintptr(flags),
	)
	return info
}

// GetFileIcon returns a walk.Icon for a given file path
func GetFileIcon(path string, isDir bool) *walk.Icon {
	var attrs uint32
	flags := uint32(SHGFI_ICON | SHGFI_SMALLICON)
	if isDir {
		flags |= SHGFI_USEFILEATTRIBUTES
		attrs = FILE_ATTRIBUTE_DIRECTORY
	} else {
		// Use extension only for speed (USEFILEATTRIBUTES doesn't need file to exist)
		flags |= SHGFI_USEFILEATTRIBUTES
		attrs = FILE_ATTRIBUTE_NORMAL
	}
	info := shGetFileInfo(path, attrs, flags)
	if info.HIcon != 0 {
		icon, err := walk.NewIconFromHICON(info.HIcon)
		if err == nil {
			return icon
		}
		procDestroyIcon.Call(uintptr(info.HIcon))
	}
	return nil
}

// IconCache caches icons by extension
type IconCache struct {
	cache     map[string]int // ext -> imageList index
	imageList *walk.ImageList
	folderIdx int
}

func NewIconCache() *IconCache {
	il, _ := walk.NewImageList(walk.Size{Width: 16, Height: 16}, 0)
	ic := &IconCache{
		cache:     make(map[string]int),
		imageList: il,
		folderIdx: -1,
	}
	// Pre-cache folder icon
	if icon := GetFileIcon("folder", true); icon != nil {
		idx, _ := il.AddIcon(icon)
		ic.folderIdx = idx
	}
	return ic
}

func (ic *IconCache) GetIndex(name string, isDir bool) int {
	if isDir || name == ".." {
		return ic.folderIdx
	}
	ext := ""
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			ext = name[i:]
			break
		}
	}
	if ext == "" {
		ext = ".file"
	}
	if idx, ok := ic.cache[ext]; ok {
		return idx
	}
	icon := GetFileIcon("dummy"+ext, false)
	if icon == nil {
		return -1
	}
	idx, err := ic.imageList.AddIcon(icon)
	if err != nil {
		return -1
	}
	ic.cache[ext] = idx
	return idx
}
