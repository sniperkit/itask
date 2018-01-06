package util

import (
	"os"
)

func EnsureDir(path string) {
	d, err := os.Open(path)
	if err != nil {
		os.MkdirAll(path, os.FileMode(0755))
	}
	d.Close()
}
