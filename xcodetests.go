package main

import (
	"os"
	"path/filepath"
)

const derivedDataPathInHome = "Library/Developer/Xcode/DerivedData"

func getDerivedDataPath() string {
	return filepath.Join(os.Getenv("HOME"), derivedDataPathInHome)
}
