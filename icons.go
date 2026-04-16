package main

import "github.com/lxn/walk"

// Shell32.dll icon indices for toolbar
var (
	iconAdd     *walk.Icon
	iconExtract *walk.Icon
	iconTest    *walk.Icon
	iconRepair  *walk.Icon
	iconInfo    *walk.Icon
	iconDelete  *walk.Icon
	iconUp      *walk.Icon
)

func initIcons() {
	iconAdd, _ = walk.NewIconFromSysDLL("shell32", 145)     // compressed folder
	iconExtract, _ = walk.NewIconFromSysDLL("shell32", 46)  // open folder
	iconTest, _ = walk.NewIconFromSysDLL("shell32", 23)     // search/verify
	iconRepair, _ = walk.NewIconFromSysDLL("shell32", 41)   // wrench/repair
	iconInfo, _ = walk.NewIconFromSysDLL("shell32", 221)    // info
	iconDelete, _ = walk.NewIconFromSysDLL("shell32", 131)  // recycle bin
	iconUp, _ = walk.NewIconFromSysDLL("shell32", 46)       // folder up
}
